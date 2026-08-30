package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// 支持的邮箱验证码场景。
const (
	SceneRegister    = "register"       // 注册校验
	SceneLogin       = "login"          // 验证码登录
	SceneResetPasswd = "reset_password" // 密码重置
	SceneChangeEmail = "change_email"   // 修改邮箱
)

// supportedScene 判断场景是否受支持。
//
// 场景名在 repository 层还决定了 Redis 键前缀，因此这里集中校验，
// 避免非法值落到默认分支导致不同用途的验证码互相覆盖。
func supportedScene(scene string) bool {
	switch scene {
	case SceneRegister, SceneLogin, SceneResetPasswd, SceneChangeEmail:
		return true
	default:
		return false
	}
}

// emailPurpose 根据场景返回邮件正文的用途描述。
func emailPurpose(scene string) string {
	switch scene {
	case SceneRegister:
		return "注册 MuseFlow 账号"
	case SceneLogin:
		return "登录 MuseFlow 账号"
	case SceneResetPasswd:
		return "重置 MuseFlow 账号密码"
	case SceneChangeEmail:
		return "修改 MuseFlow 账号邮箱"
	default:
		return "验证 MuseFlow 邮箱"
	}
}

// generateNumericCode 生成指定位数的随机数字验证码（使用密码学安全的随机源）。
func generateNumericCode(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	const digits = "0123456789"
	var sb []byte
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", fmt.Errorf("生成验证码失败: %w", err)
		}
		sb = append(sb, digits[n.Int64()])
	}
	return string(sb), nil
}
