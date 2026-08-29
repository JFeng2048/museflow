package errcode

import (
	"net/http"
)

// Response 统一响应包装。
type Response struct {
	Code    int    `json:"code" example:"2000"`
	Message string `json:"message" example:"success"`
	Data    any    `json:"data,omitempty"`
}

// Success 构造成功响应。
func Success(data any) Response {
	return Response{
		Code:    int(CodeSuccess),
		Message: Message(CodeSuccess, defaultLang),
		Data:    data,
	}
}

// SuccessWithLang 按指定语言构造成功响应。
func SuccessWithLang(lang Lang, data any) Response {
	return Response{
		Code:    int(CodeSuccess),
		Message: Message(CodeSuccess, lang),
		Data:    data,
	}
}

// Error 按业务码构造错误响应（默认语言）。
func Error(code Code) Response {
	return Response{
		Code:    int(code),
		Message: Message(code, defaultLang),
	}
}

// ErrorWithLang 按业务码与指定语言构造错误响应。
func ErrorWithLang(code Code, lang Lang) Response {
	return Response{
		Code:    int(code),
		Message: Message(code, lang),
	}
}

// Fail 以自定义消息构造错误响应（不查字典，message 原样返回）。
func Fail(code Code, message string) Response {
	return Response{
		Code:    int(code),
		Message: message,
	}
}

// HTTPStatus 将业务码映射为推荐的 HTTP 状态码。
//
// 由于业务码本身即按 HTTP 语义分段（2xxx/4xxx/5xxx），
// 这里先按段判定大方向，再对细分码做精确映射。
func (r Response) HTTPStatus() int {
	c := Code(r.Code)

	switch {
	case IsSuccess(c):
		return successHTTPStatus(c)
	case IsClientError(c):
		return clientErrorHTTPStatus(c)
	case IsServerError(c):
		return serverErrorHTTPStatus(c)
	default:
		// 未落入任何分段（如历史遗留的 0），按成功处理以兼容旧客户端
		return http.StatusOK
	}
}

// successHTTPStatus 成功段（2xxx）到 HTTP 状态码的映射。
func successHTTPStatus(c Code) int {
	switch c {
	case CodeCreated:
		return http.StatusCreated
	case CodeAccepted:
		return http.StatusAccepted
	default:
		return http.StatusOK
	}
}

// clientErrorHTTPStatus 客户端错误段（4xxx）到 HTTP 状态码的映射。
func clientErrorHTTPStatus(c Code) int {
	switch c {
	case CodeUnauthorized, CodeTokenInvalid, CodeTokenExpired,
		CodeRefreshInvalid, CodeMissingToken, CodeMissingRefresh,
		CodeMissingDevice, CodeWrongPassword:
		return http.StatusUnauthorized
	case CodeForbidden, CodeDeviceMismatch, CodeAccountDisabled:
		return http.StatusForbidden
	case CodeNotFound, CodeUserNotFound:
		return http.StatusNotFound
	case CodeConflict, CodeEmailRegistered:
		return http.StatusConflict
	case CodeTooManyRequests, CodeAccountLocked:
		return http.StatusTooManyRequests
	default:
		return http.StatusBadRequest
	}
}

// serverErrorHTTPStatus 服务端错误段（5xxx）到 HTTP 状态码的映射。
func serverErrorHTTPStatus(c Code) int {
	switch c {
	case CodeServiceUnavail:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
