import { useArticlesApi } from '~/composables/api'
import type { ApiResponse, Article, ArticleFilters, UpdateArticleData } from '~/types'

/**
 * 文章服务
 * 封装文章相关的业务逻辑
 */
export function useArticleService() {
  const api = useArticlesApi()

  /**
   * 加载文章列表
   * @param filters - 筛选条件
   * @returns API 响应
   */
  async function loadArticles(filters: ArticleFilters = {}): Promise<ApiResponse<Article[]>> {
    return api.getArticles(filters)
  }

  /**
   * 加载单篇文章
   * @param id - 文章 ID
   * @returns API 响应
   */
  async function loadArticle(id: number): Promise<ApiResponse<Article>> {
    return api.getArticle(id)
  }

  /**
   * 更新文章
   * @param id - 文章 ID
   * @param data - 更新数据
   * @returns API 响应
   */
  async function updateArticle(
    id: number,
    data: UpdateArticleData
  ): Promise<ApiResponse<Article>> {
    return api.updateArticle(id, data)
  }

  /**
   * 标记文章为已读
   * @param id - 文章 ID
   * @returns API 响应
   */
  async function markAsRead(id: number): Promise<ApiResponse<Article>> {
    return api.updateArticle(id, { read: true })
  }

  /**
   * 切换文章收藏状态
   * @param id - 文章 ID
   * @param isFavorite - 是否收藏
   * @returns API 响应
   */
  async function toggleFavorite(
    id: number,
    isFavorite: boolean
  ): Promise<ApiResponse<Article>> {
    return api.updateArticle(id, { favorite: isFavorite })
  }

  /**
   * 标记所有文章为已读
   * @param feedId - 订阅源 ID（可选）
   * @returns API 响应
   */
  async function markAllAsRead(feedId?: number): Promise<ApiResponse<void>> {
    const articlesStore = useArticlesStore()
    const ids = feedId
      ? articlesStore.articles
          .filter((a) => a.feedId === String(feedId))
          .map((a) => Number(a.id))
      : articlesStore.articles.map((a) => Number(a.id))

    return api.bulkUpdateArticles({ ids, read: true })
  }

  /**
   * 获取文章统计信息
   * @returns API 响应
   */
  async function getStats(): Promise<ApiResponse<{ unread: number }>> {
    return api.getArticlesStats()
  }

  return {
    loadArticles,
    loadArticle,
    updateArticle,
    markAsRead,
    toggleFavorite,
    markAllAsRead,
    getStats,
  }
}
