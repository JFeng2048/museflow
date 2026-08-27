package envloader

import (
	"os"
	"path/filepath"
	"testing"
)

// 临时仓库根目录 + .env 文件，作为「全局 .env」层
func setupRootEnv(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	content := "GATEWAY_PORT=5001\n" +
		"JWT_SECRET=shared-secret\n" +
		"DB_HOST=localhost\n" +
		"DB_NAME=museflow\n" +
		"USER_BCRYPT_COST=12\n" +
		"# 注释行应被忽略\n" +
		"SHARED_IGNORED=1\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o644); err != nil {
		t.Fatalf("写根配置失败: %v", err)
	}
	old := os.Getenv("MUSEFLOW_ENV_DIR")
	os.Setenv("MUSEFLOW_ENV_DIR", dir)
	return func() { os.Setenv("MUSEFLOW_ENV_DIR", old) }
}

// 在指定目录写入 .env（模拟「服务自身 .env」层）
func writeServiceEnv(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(content), 0o644); err != nil {
		t.Fatalf("写服务配置失败: %v", err)
	}
}

func TestPrefixLoadFromFile(t *testing.T) {
	cleanup := setupRootEnv(t)
	defer cleanup()

	g := New("GATEWAY", ".env")
	if got := g.Get("PORT", "X"); got != "5001" {
		t.Errorf("GATEWAY_PORT 期望 5001，实际 %q", got)
	}
	if got := g.GetCommon("JWT_SECRET", "X"); got != "shared-secret" {
		t.Errorf("JWT_SECRET 期望 shared-secret，实际 %q", got)
	}

	u := New("USER", ".env")
	if got := u.GetInt("BCRYPT_COST", -1); got != 12 {
		t.Errorf("USER_BCRYPT_COST 期望 12，实际 %d", got)
	}
}

func TestDBCommonConfig(t *testing.T) {
	cleanup := setupRootEnv(t)
	defer cleanup()

	// 数据库配置使用公共 DB_ 前缀（所有服务共用）
	db := New("DB", ".env")
	if got := db.GetCommon("DB_HOST", "X"); got != "localhost" {
		t.Errorf("DB_HOST 期望 localhost，实际 %q", got)
	}
	if got := db.GetCommon("DB_NAME", "X"); got != "museflow" {
		t.Errorf("DB_NAME 期望 museflow，实际 %q", got)
	}
}

// TestLayeredOverride 验证分层优先级：服务自身 .env 覆盖根 .env。
func TestLayeredOverride(t *testing.T) {
	cleanup := setupRootEnv(t)
	defer cleanup()

	// 服务目录与根目录不同，写入服务自身 .env 覆盖部分键
	svcDir := t.TempDir()
	writeServiceEnv(t, svcDir, "GATEWAY_PORT=6001\nDB_NAME=museflow_user\n")

	oldWd, _ := os.Getwd()
	if err := os.Chdir(svcDir); err != nil {
		t.Fatalf("切换工作目录失败: %v", err)
	}
	defer os.Chdir(oldWd)

	g := New("GATEWAY", ".env")
	if got := g.Get("PORT", "X"); got != "6001" {
		t.Errorf("服务 .env 应覆盖根 .env，GATEWAY_PORT 期望 6001，实际 %q", got)
	}

	db := New("DB", ".env")
	if got := db.GetCommon("DB_NAME", "X"); got != "museflow_user" {
		t.Errorf("服务 .env 应覆盖根 .env，DB_NAME 期望 museflow_user，实际 %q", got)
	}
	// 服务 .env 未覆盖的键仍取自根 .env
	if got := db.GetCommon("DB_HOST", "X"); got != "localhost" {
		t.Errorf("未覆盖键应取自根 .env，DB_HOST 期望 localhost，实际 %q", got)
	}
}

func TestSystemEnvOverride(t *testing.T) {
	cleanup := setupRootEnv(t)
	defer cleanup()

	t.Setenv("GATEWAY_PORT", "9999")
	g := New("GATEWAY", ".env")
	if got := g.Get("PORT", "X"); got != "9999" {
		t.Errorf("系统环境变量应覆盖文件值，期望 9999，实际 %q", got)
	}
}

func TestOtherPrefixIgnored(t *testing.T) {
	cleanup := setupRootEnv(t)
	defer cleanup()

	// SHARED_ 前缀不属于任一服务，应读不到
	g := New("GATEWAY", ".env")
	if got := g.Get("IGNORED", "fallback"); got != "fallback" {
		t.Errorf("非本前缀的键应回退默认值，实际 %q", got)
	}
}

func TestMissingFileFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("MUSEFLOW_ENV_DIR", dir)
	defer os.Unsetenv("MUSEFLOW_ENV_DIR")

	g := New("GATEWAY", ".env")
	if got := g.Get("PORT", "5001"); got != "5001" {
		t.Errorf("文件缺失应回退默认值，期望 5001，实际 %q", got)
	}
}

func TestDurationParsing(t *testing.T) {
	cleanup := setupRootEnv(t)
	defer cleanup()

	u := New("USER", ".env")
	if got := u.GetDuration("ACCESS_TTL_SECONDS", 0); got.Seconds() != 3600 {
		t.Logf("ACCESS_TTL_SECONDS 未在 .env，走默认值: %v", got)
	}
}
