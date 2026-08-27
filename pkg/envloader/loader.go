// Package envloader 提供基于前缀的环境变量加载器，供各微服务共享。
//
// 设计目标：
//  1. 分层读取 .env 文件：服务自身目录的 .env 与仓库根目录的 .env
//  2. 系统环境变量优先级最高
//  3. 提供默认值
//  4. 每个服务只读取自己的前缀（如网关 GATEWAY_、用户服务 USER_）；共享配置不带前缀（如 DB_HOST、JWT_SECRET）
//
// 加载优先级（由高到低）：
//
//	系统环境变量 > 服务自身 .env > 仓库根 .env > 默认值
//
// 每个服务在自己的目录（如 services/user-service/.env）放置专属配置，
// 仓库根目录的 .env 放置公共/默认配置；服务自身 .env 会覆盖根 .env。
// 可用 MUSEFLOW_ENV_DIR 指定仓库根目录（覆盖向上查找）。
package envloader

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Loader 前缀环境变量加载器。
type Loader struct {
	prefix string
	values map[string]string // 已去除前缀的 key -> 值（合并服务 .env 与根 .env，服务优先）
	common map[string]string // 公共配置 key（不含服务前缀）-> 值，如 DB_HOST、JWT_SECRET
}

// New 构造加载器。
//
// prefix 为配置前缀，不含下划线（如 "GATEWAY"、"USER"、"DB"）。
// fileName 为配置文件名（如 ".env"）。
//
// 会依次加载：
//   - 当前工作目录下的 fileName（服务自身 .env）
//   - 仓库根目录下的 fileName（全局 .env，可用 MUSEFLOW_ENV_DIR 指定）
//
// 文件缺失不返回错误：生产环境可直接使用系统环境变量。
func New(prefix, fileName string) *Loader {
	prefixUpper := strings.ToUpper(prefix) + "_"
	values := make(map[string]string)
	common := make(map[string]string)

	// 全局 .env（仓库根目录）作为底层
	mergeFile(resolveRepoRoot(), fileName, prefixUpper, values, common)
	// 服务自身 .env（当前工作目录）覆盖上层
	mergeFile(cwd(), fileName, prefixUpper, values, common)

	return &Loader{
		prefix: prefixUpper,
		values: values,
		common: common,
	}
}

// mergeFile 读取 dir/fileName 中的键值对，覆盖写入 values/common。
// 服务自身 .env 后加载，故其取值优先于根 .env。
func mergeFile(dir, fileName, prefixUpper string, values, common map[string]string) {
	if dir == "" {
		return
	}
	path := filepath.Join(dir, fileName)
	f, err := os.Open(path)
	if err != nil {
		// 文件不存在时使用纯默认值，不报错
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		rawKey := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, `"'`) // 去除首尾引号

		rawKeyUpper := strings.ToUpper(rawKey)
		// 公共配置（不含服务前缀）始终收集，供 GetCommon 使用
		common[rawKeyUpper] = val

		if !strings.HasPrefix(rawKeyUpper, prefixUpper) {
			continue
		}
		values[strings.TrimPrefix(rawKeyUpper, prefixUpper)] = val
	}
	// 忽略扫描错误：损坏的配置文件不影响启动
	_ = scanner.Err()
}

// Get 按优先级读取字符串：系统环境变量 > 文件 > 默认值。
func (l *Loader) Get(key, fallback string) string {
	envKey := l.prefix + strings.ToUpper(key)
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	if v, ok := l.values[strings.ToUpper(key)]; ok && v != "" {
		return v
	}
	return fallback
}

// GetBool 读取布尔值，解析失败回退默认值。
func (l *Loader) GetBool(key string, fallback bool) bool {
	v := l.Get(key, "")
	if v == "" {
		return fallback
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	return fallback
}

// GetInt 读取整数，解析失败回退默认值。
func (l *Loader) GetInt(key string, fallback int) int {
	v := l.Get(key, "")
	if v == "" {
		return fallback
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return fallback
}

// GetDuration 读取秒数并转为 time.Duration，解析失败回退默认值。
func (l *Loader) GetDuration(key string, fallback time.Duration) time.Duration {
	return time.Duration(l.GetInt(key, int(fallback.Seconds()))) * time.Second
}

// GetCommon 读取公共配置（不含服务前缀），优先级：系统环境变量 > 文件 > 默认值。
// 适用于跨层共享、不带服务前缀的配置（如 DB_HOST、JWT_SECRET）。
func (l *Loader) GetCommon(key, fallback string) string {
	keyUpper := strings.ToUpper(key)
	if v := os.Getenv(keyUpper); v != "" {
		return v
	}
	if v, ok := l.common[keyUpper]; ok && v != "" {
		return v
	}
	return fallback
}

// GetCommonInt 读取公共整数配置，解析失败回退默认值。
func (l *Loader) GetCommonInt(key string, fallback int) int {
	v := l.GetCommon(key, "")
	if v == "" {
		return fallback
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return fallback
}

// cwd 返回当前工作目录，出错时返回空串。
func cwd() string {
	d, err := os.Getwd()
	if err != nil {
		return ""
	}
	return d
}

// resolveRepoRoot 定位仓库根目录。
//
// 优先使用 MUSEFLOW_ENV_DIR 环境变量；否则从当前工作目录向上查找含 .git 的目录。
// 若均失败，返回当前工作目录（由调用方决定 .env 是否存在）。
func resolveRepoRoot() string {
	if dir := os.Getenv("MUSEFLOW_ENV_DIR"); dir != "" {
		return dir
	}
	dir := cwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd()
}
