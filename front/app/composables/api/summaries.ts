import { apiClient } from './client'
import type {
  ApiResponse,
  PaginatedApiResponse,
  AISummary,
  GenerateSummaryData,
  QueueSummaryRequest,
  SummaryBatch,
  SummaryJob,
} from '~/types'

/**
 * AI 摘要 API
 */
export function useSummariesApi() {
  /**
   * 获取 AI 摘要列表
   */
  async function getSummaries(params: {
    category_id?: number
    page?: number
    per_page?: number
    start_date?: string
    end_date?: string
  } = {}): Promise<PaginatedApiResponse<AISummary>> {
    const query = apiClient.buildQueryParams(params)
    return apiClient.get<AISummary[]>(`/summaries${query ? `?${query}` : ''}`) as any
  }

  /**
   * 获取单个 AI 摘要
   */
  async function getSummary(id: number): Promise<ApiResponse<AISummary>> {
    return apiClient.get<AISummary>(`/summaries/${id}`)
  }

  /**
   * 生成 AI 摘要
   */
  async function generateSummary(data: GenerateSummaryData): Promise<ApiResponse<AISummary>> {
    return apiClient.post<AISummary>('/summaries/generate', data)
  }

  /**
   * 自动生成 AI 摘要
   */
  async function autoGenerateSummary(
    data: GenerateSummaryData
  ): Promise<ApiResponse<AISummary>> {
    return apiClient.post<AISummary>('/summaries/auto-generate', data)
  }

  /**
   * 删除 AI 摘要
   */
  async function deleteSummary(id: number): Promise<ApiResponse<void>> {
    return apiClient.delete<void>(`/summaries/${id}`)
  }

  /**
   * 获取自动总结状态
   */
  async function getAutoSummaryStatus(): Promise<ApiResponse<any>> {
    return apiClient.get('/auto-summary/status')
  }

  /**
   * 更新自动总结配置
   */
  async function updateAutoSummaryConfig(data: {
    base_url: string
    api_key: string
    model: string
  }): Promise<ApiResponse<void>> {
    return apiClient.post('/auto-summary/config', data)
  }

  /**
   * 提交队列总结任务（多分类）
   */
  async function submitQueueSummary(
    data: QueueSummaryRequest
  ): Promise<ApiResponse<SummaryBatch>> {
    return apiClient.post<SummaryBatch>('/summaries/queue', data)
  }

  /**
   * 获取队列状态
   */
  async function getQueueStatus(): Promise<ApiResponse<SummaryBatch | null>> {
    return apiClient.get<SummaryBatch | null>('/summaries/queue/status')
  }

  /**
   * 获取单个任务详情
   */
  async function getQueueJob(jobId: string): Promise<ApiResponse<SummaryJob>> {
    return apiClient.get<SummaryJob>(`/summaries/queue/jobs/${jobId}`)
  }

  return {
    getSummaries,
    getSummary,
    generateSummary,
    autoGenerateSummary,
    deleteSummary,
    getAutoSummaryStatus,
    updateAutoSummaryConfig,
    submitQueueSummary,
    getQueueStatus,
    getQueueJob,
  }
}
