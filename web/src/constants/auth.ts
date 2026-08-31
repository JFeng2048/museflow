/**
 * 认证相关常量。
 *
 * 单独成文件是为了让 request 拦截器与 user store 共用同一份定义，
 * 避免两边各写一份字符串导致令牌存了却读不到。
 */

/** access token 在 localStorage 中的存储键。 */
export const TOKEN_KEY = 'mf.token'
