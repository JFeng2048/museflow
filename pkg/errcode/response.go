package errcode

import (
	"net/http"
)

// Response 统一响应包装。
type Response struct {
	Code    int    `json:"code" example:"0"`
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
func (r Response) HTTPStatus() int {
	switch Code(r.Code) {
	case CodeSuccess:
		return http.StatusOK
	case CodeParamInvalid:
		return http.StatusBadRequest
	case CodeUnauthorized, CodeTokenInvalid, CodeTokenExpired,
		CodeRefreshInvalid, CodeMissingToken, CodeMissingRefresh, CodeMissingDevice:
		return http.StatusUnauthorized
	case CodeForbidden, CodeDeviceMismatch:
		return http.StatusForbidden
	case CodeNotFound, CodeUserNotFound:
		return http.StatusNotFound
	case CodeConflict, CodeEmailRegistered:
		return http.StatusConflict
	case CodeServiceUnavail:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
