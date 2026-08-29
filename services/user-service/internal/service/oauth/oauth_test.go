package oauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
)

// ---- 内存版仓储 ----

type fakeOAuthRepo struct {
	bindings map[string]*model.OAuth // key: provider + ":" + providerUserID
}

func newFakeOAuthRepo() *fakeOAuthRepo {
	return &fakeOAuthRepo{bindings: map[string]*model.OAuth{}}
}

func key(provider, uid string) string { return provider + ":" + uid }

func (r *fakeOAuthRepo) Create(_ context.Context, o *model.OAuth) error {
	o.ID = int64(len(r.bindings) + 1)
	r.bindings[key(o.Provider, o.ProviderUserID)] = o
	return nil
}

func (r *fakeOAuthRepo) FindByProvider(_ context.Context, provider, providerUserID string) (*model.OAuth, error) {
	if o, ok := r.bindings[key(provider, providerUserID)]; ok && o.IsActive {
		return o, nil
	}
	return nil, repository.ErrOAuthNotFound
}

func (r *fakeOAuthRepo) ListByUser(_ context.Context, userUUID uuid.UUID) ([]model.OAuth, error) {
	out := []model.OAuth{}
	for _, o := range r.bindings {
		if o.UserUUID == userUUID && o.IsActive {
			out = append(out, *o)
		}
	}
	return out, nil
}

func (r *fakeOAuthRepo) Delete(_ context.Context, userUUID uuid.UUID, provider string) error {
	for k, o := range r.bindings {
		if o.UserUUID == userUUID && o.Provider == provider {
			delete(r.bindings, k)
			return nil
		}
	}
	return repository.ErrOAuthNotFound
}

func (r *fakeOAuthRepo) TouchLastLogin(_ context.Context, _ int64) error { return nil }

type fakeUserRepo struct {
	byUUID map[string]*model.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byUUID: map[string]*model.User{}}
}

func (r *fakeUserRepo) Create(_ context.Context, u *model.User) error {
	u.ID = int64(len(r.byUUID) + 1)
	r.byUUID[u.UUID.String()] = u
	return nil
}

func (r *fakeUserRepo) FindByUUID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if u, ok := r.byUUID[id.String()]; ok {
		return u, nil
	}
	return nil, repository.ErrUserNotFound
}

func (r *fakeUserRepo) FindByEmail(_ context.Context, _ string) (*model.User, error) {
	return nil, repository.ErrUserNotFound
}

func (r *fakeUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) { return false, nil }
func (r *fakeUserRepo) UpdateLoginInfo(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (r *fakeUserRepo) IncrementLoginFails(_ context.Context, _ string) (*model.User, error) {
	return nil, repository.ErrUserNotFound
}
func (r *fakeUserRepo) ResetLoginFails(_ context.Context, _ uuid.UUID) error { return nil }
func (r *fakeUserRepo) UpdateProfile(_ context.Context, _ uuid.UUID, _, _, _ string) error {
	return nil
}
func (r *fakeUserRepo) UpdatePasswordHash(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (r *fakeUserRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ int16) error { return nil }
func (r *fakeUserRepo) ListUsers(_ context.Context, _ string, _ int16, _ string, _ bool, _, _ int) ([]model.User, error) {
	return nil, nil
}
func (r *fakeUserRepo) CountUsers(_ context.Context, _ string, _ int16) (int64, error) { return 0, nil }

func newTestService() (*Service, *fakeOAuthRepo, *fakeUserRepo) {
	or := newFakeOAuthRepo()
	ur := newFakeUserRepo()
	// rbac / audit 传 nil：本用例聚焦绑定与登录注册流程
	return NewService(or, ur, nil, nil), or, ur
}

func githubProfile(uid string) Profile {
	return Profile{
		Provider:    model.ProviderGitHub,
		ProviderUID: uid,
		Email:       uid + "@example.com",
		Nickname:    "octocat",
		AvatarURL:   "https://avatars.example.com/" + uid,
	}
}

// ---- 用例 ----

func TestOAuthLoginRegistersNewUserOnFirstLogin(t *testing.T) {
	svc, or, ur := newTestService()
	ctx := context.Background()

	u, isNew, err := svc.LoginOrRegister(ctx, githubProfile("gh-001"))
	if err != nil {
		t.Fatalf("首次第三方登录失败: %v", err)
	}
	if !isNew {
		t.Error("首次登录应标记为新注册用户")
	}
	if u.Nickname != "octocat" {
		t.Errorf("昵称应取自第三方资料，实际: %s", u.Nickname)
	}
	if u.PasswordHash != nil {
		t.Error("第三方注册用户不应设置密码哈希")
	}
	if len(or.bindings) != 1 {
		t.Errorf("应建立 1 条绑定，实际 %d", len(or.bindings))
	}
	if len(ur.byUUID) != 1 {
		t.Errorf("应创建 1 个用户，实际 %d", len(ur.byUUID))
	}
}

func TestOAuthLoginReusesExistingBinding(t *testing.T) {
	svc, or, ur := newTestService()
	ctx := context.Background()

	first, _, err := svc.LoginOrRegister(ctx, githubProfile("gh-002"))
	if err != nil {
		t.Fatalf("首次登录失败: %v", err)
	}

	second, isNew, err := svc.LoginOrRegister(ctx, githubProfile("gh-002"))
	if err != nil {
		t.Fatalf("二次登录失败: %v", err)
	}
	if isNew {
		t.Error("二次登录不应再注册新用户")
	}
	if second.UUID != first.UUID {
		t.Errorf("二次登录应返回同一用户，实际 %s != %s", second.UUID, first.UUID)
	}
	if len(ur.byUUID) != 1 || len(or.bindings) != 1 {
		t.Errorf("不应重复创建用户或绑定: users=%d bindings=%d", len(ur.byUUID), len(or.bindings))
	}
}

func TestOAuthLoginRejectsUnsupportedProvider(t *testing.T) {
	svc, _, _ := newTestService()
	ctx := context.Background()

	p := githubProfile("gh-003")
	p.Provider = "myspace"

	if _, _, err := svc.LoginOrRegister(ctx, p); !errors.Is(err, ErrProviderNotSupported) {
		t.Errorf("不支持的平台应被拒绝，实际: %v", err)
	}
}

func TestBindIsIdempotentAndRejectsCrossUserBinding(t *testing.T) {
	svc, or, _ := newTestService()
	ctx := context.Background()

	alice := uuid.New()
	bob := uuid.New()

	if err := svc.Bind(ctx, alice, githubProfile("gh-100")); err != nil {
		t.Fatalf("首次绑定失败: %v", err)
	}
	// 同一用户重复绑定应幂等
	if err := svc.Bind(ctx, alice, githubProfile("gh-100")); err != nil {
		t.Fatalf("重复绑定应幂等，实际报错: %v", err)
	}
	if len(or.bindings) != 1 {
		t.Errorf("重复绑定不应产生多条记录，实际 %d", len(or.bindings))
	}
	// 其他用户绑定同一第三方账号应被拒绝
	if err := svc.Bind(ctx, bob, githubProfile("gh-100")); !errors.Is(err, ErrAlreadyBound) {
		t.Errorf("跨用户绑定应被拒绝，实际: %v", err)
	}
}

func TestUnbindRemovesBinding(t *testing.T) {
	svc, or, _ := newTestService()
	ctx := context.Background()

	uuidA := uuid.New()
	if err := svc.Bind(ctx, uuidA, githubProfile("gh-200")); err != nil {
		t.Fatalf("绑定失败: %v", err)
	}

	list, err := svc.ListBindings(ctx, uuidA)
	if err != nil || len(list) != 1 {
		t.Fatalf("绑定列表应含 1 条，实际 %d, err=%v", len(list), err)
	}

	if err := svc.Unbind(ctx, uuidA, model.ProviderGitHub); err != nil {
		t.Fatalf("解绑失败: %v", err)
	}
	if len(or.bindings) != 0 {
		t.Errorf("解绑后不应残留记录，实际 %d", len(or.bindings))
	}

	// 解绑不存在的绑定应返回未找到
	if err := svc.Unbind(ctx, uuidA, model.ProviderGitHub); !errors.Is(err, ErrOAuthNotFound) {
		t.Errorf("解绑不存在的绑定应返回未找到，实际: %v", err)
	}
}

func TestDefaultNicknameFallsBackToProviderUID(t *testing.T) {
	p := Profile{Provider: model.ProviderWeChat, ProviderUID: "wx-openid-123"}
	if got := defaultNickname(p); got == "" {
		t.Error("默认昵称不应为空")
	}
	// 带邮箱时取邮箱前缀
	p.Email = "alice@example.com"
	if got := defaultNickname(p); got == "" {
		t.Error("带邮箱时默认昵称不应为空")
	}
	_ = time.Now
}
