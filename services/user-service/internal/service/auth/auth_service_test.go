package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/dto"
	"github.com/museflow/user-service/internal/service/token"
)

// ---- 内存版 UserRepository ----

type fakeUserRepo struct {
	byEmail map[string]*model.User
	byUUID  map[string]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byEmail: map[string]*model.User{},
		byUUID:  map[string]*model.User{},
	}
}

func (r *fakeUserRepo) Create(_ context.Context, u *model.User) error {
	r.byEmail[u.Email] = u
	r.byUUID[u.UUID.String()] = u
	return nil
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, email string) (*model.User, error) {
	if u, ok := r.byEmail[email]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *fakeUserRepo) FindByUUID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if u, ok := r.byUUID[id.String()]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *fakeUserRepo) ExistsByEmail(_ context.Context, email string) (bool, error) {
	_, ok := r.byEmail[email]
	return ok, nil
}

func (r *fakeUserRepo) UpdateLoginInfo(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}

// ---- 内存版 TokenStore ----

type fakeTokenStore struct {
	refreshValid map[string]bool
	blacklist    map[string]bool
	userTokens   map[string][]repository.TokenMeta
}

func newFakeTokenStore() *fakeTokenStore {
	return &fakeTokenStore{
		refreshValid: map[string]bool{},
		blacklist:    map[string]bool{},
		userTokens:   map[string][]repository.TokenMeta{},
	}
}

func (s *fakeTokenStore) SetRefreshValid(_ context.Context, tokenID string, _ time.Duration) error {
	s.refreshValid[tokenID] = true
	return nil
}

func (s *fakeTokenStore) IsRefreshValid(_ context.Context, tokenID string) (bool, error) {
	return s.refreshValid[tokenID], nil
}

func (s *fakeTokenStore) DeleteRefreshValid(_ context.Context, tokenID string) error {
	delete(s.refreshValid, tokenID)
	return nil
}

func (s *fakeTokenStore) BlacklistAccess(_ context.Context, jti string, ttl time.Duration) error {
	if ttl > 0 {
		s.blacklist[jti] = true
	}
	return nil
}

func (s *fakeTokenStore) IsAccessBlacklisted(_ context.Context, jti string) (bool, error) {
	return s.blacklist[jti], nil
}

func (s *fakeTokenStore) AppendUserToken(_ context.Context, userID string, meta repository.TokenMeta, _ time.Duration) error {
	s.userTokens[userID] = append(s.userTokens[userID], meta)
	return nil
}

func (s *fakeTokenStore) RemoveUserToken(_ context.Context, userID, tokenID string) error {
	kept := make([]repository.TokenMeta, 0)
	for _, m := range s.userTokens[userID] {
		if m.TokenID != tokenID {
			kept = append(kept, m)
		}
	}
	s.userTokens[userID] = kept
	return nil
}

func (s *fakeTokenStore) TouchUserToken(_ context.Context, _, _ string, _ time.Duration) error {
	return nil
}

// ---- 测试辅助 ----

func newTestService() (*AuthService, *fakeTokenStore) {
	store := newFakeTokenStore()
	tm := token.NewTokenManager("test-secret", time.Hour, 30*24*time.Hour)
	// bcrypt 使用最小成本加速测试
	svc := NewAuthService(newFakeUserRepo(), store, tm, bcrypt.MinCost)
	return svc, store
}

func testDevice() dto.Device {
	return dto.Device{DeviceID: "device-1", UserAgent: "Go-Test", IP: "127.0.0.1", DeviceName: "test"}
}

// ---- 用例 ----

func TestRegisterHashesPasswordAndRejectsDuplicate(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	u, err := svc.Register(ctx, "Author@MuseFlow.ai", "P@ssw0rd123", "")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 邮箱应被规范化为小写
	if u.Email != "author@museflow.ai" {
		t.Errorf("邮箱未规范化: %s", u.Email)
	}
	// 昵称应由邮箱前缀兜底
	if u.Nickname != "author" {
		t.Errorf("昵称兜底错误: %s", u.Nickname)
	}
	// 密码必须是 bcrypt 哈希而非明文
	if u.PasswordHash == nil || *u.PasswordHash == "P@ssw0rd123" {
		t.Fatal("密码未被加密存储")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte("P@ssw0rd123")); err != nil {
		t.Errorf("bcrypt 哈希无法校验: %v", err)
	}

	// 大小写不同的同一邮箱应判定为重复
	if _, err := svc.Register(ctx, "author@museflow.ai", "another", ""); !errors.Is(err, ErrEmailExists) {
		t.Errorf("期望 ErrEmailExists，实际: %v", err)
	}
}

func TestLoginWrongPasswordAndUnknownEmailReturnSameError(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "a@b.com", "correct-password", "n"); err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	_, _, errWrong := svc.Login(ctx, "a@b.com", "wrong-password", testDevice())
	_, _, errUnknown := svc.Login(ctx, "nobody@b.com", "whatever", testDevice())

	// 两者必须返回相同错误，避免邮箱枚举
	if !errors.Is(errWrong, ErrInvalidCredentials) || !errors.Is(errUnknown, ErrInvalidCredentials) {
		t.Errorf("期望统一的 ErrInvalidCredentials，实际: %v / %v", errWrong, errUnknown)
	}
}

func TestLoginIssuesUsableTokenPair(t *testing.T) {
	svc, store := newTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "a@b.com", "pw12345678", "n"); err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	pair, u, err := svc.Login(ctx, "a@b.com", "pw12345678", testDevice())
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// access token 应能通过校验并解析出用户 uuid
	uid, err := svc.ValidateAccess(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("access token 校验失败: %v", err)
	}
	if uid != u.UUID.String() {
		t.Errorf("uuid 不匹配: %s != %s", uid, u.UUID)
	}
	// refresh token 必须已写入白名单
	if len(store.refreshValid) != 1 {
		t.Errorf("refresh 白名单数量异常: %d", len(store.refreshValid))
	}
	if pair.ExpiresIn != 3600 {
		t.Errorf("access 有效期错误: %d", pair.ExpiresIn)
	}
}

func TestRefreshRejectsMismatchedDeviceAndAccessTokenMisuse(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "a@b.com", "pw12345678", "n"); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	pair, _, err := svc.Login(ctx, "a@b.com", "pw12345678", testDevice())
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 正常刷新应成功
	if _, _, err := svc.Refresh(ctx, pair.RefreshToken, testDevice()); err != nil {
		t.Fatalf("正常刷新失败: %v", err)
	}

	// 设备 ID 不一致 -> 拒绝
	other := testDevice()
	other.DeviceID = "device-2"
	if _, _, err := svc.Refresh(ctx, pair.RefreshToken, other); !errors.Is(err, ErrDeviceMismatch) {
		t.Errorf("设备 ID 不匹配未被拒绝: %v", err)
	}

	// 指纹变化（IP 改变）-> 拒绝
	movedIP := testDevice()
	movedIP.IP = "10.0.0.9"
	if _, _, err := svc.Refresh(ctx, pair.RefreshToken, movedIP); !errors.Is(err, ErrDeviceMismatch) {
		t.Errorf("设备指纹不匹配未被拒绝: %v", err)
	}

	// access token 不能当作 refresh token 使用
	if _, _, err := svc.Refresh(ctx, pair.AccessToken, testDevice()); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("令牌类型混用未被拒绝: %v", err)
	}
}

func TestLogoutBlacklistsAccessAndRevokesRefresh(t *testing.T) {
	svc, store := newTestService()
	ctx := context.Background()

	if _, err := svc.Register(ctx, "a@b.com", "pw12345678", "n"); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	pair, u, err := svc.Login(ctx, "a@b.com", "pw12345678", testDevice())
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	if err := svc.Logout(ctx, pair.AccessToken, pair.RefreshToken); err != nil {
		t.Fatalf("登出失败: %v", err)
	}

	// 关键断言：登出后 access token 必须立即失效（黑名单生效）
	if _, err := svc.ValidateAccess(ctx, pair.AccessToken); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("登出后 access token 仍然有效: %v", err)
	}
	// refresh token 白名单应被删除，无法再换取新令牌
	if _, _, err := svc.Refresh(ctx, pair.RefreshToken, testDevice()); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("登出后仍可刷新: %v", err)
	}
	// 设备会话列表应被清空
	if len(store.userTokens[u.UUID.String()]) != 0 {
		t.Errorf("设备会话未被移除: %d", len(store.userTokens[u.UUID.String()]))
	}
}

func TestValidateAccessRejectsTamperedToken(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	// 用不同密钥签发的令牌不应通过校验
	forged := token.NewTokenManager("attacker-secret", time.Hour, time.Hour)
	tokenStr, err := forged.GenerateAccess(uuid.NewString(), uuid.NewString())
	if err != nil {
		t.Fatalf("生成伪造令牌失败: %v", err)
	}

	if _, err := svc.ValidateAccess(ctx, tokenStr); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("伪造签名令牌未被拒绝: %v", err)
	}
}

func TestExpiredAccessTokenIsRejected(t *testing.T) {
	store := newFakeTokenStore()
	// 负有效期立即过期
	tm := token.NewTokenManager("test-secret", -time.Minute, time.Hour)
	svc := NewAuthService(newFakeUserRepo(), store, tm, bcrypt.MinCost)

	tokenStr, err := tm.GenerateAccess(uuid.NewString(), uuid.NewString())
	if err != nil {
		t.Fatalf("生成令牌失败: %v", err)
	}

	if _, err := svc.ValidateAccess(context.Background(), tokenStr); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("过期令牌未被拒绝: %v", err)
	}
}
