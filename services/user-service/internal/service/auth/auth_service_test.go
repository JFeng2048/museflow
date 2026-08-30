package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/pkg/turnstile"
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

func (r *fakeUserRepo) SaveMFASecret(_ context.Context, id uuid.UUID, secret string) error {
	u, ok := r.byUUID[id.String()]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.MFASecret = &secret
	return nil
}

func (r *fakeUserRepo) EnableMFA(_ context.Context, id uuid.UUID, codes []string) error {
	u, ok := r.byUUID[id.String()]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.MFAEnabled = true
	u.MFARecoveryCodes = codes
	return nil
}

func (r *fakeUserRepo) DisableMFA(_ context.Context, id uuid.UUID) error {
	u, ok := r.byUUID[id.String()]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.MFAEnabled = false
	u.MFASecret = nil
	u.MFARecoveryCodes = nil
	return nil
}

func (r *fakeUserRepo) UpdateRecoveryCodes(_ context.Context, id uuid.UUID, codes []string) error {
	u, ok := r.byUUID[id.String()]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.MFARecoveryCodes = codes
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

func (r *fakeUserRepo) SetEmailVerified(_ context.Context, id uuid.UUID, verified bool) error {
	u, ok := r.byUUID[id.String()]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.EmailVerified = verified
	return nil
}

func (r *fakeUserRepo) UpdateEmail(_ context.Context, id uuid.UUID, email string) error {
	u, ok := r.byUUID[id.String()]
	if !ok {
		return repository.ErrUserNotFound
	}
	u.Email = email
	u.EmailVerified = true
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

func newTestService() (*AuthService, *fakeTokenStore, *fakeCodeStore) {
	store := newFakeTokenStore()
	tm := token.NewTokenManager("test-secret", time.Hour, 30*24*time.Hour, 5*time.Minute)
	codes := newFakeCodeStore()
	// bcrypt 使用最小成本加速测试（rbac / audit / oauth 传 nil：测试聚焦认证主流程）
	svc := NewAuthService(newFakeUserRepo(), store, tm, nil, nil, nil, codes, newFakeQueue(), &stubCaptcha{}, testResetConfig(), testEmailCodeConfig(), testMFAConfig(), bcrypt.MinCost)
	return svc, store, codes
}

// testRegisterCode 注册用例使用的固定验证码（预置到验证码存储）。
const testRegisterCode = "123456"

// registerOK 通过预置邮箱验证码完成注册，返回创建的用户。
func registerOK(t *testing.T, svc *AuthService, codes *fakeCodeStore, email, password, nickname string) *model.User {
	t.Helper()
	// Register 内部会规范化邮箱（小写），验证码需以规范化后的邮箱预置
	key := normalizeEmail(email)
	codes.codes["register:"+key] = testRegisterCode
	u, err := svc.Register(context.Background(), email, password, nickname, testRegisterCode)
	if err != nil {
		t.Fatalf("注册失败 %s: %v", email, err)
	}
	return u
}

func testDevice() dto.Device {
	return dto.Device{DeviceID: "device-1", UserAgent: "Go-Test", IP: "127.0.0.1", DeviceName: "test"}
}

// ---- 用例 ----

func TestRegisterHashesPasswordAndRejectsDuplicate(t *testing.T) {
	svc, _, codes := newTestService()

	u := registerOK(t, svc, codes, "Author@MuseFlow.ai", "P@ssw0rd123", "")
	ctx := context.Background()

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

	// 大小写不同的同一邮箱应判定为重复（需重新预置验证码，因为首次注册已消费）
	codes.codes["register:author@museflow.ai"] = testRegisterCode
	if _, err := svc.Register(ctx, "author@museflow.ai", "another", "", testRegisterCode); !errors.Is(err, ErrEmailExists) {
		t.Errorf("期望 ErrEmailExists，实际: %v", err)
	}
}

// registerWithCode 预置邮箱验证码后注册（忽略返回的用户）。
func registerWithCode(t *testing.T, svc *AuthService, codes *fakeCodeStore, email, password, nickname string) {
	t.Helper()
	registerOK(t, svc, codes, email, password, nickname)
}

func TestLoginWrongPasswordAndUnknownEmailReturnSameError(t *testing.T) {
	svc, _, codes := newTestService()
	ctx := context.Background()

	registerWithCode(t, svc, codes, "a@b.com", "correct-password", "n")

	_, errWrong := svc.Login(ctx, "a@b.com", "wrong-password", testDevice())
	_, errUnknown := svc.Login(ctx, "nobody@b.com", "whatever", testDevice())

	// 两者必须返回相同错误，避免邮箱枚举
	if !errors.Is(errWrong, ErrInvalidCredentials) || !errors.Is(errUnknown, ErrInvalidCredentials) {
		t.Errorf("期望统一的 ErrInvalidCredentials，实际: %v / %v", errWrong, errUnknown)
	}
}

func TestLoginIssuesUsableTokenPair(t *testing.T) {
	svc, store, codes := newTestService()
	ctx := context.Background()

	registerWithCode(t, svc, codes, "a@b.com", "pw12345678", "n")

	res, err := svc.Login(ctx, "a@b.com", "pw12345678", testDevice())
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	if res.RequiresMFA {
		t.Fatal("未开启 2FA 时不应要求二次验证")
	}
	pair := res.TokenPair

	// access token 应能通过校验并解析出用户 uuid
	uid, err := svc.ValidateAccess(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("access token 校验失败: %v", err)
	}
	if uid != res.User.UUID.String() {
		t.Errorf("uuid 不匹配: %s != %s", uid, res.User.UUID)
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
	svc, _, codes := newTestService()
	ctx := context.Background()

	registerWithCode(t, svc, codes, "a@b.com", "pw12345678", "n")
	res, err := svc.Login(ctx, "a@b.com", "pw12345678", testDevice())
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	pair := res.TokenPair

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
	svc, store, codes := newTestService()
	ctx := context.Background()

	registerWithCode(t, svc, codes, "a@b.com", "pw12345678", "n")
	res, err := svc.Login(ctx, "a@b.com", "pw12345678", testDevice())
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}
	pair := res.TokenPair
	u := res.User

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
	svc, _, _ := newTestService()
	ctx := context.Background()

	// 用不同密钥签发的令牌不应通过校验
	forged := token.NewTokenManager("attacker-secret", time.Hour, time.Hour, 5*time.Minute)
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
	tm := token.NewTokenManager("test-secret", -time.Minute, time.Hour, 5*time.Minute)
	svc := NewAuthService(newFakeUserRepo(), store, tm, nil, nil, nil, newFakeCodeStore(), newFakeQueue(), &stubCaptcha{}, testResetConfig(), testEmailCodeConfig(), testMFAConfig(), bcrypt.MinCost)

	tokenStr, err := tm.GenerateAccess(uuid.NewString(), uuid.NewString())
	if err != nil {
		t.Fatalf("生成令牌失败: %v", err)
	}

	if _, err := svc.ValidateAccess(context.Background(), tokenStr); !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("过期令牌未被拒绝: %v", err)
	}
}

func TestUpdateProfileChangesProvidedFieldsOnly(t *testing.T) {
	svc, _, codes := newTestService()
	ctx := context.Background()

	u := registerOK(t, svc, codes, "profile@museflow.ai", "pw12345678", "原始昵称")

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
	svc, _, codes := newTestService()
	ctx := context.Background()

	u := registerOK(t, svc, codes, "pwd@museflow.ai", "oldpass1234", "n")

	// 旧密码错误 -> 拒绝
	if err := svc.ChangePassword(ctx, u.UUID.String(), "wrong-old", "newpass5678"); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("旧密码错误未被拒绝: %v", err)
	}

	// 旧密码正确 -> 修改成功，新密码可登录
	if err := svc.ChangePassword(ctx, u.UUID.String(), "oldpass1234", "newpass5678"); err != nil {
		t.Fatalf("修改密码失败: %v", err)
	}
	if _, err := svc.Login(ctx, "pwd@museflow.ai", "newpass5678", testDevice()); err != nil {
		t.Errorf("新密码无法登录: %v", err)
	}
	// 旧密码应失效
	if _, err := svc.Login(ctx, "pwd@museflow.ai", "oldpass1234", testDevice()); !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("旧密码仍可登录: %v", err)
	}
}

func TestLoginLocksAccountAfterMaxFailures(t *testing.T) {
	repo := newFakeUserRepo()
	store := newFakeTokenStore()
	tm := token.NewTokenManager("test-secret", time.Hour, time.Hour, 5*time.Minute)
	codes := newFakeCodeStore()
	svc := NewAuthService(repo, store, tm, nil, nil, nil, codes, newFakeQueue(), &stubCaptcha{}, testResetConfig(), testEmailCodeConfig(), testMFAConfig(), bcrypt.MinCost)
	ctx := context.Background()

	codes.codes["register:lock@museflow.ai"] = testRegisterCode
	if _, err := svc.Register(ctx, "lock@museflow.ai", "pw12345678", "n", testRegisterCode); err != nil {
		t.Fatalf("注册失败: %v", err)
	}

	// 连续失败（repository 层阈值为 5 次）
	for i := 0; i < 5; i++ {
		if _, err := svc.Login(ctx, "lock@museflow.ai", "bad-password", testDevice()); !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("第 %d 次登录失败应返回凭证错误，实际: %v", i+1, err)
		}
	}

	// 达到阈值后锁定：即使密码正确也应拒绝
	if _, err := svc.Login(ctx, "lock@museflow.ai", "pw12345678", testDevice()); err == nil {
		t.Errorf("账号锁定后仍可登录")
	}
}

func TestListSessionsReturnsActiveDevices(t *testing.T) {
	svc, _, codes := newTestService()
	ctx := context.Background()

	u := registerOK(t, svc, codes, "sess@museflow.ai", "pw12345678", "n")
	if _, err := svc.Login(ctx, "sess@museflow.ai", "pw12345678", testDevice()); err != nil {
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

// ---- 邮箱验证码（注册校验 / 补验证 / 免密登录） ----

func TestSendVerifyCodeStoresCodeAndEnqueuesTask(t *testing.T) {
	svc, codes, producer, _ := newResetTestService()
	ctx := context.Background()

	taskID, expiresIn, err := svc.SendVerifyCode(ctx, SendVerifyCodeInput{Email: "Code@MuseFlow.ai", Scene: "register"})
	if err != nil {
		t.Fatalf("发送验证码失败: %v", err)
	}
	if taskID == "" {
		t.Fatal("应返回用于订阅进度的 task_id")
	}
	if expiresIn != int64(svc.emailCfg.CodeTTL.Seconds()) {
		t.Errorf("有效期秒数不符: %d != %d", expiresIn, int64(svc.emailCfg.CodeTTL.Seconds()))
	}

	// 邮箱在存储与任务载荷中均被规范化为小写
	code, _ := codes.GetCode(ctx, "register", "code@museflow.ai")
	if len(code) != svc.emailCfg.CodeLength {
		t.Errorf("应生成 %d 位验证码，实际 %q", svc.emailCfg.CodeLength, code)
	}
	payload := producer.lastVerifyCode()
	if payload == nil {
		t.Fatal("未投递邮箱验证码任务")
	}
	if payload.To != "code@museflow.ai" {
		t.Errorf("收件人未规范化: %s", payload.To)
	}
	if payload.Code != code {
		t.Errorf("任务载荷应携带验证码 %s，实际: %s", code, payload.Code)
	}
	if payload.Scene != "register" {
		t.Errorf("场景不符，实际: %s", payload.Scene)
	}
	if !strings.Contains(payload.Purpose, "注册") {
		t.Errorf("用途描述不含场景信息: %s", payload.Purpose)
	}
}

func TestSendVerifyCodeRollsBackWhenQueueUnavailable(t *testing.T) {
	svc, codes, producer, _ := newResetTestService()
	ctx := context.Background()

	producer.failNext = true
	if _, _, err := svc.SendVerifyCode(ctx, SendVerifyCodeInput{Email: "rollback@museflow.ai", Scene: "register"}); err == nil {
		t.Fatal("队列不可用时应当失败")
	}

	// 验证码与冷却锁都应回滚，用户无需等待一个无效的冷却周期
	if code, _ := codes.GetCode(ctx, "register", "rollback@museflow.ai"); code != "" {
		t.Errorf("入队失败后应删除验证码，实际仍存在: %s", code)
	}
	if codes.locks["register:rollback@museflow.ai"] {
		t.Error("入队失败后应释放重发冷却锁")
	}
}

func TestSendVerifyCodeRejectsUnsupportedScene(t *testing.T) {
	svc, _, _, _ := newResetTestService()
	// 非法场景应被拒绝（不应静默成功）
	if _, _, err := svc.SendVerifyCode(context.Background(), SendVerifyCodeInput{Email: "x@y.com", Scene: "hack"}); err == nil {
		t.Fatal("非法场景应被拒绝")
	}
}

func TestSendVerifyCodePassesCaptchaContext(t *testing.T) {
	svc, _, _, captcha := newResetTestService()
	ctx := context.Background()

	const ip = "203.0.113.7"
	if _, _, err := svc.SendVerifyCode(ctx, SendVerifyCodeInput{
		Email: "captcha@museflow.ai", Scene: "register", CaptchaToken: "tok-abc", ClientIP: ip,
	}); err != nil {
		t.Fatalf("发送验证码失败: %v", err)
	}

	// 令牌、IP 与场景（action）都应原样透传给校验服务
	if captcha.verified != 1 {
		t.Errorf("应执行一次人机验证，实际 %d 次", captcha.verified)
	}
	if captcha.lastTok != "tok-abc" {
		t.Errorf("令牌未透传，实际: %q", captcha.lastTok)
	}
	if captcha.lastIP != ip {
		t.Errorf("客户端 IP 未透传，实际: %q", captcha.lastIP)
	}
	if captcha.lastAct != "register" {
		t.Errorf("action 应取场景名以便前端 widget 对齐，实际: %q", captcha.lastAct)
	}
}

func TestSendVerifyCodeRejectsWhenCaptchaFails(t *testing.T) {
	svc, codes, producer, captcha := newResetTestService()
	ctx := context.Background()

	captcha.failErr = turnstile.ErrTokenInvalid
	_, _, err := svc.SendVerifyCode(ctx, SendVerifyCodeInput{
		Email: "bot@museflow.ai", Scene: "register", CaptchaToken: "stale-token",
	})
	if !errors.Is(err, turnstile.ErrTokenInvalid) {
		t.Fatalf("人机验证未通过时应返回对应错误，实际: %v", err)
	}

	// 关键：未通过人机验证时不得生成验证码，否则机器人仍能拿到有效验证码
	if code, _ := codes.GetCode(ctx, "register", "bot@museflow.ai"); code != "" {
		t.Errorf("人机验证未通过时不应生成验证码，实际: %s", code)
	}
	// 也不得占用重发冷却，否则机器人能把真实用户挡在冷却期内
	if codes.locks["register:bot@museflow.ai"] {
		t.Error("人机验证未通过时不应占用重发冷却锁")
	}
	// 更不应产生发信任务
	if n := len(producer.verifyCodes); n != 0 {
		t.Errorf("人机验证未通过时不应投递发信任务，实际 %d 个", n)
	}
}

func TestSendVerifyCodeSkipsCaptchaWhenDisabled(t *testing.T) {
	// captcha 为 nil 表示服务端未启用人机验证（等价于开发态的 noop 客户端）
	store := newFakeCodeStore()
	producer := newFakeQueue()
	tm := token.NewTokenManager("test-secret", time.Hour, time.Hour, 5*time.Minute)
	svc := NewAuthService(
		newFakeUserRepo(), newFakeTokenStore(), tm,
		nil, nil, nil,
		store, producer, nil, // 人机验证传 nil：跳过校验
		testResetConfig(), testEmailCodeConfig(), testMFAConfig(), bcrypt.MinCost,
	)

	if _, _, err := svc.SendVerifyCode(context.Background(), SendVerifyCodeInput{
		Email: "dev@museflow.ai", Scene: "login",
	}); err != nil {
		t.Fatalf("未启用人机验证时不应阻断发送: %v", err)
	}
	if len(producer.verifyCodes) != 1 {
		t.Error("未启用人机验证时应正常投递发信任务")
	}
}

func TestChangeEmailUpdatesEmailAndVerifies(t *testing.T) {
	svc, codes, _, _ := newResetTestService()
	ctx := context.Background()

	// 准备一个已登录用户
	u := registerOK(t, svc, codes, "old@museflow.ai", "oldpass1234", "n")

	// 向新邮箱发送 change_email 场景验证码
	if _, _, err := svc.SendVerifyCode(ctx, SendVerifyCodeInput{Email: "new@museflow.ai", Scene: "change_email"}); err != nil {
		t.Fatalf("发送验证码失败: %v", err)
	}
	code, _ := codes.GetCode(ctx, "change_email", "new@museflow.ai")

	// 校验通过后应更新邮箱并标记已验证
	if err := svc.ChangeEmail(ctx, u.UUID, "new@museflow.ai", code); err != nil {
		t.Fatalf("修改邮箱失败: %v", err)
	}
	got, _ := svc.users.FindByUUID(ctx, u.UUID)
	if got.Email != "new@museflow.ai" {
		t.Errorf("邮箱未更新，实际: %s", got.Email)
	}
	if !got.EmailVerified {
		t.Error("修改邮箱后仍未标记已验证")
	}

	// 验证码一次性：用同一验证码尝试改成第三个邮箱应失败（码已删除）
	if err := svc.ChangeEmail(ctx, u.UUID, "third@museflow.ai", code); !errors.Is(err, ErrCodeNotSent) {
		t.Errorf("验证码不应可重复使用，实际: %v", err)
	}
}

func TestChangeEmailRejectsAlreadyUsedEmail(t *testing.T) {
	svc, codes, _, _ := newResetTestService()
	ctx := context.Background()

	registerOK(t, svc, codes, "owner@museflow.ai", "oldpass1234", "n")
	u := registerOK(t, svc, codes, "other@museflow.ai", "oldpass1234", "n")

	if _, _, err := svc.SendVerifyCode(ctx, SendVerifyCodeInput{Email: "owner@museflow.ai", Scene: "change_email"}); err != nil {
		t.Fatalf("发送验证码失败: %v", err)
	}
	code, _ := codes.GetCode(ctx, "change_email", "owner@museflow.ai")

	// 试图把 other 的邮箱改成已被 owner 占用的邮箱 -> 拒绝
	if err := svc.ChangeEmail(ctx, u.UUID, "owner@museflow.ai", code); !errors.Is(err, ErrEmailAlreadyUsed) {
		t.Errorf("占用邮箱应被拒绝，实际: %v", err)
	}
}

func TestLoginWithCodeIssuesUsableTokens(t *testing.T) {
	svc, codes, _, _ := newResetTestService()
	ctx := context.Background()

	registerOK(t, svc, codes, "codeuser@museflow.ai", "pw12345678", "n")
	if _, _, err := svc.SendVerifyCode(ctx, SendVerifyCodeInput{Email: "codeuser@museflow.ai", Scene: "login"}); err != nil {
		t.Fatalf("发送登录验证码失败: %v", err)
	}
	code, _ := codes.GetCode(ctx, "login", "codeuser@museflow.ai")

	res, err := svc.LoginWithCode(ctx, "codeuser@museflow.ai", code, testDevice())
	if err != nil {
		t.Fatalf("验证码登录失败: %v", err)
	}
	if res.RequiresMFA {
		t.Fatal("未开启 2FA 时不应要求二次验证")
	}
	if _, err := svc.ValidateAccess(ctx, res.TokenPair.AccessToken); err != nil {
		t.Errorf("access token 校验失败: %v", err)
	}
}

func TestLoginWithCodeRejectsWrongCodeAndUnknownEmail(t *testing.T) {
	svc, codes, _, _ := newResetTestService()
	ctx := context.Background()

	registerOK(t, svc, codes, "codeuser2@museflow.ai", "pw12345678", "n")
	if _, _, err := svc.SendVerifyCode(ctx, SendVerifyCodeInput{Email: "codeuser2@museflow.ai", Scene: "login"}); err != nil {
		t.Fatalf("发送登录验证码失败: %v", err)
	}

	// 错误验证码 -> 拒绝
	if _, err := svc.LoginWithCode(ctx, "codeuser2@museflow.ai", "000000", testDevice()); !errors.Is(err, ErrCodeMismatch) {
		t.Errorf("错误验证码应被拒绝，实际: %v", err)
	}
	// 未知邮箱 -> 先校验验证码，统一返回验证码错误（避免邮箱枚举）
	if _, err := svc.LoginWithCode(ctx, "nobody@museflow.ai", "123456", testDevice()); !errors.Is(err, ErrCodeNotSent) {
		t.Errorf("未知邮箱应返回验证码错误，实际: %v", err)
	}
}
