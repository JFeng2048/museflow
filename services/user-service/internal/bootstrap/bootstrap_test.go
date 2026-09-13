package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
	"github.com/museflow/user-service/internal/service/rbac"
)

// ---- 测试替身：只实现 bootstrap 用到的少数方法 ----

type fakeUsers struct {
	user       *model.User
	created    *model.User
	updateCnt  int
	updated    string
	updatedFor uuid.UUID
}

func (f *fakeUsers) FindByEmail(_ context.Context, email string) (*model.User, error) {
	if f.user == nil || !strings.EqualFold(f.user.Email, email) {
		return nil, repository.ErrUserNotFound
	}
	return f.user, nil
}

func (f *fakeUsers) Create(_ context.Context, u *model.User) error {
	if u.UUID == uuid.Nil {
		u.UUID = uuid.New()
	}
	f.created = u
	f.user = u
	return nil
}

func (f *fakeUsers) UpdatePasswordHash(_ context.Context, id uuid.UUID, hash string) error {
	f.updateCnt++
	f.updated = hash
	f.updatedFor = id
	if f.user != nil {
		f.user.PasswordHash = &hash
	}
	return nil
}

type fakeRBAC struct {
	roles        []model.Role
	createdRole  []model.Role
	perms        []model.Permission
	createdPerms []model.Permission
	rolePerms    map[int16][]string
	setCalls     map[int16][]string
	roleCodes    []string
	assignCalls  []string
}

func (f *fakeRBAC) ListRoles(context.Context) ([]model.Role, error) { return f.roles, nil }

func (f *fakeRBAC) CreateRole(_ context.Context, role *model.Role) error {
	role.ID = int16(len(f.roles) + 1)
	f.roles = append(f.roles, *role)
	f.createdRole = append(f.createdRole, *role)
	return nil
}

func (f *fakeRBAC) ListPermissions(context.Context) ([]model.Permission, error) { return f.perms, nil }

func (f *fakeRBAC) CreatePermissions(_ context.Context, perms []model.Permission) error {
	for i, p := range perms {
		if p.ID == 0 {
			p.ID = int16(len(f.perms) + i + 1)
		}
		f.perms = append(f.perms, p)
		f.createdPerms = append(f.createdPerms, p)
	}
	return nil
}

func (f *fakeRBAC) GetRolePermissions(_ context.Context, roleID int16) ([]string, error) {
	return f.rolePerms[roleID], nil
}

func (f *fakeRBAC) SetRolePermissions(_ context.Context, roleID int16, permCodes []string) error {
	if f.rolePerms == nil {
		f.rolePerms = map[int16][]string{}
	}
	if f.setCalls == nil {
		f.setCalls = map[int16][]string{}
	}
	f.rolePerms[roleID] = permCodes
	f.setCalls[roleID] = permCodes
	return nil
}

func (f *fakeRBAC) GetUserRoleCodes(context.Context, uuid.UUID) ([]string, error) {
	return f.roleCodes, nil
}

func (f *fakeRBAC) AssignRole(_ context.Context, _ uuid.UUID, roleCode string, _ uuid.UUID) error {
	f.assignCalls = append(f.assignCalls, roleCode)
	f.roleCodes = append(f.roleCodes, roleCode)
	return nil
}

func testConfig() Config {
	return Config{Email: "Admin@Museflow.com", Password: "admin@museflow.com", BcryptCost: bcrypt.MinCost}
}

// ---- 用例 ----

func TestRunCreatesSystemRolesAndAdmin(t *testing.T) {
	users := &fakeUsers{}
	store := &fakeRBAC{}

	if err := Run(context.Background(), users, store, testConfig()); err != nil {
		t.Fatalf("播种失败: %v", err)
	}

	// 三个系统角色都应补建，且标记为系统角色
	if len(store.createdRole) != 3 {
		t.Fatalf("应创建 3 个系统角色，实际 %d", len(store.createdRole))
	}
	for _, want := range []string{rbac.RoleSuperAdmin, rbac.RoleAdmin, rbac.RoleUser} {
		found := false
		for _, got := range store.createdRole {
			if got.Code == want {
				found = got.IsSystem
			}
		}
		if !found {
			t.Errorf("角色 %s 未创建或未标记为系统角色", want)
		}
	}

	// 管理员账号：邮箱小写归一、密码为 bcrypt 哈希、邮箱视为已验证、状态正常
	admin := users.created
	if admin == nil {
		t.Fatal("未创建管理员账号")
	}
	if admin.Email != "admin@museflow.com" {
		t.Errorf("邮箱未归一化: %s", admin.Email)
	}
	if admin.PasswordHash == nil || *admin.PasswordHash == testConfig().Password {
		t.Fatal("管理员密码未加密存储")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*admin.PasswordHash), []byte(testConfig().Password)); err != nil {
		t.Errorf("bcrypt 哈希无法校验: %v", err)
	}
	if !admin.EmailVerified {
		t.Error("管理员账号应直接标记邮箱已验证，否则无法用密码登录")
	}
	if admin.Status != model.StatusNormal {
		t.Errorf("管理员状态应为正常，实际 %d", admin.Status)
	}
	if admin.Nickname != DefaultAdminNickname {
		t.Errorf("昵称为空时应回退默认值，实际 %q", admin.Nickname)
	}

	// 授予 super_admin
	if len(store.assignCalls) != 1 || store.assignCalls[0] != rbac.RoleSuperAdmin {
		t.Errorf("应授予 super_admin，实际 %v", store.assignCalls)
	}
}

func TestRunUsesConfiguredNickname(t *testing.T) {
	users := &fakeUsers{}
	store := &fakeRBAC{}

	cfg := testConfig()
	cfg.Nickname = "运维管理员"
	if err := Run(context.Background(), users, store, cfg); err != nil {
		t.Fatalf("播种失败: %v", err)
	}
	if users.created == nil || users.created.Nickname != "运维管理员" {
		t.Fatalf("应使用 USER_ADMIN_NICKNAME 配置的昵称，实际 %+v", users.created)
	}
}

func TestRunSeedsPermissionsAndRoleMappings(t *testing.T) {
	users := &fakeUsers{}
	store := &fakeRBAC{}

	if err := Run(context.Background(), users, store, testConfig()); err != nil {
		t.Fatalf("播种失败: %v", err)
	}

	// 权限：与 SQL 种子一致的 17 条，字段完整
	if len(store.createdPerms) != 17 {
		t.Fatalf("应创建 17 条权限，实际 %d", len(store.createdPerms))
	}
	for _, p := range store.createdPerms {
		if p.Code == "" || p.Name == "" || p.Resource == "" || p.Action == "" {
			t.Errorf("权限字段不完整: %+v", p)
		}
	}

	// 角色权限映射：角色由 CreateRole 依次拿到 id 1/2/3
	// super_admin 全部 17 条、user 8 条、admin 在种子里没有权限
	if got := store.setCalls[1]; len(got) != 17 {
		t.Errorf("super_admin 应得到全部权限，实际 %d 条", len(got))
	}
	if got := store.setCalls[3]; len(got) != 8 {
		t.Errorf("user 应得到 8 条权限，实际 %d 条", len(got))
	}
	if _, ok := store.setCalls[2]; ok {
		t.Error("admin 在种子数据里没有权限，不应写入映射")
	}
}

func TestRunOnlyFillsMissingPermissionsAndKeepsCustomizedRoles(t *testing.T) {
	// 三个角色都已存在且都被后台配置过权限：super_admin 被收窄、user 被调整
	store := &fakeRBAC{
		roles: []model.Role{
			{ID: 1, Code: rbac.RoleSuperAdmin, IsSystem: true},
			{ID: 2, Code: rbac.RoleAdmin, IsSystem: true},
			{ID: 3, Code: rbac.RoleUser, IsSystem: true},
		},
		perms: []model.Permission{{ID: 1, Code: "user:read"}},
		rolePerms: map[int16][]string{
			1: {"system:admin"},
			3: {"novel:read"},
		},
	}

	if err := Run(context.Background(), &fakeUsers{}, store, testConfig()); err != nil {
		t.Fatalf("播种失败: %v", err)
	}

	if len(store.createdPerms) != 16 {
		t.Errorf("只应补齐缺失的 16 条权限，实际 %d", len(store.createdPerms))
	}
	if len(store.setCalls) != 0 {
		t.Errorf("角色已有权限时不应覆盖，实际写入 %v", store.setCalls)
	}
}

func TestRunFillsRoleDefaultsWhenRoleHasNoPermissions(t *testing.T) {
	// 角色存在但权限为空（例如只清了 role_permission）-> 应补回默认映射
	store := &fakeRBAC{
		roles: []model.Role{{ID: 1, Code: rbac.RoleSuperAdmin, IsSystem: true}},
		perms: systemPermissions,
	}

	if err := Run(context.Background(), &fakeUsers{}, store, testConfig()); err != nil {
		t.Fatalf("播种失败: %v", err)
	}

	if len(store.createdPerms) != 0 {
		t.Errorf("权限已齐备时不应重复创建，实际 %d", len(store.createdPerms))
	}
	if got := store.setCalls[1]; len(got) != len(systemPermissions) {
		t.Errorf("权限为空的角色应补回全部默认权限，实际 %d 条", len(got))
	}
}

func TestRunSkipsExistingRolesAndKeepsPassword(t *testing.T) {
	hash := "existing-hash"
	users := &fakeUsers{user: &model.User{
		UUID:         uuid.New(),
		Email:        "admin@museflow.com",
		PasswordHash: &hash,
		Nickname:     "系统管理员",
		Status:       model.StatusNormal,
	}}
	store := &fakeRBAC{
		roles:     []model.Role{{Code: rbac.RoleSuperAdmin}, {Code: rbac.RoleAdmin}, {Code: rbac.RoleUser}},
		roleCodes: []string{rbac.RoleSuperAdmin},
	}

	if err := Run(context.Background(), users, store, testConfig()); err != nil {
		t.Fatalf("播种失败: %v", err)
	}

	if len(store.createdRole) != 0 {
		t.Errorf("角色已齐备时不应再创建，实际创建 %d 个", len(store.createdRole))
	}
	if users.updateCnt != 0 {
		t.Error("已存在账号的密码不应被覆盖")
	}
	if users.created != nil {
		t.Error("账号已存在时不应重复创建")
	}
	if len(store.assignCalls) != 0 {
		t.Errorf("已拥有 super_admin 时不应重复授予，实际 %v", store.assignCalls)
	}
}

func TestRunGrantsRoleWhenMissing(t *testing.T) {
	hash := "existing-hash"
	users := &fakeUsers{user: &model.User{
		UUID:         uuid.New(),
		Email:        "admin@museflow.com",
		PasswordHash: &hash,
	}}
	store := &fakeRBAC{roleCodes: []string{rbac.RoleUser}}

	if err := Run(context.Background(), users, store, testConfig()); err != nil {
		t.Fatalf("播种失败: %v", err)
	}
	if len(store.assignCalls) != 1 || store.assignCalls[0] != rbac.RoleSuperAdmin {
		t.Errorf("缺少 super_admin 时应补授，实际 %v", store.assignCalls)
	}
}

func TestRunResetsPasswordWhenEnabled(t *testing.T) {
	hash := "existing-hash"
	users := &fakeUsers{user: &model.User{
		UUID:         uuid.New(),
		Email:        "admin@museflow.com",
		PasswordHash: &hash,
	}}
	store := &fakeRBAC{roleCodes: []string{rbac.RoleSuperAdmin}}

	cfg := testConfig()
	cfg.ResetPassword = true
	if err := Run(context.Background(), users, store, cfg); err != nil {
		t.Fatalf("播种失败: %v", err)
	}
	if users.updateCnt != 1 {
		t.Fatalf("开启重置时应改写密码一次，实际 %d", users.updateCnt)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(users.updated), []byte(cfg.Password)); err != nil {
		t.Errorf("重置后的哈希无法用配置密码校验: %v", err)
	}
}

func TestRunFillsPasswordWhenAccountHasNone(t *testing.T) {
	users := &fakeUsers{user: &model.User{UUID: uuid.New(), Email: "admin@museflow.com"}}
	store := &fakeRBAC{roleCodes: []string{rbac.RoleSuperAdmin}}

	if err := Run(context.Background(), users, store, testConfig()); err != nil {
		t.Fatalf("播种失败: %v", err)
	}
	if users.updateCnt != 1 {
		t.Errorf("账号无密码时应补设，实际改写 %d 次", users.updateCnt)
	}
}

func TestRunWithoutEmailOnlyEnsuresRoles(t *testing.T) {
	users := &fakeUsers{}
	store := &fakeRBAC{}

	cfg := testConfig()
	cfg.Email = ""
	if err := Run(context.Background(), users, store, cfg); err != nil {
		t.Fatalf("播种失败: %v", err)
	}
	if len(store.createdRole) != 3 {
		t.Errorf("未配置管理员时仍应补齐角色，实际创建 %d 个", len(store.createdRole))
	}
	if users.created != nil {
		t.Error("未配置 USER_ADMIN_EMAIL 时不应创建账号")
	}
}

func TestRunRequiresPasswordWhenCreating(t *testing.T) {
	cfg := testConfig()
	cfg.Password = ""

	err := Run(context.Background(), &fakeUsers{}, &fakeRBAC{}, cfg)
	if err == nil || !strings.Contains(err.Error(), "USER_ADMIN_PASSWORD") {
		t.Fatalf("密码为空且账号不存在时应报错并给出键名，实际: %v", err)
	}
}

func TestRunRejectsOverlongPassword(t *testing.T) {
	cfg := testConfig()
	cfg.Password = strings.Repeat("a", maxPasswordBytes+1)

	err := Run(context.Background(), &fakeUsers{}, &fakeRBAC{}, cfg)
	if err == nil {
		t.Fatal("超长密码应被拒绝，避免 bcrypt 静默截断")
	}
}

func TestRunRejectsNilDeps(t *testing.T) {
	if err := Run(context.Background(), nil, &fakeRBAC{}, testConfig()); err == nil {
		t.Error("用户仓储为空时应报错")
	}
	if err := Run(context.Background(), &fakeUsers{}, nil, testConfig()); err == nil {
		t.Error("RBAC 仓储为空时应报错")
	}
}
