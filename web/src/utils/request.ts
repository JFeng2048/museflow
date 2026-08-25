import axios, { type AxiosRequestConfig } from "axios"

const baseURL = import.meta.env.VITE_API_BASE_URL || "/api"

const instance = axios.create({
  baseURL,
  timeout: 15000,
  headers: { "Content-Type": "application/json" },
})

// 占位拦截器：接入真实后端时在此挂载 token 注入与统一错误处理。
instance.interceptors.request.use((config) => {
  const token = localStorage.getItem("museflow:token")
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// 统一解包响应体，业务层拿到的就是 data 本身。
instance.interceptors.response.use(
  (response) => response.data,
  (error) => Promise.reject(error),
)

// 让 request 的方法直接返回解包后的数据，省去各业务层再取 .data。
export interface AppRequest {
  get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  head<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T>
  post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
  patch<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T>
}

const request = instance as unknown as AppRequest

export default request