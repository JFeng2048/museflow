// Package upstream 调用上游大模型厂商的模型目录接口。
//
// 存在的理由：后台「渠道」与「模型」是两张表，逐条手抄模型标识既慢又容易打错。
// 拿到 base_url + api_key 之后直接问上游要一份目录，管理员勾选即可批量登记，
// api_model 也就不再需要肉眼校对。
//
// 边界约定：
//   - 本包只发一个 GET 请求（/models 或等价路径），不参与任何对话补全；
//   - 密钥只在请求头里出现一次，不写日志、不进响应；
//   - 上游的非 2xx 一律折算成 ErrUpstreamFailed，由调用方决定怎么提示。
package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// DefaultTimeout 上游调用超时。
	//
	// 拉目录是一次用户可感知的同步操作：太短会把慢厂商误判成不可用，
	// 太长会让后台的按钮一直转下去。15s 覆盖了绝大多数厂商的首字节时间。
	DefaultTimeout = 15 * time.Second

	// MaxResponseBytes 响应体大小上限。
	// 正常模型目录几百 KB；限制是为了避免把一整个错误页读进内存。
	MaxResponseBytes = 4 << 20

	// MaxModels 返回条目上限。厂商目录动辄上千条，而后台只需要勾选常用的那批，
	// 全量返回只会让前端表格难用。
	MaxModels = 500
)

// ErrUpstreamFailed 上游调用失败：网络不可达、超时、非 2xx、响应无法解析。
var ErrUpstreamFailed = errors.New("拉取上游模型列表失败")

// Options 一次上游调用所需的目标与凭证。
type Options struct {
	Protocol string // openai / anthropic / gemini / custom
	BaseURL  string // 已归一化（无结尾斜杠）
	APIKey   string // 可空：本地部署的 Ollama 等渠道不需要密钥
}

// Model 上游返回的单条模型目录项。
type Model struct {
	// ID 上游模型标识，如 gpt-4o；后续创建模型时直接作为 api_model。
	ID string
	// Object 对象类型，如 model / embedding；上游未返回时为空。
	Object string
	// OwnedBy 归属方，如 openai / google；上游未返回时为空。
	OwnedBy string
	// CreatedAt 上游创建时间（RFC3339）；上游未返回时为空。
	CreatedAt string
}

// Client 上游模型目录客户端，实例可全局复用（内部只有无状态的 http.Client）。
type Client struct {
	http *http.Client
}

// NewClient 构造客户端，timeout <= 0 时取 DefaultTimeout。
func NewClient(timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Client{http: &http.Client{Timeout: timeout}}
}

// ListModels 请求上游模型目录并归一化成统一结构。
//
// 两种上游返回格式都能吃：OpenAI / Anthropic 的 {"data":[...]} 与
// Gemini 的 {"models":[...]}。判断依据是哪个数组非空，不看 protocol——
// 兼容厂商抄什么形状的都有，按协议猜反而会把能用的渠道判死。
func (c *Client) ListModels(ctx context.Context, opt Options) ([]Model, error) {
	endpoint, err := modelsEndpoint(opt.Protocol, opt.BaseURL)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("%w：请求地址非法: %v", ErrUpstreamFailed, err)
	}
	for k, v := range authHeaders(opt.Protocol, opt.APIKey) {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w：%v", ErrUpstreamFailed, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, MaxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("%w：读取响应失败: %v", ErrUpstreamFailed, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("%w：%s", ErrUpstreamFailed, describeFailure(resp.StatusCode, body))
	}

	items, err := parseCatalog(body)
	if err != nil {
		return nil, fmt.Errorf("%w：%v", ErrUpstreamFailed, err)
	}
	return items, nil
}

// modelsEndpoint 按协议拼出模型目录地址。
//
// 共同规则是先去掉结尾斜杠，再按协议补版本段：
//   - openai / custom：base_url 通常已带 /v1（如 https://api.openai.com/v1），
//     只填了域名时补上 /v1，否则会拼出 https://api.example.com/models 这种 404；
//   - anthropic：列表接口固定挂在 /v1 下，同样容忍 base_url 已带 /v1；
//   - gemini：版本段是 v1beta，换版本会整体失效，因此单独写死。
func modelsEndpoint(protocol, baseURL string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", fmt.Errorf("%w：base_url 不能为空", ErrUpstreamFailed)
	}

	parsed, err := url.ParseRequestURI(base)
	if err != nil {
		return "", fmt.Errorf("%w：base_url 不是合法地址: %v", ErrUpstreamFailed, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%w：base_url 只支持 http / https", ErrUpstreamFailed)
	}

	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "gemini":
		return base + "/v1beta/models", nil
	case "anthropic":
		return withVersion(base, parsed.Path, "v1"), nil
	default:
		return withVersion(base, parsed.Path, "v1"), nil
	}
}

// withVersion 在 base 与列表路径之间插入版本段，已带该版本段时不重复追加。
func withVersion(base, path, segment string) string {
	trimmed := strings.TrimSuffix(strings.ToLower(path), "/")
	switch trimmed {
	case "", "/":
		// 只填了域名：补上版本段，否则会拼出 https://api.example.com/models 这种 404。
		return base + "/" + segment + "/models"
	case "/" + segment:
		// 已带该版本段：直接挂在它下面，再补一次就成了 /v1/v1/models。
		return base + "/models"
	default:
		// 自定义了前缀路径（如 /api）：按行业惯例仍在其后补版本段。
		return base + "/" + segment + "/models"
	}
}

// authHeaders 组装请求头。
//
// Gemini 用 x-goog-api-key，其余协议一律 Authorization: Bearer——
// 这是各家事实上的约定，包括绝大多数自称「OpenAI 兼容」的国内厂商。
// 密钥为空时一个认证头都不加：本地 Ollama 这类渠道本来就该匿名访问。
func authHeaders(protocol, apiKey string) map[string]string {
	headers := map[string]string{"Accept": "application/json"}

	key := strings.TrimSpace(apiKey)
	if key == "" {
		return headers
	}
	if strings.EqualFold(strings.TrimSpace(protocol), "gemini") {
		headers["x-goog-api-key"] = key
		return headers
	}
	headers["Authorization"] = "Bearer " + key
	return headers
}

// catalogPayload 上游模型目录响应。
//
// displayName 与 created_at 两套命名都要认：Anthropic 用下划线、
// Gemini 用小驼峰，缺一个字段就会把好数据丢掉。
type catalogPayload struct {
	Data   []catalogEntry `json:"data"`
	Models []catalogEntry `json:"models"`
	Error  *catalogError  `json:"error"`
}

type catalogEntry struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Object       string `json:"object"`
	OwnedBy      string `json:"owned_by"`
	DisplayName  string `json:"display_name"`
	DisplayNameC string `json:"displayName"`
	Created      int64  `json:"created"`
	CreatedAt    string `json:"created_at"`
}

type catalogError struct {
	Message string `json:"message"`
	Code    any    `json:"code"`
	Status  string `json:"status"`
}

// parseCatalog 解析目录响应。
func parseCatalog(body []byte) ([]Model, error) {
	var payload catalogPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("响应不是合法的 JSON: %v", err)
	}
	if payload.Error != nil && payload.Error.Message != "" {
		return nil, fmt.Errorf("上游返回错误: %s", payload.Error.Message)
	}

	entries := payload.Data
	if len(entries) == 0 {
		entries = payload.Models
	}

	out := make([]Model, 0, len(entries))
	for _, e := range entries {
		model, ok := toModel(e)
		if !ok {
			continue
		}
		out = append(out, model)
		if len(out) >= MaxModels {
			break
		}
	}
	if len(out) == 0 {
		return nil, errors.New("上游没有返回任何模型")
	}
	return out, nil
}

// toModel 把一条上游记录归一化成 Model。
//
// 返回 false 表示这条记录没有可用标识（Gemini 的 name 为空等），直接跳过：
// 登记一个空 api_model 的模型只会让后续调用失败。
func toModel(e catalogEntry) (Model, bool) {
	id := strings.TrimSpace(e.ID)
	if id == "" {
		// Gemini 的标识形如 "models/gemini-2.0-flash"，调用时要的是后半段。
		id = strings.TrimPrefix(strings.TrimSpace(e.Name), "models/")
	}
	if id == "" {
		return Model{}, false
	}

	out := Model{ID: id, Object: strings.TrimSpace(e.Object)}
	out.OwnedBy = strings.TrimSpace(e.OwnedBy)
	if out.OwnedBy == "" {
		out.OwnedBy = strings.TrimSpace(e.DisplayNameC)
	}
	switch {
	case e.Created > 0:
		out.CreatedAt = time.Unix(e.Created, 0).UTC().Format(time.RFC3339)
	case e.CreatedAt != "":
		out.CreatedAt = strings.TrimSpace(e.CreatedAt)
	}
	return out, true
}

// describeFailure 从非 2xx 响应里提取可读原因。
//
// 优先用上游的 error.message：401 时它会直接说明是密钥无效还是项目未授权，
// 比一句「HTTP 401」有用得多。取不到就退回状态码。
func describeFailure(status int, body []byte) string {
	var payload catalogError
	if err := json.Unmarshal(body, &payload); err == nil && payload.Message != "" {
		return fmt.Sprintf("HTTP %d %s", status, strings.TrimSpace(payload.Message))
	}

	text := strings.TrimSpace(string(body))
	if len(text) > 200 {
		text = text[:200]
	}
	if text == "" {
		return "HTTP " + strconv.Itoa(status)
	}
	return fmt.Sprintf("HTTP %d %s", status, text)
}
