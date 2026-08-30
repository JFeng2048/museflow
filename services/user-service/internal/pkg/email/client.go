// Package email 封装邮件投递能力。
//
// 设计要点：
//   - 以 Sender 接口抽象，便于替换为第三方邮件服务或测试替身。
//   - 未配置 SMTP 主机时自动降级为「日志模式」：把邮件内容写入日志，
//     保证本地开发 / 联调在没有邮件服务时也能走通完整流程。
//   - 正文采用 multipart/alternative（HTML + 纯文本），主题按 RFC 2047 编码，
//     保证中文主题与富文本正文在各类邮件客户端中正常显示。
//   - Send 接受 context，Worker 的任务超时可以及时取消阻塞的 SMTP 调用。
package email

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"
	"time"

	"github.com/museflow/pkg/logger"
)

// Message 待发送的邮件。
type Message struct {
	To      string // 收件人
	Subject string // 主题（支持中文，内部按 RFC 2047 编码）
	// HTML 富文本正文。
	HTML string
	// Text 纯文本正文，作为不支持 HTML 的客户端的降级内容。
	Text string
}

// Sender 邮件发送抽象。
type Sender interface {
	// Send 发送邮件；实现应尊重 ctx 的取消与超时。
	Send(ctx context.Context, msg Message) error
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
func (c SMTPConfig) Enabled() bool { return c.Host != "" }

// addr 返回 host:port 形式的连接地址。
func (c SMTPConfig) addr() string { return fmt.Sprintf("%s:%d", c.Host, c.Port) }

// sender 返回实际发件人地址。
func (c SMTPConfig) sender() string {
	if c.From != "" {
		return c.From
	}
	return c.Username
}

// smtpSender 基于 net/smtp 的实现。
type smtpSender struct {
	cfg SMTPConfig
}

// NewSender 构造邮件发送器；未配置 SMTP 时返回日志模式实现。
func NewSender(cfg SMTPConfig) Sender {
	if !cfg.Enabled() {
		return &logSender{}
	}
	return &smtpSender{cfg: cfg}
}

// Send 发送邮件。
//
// net/smtp 不支持 context，这里把阻塞调用放到独立协程，
// 通过 select 响应 ctx 的取消 / 超时，避免 Worker 被慢 SMTP 长期占用。
func (s *smtpSender) Send(ctx context.Context, msg Message) error {
	if msg.To == "" {
		return fmt.Errorf("收件人为空")
	}

	raw, err := buildMIMEMessage(s.cfg.sender(), msg)
	if err != nil {
		return err
	}

	errc := make(chan error, 1)
	go func() { errc <- s.send(raw, msg.To) }()

	select {
	case err := <-errc:
		if err != nil {
			return fmt.Errorf("发送邮件失败: %w", err)
		}
		return nil
	case <-ctx.Done():
		return fmt.Errorf("发送邮件被中断: %w", ctx.Err())
	}
}

// send 执行一次 SMTP 会话。
//
// 端口 465 走隐式 TLS，其余端口先建立明文连接再用 STARTTLS 升级；
// 升级失败时退回明文，兼容部分内网邮件网关。
func (s *smtpSender) send(raw []byte, to string) error {
	addr := s.cfg.addr()
	from := s.cfg.sender()
	auth := smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)

	if s.cfg.Port == 465 {
		return sendOverTLS(addr, auth, from, []string{to}, raw)
	}

	client, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return err
	}
	// 服务端支持 STARTTLS 时升级连接，凭据不明文外泄
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: s.cfg.Host}); err != nil {
			return err
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// sendOverTLS 建立隐式 TLS 连接后发送（端口 465）。
func sendOverTLS(addr string, auth smtp.Auth, from string, to []string, raw []byte) error {
	host, _, err := splitHostPort(addr)
	if err != nil {
		return err
	}
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Hello("localhost"); err != nil {
		return err
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// splitHostPort 拆分 host:port，忽略 IPv6 方括号。
func splitHostPort(addr string) (string, string, error) {
	idx := strings.LastIndex(addr, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("非法的 SMTP 地址: %s", addr)
	}
	host := addr[:idx]
	host = strings.Trim(host, "[]")
	return host, addr[idx+1:], nil
}

// logSender 日志模式：未配置 SMTP 时使用，把内容写入日志而非真实发送。
type logSender struct{}

func (s *logSender) Send(_ context.Context, msg Message) error {
	// 开发环境提示：验证码等敏感内容只写入日志，生产环境请配置 SMTP
	logger.Warn("邮件服务未配置（SMTP_HOST 为空），已降级为日志模式：内容已打印至下方日志，请在生产环境配置 SMTP 以便正常发信",
		"to", msg.To, "subject", msg.Subject, "text", msg.Text)
	return nil
}

// buildMIMEMessage 组装 multipart/alternative 邮件体。
func buildMIMEMessage(from string, msg Message) ([]byte, error) {
	var buf bytes.Buffer
	w := textproto.NewWriter(bufio.NewWriter(&buf))

	headers := map[string]string{
		"From":         from,
		"To":           msg.To,
		"Subject":      encodeHeader(msg.Subject),
		"Date":         time.Now().Format(time.RFC1123Z),
		"MIME-Version": "1.0",
	}
	for k, v := range headers {
		if err := w.PrintfLine("%s: %s", k, v); err != nil {
			return nil, err
		}
	}

	mp := multipart.NewWriter(&buf)
	if err := w.PrintfLine("Content-Type: multipart/alternative; boundary=%s", mp.Boundary()); err != nil {
		return nil, err
	}
	if err := w.PrintfLine(""); err != nil {
		return nil, err
	}

	if err := writePart(mp, "text/plain; charset=UTF-8", msg.Text); err != nil {
		return nil, err
	}
	if msg.HTML != "" {
		if err := writePart(mp, "text/html; charset=UTF-8", msg.HTML); err != nil {
			return nil, err
		}
	}
	if err := mp.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writePart 写入 multipart 的一个分块。
func writePart(mp *multipart.Writer, contentType, body string) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", contentType)
	h.Set("Content-Transfer-Encoding", "8bit")
	part, err := mp.CreatePart(h)
	if err != nil {
		return err
	}
	_, err = part.Write([]byte(body))
	return err
}

// encodeHeader 按 RFC 2047 编码非 ASCII 邮件头。
func encodeHeader(s string) string {
	if s == "" {
		return ""
	}
	if isASCII(s) {
		return s
	}
	return mime.QEncoding.Encode("utf-8", s)
}

// isASCII 判断字符串是否全为 ASCII 可打印字符。
func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}
