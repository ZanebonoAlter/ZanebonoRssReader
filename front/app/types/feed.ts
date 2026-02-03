/**
 * 订阅源相关类型定义
 */

/**
 * 订阅源数据模型
 */
export interface RssFeed {
  id: string
  title: string
  description: string
  url: string
  category: string
  icon?: string
  color?: string
  lastUpdated: string
  articleCount: number
  unreadCount?: number
  maxArticles?: number
  refreshInterval?: number
  refreshStatus?: 'idle' | 'refreshing' | 'success' | 'error'
  refreshError?: string
  lastRefreshAt?: string
  aiSummaryEnabled?: boolean
}

/**
 * 订阅源创建数据
 */
export interface CreateFeedData {
  url: string
  category_id?: number
  title?: string
  description?: string
  icon?: string
  color?: string
}

/**
 * 订阅源更新数据
 */
export interface UpdateFeedData {
  url?: string
  category_id?: number
  title?: string
  description?: string
  icon?: string
  color?: string
  max_articles?: number
  refresh_interval?: number
  ai_summary_enabled?: boolean
}

/**
 * RSS 响应数据
 */
export interface FeedResponse {
  status: string
  feed: {
    title: string
    description: string
    image?: string
  }
  items: FeedItem[]
}

/**
 * RSS 文章条目
 */
export interface FeedItem {
  title: string
  link: string
  pubDate: string
  description?: string
  content?: string
  author?: string
  thumbnail?: string
  enclosure?: {
    link: string
  }
}
