package errcode

import "github.com/gin-gonic/gin"

// LangFromGin 从 gin 请求的 Accept-Language 头解析语言。
func LangFromGin(c *gin.Context) Lang {
	return ParseAcceptLanguage(c.GetHeader("Accept-Language"))
}

// SuccessGin 根据请求语言构造成功响应。
func SuccessGin(c *gin.Context, data any) Response {
	return SuccessWithLang(LangFromGin(c), data)
}

// ErrorGin 根据请求语言构造错误响应。
func ErrorGin(c *gin.Context, code Code) Response {
	return ErrorWithLang(code, LangFromGin(c))
}
