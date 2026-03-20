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

export function useSummariesApi() {
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

  async function getSummary(id: number): Promise<ApiResponse<AISummary>> {
    return apiClient.get<AISummary>(`/summaries/${id}`)
  }

  async function generateSummary(data: GenerateSummaryData): Promise<ApiResponse<AISummary>> {
    return apiClient.post<AISummary>('/summaries/generate', data)
  }

  async function autoGenerateSummary(data: GenerateSummaryData): Promise<ApiResponse<AISummary>> {
    return apiClient.post<AISummary>('/summaries/auto-generate', data)
  }

  async function deleteSummary(id: number): Promise<ApiResponse<void>> {
    return apiClient.delete<void>(`/summaries/${id}`)
  }

  async function getAutoSummaryStatus(): Promise<ApiResponse<any>> {
    return apiClient.get('/auto-summary/status')
  }

  async function updateAutoSummaryConfig(data: {
    base_url?: string
    api_key?: string
    model?: string
    time_range?: number
  }): Promise<ApiResponse<void>> {
    return apiClient.post('/auto-summary/config', data)
  }

  async function submitQueueSummary(data: QueueSummaryRequest): Promise<ApiResponse<SummaryBatch>> {
    return apiClient.post<SummaryBatch>('/summaries/queue', data)
  }

  async function getQueueStatus(): Promise<ApiResponse<SummaryBatch | null>> {
    return apiClient.get<SummaryBatch | null>('/summaries/queue/status')
  }

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
