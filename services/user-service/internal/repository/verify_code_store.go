package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis 键前缀（验证码相关）。按场景区分，避免不同用途的验证码互相覆盖。
const (
	// resetCodePrefix 密码重置验证码：pwd:reset:code:{email} -> 验证码
	resetCodePrefix = "pwd:reset:code:"
	// registerCodePrefix 注册邮箱校验验证码：email:verify:code:{email} -> 验证码
	registerCodePrefix = "email:verify:code:"
	// loginCodePrefix 验证码登录验证码：email:login:code:{email} -> 验证码
	loginCodePrefix = "email:login:code:"
	// changeEmailCodePrefix 修改邮箱验证码：email:change:code:{email} -> 验证码
	changeEmailCodePrefix = "email:change:code:"
	// resetCodeLimitPrefix 发送频率限制：pwd:reset:limit:{email} -> 1
	resetCodeLimitPrefix = "pwd:reset:limit:"
	// registerCodeLimitPrefix 注册验证码发送频率限制
	registerCodeLimitPrefix = "email:verify:limit:"
	// loginCodeLimitPrefix 验证码登录发送频率限制
	loginCodeLimitPrefix = "email:login:limit:"
	// changeEmailLimitPrefix 修改邮箱验证码发送频率限制
	changeEmailLimitPrefix = "email:change:limit:"
)

// codePrefixForScene 根据场景返回验证码键前缀。
func codePrefixForScene(scene string) string {
	switch scene {
	case "register":
		return registerCodePrefix
	case "login":
		return loginCodePrefix
	case "change_email":
		return changeEmailCodePrefix
	default:
		return resetCodePrefix
	}
}

// limitPrefixForScene 根据场景返回防重发键前缀。
func limitPrefixForScene(scene string) string {
	switch scene {
	case "register":
		return registerCodeLimitPrefix
	case "login":
		return loginCodeLimitPrefix
	case "change_email":
		return changeEmailLimitPrefix
	default:
		return resetCodeLimitPrefix
	}
}

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
	// UnlockResend 释放防重发锁：入队等下游步骤失败时调用，
	// 避免用户被无谓的冷却期挡住而无法重试。
	UnlockResend(ctx context.Context, scene, target string) error
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
	return s.rdb.Set(ctx, keyFor(codePrefixForScene(scene), scene, target), code, ttl).Err()
}

func (s *redisVerifyCodeStore) GetCode(ctx context.Context, scene, target string) (string, error) {
	code, err := s.rdb.Get(ctx, keyFor(codePrefixForScene(scene), scene, target)).Result()
	if err == redis.Nil {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return code, nil
}

func (s *redisVerifyCodeStore) DeleteCode(ctx context.Context, scene, target string) error {
	return s.rdb.Del(ctx, keyFor(codePrefixForScene(scene), scene, target)).Err()
}

// TryLockResend 通过 SetNX 实现冷却期内的防重发。
func (s *redisVerifyCodeStore) TryLockResend(ctx context.Context, scene, target string, cooldown time.Duration) (bool, error) {
	return s.rdb.SetNX(ctx, keyFor(limitPrefixForScene(scene), scene, target), 1, cooldown).Result()
}

// UnlockResend 删除防重发锁键。
func (s *redisVerifyCodeStore) UnlockResend(ctx context.Context, scene, target string) error {
	return s.rdb.Del(ctx, keyFor(limitPrefixForScene(scene), scene, target)).Err()
}
