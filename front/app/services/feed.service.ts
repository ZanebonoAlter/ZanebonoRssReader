import { useFeedsApi } from '~/composables/api'
import type {
  ApiResponse,
  RssFeed,
  CreateFeedData,
  UpdateFeedData,
  PaginationParams,
} from '~/types'

/**
 * 订阅源服务
 * 封装订阅源相关的业务逻辑
 */
export function useFeedService() {
  const api = useFeedsApi()

  /**
   * 加载订阅源列表
   * @param params - 分页和筛选参数
   * @returns API 响应
   */
  async function loadFeeds(
    params: PaginationParams = {}
  ): Promise<ApiResponse<RssFeed[]>> {
    return api.getFeeds(params)
  }

  /**
   * 预览订阅源
   * @param url - RSS 订阅源地址
   * @returns API 响应
   */
  async function fetchFeed(url: string): Promise<ApiResponse<any>> {
    return api.fetchFeed(url)
  }

  /**
   * 创建订阅源
   * @param data - 订阅源数据
   * @returns API 响应
   */
  async function createFeed(data: CreateFeedData): Promise<ApiResponse<RssFeed>> {
    return api.createFeed(data)
  }

  /**
   * 更新订阅源
   * @param id - 订阅源 ID
   * @param data - 更新数据
   * @returns API 响应
   */
  async function updateFeed(
    id: number,
    data: UpdateFeedData
  ): Promise<ApiResponse<RssFeed>> {
    return api.updateFeed(id, data)
  }

  /**
   * 删除订阅源
   * @param id - 订阅源 ID
   * @returns API 响应
   */
  async function deleteFeed(id: number): Promise<ApiResponse<void>> {
    return api.deleteFeed(id)
  }

  /**
   * 刷新订阅源
   * @param id - 订阅源 ID
   * @returns API 响应
   */
  async function refreshFeed(id: number): Promise<ApiResponse<{ message?: string }>> {
    return api.refreshFeed(id)
  }

  /**
   * 刷新所有订阅源
   * @returns API 响应
   */
  async function refreshAllFeeds(): Promise<ApiResponse<{ message?: string }>> {
    return api.refreshAllFeeds()
  }

  return {
    loadFeeds,
    fetchFeed,
    createFeed,
    updateFeed,
    deleteFeed,
    refreshFeed,
    refreshAllFeeds,
  }
}
