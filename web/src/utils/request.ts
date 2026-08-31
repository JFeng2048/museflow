import axios, { type AxiosRequestConfig } from 'axios'
import { TOKEN_KEY } from '@/constants/auth'

/**
 * 后端网关的全局路由前缀为 /api/v1（services/api-gateway/internal/router/router.go），
 * 因此默认 baseURL 带 /v1；业务层写相对路径即可（如 `/auth/login`）。
 */
const baseURL = import.meta.env.VITE_API_BASE_URL || '/api/v1'

const instance = axios.create({
  baseURL,
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
  // 刷新令牌由后端写入 HttpOnly Cookie，跨域请求必须携带凭证
  withCredentials: true,
})

instance.interceptors.request.use((config) => {
  const token = localStorage.getItem(TOKEN_KEY)
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

/** 后端统一响应信封（pkg/errcode.Response）。 */
export interface ApiEnvelope<T = unknown> {
  code: number
  message: string
  data?: T
}

/** 业务码分段：2xxx 成功（CodeSuccess=2000 / Created=2001 / Accepted=2002）。 */
const SEGMENT_SUCCESS = 2000
const SEGMENT_CLIENT = 3000

function isEnvelope(body: unknown): body is ApiEnvelope {
  return !!body && typeof body === 'object' && typeof (body as ApiEnvelope).code === 'number'
}

function isSuccessCode(code: number) {
  return code >= SEGMENT_SUCCESS && code < SEGMENT_CLIENT
}

/** 附加业务码与 HTTP 状态码的错误，便于业务层按 code 分支处理。 */
export interface ApiError extends Error {
  code?: number
  status?: number
}

function toApiError(message: string, code?: number, status?: number): ApiError {
  return Object.assign(new Error(message), { code, status })
}

/**
 * 拆信封：成功返回 data 本身，业务失败抛 Error。
 * 非信封响应（健康检查等）原样返回，避免误伤。
 */
export function unwrap<T>(body: unknown): T {
  if (!isEnvelope(body)) return body as T
  if (isSuccessCode(body.code)) return body.data as T
  throw toApiError(body.message || '请求失败', body.code)
}

/** 网络/HTTP 错误归一化：优先取后端信封里的中文提示。 */
function normalizeError(error: unknown): ApiError {
  if (axios.isAxiosError(error)) {
    const body = error.response?.data
    const status = error.response?.status
    if (isEnvelope(body)) {
      return toApiError(body.message || error.message, body.code, status)
    }
    return toApiError(error.message, undefined, status)
  }
  if (error instanceof Error) return error
  return toApiError(String(error))
}

/**
 * 访问令牌过期时（HTTP 401）用 HttpOnly Cookie 里的刷新令牌换发新令牌。
 * 并发请求共享同一次刷新（single-flight），避免多个 401 同时触发多次刷新。
 */
let refreshing: Promise<string> | null = null

function refreshAccessToken(): Promise<string> {
  if (!refreshing) {
    refreshing = instance
      .post<ApiEnvelope<{ access_token: string }>>('/common/refresh')
      .then((res) => {
        const data = unwrap<{ access_token: string }>(res.data)
        if (!data?.access_token) throw toApiError('登录已过期，请重新登录')
        localStorage.setItem(TOKEN_KEY, data.access_token)
        return data.access_token
      })
      .finally(() => {
        refreshing = null
      })
  }
  return refreshing
}

/**
 * 这些接口返回 401 属于「账号密码错误 / 票据失效」等正常业务结果，
 * 不应触发刷新重试，否则会把登录失败伪装成刷新失败。
 */
const NO_REFRESH_RETRY = [
  /^\/common\/refresh/,
  /^\/auth\/login/,
  /^\/auth\/register/,
  /^\/auth\/mfa\/verify-login/,
  /^\/auth\/password\/reset/,
  /^\/auth\/logout/,
]

function canRefresh(url: string) {
  return !NO_REFRESH_RETRY.some((re) => re.test(url))
}

/**
 * 执行一次请求：拆信封；401 且允许刷新时先换令牌再重试一次。
 * 刷新本身失败会直接向上抛出，由调用方（登录态失效处理）接管。
 */
async function call<T>(url: string, send: () => Promise<{ data: unknown }>): Promise<T> {
  try {
    const res = await send()
    return unwrap<T>(res.data)
  } catch (error) {
    const apiError = normalizeError(error)
    if (apiError.status === 401 && canRefresh(url)) {
      await refreshAccessToken()
      const res = await send()
      return unwrap<T>(res.data)
    }
    throw apiError
  }
}

/** 让 request 的方法直接返回解包后的数据，省去各业务层再取 .data。 */
export interface AppRequest {
  get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  head<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
}

const request: AppRequest = {
  get: (url, config) => call(url, () => instance.get(url, config)),
  delete: (url, config) => call(url, () => instance.delete(url, config)),
  head: (url, config) => call(url, () => instance.head(url, config)),
  post: (url, data, config) => call(url, () => instance.post(url, data, config)),
  put: (url, data, config) => call(url, () => instance.put(url, data, config)),
  patch: (url, data, config) => call(url, () => instance.patch(url, data, config)),
}

export default request
