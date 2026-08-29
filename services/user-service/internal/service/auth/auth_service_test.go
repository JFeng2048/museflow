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

func (r *fakeUserRepo) IncrementLoginFails(_ context.Context, email string) (*model.User, error) {
	u, ok := r.byEmail[email]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	u.LoginFailCount++
	// 与真实仓储一致：达到 5 次即锁定 15 分钟
	if u.LoginFailCount >= 5 {
		lu := time.Now().Add(15 * time.Minute)
		u.LockedUntil = &lu
	}
	return u, nil
}

func (r *fakeUserRepo) ResetLoginFails(_ context.Context, id uuid.UUID) error {
	if u, ok := r.byUUID[id.String()]; ok {
		u.LoginFailCount = 0
		u.LockedUntil = nil
	}
	return nil
}

func (r *fakeUserRepo) UpdateProfile(_ context.Context, id uuid.UUID, nickname, avatarURL, bio string) error {
	u, ok := r.byUUID[id.String()]
	if !ok {
		return repository.ErrUserNotFound
	}
	if nickname != "" {
		u.Nickname = nickname
	}
	if avatarURL != "" {
		u.AvatarURL = &avatarURL
	}
	if bio != "" {
		u.Bio = &bio
	}
	return nil
}

func (r *fakeUserRepo) UpdatePasswordHash(_ context.Context, id uuid.UUID, hash string) error {
	u, ok := r.byUUID[id.String()]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.PasswordHash = &hash
	return nil
}

func (r *fakeUserRepo) UpdateStatus(_ context.Context, id uuid.UUID, status int16) error {
	u, ok := r.byUUID[id.String()]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.Status = status
	return nil
}

func (r *fakeUserRepo) ListUsers(_ context.Context, _ string, _ int16, _ string, _ bool, _, _ int) ([]model.User, error) {
	out := make([]model.User, 0, len(r.byUUID))
	for _, u := range r.byUUID {
		out = append(out, *u)
	}
	return out, nil
}

func (r *fakeUserRepo) CountUsers(_ context.Context, _ string, _ int16) (int64, error) {
	return int64(len(r.byUUID)), nil
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

func (s *fakeTokenStore) GetUserPermissions(_ context.Context, userID string) ([]string, error) {
	return nil, nil
}

func (s *fakeTokenStore) SetUserPermissions(_ context.Context, userID string, _ []string, _ time.Duration) error {
	return nil
}

func (s *fakeTokenStore) ClearUserPermissions(_ context.Context, userID string) error {
	return nil
}

func (s *fakeTokenStore) ListUserTokens(_ context.Context, userID string) ([]repository.TokenMeta, error) {
	return s.userTokens[userID], nil
}

// ---- 测试辅助 ----

func newTestService() (*AuthService, *fakeTokenStore) {
	store := newFakeTokenStore()
	tm := token.NewTokenManager("test-secret", time.Hour, 30*24*time.Hour)
	// bcrypt 使用最小成本加速测试
	svc := NewAuthService(newFakeUserRepo(), store, tm, nil, bcrypt.MinCost)
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
	svc := NewAuthService(newFakeUserRepo(), store, tm, nil, bcrypt.MinCost)

	tokenStr, err := tm.GenerateAccess(uuid.NewString(), uuid.NewString())
	if err != nil {
		t.Fatalf("生成令牌失败: %v", err)
	}

	if _, err := svc.ValidateAccess(context.Background(), tokenStr); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("过期令牌未被拒绝: %v", err)
	}
}

func TestUpdateProfileChangesProvidedFieldsOnly(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	u, err := svc.Register(ctx, "profile@museflow.ai", "pw12345678", "原始昵称")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 只传昵称，头像与简介应保持为空（不覆盖）
	got, err := svc.UpdateProfile(ctx, u.UUID.String(), "新昵称", "", "")
	if err != nil {
		t.Fatalf("更新资料失败: %v", err)
	}
	if got.Nickname != "新昵称" {
		t.Errorf("昵称未更新: %s", got.Nickname)
	}
	if got.AvatarURL != nil {
		t.Errorf("头像不应被空值覆盖: %v", got.AvatarURL)
	}
}

func TestChangePasswordRequiresCorrectOldPassword(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	u, err := svc.Register(ctx, "pwd@museflow.ai", "oldpass1234", "n")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 旧密码错误 -> 拒绝
	if err := svc.ChangePassword(ctx, u.UUID.String(), "wrong-old", "newpass5678"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("旧密码错误未被拒绝: %v", err)
	}

	// 旧密码正确 -> 修改成功，新密码可登录
	if err := svc.ChangePassword(ctx, u.UUID.String(), "oldpass1234", "newpass5678"); err != nil {
		t.Fatalf("修改密码失败: %v", err)
	}
	if _, _, err := svc.Login(ctx, "pwd@museflow.ai", "newpass5678", testDevice()); err != nil {
		t.Errorf("新密码无法登录: %v", err)
	}
	// 旧密码应失效
	if _, _, err := svc.Login(ctx, "pwd@museflow.ai", "oldpass1234", testDevice()); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("旧密码仍可登录: %v", err)
	}
}

func TestLoginLocksAccountAfterMaxFailures(t *testing.T) {
	repo := newFakeUserRepo()
	store := newFakeTokenStore()
	tm := token.NewTokenManager("test-secret", time.Hour, time.Hour)
	svc := NewAuthService(repo, store, tm, nil, bcrypt.MinCost)
	ctx := context.Background()

	if _, err := svc.Register(ctx, "lock@museflow.ai", "pw12345678", "n"); err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 连续失败（repository 层阈值为 5 次）
	for i := 0; i < 5; i++ {
		if _, _, err := svc.Login(ctx, "lock@museflow.ai", "bad-password", testDevice()); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("第 %d 次登录失败应返回凭证错误，实际: %v", i+1, err)
		}
	}

	// 达到阈值后锁定：即使密码正确也应拒绝
	if _, _, err := svc.Login(ctx, "lock@museflow.ai", "pw12345678", testDevice()); err == nil {
		t.Errorf("账号锁定后仍可登录")
	}
}

func TestListSessionsReturnsActiveDevices(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	u, err := svc.Register(ctx, "sess@museflow.ai", "pw12345678", "n")
	if err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	if _, _, err := svc.Login(ctx, "sess@museflow.ai", "pw12345678", testDevice()); err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	sessions, err := svc.ListSessions(ctx, u.UUID.String())
	if err != nil {
		t.Fatalf("查询会话失败: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("期望 1 个会话，实际 %d", len(sessions))
	}
	if sessions[0].DeviceID != testDevice().DeviceID {
		t.Errorf("会话设备 ID 不匹配: %s", sessions[0].DeviceID)
	}

	// 吊销后可再次登录（会话清空）
	if err := svc.RevokeSession(ctx, u.UUID.String(), sessions[0].TokenID); err != nil {
		t.Fatalf("吊销会话失败: %v", err)
	}
	left, _ := svc.ListSessions(ctx, u.UUID.String())
	if len(left) != 0 {
		t.Errorf("吊销后仍残留会话: %d", len(left))
	}
}
