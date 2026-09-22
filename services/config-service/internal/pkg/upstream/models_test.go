package upstream

import (
	"strconv"
	"testing"
)

// itoa 仅为测试构造数据服务。
func itoa(i int) string { return strconv.Itoa(i) }

func TestModelsEndpoint(t *testing.T) {
	cases := []struct {
		name     string
		protocol string
		baseURL  string
		want     string
		wantErr  bool
	}{
		{
			name:     "openai 已带 /v1",
			protocol: "openai",
			baseURL:  "https://api.openai.com/v1",
			want:     "https://api.openai.com/v1/models",
		},
		{
			name:     "openai 只填域名",
			protocol: "openai",
			baseURL:  "https://api.openai.com",
			want:     "https://api.openai.com/v1/models",
		},
		{
			name:     "openai 结尾斜杠",
			protocol: "openai",
			baseURL:  "https://api.openai.com/v1/",
			want:     "https://api.openai.com/v1/models",
		},
		{
			name:     "custom 自定义前缀",
			protocol: "custom",
			baseURL:  "https://gateway.internal/api",
			want:     "https://gateway.internal/api/v1/models",
		},
		{
			name:     "anthropic 已带 /v1",
			protocol: "anthropic",
			baseURL:  "https://api.anthropic.com/v1",
			want:     "https://api.anthropic.com/v1/models",
		},
		{
			name:     "anthropic 只填域名",
			protocol: "anthropic",
			baseURL:  "https://api.anthropic.com",
			want:     "https://api.anthropic.com/v1/models",
		},
		{
			name:     "gemini 固定 v1beta",
			protocol: "gemini",
			baseURL:  "https://generativelanguage.googleapis.com",
			want:     "https://generativelanguage.googleapis.com/v1beta/models",
		},
		{
			name:     "协议大小写与空白",
			protocol: " OpenAI ",
			baseURL:  "https://api.openai.com/v1",
			want:     "https://api.openai.com/v1/models",
		},
		{
			name:     "空 base_url",
			protocol: "openai",
			baseURL:  "   ",
			wantErr:  true,
		},
		{
			name:     "非 http 协议",
			protocol: "openai",
			baseURL:  "ftp://api.openai.com",
			wantErr:  true,
		},
		{
			name:     "不是合法地址",
			protocol: "openai",
			baseURL:  "api.openai.com/v1",
			wantErr:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := modelsEndpoint(tc.protocol, tc.baseURL)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("期望报错，实际得到 %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("意外错误: %v", err)
			}
			if got != tc.want {
				t.Errorf("path 不匹配:\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func TestAuthHeaders(t *testing.T) {
	got := authHeaders("gemini", " key-123 ")
	if got["x-goog-api-key"] != "key-123" {
		t.Errorf("gemini 应使用 x-goog-api-key，实际为 %v", got)
	}
	if _, ok := got["Authorization"]; ok {
		t.Errorf("gemini 不应带 Authorization，实际为 %v", got)
	}

	got = authHeaders("anthropic", "sk-abc")
	if got["Authorization"] != "Bearer sk-abc" {
		t.Errorf("Authorization 头不正确: %q", got["Authorization"])
	}

	// 空密钥一个认证头都不加：本地 Ollama 一类渠道本来就该匿名访问
	got = authHeaders("openai", "   ")
	if _, ok := got["Authorization"]; ok {
		t.Errorf("空密钥不应带认证头，实际为 %v", got)
	}
}

func TestParseCatalog(t *testing.T) {
	openaiStyle := []byte(`{"object":"list","data":[{"id":"gpt-4o","object":"model","owned_by":"openai","created":1717200000}]}`)
	items, err := parseCatalog(openaiStyle)
	if err != nil {
		t.Fatalf("解析 openai 风格失败: %v", err)
	}
	if len(items) != 1 || items[0].ID != "gpt-4o" || items[0].OwnedBy != "openai" {
		t.Fatalf("openai 风格解析结果不对: %+v", items)
	}

	// Gemini 的标识形如 "models/xxx"，调用时要的是后半段
	geminiStyle := []byte(`{"models":[{"name":"models/gemini-2.0-flash","displayName":"Gemini 2.0 Flash"}]}`)
	items, err = parseCatalog(geminiStyle)
	if err != nil {
		t.Fatalf("解析 gemini 风格失败: %v", err)
	}
	if len(items) != 1 || items[0].ID != "gemini-2.0-flash" || items[0].OwnedBy != "Gemini 2.0 Flash" {
		t.Fatalf("gemini 风格解析结果不对: %+v", items)
	}

	// 上游把错误放在 200 响应体里的情况
	if _, err := parseCatalog([]byte(`{"error":{"message":"Invalid API key"}}`)); err == nil {
		t.Error("上游错误响应应当报错")
	}

	// 没有可用标识的记录应被跳过而不是登记成空 api_model
	items, err = parseCatalog([]byte(`{"data":[{"object":"model"},{"id":"ok-1"}]}`))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(items) != 1 || items[0].ID != "ok-1" {
		t.Fatalf("空标识记录未被跳过: %+v", items)
	}
}

func TestParseCatalogCapsAtMaxModels(t *testing.T) {
	body := []byte(`{"data":[`)
	for i := 0; i < MaxModels+50; i++ {
		if i > 0 {
			body = append(body, ',')
		}
		body = append(body, []byte(`{"id":"m-`+itoa(i)+`"}`)...)
	}
	body = append(body, ']', '}')

	items, err := parseCatalog(body)
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(items) != MaxModels {
		t.Errorf("条目数应被截到 %d，实际 %d", MaxModels, len(items))
	}
}
