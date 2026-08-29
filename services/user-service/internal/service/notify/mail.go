// Package notify 提供邮件通知能力（当前用于发送密码重置验证码）。
//
// 设计要点：
//   - 以接口 EmailSender 抽象，便于替换为第三方邮件服务或测试替身。
//   - 未配置 SMTP 主机时自动降级为「日志模式」：把验证码打到日志，
//     保证本地开发 / 联调在没有邮件服务时也能走通完整流程。
//   - 邮件发送失败不影响验证码已生成的事实，调用方按需决定是否回滚。
package notify

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"strings"
	"time"

	"github.com/museflow/pkg/logger"
)

// EmailSender 邮件发送抽象。
type EmailSender interface {
	// Send 发送纯文本邮件；to 为收件人，subject 主题，body 正文。
	Send(to, subject, body string) error
}

// SMTPConfig SMTP 配置。
type SMTPConfig struct {
	Host     string // SMTP 主机，如 smtp.gmail.com
	Port     int    // SMTP 端口，如 587
	Username string // 登录账号
	Password string // 登录密码 / 授权码
	From     string // 发件人地址，为空时回退 Username
}

// Enabled 判断是否已配置 SMTP 主机（未配置则走日志模式）。
func (c SMTPConfig) Enabled() bool {
	return c.Host != ""
}

// smtpSender 基于 net/smtp 的实现。
type smtpSender struct {
	cfg SMTPConfig
}

// NewEmailSender 构造邮件发送器；未配置 SMTP 时返回日志模式实现。
func NewEmailSender(cfg SMTPConfig) EmailSender {
	if !cfg.Enabled() {
		return &logSender{}
	}
	from := cfg.From
	if from == "" {
		from = cfg.Username
	}
	return &smtpSender{cfg: SMTPConfig{Host: cfg.Host, Port: cfg.Port, Username: cfg.Username, Password: cfg.Password, From: from}}
}

func (s *smtpSender) Send(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	msg := strings.Join([]string{
		fmt.Sprintf("From: %s", s.cfg.From),
		fmt.Sprintf("To: %s", to),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		body,
	}, "\r\n")

	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	if err := smtp.SendMail(addr, auth, s.cfg.From, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("发送邮件失败: %w", err)
	}
	return nil
}

// logSender 日志模式：未配置 SMTP 时使用，把内容写入日志而非真实发送。
type logSender struct{}

func (s *logSender) Send(to, subject, body string) error {
	logger.Warn("邮件服务未配置（SMTP_HOST 为空），已降级为日志模式",
		"to", to, "subject", subject, "body", body)
	return nil
}

// GenerateNumericCode 生成指定位数的随机数字验证码。
func GenerateNumericCode(length int) (string, error) {
	if length <= 0 {
		length = 6
	}
	const digits = "0123456789"
	var sb strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", fmt.Errorf("生成验证码失败: %w", err)
		}
		sb.WriteByte(digits[n.Int64()])
	}
	return sb.String(), nil
}

// ResetPasswordBody 构造密码重置邮件正文。
func ResetPasswordBody(code string, ttl time.Duration) string {
	minutes := int(ttl.Minutes())
	if minutes < 1 {
		minutes = 1
	}
	return fmt.Sprintf(
		"您正在重置 MuseFlow 账号密码。\n\n"+
			"验证码：%s\n"+
			"有效期：%d 分钟\n\n"+
			"如果这不是您本人的操作，请忽略此邮件，您的密码不会发生变化。\n",
		code, minutes)
}
