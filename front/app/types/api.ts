/**
 * API 请求和响应类型定义
 */

/**
 * API 通用响应格式
 */
export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  message?: string
  error?: string
}

/**
 * 分页参数
 */
export interface PaginationParams {
  page?: number
  per_page?: number
}

/**
 * 分页响应数据
 */
export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  per_page: number
}
