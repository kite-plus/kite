/**
 * Go 后端统一响应结构
 * 所有 API 接口均返回此格式
 */
export interface ApiResponse<T = unknown> {
  code: number
  data: T
  msg: string
}
