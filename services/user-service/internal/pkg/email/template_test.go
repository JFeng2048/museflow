package email

import (
	"strings"
	"testing"
)

func TestNewRendererParsesAllTemplates(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("解析模板失败: %v", err)
	}

	cases := []struct {
		name string
		data any
		want []string // 正文中必须出现的片段
	}{
		{TemplateVerifyCode, VerifyCodeData{Purpose: "注册 MuseFlow 账号", Code: "123 456", Minutes: 10}, []string{"注册 MuseFlow 账号", "123 456", "10"}},
		{TemplateResetPassword, VerifyCodeData{Code: "654 321", Minutes: 5}, []string{"654 321", "5"}},
		{TemplateWelcome, WelcomeData{Nickname: "墨流", Email: "author@museflow.ai"}, []string{"墨流", "author@museflow.ai"}},
	}

	for _, tc := range cases {
		html, text, err := r.Render(tc.name, tc.data)
		if err != nil {
			t.Fatalf("渲染 %s 失败: %v", tc.name, err)
		}
		for _, want := range tc.want {
			if !strings.Contains(html, want) {
				t.Errorf("模板 %s 的 HTML 缺少 %q", tc.name, want)
			}
			if !strings.Contains(text, want) {
				t.Errorf("模板 %s 的纯文本缺少 %q", tc.name, want)
			}
		}
	}
}

func TestRenderRejectsUnknownTemplate(t *testing.T) {
	r, err := NewRenderer()
	if err != nil {
		t.Fatalf("解析模板失败: %v", err)
	}
	if _, _, err := r.Render("not_exist", nil); err == nil {
		t.Error("未知模板应返回错误")
	}
}

func TestTemplateForScene(t *testing.T) {
	if got := TemplateForScene("reset_password"); got != TemplateResetPassword {
		t.Errorf("密码重置应使用专属模板，实际: %s", got)
	}
	for _, scene := range []string{"register", "login", "change_email"} {
		if got := TemplateForScene(scene); got != TemplateVerifyCode {
			t.Errorf("场景 %s 应使用通用验证码模板，实际: %s", scene, got)
		}
	}
}

func TestFormatCode(t *testing.T) {
	cases := map[string]string{
		"123456": "123 456",
		"1234":   "123 4",
		"12":     "12",
		"":       "",
	}
	for in, want := range cases {
		if got := FormatCode(in); got != want {
			t.Errorf("FormatCode(%q) = %q，期望 %q", in, got, want)
		}
	}
}

func TestTTLMinutes(t *testing.T) {
	if got := TTLMinutes(600e9); got != 10 {
		t.Errorf("600 秒应为 10 分钟，实际 %d", got)
	}
	// 不足 1 分钟按 1 分钟展示，避免出现「0 分钟有效」
	if got := TTLMinutes(1e9); got != 1 {
		t.Errorf("1 秒应至少展示 1 分钟，实际 %d", got)
	}
}

func TestBuildMIMEMessageEncodesChineseSubject(t *testing.T) {
	raw, err := buildMIMEMessage("noreply@museflow.ai", Message{
		To:      "author@museflow.ai",
		Subject: "MuseFlow 邮箱验证码",
		HTML:    "<html><body>验证码</body></html>",
		Text:    "验证码",
	})
	if err != nil {
		t.Fatalf("组装邮件失败: %v", err)
	}
	out := string(raw)

	// 中文主题需按 RFC 2047 编码，不能直接出现原始中文
	if strings.Contains(out, "Subject: MuseFlow 邮箱验证码") {
		t.Error("中文主题应按 RFC 2047 编码")
	}
	if !strings.Contains(out, "=?utf-8?") {
		t.Errorf("主题缺少 RFC 2047 编码标记，实际头部: %s", headOf(out))
	}
	if !strings.Contains(out, "multipart/alternative") {
		t.Error("应生成 multipart/alternative 邮件体")
	}
	if !strings.Contains(out, "text/html; charset=UTF-8") || !strings.Contains(out, "text/plain; charset=UTF-8") {
		t.Error("应同时包含 HTML 与纯文本分块")
	}
}

func TestBuildMIMEMessageKeepsASCIISubject(t *testing.T) {
	raw, err := buildMIMEMessage("noreply@museflow.ai", Message{To: "a@b.com", Subject: "Welcome", Text: "hi"})
	if err != nil {
		t.Fatalf("组装邮件失败: %v", err)
	}
	// 纯 ASCII 主题无需编码，保持可读
	if !strings.Contains(string(raw), "Subject: Welcome") {
		t.Error("纯 ASCII 主题不应被编码")
	}
}

func TestNewSenderFallsBackToLogMode(t *testing.T) {
	// 未配置 SMTP 主机时不应返回真实 SMTP 发送器
	if _, ok := NewSender(SMTPConfig{}).(*logSender); !ok {
		t.Error("未配置 SMTP 时应降级为日志模式")
	}
	if _, ok := NewSender(SMTPConfig{Host: "smtp.example.com", Port: 587}).(*smtpSender); !ok {
		t.Error("配置了 SMTP 时应返回 SMTP 发送器")
	}
}

// headOf 截取邮件头部，便于断言失败时定位问题。
func headOf(s string) string {
	idx := strings.Index(s, "\r\n\r\n")
	if idx < 0 {
		if len(s) > 200 {
			return s[:200]
		}
		return s
	}
	return s[:idx]
}
