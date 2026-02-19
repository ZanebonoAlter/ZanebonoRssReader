import { apiClient } from './client'
import type {
  ApiResponse,
  PaginatedApiResponse,
  Article,
  ArticleFilters,
  UpdateArticleData,
  BulkUpdateArticlesData,
} from '~/types'

/**
 * 文章 API
 */
export function useArticlesApi() {
  /**
   * 获取文章列表
   */
  async function getArticles(filters: ArticleFilters = {}): Promise<PaginatedApiResponse<Article>> {
    const query = apiClient.buildQueryParams(filters)
    return apiClient.get<Article[]>(`/articles${query ? `?${query}` : ''}`) as any
  }

  /**
   * 获取单篇文章
   */
  async function getArticle(id: number): Promise<ApiResponse<Article>> {
    return apiClient.get<Article>(`/articles/${id}`)
  }

  /**
   * 更新文章
   */
  async function updateArticle(
    id: number,
    data: UpdateArticleData
  ): Promise<ApiResponse<Article>> {
    return apiClient.put<Article>(`/articles/${id}`, data)
  }

  /**
   * 批量更新文章
   */
  async function bulkUpdateArticles(data: BulkUpdateArticlesData): Promise<ApiResponse<void>> {
    return apiClient.put<void>('/articles/bulk-update', data)
  }

  /**
   * 获取文章统计信息
   */
  async function getArticlesStats(): Promise<ApiResponse<{ total: number; unread: number; favorite: number }>> {
    return apiClient.get<{ total: number; unread: number; favorite: number }>('/articles/stats')
  }

  return {
    getArticles,
    getArticle,
    updateArticle,
    bulkUpdateArticles,
    getArticlesStats,
  }
}
