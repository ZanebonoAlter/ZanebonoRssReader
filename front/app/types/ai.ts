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
  category_ids?: number[]  // 多分类选择
  time_range?: number
  base_url: string
  api_key: string
  model: string
}

/**
 * 总结任务状态
 */
export type SummaryJobStatus = 'pending' | 'processing' | 'completed' | 'failed'

/**
 * 单个总结任务
 */
export interface SummaryJob {
  id: string
  batch_id: string
  category_id: number | null
  category_name: string
  status: SummaryJobStatus
  error_message?: string
  error_code?: string
  result_id?: number
  created_at: string
  updated_at: string
  completed_at?: string
}

/**
 * 总结批次
 */
export interface SummaryBatch {
  id: string
  status: 'pending' | 'processing' | 'completed'
  total_jobs: number
  completed_jobs: number
  failed_jobs: number
  created_at: string
  completed_at?: string
  jobs: SummaryJob[]
}

/**
 * 队列总结请求数据
 */
export interface QueueSummaryRequest {
  category_ids: number[]
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
