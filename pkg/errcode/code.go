// Package errcode 定义所有服务共用的业务码（0-7999）及统一响应封装。
//
// 业务码分区约定：
//   - 0      : 成功
//   - 1000-1999: 通用/系统级错误
//   - 2000-2999: 用户与认证服务相关错误
//   - 3000-7999: 预留给其它服务
package errcode

// Code 业务码类型。0 表示成功。
type Code int

// 通用业务码。
const (
	// CodeSuccess 成功。
	CodeSuccess Code = 0

	// 通用/系统级错误 1000-1999。
	CodeParamInvalid    Code = 1000 // 参数校验失败
	CodeUnauthorized    Code = 1001 // 未认证或令牌失效
	CodeForbidden       Code = 1002 // 无权限
	CodeNotFound        Code = 1003 // 资源不存在
	CodeConflict        Code = 1004 // 资源冲突（如邮箱已注册）
	CodeTooManyRequests Code = 1005 // 请求过于频繁
	CodeInternal        Code = 1099 // 服务内部错误
	CodeServiceUnavail  Code = 1098 // 下游服务暂不可用

	// 用户与认证服务 2000-2999。
	CodeUserNotFound     Code = 2000 // 用户不存在
	CodeWrongPassword    Code = 2001 // 邮箱或密码错误
	CodeEmailRegistered  Code = 2002 // 邮箱已被注册
	CodeTokenInvalid     Code = 2003 // 令牌无效或已过期
	CodeTokenExpired     Code = 2004 // 令牌已过期
	CodeRefreshInvalid   Code = 2005 // 刷新令牌无效
	CodeDeviceMismatch   Code = 2006 // 设备校验失败
	CodeMissingToken     Code = 2007 // 缺少访问令牌
	CodeMissingRefresh   Code = 2008 // 缺少刷新令牌
	CodeMissingDevice    Code = 2009 // 缺少设备标识
)
