// RSS Feed Types
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

// Article Types
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
  read: boolean
  favorite: boolean
  imageUrl?: string
}

// Category Types
export interface Category {
  id: string
  name: string
  slug: string
  icon: string
  color: string
  description: string
  feedCount: number
}

// Filter and Sort Types
export type SortOption = 'latest' | 'popular' | 'unread'
export type FilterOption = 'all' | 'unread' | 'favorites'

export interface FilterState {
  sort: SortOption
  filter: FilterOption
  category: string | null
  search: string
}

// API Response Types
export interface RssResponse {
  feed: RssFeed
  articles: Article[]
}

export interface FeedResponse {
  status: string
  feed: {
    title: string
    description: string
    image?: string
  }
  items: Array<{
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
  }>
}
