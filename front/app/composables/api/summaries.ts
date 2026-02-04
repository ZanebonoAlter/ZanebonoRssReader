import { apiClient } from './client'
import type {
  ApiResponse,
  PaginatedApiResponse,
  AISummary,
  GenerateSummaryData,
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

  return {
    getSummaries,
    getSummary,
    generateSummary,
    autoGenerateSummary,
    deleteSummary,
    getAutoSummaryStatus,
    updateAutoSummaryConfig,
  }
}
