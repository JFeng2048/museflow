package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis 键前缀（验证码相关）。
const (
	// resetCodePrefix 密码重置验证码：pwd:reset:code:{email} -> 验证码
	resetCodePrefix = "pwd:reset:code:"
	// resetCodeLimitPrefix 发送频率限制：pwd:reset:limit:{email} -> 1
	resetCodeLimitPrefix = "pwd:reset:limit:"
)

// VerifyCodeStore 验证码存储接口。
//
// 验证码属于短时效临时数据，存 Redis 并依赖 TTL 自动过期，
// 不占用数据库表（与角色/权限等需要持久化的业务数据区分）。
type VerifyCodeStore interface {
	// SaveCode 保存验证码，ttl 为有效期
	SaveCode(ctx context.Context, scene, target, code string, ttl time.Duration) error
	// GetCode 读取验证码，不存在返回空字符串
	GetCode(ctx context.Context, scene, target string) (string, error)
	// DeleteCode 删除验证码（校验成功后调用，防止重复使用）
	DeleteCode(ctx context.Context, scene, target string) error
	// TryLockResend 防重发：在 cooldown 内重复请求返回 false
	TryLockResend(ctx context.Context, scene, target string, cooldown time.Duration) (bool, error)
}

type redisVerifyCodeStore struct {
	rdb *redis.Client
}

// NewVerifyCodeStore 构造基于 Redis 的验证码存储。
func NewVerifyCodeStore(rdb *redis.Client) VerifyCodeStore {
	return &redisVerifyCodeStore{rdb: rdb}
}

func keyFor(prefix, scene, target string) string {
	return fmt.Sprintf("%s%s:%s", prefix, scene, target)
}

func (s *redisVerifyCodeStore) SaveCode(ctx context.Context, scene, target, code string, ttl time.Duration) error {
	return s.rdb.Set(ctx, keyFor(resetCodePrefix, scene, target), code, ttl).Err()
}

func (s *redisVerifyCodeStore) GetCode(ctx context.Context, scene, target string) (string, error) {
	code, err := s.rdb.Get(ctx, keyFor(resetCodePrefix, scene, target)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return code, nil
}

func (s *redisVerifyCodeStore) DeleteCode(ctx context.Context, scene, target string) error {
	return s.rdb.Del(ctx, keyFor(resetCodePrefix, scene, target)).Err()
}

// TryLockResend 通过 SetNX 实现冷却期内的防重发。
func (s *redisVerifyCodeStore) TryLockResend(ctx context.Context, scene, target string, cooldown time.Duration) (bool, error) {
	return s.rdb.SetNX(ctx, keyFor(resetCodeLimitPrefix, scene, target), 1, cooldown).Result()
}
