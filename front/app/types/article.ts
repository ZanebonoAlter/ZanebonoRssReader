/**
 * 文章相关类型定义
 */

/**
 * 文章数据模型
 */
export interface Article {
  id: string
  feedId: string
  title: string
  description: string
  content: string
  link: string
  pubDate: string
  author?: string
  category: string
  read?: boolean
  favorite?: boolean
  content_status?: 'complete' | 'incomplete' | 'pending' | 'failed'
  full_content?: string
  content_fetched_at?: string
  completion_attempts?: number
  completion_error?: string
  ai_content_summary?: string
  imageUrl?: string
}

/**
 * 文章筛选条件
 */
export interface ArticleFilters {
  page?: number
  per_page?: number
  feed_id?: number
  category_id?: number
  uncategorized?: boolean
  read?: boolean
  favorite?: boolean
  search?: string
  start_date?: string
  end_date?: string
}

/**
 * 文章更新数据
 */
export interface UpdateArticleData {
  read?: boolean
  favorite?: boolean
}

/**
 * 批量更新文章数据
 */
export interface BulkUpdateArticlesData {
  ids: number[]
  read?: boolean
  favorite?: boolean
}
