package rbac

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/museflow/user-service/internal/model"
	"github.com/museflow/user-service/internal/repository"
)

// 测试替身：嵌入接口以满足未用到的方法，只覆盖用例需要的那几个；
// 若用例意外调用到未覆盖的方法会 panic，正好暴露过度依赖。

type fakeRepo struct {
	repository.RBACRepository
	roleCodes   []string
	permCodes   []string
	permissions []model.Permission
}

func (f *fakeRepo) GetUserRoleCodes(context.Context, uuid.UUID) ([]string, error) {
	return f.roleCodes, nil
}

func (f *fakeRepo) GetUserPermissionCodes(context.Context, uuid.UUID) ([]string, error) {
	return f.permCodes, nil
}

func (f *fakeRepo) ListPermissions(context.Context) ([]model.Permission, error) {
	return f.permissions, nil
}

type fakeCache struct {
	repository.TokenStore
	perms      map[string][]string
	setCalls   int
	clearCalls int
}

func (f *fakeCache) GetUserPermissions(_ context.Context, userID string) ([]string, error) {
	return f.perms[userID], nil
}

func (f *fakeCache) SetUserPermissions(_ context.Context, userID string, perms []string, _ time.Duration) error {
	if f.perms == nil {
		f.perms = map[string][]string{}
	}
	f.perms[userID] = perms
	f.setCalls++
	return nil
}

func (f *fakeCache) ClearUserPermissions(_ context.Context, userID string) error {
	delete(f.perms, userID)
	f.clearCalls++
	return nil
}

// ---- 用例 ----

// super_admin 是通配：不依赖权限表里逐条授权，新增权限也直接放行。
func TestSuperAdminBypassesPermissionCheck(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{
		roleCodes:   []string{RoleSuperAdmin},
		permissions: []model.Permission{{Code: "user:read"}, {Code: "novel:read"}},
	}
	cache := &fakeCache{}
	svc := NewService(repo, cache, time.Hour)
	ctx := context.Background()

	// 连权限表里不存在的权限码也应通过——证明是通配而非枚举
	if ok, err := svc.CheckPermission(ctx, id, "future:something"); err != nil || !ok {
		t.Fatalf("super_admin 应免权限校验，ok=%v err=%v", ok, err)
	}

	// 缓存里存通配标记，而不是把权限逐条展开
	if got := cache.perms[id.String()]; len(got) != 1 || got[0] != PermWildcard {
		t.Errorf("缓存应为通配标记，实际 %v", got)
	}

	// 对外查询时展开为权限表中的全部权限，前端菜单无需特殊处理
	codes, err := svc.GetUserPermissions(ctx, id)
	if err != nil {
		t.Fatalf("查询权限失败: %v", err)
	}
	if len(codes) != 2 {
		t.Errorf("应展开为全部权限，实际 %v", codes)
	}
}

func TestNormalUserUsesRolePermissions(t *testing.T) {
	id := uuid.New()
	repo := &fakeRepo{
		roleCodes: []string{RoleUser},
		permCodes: []string{"novel:read", "novel:write"},
	}
	svc := NewService(repo, &fakeCache{}, time.Hour)
	ctx := context.Background()

	if ok, _ := svc.CheckPermission(ctx, id, "novel:read"); !ok {
		t.Error("拥有的权限应通过")
	}
	if ok, _ := svc.CheckPermission(ctx, id, "system:admin"); ok {
		t.Error("未拥有的权限不应通过")
	}
	codes, err := svc.GetUserPermissions(ctx, id)
	if err != nil {
		t.Fatalf("查询权限失败: %v", err)
	}
	if len(codes) != 2 {
		t.Errorf("普通用户应原样返回角色权限，实际 %v", codes)
	}
}

func TestCachedPermissionsSkipDatabase(t *testing.T) {
	id := uuid.New()
	// repo 里没有任何角色/权限数据：命中缓存后结果应仍来自缓存
	repo := &fakeRepo{}
	cache := &fakeCache{perms: map[string][]string{id.String(): {"novel:write"}}}
	svc := NewService(repo, cache, time.Hour)

	codes, err := svc.GetUserPermissions(context.Background(), id)
	if err != nil {
		t.Fatalf("查询权限失败: %v", err)
	}
	if len(codes) != 1 || codes[0] != "novel:write" {
		t.Errorf("应直接使用缓存结果，实际 %v", codes)
	}
	if cache.setCalls != 0 {
		t.Errorf("命中缓存时不应回写，实际写入 %d 次", cache.setCalls)
	}
}
