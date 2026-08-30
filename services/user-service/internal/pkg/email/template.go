package email

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"strings"
	"time"
)

//go:embed templates/*.html templates/*.txt
var templateFS embed.FS

// 模板名称。HTML 与同名 .txt 成对出现，分别作为富文本与纯文本正文。
const (
	// TemplateVerifyCode 邮箱验证码（注册 / 登录 / 修改邮箱）。
	TemplateVerifyCode = "verify_code"
	// TemplateResetPassword 密码重置验证码（措辞更强调账号安全）。
	TemplateResetPassword = "reset_password"
	// TemplateWelcome 注册欢迎邮件。
	TemplateWelcome = "welcome"
)

// Renderer 邮件模板渲染器。
//
// 模板在构造时一次性解析并缓存，运行期并发安全（html/template 的 Execute 只读）。
type Renderer struct {
	tmpl *template.Template
}

// NewRenderer 解析内嵌模板。模板文件缺失或语法错误时返回错误，
// 让 Worker 在启动阶段就暴露问题，而不是等到发信时才发现。
func NewRenderer() (*Renderer, error) {
	tmpl, err := template.New("email").
		Funcs(template.FuncMap{
			"year": func() int { return time.Now().Year() },
		}).
		ParseFS(templateFS, "templates/*.html", "templates/*.txt")
	if err != nil {
		return nil, fmt.Errorf("解析邮件模板失败: %w", err)
	}
	return &Renderer{tmpl: tmpl}, nil
}

// Render 渲染指定模板的 HTML 与纯文本正文。
func (r *Renderer) Render(name string, data any) (html string, text string, err error) {
	html, err = r.render(name+".html", data)
	if err != nil {
		return "", "", err
	}
	text, err = r.render(name+".txt", data)
	if err != nil {
		return "", "", err
	}
	return html, text, nil
}

func (r *Renderer) render(file string, data any) (string, error) {
	if r.tmpl.Lookup(file) == nil {
		return "", fmt.Errorf("邮件模板不存在: %s", file)
	}
	var buf bytes.Buffer
	if err := r.tmpl.ExecuteTemplate(&buf, file, data); err != nil {
		return "", fmt.Errorf("渲染邮件模板 %s 失败: %w", file, err)
	}
	return buf.String(), nil
}

// VerifyCodeData 验证码邮件模板数据。
type VerifyCodeData struct {
	// Purpose 用途描述，如「注册 MuseFlow 账号」。
	Purpose string
	// Nickname 收件人昵称，为空时模板回退为通用问候。
	Nickname string
	// Code 验证码（调用方应先用 FormatCode 分组，便于阅读）。
	Code string
	// Minutes 有效期（分钟）。
	Minutes int
}

// WelcomeData 欢迎邮件模板数据。
type WelcomeData struct {
	Nickname string
	Email    string
}

// TemplateForScene 按验证码场景选择模板。
//
// 密码重置单独一套模板，正文会强调「未发起请忽略」等安全提示。
func TemplateForScene(scene string) string {
	if scene == "reset_password" {
		return TemplateResetPassword
	}
	return TemplateVerifyCode
}

// VerifyCodeSubject 生成验证码邮件主题。
func VerifyCodeSubject(scene string) string {
	switch scene {
	case "reset_password":
		return "重置您的 MuseFlow 账号密码"
	case "change_email":
		return "验证您的新邮箱"
	case "login":
		return "MuseFlow 登录验证码"
	default:
		return "MuseFlow 邮箱验证码"
	}
}

// FormatCode 按 3 位一组格式化验证码，便于在邮件中辨认。
func FormatCode(code string) string {
	var sb strings.Builder
	for i, c := range code {
		if i > 0 && i%3 == 0 {
			sb.WriteByte(' ')
		}
		sb.WriteRune(c)
	}
	return sb.String()
}

// TTLMinutes 将有效期换算为分钟，不足 1 分钟按 1 分钟展示。
func TTLMinutes(ttl time.Duration) int {
	m := int(ttl.Minutes())
	if m < 1 {
		return 1
	}
	return m
}
