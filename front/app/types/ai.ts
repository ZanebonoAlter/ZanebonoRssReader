/**
 * AI 相关类型定义
 */

/**
 * AI 总结数据模型
 */
export interface AISummary {
  id: number
  category_id: number | null
  title: string
  summary: string
  key_points: string
  articles: string
  article_count: number
  time_range: number
  created_at: string
  updated_at: string
  category_name: string
}

/**
 * AI 总结请求数据
 */
export interface AISummaryRequest {
  base_url: string
  api_key: string
  model: string
  title: string
  content: string
  language?: string
}

/**
 * AI 总结响应数据
 */
export interface AISummaryData {
  one_sentence: string
  key_points: string[]
  takeaways: string[]
  tags: string[]
}

/**
 * AI 生成总结请求数据
 */
export interface GenerateSummaryData {
  category_id?: number | null
  time_range?: number
  base_url: string
  api_key: string
  model: string
}

/**
 * AI 设置数据
 */
export interface AISettings {
  baseURL: string
  apiKey: string
  model: string
  summaryEnabled?: boolean
}
