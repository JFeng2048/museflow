package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis 键前缀。
const (
	// refreshValidPrefix refresh token 白名单：存在即有效，删除即失效。
	refreshValidPrefix = "auth:refresh:valid:"
	// accessBlacklistPrefix access token 黑名单：登出后在 token 自然过期前拦截。
	accessBlacklistPrefix = "auth:access:blacklist:"
	// userTokensPrefix 用户 token 元数据列表，用于设备管理与批量吊销。
	userTokensPrefix = "auth:user:tokens:"
	// permissionCachePrefix 用户权限缓存：perm:user:{userUUID} -> "perm1,perm2"
	permissionCachePrefix = "perm:user:"
)

// TokenMeta 单个登录会话（设备）的元数据。
type TokenMeta struct {
	TokenID         string    `json:"tokenId"`
	DeviceID        string    `json:"deviceId"`
	DeviceName      string    `json:"deviceName"`
	LoginTime       time.Time `json:"loginTime"`
	LastRefreshTime time.Time `json:"lastRefreshTime"`
}

// TokenStore 双令牌状态存储接口。
type TokenStore interface {
	// SetRefreshValid 写入 refresh token 白名单标记
	SetRefreshValid(ctx context.Context, tokenID string, ttl time.Duration) error
	// IsRefreshValid 判断 refresh token 是否仍在白名单中
	IsRefreshValid(ctx context.Context, tokenID string) (bool, error)
	// DeleteRefreshValid 删除白名单标记（登出/吊销）
	DeleteRefreshValid(ctx context.Context, tokenID string) error

	// BlacklistAccess 将 access token 加入黑名单，TTL 为其剩余有效期
	BlacklistAccess(ctx context.Context, jti string, ttl time.Duration) error
	// IsAccessBlacklisted 判断 access token 是否已被拉黑
	IsAccessBlacklisted(ctx context.Context, jti string) (bool, error)

	// AppendUserToken 追加设备元数据到用户会话列表
	AppendUserToken(ctx context.Context, userID string, meta TokenMeta, ttl time.Duration) error
	// RemoveUserToken 从用户会话列表移除指定 tokenID
	RemoveUserToken(ctx context.Context, userID, tokenID string) error
	// TouchUserToken 更新指定 tokenID 的最后刷新时间
	TouchUserToken(ctx context.Context, userID, tokenID string, ttl time.Duration) error
	// ListUserTokens 返回用户当前活跃会话列表
	ListUserTokens(ctx context.Context, userID string) ([]TokenMeta, error)

	// GetUserPermissions 读取用户权限缓存（perm:user:{userID}），未命中返回空切片。
	// 注意：缓存不可用（Redis 故障）时返回 (nil, nil) 交由上层降级查库。
	GetUserPermissions(ctx context.Context, userID string) ([]string, error)
	// SetUserPermissions 写入用户权限缓存，perm 为权限编码列表（逗号分隔存储）。
	SetUserPermissions(ctx context.Context, userID string, perms []string, ttl time.Duration) error
	// ClearUserPermissions 删除用户权限缓存（权限变更时调用）。
	ClearUserPermissions(ctx context.Context, userID string) error
}

type redisTokenStore struct {
	rdb *redis.Client
}

// NewTokenStore 构造基于 Redis 的令牌状态存储。
func NewTokenStore(rdb *redis.Client) TokenStore {
	return &redisTokenStore{rdb: rdb}
}

func (s *redisTokenStore) SetRefreshValid(ctx context.Context, tokenID string, ttl time.Duration) error {
	return s.rdb.Set(ctx, refreshValidPrefix+tokenID, "active", ttl).Err()
}

func (s *redisTokenStore) IsRefreshValid(ctx context.Context, tokenID string) (bool, error) {
	n, err := s.rdb.Exists(ctx, refreshValidPrefix+tokenID).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *redisTokenStore) DeleteRefreshValid(ctx context.Context, tokenID string) error {
	return s.rdb.Del(ctx, refreshValidPrefix+tokenID).Err()
}

// BlacklistAccess ttl <= 0 表示 token 已自然过期，无需入黑名单。
func (s *redisTokenStore) BlacklistAccess(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	return s.rdb.Set(ctx, accessBlacklistPrefix+jti, "revoked", ttl).Err()
}

func (s *redisTokenStore) IsAccessBlacklisted(ctx context.Context, jti string) (bool, error) {
	n, err := s.rdb.Exists(ctx, accessBlacklistPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *redisTokenStore) AppendUserToken(ctx context.Context, userID string, meta TokenMeta, ttl time.Duration) error {
	metas, err := s.loadUserTokens(ctx, userID)
	if err != nil {
		return err
	}

	// 同一 tokenID 视为覆盖，避免重复
	filtered := make([]TokenMeta, 0, len(metas)+1)
	for _, m := range metas {
		if m.TokenID != meta.TokenID {
			filtered = append(filtered, m)
		}
	}
	filtered = append(filtered, meta)

	return s.saveUserTokens(ctx, userID, filtered, ttl)
}

func (s *redisTokenStore) RemoveUserToken(ctx context.Context, userID, tokenID string) error {
	metas, err := s.loadUserTokens(ctx, userID)
	if err != nil {
		return err
	}

	filtered := make([]TokenMeta, 0, len(metas))
	for _, m := range metas {
		if m.TokenID != tokenID {
			filtered = append(filtered, m)
		}
	}

	key := userTokensPrefix + userID
	if len(filtered) == 0 {
		return s.rdb.Del(ctx, key).Err()
	}

	// 保留原有 TTL，避免登出一个设备后整个列表提前过期
	ttl, err := s.rdb.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	return s.saveUserTokens(ctx, userID, filtered, ttl)
}

func (s *redisTokenStore) TouchUserToken(ctx context.Context, userID, tokenID string, ttl time.Duration) error {
	metas, err := s.loadUserTokens(ctx, userID)
	if err != nil {
		return err
	}

	found := false
	for i := range metas {
		if metas[i].TokenID == tokenID {
			metas[i].LastRefreshTime = time.Now()
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	return s.saveUserTokens(ctx, userID, metas, ttl)
}

// ListUserTokens 返回用户当前活跃会话列表。
func (s *redisTokenStore) ListUserTokens(ctx context.Context, userID string) ([]TokenMeta, error) {
	return s.loadUserTokens(ctx, userID)
}

func (s *redisTokenStore) loadUserTokens(ctx context.Context, userID string) ([]TokenMeta, error) {
	raw, err := s.rdb.Get(ctx, userTokensPrefix+userID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var metas []TokenMeta
	if err := json.Unmarshal(raw, &metas); err != nil {
		// 数据损坏时按空列表处理，避免阻塞登录
		return nil, nil
	}
	return metas, nil
}

func (s *redisTokenStore) saveUserTokens(ctx context.Context, userID string, metas []TokenMeta, ttl time.Duration) error {
	raw, err := json.Marshal(metas)
	if err != nil {
		return fmt.Errorf("序列化用户会话列表失败: %w", err)
	}
	return s.rdb.Set(ctx, userTokensPrefix+userID, raw, ttl).Err()
}

// GetUserPermissions 读取用户权限缓存；未命中或 Redis 故障时返回空切片（降级查库）。
func (s *redisTokenStore) GetUserPermissions(ctx context.Context, userID string) ([]string, error) {
	raw, err := s.rdb.Get(ctx, permissionCachePrefix+userID).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		// Redis 不可用时降级到数据库查询，不返回错误阻断主流程
		return nil, nil
	}
	if raw == "" {
		return nil, nil
	}
	return splitPerms(raw), nil
}

// SetUserPermissions 写入用户权限缓存（权限编码用逗号拼接存储）。
func (s *redisTokenStore) SetUserPermissions(ctx context.Context, userID string, perms []string, ttl time.Duration) error {
	if len(perms) == 0 {
		// 无权限时写入空串，避免缓存穿透反复查库
		return s.rdb.Set(ctx, permissionCachePrefix+userID, "", ttl).Err()
	}
	return s.rdb.Set(ctx, permissionCachePrefix+userID, joinPerms(perms), ttl).Err()
}

// ClearUserPermissions 删除用户权限缓存（权限变更时调用）。
// 删除失败（Redis 故障）返回 nil，由调用方按"不阻断主流程"处理。
func (s *redisTokenStore) ClearUserPermissions(ctx context.Context, userID string) error {
	return s.rdb.Del(ctx, permissionCachePrefix+userID).Err()
}

func joinPerms(perms []string) string {
	out := ""
	for i, p := range perms {
		if i > 0 {
			out += ","
		}
		out += p
	}
	return out
}

func splitPerms(raw string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(raw); i++ {
		if raw[i] == ',' {
			if seg := raw[start:i]; seg != "" {
				parts = append(parts, seg)
			}
			start = i + 1
		}
	}
	if seg := raw[start:]; seg != "" {
		parts = append(parts, seg)
	}
	return parts
}
