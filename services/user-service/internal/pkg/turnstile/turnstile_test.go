package turnstile

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// newTestClient 构造指向本地测试服务的客户端。
func newTestClient(t *testing.T, secret, endpoint string, hosts ...string) *Client {
	t.Helper()
	return New(Config{
		Secret:           secret,
		Endpoint:         endpoint,
		Timeout:          2 * time.Second,
		AllowedHostnames: hosts,
	})
}

// fakeCloudflare 模拟 siteverify 接口，返回预设响应。
func fakeCloudflare(t *testing.T, handler func(url.Values) verifyResponse) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("siteverify 应使用 POST，实际 %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Errorf("siteverify 应使用表单编码，实际 %s", ct)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("解析表单失败: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(handler(r.PostForm))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestVerifySkipsWhenSecretUnset(t *testing.T) {
	// 未配置密钥：降级放行，且不发起任何网络请求
	c := New(Config{})
	if c.Verify(context.Background(), "", "", "") != nil {
		t.Error("未配置密钥时应跳过校验（放行）")
	}
	if NewNoop().Verify(context.Background(), "", "", "") != nil {
		t.Error("noop 客户端应恒放行")
	}
}

func TestVerifyRejectsMissingToken(t *testing.T) {
	called := false
	srv := fakeCloudflare(t, func(url.Values) verifyResponse {
		called = true
		return verifyResponse{Success: true}
	})
	c := newTestClient(t, "secret", srv.URL)

	if err := c.Verify(context.Background(), "", "", ""); err != ErrTokenMissing {
		t.Errorf("缺少令牌应返回 ErrTokenMissing，实际: %v", err)
	}
	if called {
		t.Error("缺少令牌时不应请求校验服务")
	}
}

func TestVerifyAcceptsValidToken(t *testing.T) {
	var got url.Values
	srv := fakeCloudflare(t, func(form url.Values) verifyResponse {
		got = form
		return verifyResponse{Success: true, Hostname: "museflow.ai", Action: "register"}
	})
	c := newTestClient(t, "my-secret", srv.URL)

	err := c.Verify(context.Background(), "tok-123", "203.0.113.7", "register")
	if err != nil {
		t.Fatalf("合法令牌应通过: %v", err)
	}

	// 密钥、令牌与 IP 都要按 siteverify 约定提交
	if got.Get("secret") != "my-secret" {
		t.Errorf("密钥未提交: %q", got.Get("secret"))
	}
	if got.Get("response") != "tok-123" {
		t.Errorf("令牌未提交: %q", got.Get("response"))
	}
	if got.Get("remoteip") != "203.0.113.7" {
		t.Errorf("客户端 IP 未提交: %q", got.Get("remoteip"))
	}
}

func TestVerifyOmitsLoopbackRemoteIP(t *testing.T) {
	var got url.Values
	srv := fakeCloudflare(t, func(form url.Values) verifyResponse {
		got = form
		return verifyResponse{Success: true}
	})
	c := newTestClient(t, "secret", srv.URL)

	if err := c.Verify(context.Background(), "tok", "::1", ""); err != nil {
		t.Fatalf("省略回环 IP 后应能通过: %v", err)
	}
	if got.Get("remoteip") != "" {
		t.Errorf("回环地址不应提交给 siteverify，实际: %q", got.Get("remoteip"))
	}
}

func TestVerifyParsesHTTP400JSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(verifyResponse{
			Success:    false,
			ErrorCodes: []string{"invalid-input-secret"},
		})
	}))
	t.Cleanup(srv.Close)
	c := newTestClient(t, "site-key-not-secret", srv.URL)

	if err := c.Verify(context.Background(), "tok", "127.0.0.1", ""); err != ErrServiceUnavailable {
		t.Errorf("密钥无效应返回 ErrServiceUnavailable，实际: %v", err)
	}
}

func TestVerifyRejectsInvalidToken(t *testing.T) {
	srv := fakeCloudflare(t, func(url.Values) verifyResponse {
		return verifyResponse{Success: false, ErrorCodes: []string{"invalid-input-response"}}
	})
	c := newTestClient(t, "secret", srv.URL)

	if err := c.Verify(context.Background(), "bad", "", ""); err != ErrTokenInvalid {
		t.Errorf("无效令牌应返回 ErrTokenInvalid，实际: %v", err)
	}
}

func TestVerifyRejectsReplayedToken(t *testing.T) {
	// timeout-or-duplicate 表示令牌已用过，同样按无效处理
	srv := fakeCloudflare(t, func(url.Values) verifyResponse {
		return verifyResponse{Success: false, ErrorCodes: []string{"timeout-or-duplicate"}}
	})
	c := newTestClient(t, "secret", srv.URL)

	if err := c.Verify(context.Background(), "reused", "", ""); err != ErrTokenInvalid {
		t.Errorf("重放令牌应返回 ErrTokenInvalid，实际: %v", err)
	}
}

func TestVerifyRejectsActionMismatch(t *testing.T) {
	srv := fakeCloudflare(t, func(url.Values) verifyResponse {
		return verifyResponse{Success: true, Action: "login"}
	})
	c := newTestClient(t, "secret", srv.URL)

	// 期望 register 但令牌是 login：说明令牌是从别处拿来的，应拒绝
	if err := c.Verify(context.Background(), "tok", "", "register"); err != ErrActionMismatch {
		t.Errorf("action 不匹配应返回 ErrActionMismatch，实际: %v", err)
	}
}

func TestVerifySkipsActionCheckWhenTokenHasNoAction(t *testing.T) {
	// 旧版 widget 未设置 action 时响应为空，此时不应误伤
	srv := fakeCloudflare(t, func(url.Values) verifyResponse {
		return verifyResponse{Success: true, Action: ""}
	})
	c := newTestClient(t, "secret", srv.URL)

	if err := c.Verify(context.Background(), "tok", "", "register"); err != nil {
		t.Errorf("令牌未携带 action 时应跳过该校验，实际: %v", err)
	}
}

func TestVerifyEnforcesHostnameAllowlist(t *testing.T) {
	srv := fakeCloudflare(t, func(url.Values) verifyResponse {
		return verifyResponse{Success: true, Hostname: "evil.example.com"}
	})
	c := newTestClient(t, "secret", srv.URL, "museflow.ai", "www.museflow.ai")

	if err := c.Verify(context.Background(), "tok", "", ""); err != ErrHostnameMismatch {
		t.Errorf("不在白名单的域名应返回 ErrHostnameMismatch，实际: %v", err)
	}
}

func TestVerifyAllowsListedHostname(t *testing.T) {
	srv := fakeCloudflare(t, func(url.Values) verifyResponse {
		return verifyResponse{Success: true, Hostname: "MUSEFLOW.AI"}
	})
	// 白名单未配置时应放行；配置后按忽略大小写 + 忽略末尾点匹配
	c := newTestClient(t, "secret", srv.URL, "museflow.ai.")
	if err := c.Verify(context.Background(), "tok", "", ""); err != nil {
		t.Errorf("白名单内的域名应放行，实际: %v", err)
	}
}

func TestVerifyFailsClosedOnServiceError(t *testing.T) {
	// Cloudflare 故障：拒绝而不是放行，避免校验挂了就敞开口子
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := newTestClient(t, "secret", srv.URL)

	if err := c.Verify(context.Background(), "tok", "", ""); err != ErrServiceUnavailable {
		t.Errorf("校验服务异常应返回 ErrServiceUnavailable，实际: %v", err)
	}
}

func TestVerifyFailsClosedOnNetworkError(t *testing.T) {
	// 指向未监听的地址，模拟网络不可达
	c := newTestClient(t, "secret", "http://127.0.0.1:1/siteverify")
	if err := c.Verify(context.Background(), "tok", "", ""); err != ErrServiceUnavailable {
		t.Errorf("网络不可达应返回 ErrServiceUnavailable，实际: %v", err)
	}
}

func TestVerifyHonoursContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(500 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(verifyResponse{Success: true})
	}))
	defer srv.Close()
	c := newTestClient(t, "secret", srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := c.Verify(ctx, "tok", "", ""); err != ErrServiceUnavailable {
		t.Errorf("ctx 已取消应返回 ErrServiceUnavailable，实际: %v", err)
	}
}

func TestSanitizeRemoteIP(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"203.0.113.7", "203.0.113.7"},
		{"::1", ""},
		{"127.0.0.1", ""},
		{"[::1]:5173", ""},
		{"192.168.1.8", ""},
		{"  203.0.113.7, 10.0.0.1 ", "203.0.113.7"},
		{"localhost", ""},
	}
	for _, tc := range cases {
		if got := sanitizeRemoteIP(tc.in); got != tc.want {
			t.Errorf("sanitizeRemoteIP(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSplitHostnameHelper(t *testing.T) {
	// 直接覆盖匹配函数：不区分大小写、忽略末尾点
	if !containsHostname([]string{"museflow.ai"}, "MUSEFLOW.AI.") {
		t.Error("hostname 匹配应忽略大小写与末尾点")
	}
	if containsHostname([]string{"museflow.ai"}, "evil.com") {
		t.Error("不在白名单的域名不应匹配")
	}
}

func TestDescribeErrorCodesTranslatesKnownCodes(t *testing.T) {
	got := describeErrorCodes([]string{"timeout-or-duplicate", "unknown-code"})
	if len(got) != 2 {
		t.Fatalf("应保留全部错误码，实际: %v", got)
	}
	if got[0] != "timeout-or-duplicate=验证令牌已过期或已被使用过（令牌一次性）" {
		t.Errorf("已知错误码应翻译成中文，实际: %q", got[0])
	}
	if got[1] != "unknown-code" {
		t.Errorf("未知错误码应原样保留，实际: %q", got[1])
	}
}
