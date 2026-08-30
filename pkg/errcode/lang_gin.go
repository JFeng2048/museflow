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

// AcceptedGin 构造「已受理、异步处理中」响应（业务码 2002 / HTTP 202）。
//
// 适用于请求已受理但结果需异步获取的接口：例如投递异步任务后返回 task_id，
// 客户端再凭 task_id 订阅进度。
func AcceptedGin(c *gin.Context, data any) Response {
	lang := LangFromGin(c)
	return Response{
		Code:    int(CodeAccepted),
		Message: Message(CodeAccepted, lang),
		Data:    data,
	}
}

// ErrorGin 根据请求语言构造错误响应。
func ErrorGin(c *gin.Context, code Code) Response {
	return ErrorWithLang(code, LangFromGin(c))
}
