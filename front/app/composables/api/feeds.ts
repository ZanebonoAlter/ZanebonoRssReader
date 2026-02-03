import { apiClient } from './client'
import type {
  ApiResponse,
  CreateFeedData,
  UpdateFeedData,
  RssFeed,
  PaginationParams,
  PaginatedResponse,
} from '~/types'

/**
 * 订阅源 API
 */
export function useFeedsApi() {
  /**
   * 获取订阅源列表
   */
  async function getFeeds(params: PaginationParams = {}): Promise<ApiResponse<RssFeed[]>> {
    const query = apiClient.buildQueryParams(params)
    return apiClient.get<RssFeed[]>(`/feeds${query ? `?${query}` : ''}`)
  }

  /**
   * 获取订阅源信息（预览）
   */
  async function fetchFeed(url: string): Promise<ApiResponse<any>> {
    return apiClient.post('/feeds/fetch', { url })
  }

  /**
   * 创建订阅源
   */
  async function createFeed(data: CreateFeedData): Promise<ApiResponse<RssFeed>> {
    return apiClient.post<RssFeed>('/feeds', data)
  }

  /**
   * 更新订阅源
   */
  async function updateFeed(id: number, data: UpdateFeedData): Promise<ApiResponse<RssFeed>> {
    return apiClient.put<RssFeed>(`/feeds/${id}`, data)
  }

  /**
   * 删除订阅源
   */
  async function deleteFeed(id: number): Promise<ApiResponse<void>> {
    return apiClient.delete<void>(`/feeds/${id}`)
  }

  /**
   * 刷新订阅源
   */
  async function refreshFeed(id: number): Promise<ApiResponse<{ message?: string }>> {
    return apiClient.post<{ message?: string }>(`/feeds/${id}/refresh`)
  }

  /**
   * 刷新所有订阅源
   */
  async function refreshAllFeeds(): Promise<ApiResponse<{ message?: string }>> {
    return apiClient.post<{ message?: string }>('/feeds/refresh-all')
  }

  return {
    getFeeds,
    fetchFeed,
    createFeed,
    updateFeed,
    deleteFeed,
    refreshFeed,
    refreshAllFeeds,
  }
}
