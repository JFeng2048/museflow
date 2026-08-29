// Package errcode 定义所有服务共用的业务码及统一响应封装。
//
// 业务码采用「HTTP 状态码语义 + 业务细分」的分段设计：
//   - 2xxx：成功（镜像 HTTP 2xx，如 2000=OK、2001=Created）
//   - 4xxx：客户端错误（镜像 HTTP 4xx，如 4000=BadRequest、4001=Unauthorized）
//   - 5xxx：服务端错误（镜像 HTTP 5xx，如 5000=Internal、5003=ServiceUnavailable）
//
// 业务细分码在对应分段内继续扩展（如用户与认证相关为 41xx，仍属 4xxx 客户端错误），
// 保证「看千位即知 HTTP 语义」，便于网关统一映射与前端按段处理。
package errcode

// Code 业务码类型。2000 表示成功。
type Code int

// 业务码分段常量。
const (
	// segmentSuccess 成功段（2xxx）。
	segmentSuccess = 2000
	// segmentClientError 客户端错误段（4xxx）。
	segmentClientError = 4000
	// segmentServerError 服务端错误段（5xxx）。
	segmentServerError = 5000
)

// 通用业务码。
const (
	// ---- 成功 2xxx ----
	// CodeSuccess 通用成功（对应 HTTP 200）。
	CodeSuccess Code = 2000
	// CodeCreated 创建成功（对应 HTTP 201）。
	CodeCreated Code = 2001
	// CodeAccepted 请求已接受、异步处理中（对应 HTTP 202）。
	CodeAccepted Code = 2002

	// ---- 客户端错误 4xxx ----
	// CodeParamInvalid 参数校验失败（对应 HTTP 400）。
	CodeParamInvalid Code = 4000
	// CodeUnauthorized 未认证或令牌失效（对应 HTTP 401）。
	CodeUnauthorized Code = 4001
	// CodeForbidden 无权限（对应 HTTP 403）。
	CodeForbidden Code = 4003
	// CodeNotFound 资源不存在（对应 HTTP 404）。
	CodeNotFound Code = 4004
	// CodeConflict 资源冲突，如邮箱已注册（对应 HTTP 409）。
	CodeConflict Code = 4009
	// CodeTooManyRequests 请求过于频繁（对应 HTTP 429）。
	CodeTooManyRequests Code = 4029

	// ---- 服务端错误 5xxx ----
	// CodeInternal 服务内部错误（对应 HTTP 500）。
	CodeInternal Code = 5000
	// CodeServiceUnavail 下游服务暂不可用（对应 HTTP 503）。
	CodeServiceUnavail Code = 5003

	// ---- 用户与认证业务错误 41xx（仍属客户端错误 4xxx）----
	// CodeUserNotFound 用户不存在。
	CodeUserNotFound Code = 4100
	// CodeWrongPassword 邮箱或密码错误。
	CodeWrongPassword Code = 4101
	// CodeEmailRegistered 邮箱已被注册。
	CodeEmailRegistered Code = 4102
	// CodeTokenInvalid 令牌无效或已过期。
	CodeTokenInvalid Code = 4103
	// CodeTokenExpired 令牌已过期。
	CodeTokenExpired Code = 4104
	// CodeRefreshInvalid 刷新令牌无效。
	CodeRefreshInvalid Code = 4105
	// CodeDeviceMismatch 设备校验失败。
	CodeDeviceMismatch Code = 4106
	// CodeMissingToken 缺少访问令牌。
	CodeMissingToken Code = 4107
	// CodeMissingRefresh 缺少刷新令牌。
	CodeMissingRefresh Code = 4108
	// CodeMissingDevice 缺少设备标识。
	CodeMissingDevice Code = 4109
	// CodeAccountLocked 账号因多次登录失败被锁定。
	CodeAccountLocked Code = 4110
	// CodeAccountDisabled 账号已冻结或注销。
	CodeAccountDisabled Code = 4111
)

// IsSuccess 判断业务码是否表示成功（2xxx 段）。
func IsSuccess(code Code) bool {
	return int(code) >= segmentSuccess && int(code) < segmentClientError
}

// IsClientError 判断业务码是否为客户端错误（4xxx 段）。
func IsClientError(code Code) bool {
	return int(code) >= segmentClientError && int(code) < segmentServerError
}

// IsServerError 判断业务码是否为服务端错误（5xxx 段）。
func IsServerError(code Code) bool {
	return int(code) >= segmentServerError
}
