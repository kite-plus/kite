/**
 * API 客户端基础设施
 * 封装 fetch 调用后端 API，处理统一响应、camelCase/snake_case 转换和 401 自动跳转
 */
import type { ApiResponse } from '@/types/api'

/** API 基础路径 */
const API_BASE = '/api/v1'

/** 从 cookie 中读取 CSRF token */
function getCSRFToken(): string {
  const match = document.cookie.match(/(^|;\s*)csrf_token=([^;]+)/)
  return match ? match[2] : ''
}

/** snake_case → camelCase */
function toCamelCase(str: string): string {
  return str.replace(/_([a-z])/g, (_, c) => c.toUpperCase())
}

/** camelCase → snake_case */
function toSnakeCase(str: string): string {
  return str.replace(/[A-Z]/g, (c) => `_${c.toLowerCase()}`)
}

/** 递归转换对象的 key */
function convertKeys(obj: unknown, converter: (key: string) => string): unknown {
  if (obj === null || obj === undefined) return obj
  if (Array.isArray(obj)) return obj.map((item) => convertKeys(item, converter))
  if (typeof obj === 'object') {
    const result: Record<string, unknown> = {}
    for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
      result[converter(key)] = convertKeys(value, converter)
    }
    return result
  }
  return obj
}

/** 将后端 snake_case 响应转为前端 camelCase */
export function camelizeResponse<T>(data: unknown): T {
  return convertKeys(data, toCamelCase) as T
}

/** 将前端 camelCase 请求体转为后端 snake_case */
export function snakeifyRequest(data: unknown): unknown {
  return convertKeys(data, toSnakeCase)
}

/** API 错误类 */
export class ApiError extends Error {
  code: number
  constructor(code: number, msg: string) {
    super(msg)
    this.code = code
    this.name = 'ApiError'
  }
}

/**
 * 处理 401 未授权：自动跳转登录页
 * 排除 /auth/me 和 /auth/login 请求，避免循环跳转
 */
function handleUnauthorized(path: string): void {
  // 排除鉴权相关路径，避免循环
  if (path.includes('/auth/')) return
  // 获取当前 base path
  const base = import.meta.env.BASE_URL.replace(/\/$/, '')
  window.location.href = `${base}/login`
}

/**
 * 通用请求处理：解析响应、处理 401、抛出错误
 */
async function handleResponse<T>(res: Response, path: string): Promise<T> {
  // 401 且非鉴权接口：自动跳转登录页
  if (res.status === 401 && !path.includes('/auth/')) {
    handleUnauthorized(path)
    throw new ApiError(401, 'unauthorized')
  }

  const json: ApiResponse<unknown> = await res.json()

  if (json.code < 200 || json.code >= 300) {
    throw new ApiError(json.code, json.msg)
  }

  return camelizeResponse<T>(json.data)
}

/**
 * 通用 GET 请求
 */
export async function apiGet<T>(path: string, params?: Record<string, string | number | undefined>): Promise<T> {
  const url = new URL(`${API_BASE}${path}`, window.location.origin)
  if (params) {
    for (const [key, value] of Object.entries(params)) {
      if (value !== undefined && value !== '') {
        url.searchParams.set(toSnakeCase(key), String(value))
      }
    }
  }

  const res = await fetch(url.toString())
  return handleResponse<T>(res, path)
}

/**
 * 通用 POST 请求
 */
export async function apiPost<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRFToken() },
    body: body ? JSON.stringify(snakeifyRequest(body)) : undefined,
  })
  return handleResponse<T>(res, path)
}

/**
 * 通用 PUT 请求
 */
export async function apiPut<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRFToken() },
    body: body ? JSON.stringify(snakeifyRequest(body)) : undefined,
  })
  return handleResponse<T>(res, path)
}

/**
 * 通用 PATCH 请求
 */
export async function apiPatch<T>(path: string, body?: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCSRFToken() },
    body: body ? JSON.stringify(snakeifyRequest(body)) : undefined,
  })
  return handleResponse<T>(res, path)
}

/**
 * 通用 DELETE 请求
 */
export async function apiDelete<T = void>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: 'DELETE',
    headers: { 'X-CSRF-Token': getCSRFToken() },
  })
  return handleResponse<T>(res, path)
}

/** 上传结果类型 */
export interface UploadResult {
  url: string
  filename: string
  size: number
}

/**
 * 上传图片
 */
export async function apiUpload(file: File): Promise<UploadResult> {
  const formData = new FormData()
  formData.append('file', file)

  const res = await fetch(`${API_BASE}/admin/upload/image`, {
    method: 'POST',
    headers: { 'X-CSRF-Token': getCSRFToken() },
    body: formData,
  })
  return handleResponse<UploadResult>(res, '/admin/upload/image')
}
