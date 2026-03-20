/**
 * Article type definitions.
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
  summaryStatus?: 'complete' | 'incomplete' | 'pending' | 'failed'
  summaryGeneratedAt?: string
  completionAttempts?: number
  completionError?: string
  aiContentSummary?: string
  firecrawlStatus?: 'pending' | 'processing' | 'completed' | 'failed'
  firecrawlError?: string
  firecrawlContent?: string
  firecrawlCrawledAt?: string
  imageUrl?: string
}

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

export interface UpdateArticleData {
  read?: boolean
  favorite?: boolean
}

export interface BulkUpdateArticlesData {
  ids: number[]
  read?: boolean
  favorite?: boolean
}
