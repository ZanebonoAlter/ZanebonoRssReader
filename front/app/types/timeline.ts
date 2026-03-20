/**
 * Timeline type definitions for topic graph feature.
 */

import type { TopicCategory } from '~/api/topicGraph'

export type DateRangeFilter = 'today' | 'week' | 'month' | 'custom' | null

export interface TimelineFilters {
  dateRange: DateRangeFilter
  startDate?: string
  endDate?: string
  sources: string[]
}

export interface TimelineArticleTag {
  slug: string
  label: string
  category: TopicCategory
}

export interface TimelineDigestSourceArticle {
  id: number
  title: string
  link: string
}

export interface TimelineDigest {
  id: string
  title: string
  summary: string
  createdAt: string
  feedName: string
  categoryName: string
  articleCount: number
  tags: TimelineArticleTag[]
  articles: TimelineDigestSourceArticle[]
}

export interface TimelineDigestSelection extends TimelineDigest {
  matchedArticleIds: number[]
}
