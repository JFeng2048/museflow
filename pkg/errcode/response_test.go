package errcode

import (
	"net/http"
	"testing"
)

func TestHTTPStatusMapsSuccessSegment(t *testing.T) {
	cases := map[Code]int{
		CodeSuccess:   http.StatusOK,
		CodeCreated:   http.StatusCreated,
		CodeAccepted:  http.StatusAccepted,
		Code(2999):    http.StatusOK, // 未定义的成功码回退 200
	}
	for code, want := range cases {
		got := Error(code).HTTPStatus()
		if got != want {
			t.Errorf("业务码 %d: HTTP 状态期望 %d，实际 %d", code, want, got)
		}
	}
}

func TestHTTPStatusMapsClientErrorSegment(t *testing.T) {
	cases := map[Code]int{
		CodeParamInvalid:    http.StatusBadRequest,
		CodeUnauthorized:    http.StatusUnauthorized,
		CodeForbidden:       http.StatusForbidden,
		CodeNotFound:        http.StatusNotFound,
		CodeConflict:        http.StatusConflict,
		CodeTooManyRequests: http.StatusTooManyRequests,
		// 用户与认证细分码同样落在 4xxx 段
		CodeUserNotFound:    http.StatusNotFound,
		CodeWrongPassword:   http.StatusUnauthorized,
		CodeEmailRegistered: http.StatusConflict,
		CodeTokenInvalid:    http.StatusUnauthorized,
		CodeDeviceMismatch:  http.StatusForbidden,
		CodeAccountLocked:   http.StatusTooManyRequests,
		CodeAccountDisabled: http.StatusForbidden,
		Code(4199):          http.StatusBadRequest, // 未定义细分码回退 400
	}
	for code, want := range cases {
		got := Error(code).HTTPStatus()
		if got != want {
			t.Errorf("业务码 %d: HTTP 状态期望 %d，实际 %d", code, want, got)
		}
	}
}

func TestHTTPStatusMapsServerErrorSegment(t *testing.T) {
	cases := map[Code]int{
		CodeInternal:       http.StatusInternalServerError,
		CodeServiceUnavail: http.StatusServiceUnavailable,
		Code(5999):         http.StatusInternalServerError,
	}
	for code, want := range cases {
		got := Error(code).HTTPStatus()
		if got != want {
			t.Errorf("业务码 %d: HTTP 状态期望 %d，实际 %d", code, want, got)
		}
	}
}

func TestSegmentPredicates(t *testing.T) {
	if !IsSuccess(CodeSuccess) || !IsSuccess(CodeCreated) {
		t.Error("2xxx 应判定为成功段")
	}
	if !IsClientError(CodeParamInvalid) || !IsClientError(CodeUserNotFound) {
		t.Error("4xxx 应判定为客户端错误段（含 41xx 业务细分）")
	}
	if !IsServerError(CodeInternal) || !IsServerError(CodeServiceUnavail) {
		t.Error("5xxx 应判定为服务端错误段")
	}
	if IsSuccess(CodeParamInvalid) || IsClientError(CodeInternal) || IsServerError(CodeSuccess) {
		t.Error("分段判定不应跨段命中")
	}
}

func TestSuccessResponseUsesNewCode(t *testing.T) {
	resp := Success(nil)
	if resp.Code != 2000 {
		t.Errorf("成功响应码应为 2000，实际 %d", resp.Code)
	}
	if resp.Code != int(CodeSuccess) {
		t.Errorf("成功响应码应与 CodeSuccess 一致，实际 %d", resp.Code)
	}
	if resp.HTTPStatus() != http.StatusOK {
		t.Errorf("成功响应 HTTP 状态应为 200，实际 %d", resp.HTTPStatus())
	}
}

func TestMessageFallsBackForUnknownCode(t *testing.T) {
	// 未登记的业务码应返回兜底文案，而非空字符串
	if msg := Message(Code(9999), LangZH); msg == "" {
		t.Error("未登记业务码应返回兜底中文文案")
	}
	if msg := Message(Code(9999), LangEN); msg == "" {
		t.Error("未登记业务码应返回兜底英文文案")
	}
	// 已登记的业务码应返回对应文案
	if msg := Message(CodeAccountLocked, LangZH); msg == "" {
		t.Error("账号锁定码缺少中文文案")
	}
}
