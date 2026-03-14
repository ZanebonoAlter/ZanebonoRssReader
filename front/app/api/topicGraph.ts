import { apiClient } from './client'
import type { TimelineFilters } from '~/types/timeline'

export type TopicGraphType = 'daily' | 'weekly'
export type TopicCategory = 'event' | 'person' | 'keyword'
export type AnalysisType = 'event' | 'person' | 'keyword'
export type TopicAnalysisType = AnalysisType
export type TopicKind = 'topic' | 'entity' | 'keyword'

export interface AnalysisStatusResponse {
  status: 'pending' | 'processing' | 'completed' | 'failed'
  progress: number
  error: string | null
  result: any | null
}

export interface RebuildAnalysisRequest {
  windowType: string
  anchorDate: string
}

export interface TopicTag {
  id?: number
  label: string
  slug: string
  category: TopicCategory
  kind?: TopicKind
  icon?: string
  aliases?: string[]
  score: number
}

export interface GraphNode {
  id: string
  label: string
  slug?: string
  kind: 'topic' | 'feed'
  category?: TopicCategory
  icon?: string
  weight: number
  summary_count?: number
  color?: string
  feed_name?: string
  category_name?: string
}

export type TopicGraphNode = GraphNode

export interface TopicGraphEdge {
  id: string
  source: string
  target: string
  kind: 'topic_topic' | 'topic_feed'
  weight: number
}

export interface TopicGraphPayload {
  type: TopicGraphType
  anchor_date: string
  period_label: string
  nodes: GraphNode[]
  edges: TopicGraphEdge[]
  topic_count: number
  summary_count: number
  feed_count: number
  top_topics: TopicTag[]
}

export interface TopicsByCategoryPayload {
  events: TopicTag[]
  people: TopicTag[]
  keywords: TopicTag[]
}

export interface TopicGraphSummaryCard {
  id: number
  title: string
  summary: string
  feed_name: string
  feed_color: string
  category_name: string
  article_count: number
  created_at: string
  topics: TopicTag[]
  articles: Array<{
    id: number
    title: string
    link: string
  }>
}

export interface TopicHistoryPoint {
  anchor_date: string
  count: number
  label: string
}

export interface RelatedTag {
  id: number
  label: string
  slug: string
  category: TopicCategory
  kind?: TopicKind
  cooccurrence: number
}

export interface TopicGraphDetailPayload {
  topic: TopicTag
  articles: Array<{
    id: number
    title: string
    link: string
  }>
  total_articles: number
  related_tags: RelatedTag[]
  summaries: TopicGraphSummaryCard[]
  history: TopicHistoryPoint[]
  related_topics: TopicTag[]
  search_links: Record<string, string>
  app_links: Record<string, string>
}

export interface GetTopicAnalysisParams {
  tagID: number
  analysisType: TopicAnalysisType
  windowType: TopicGraphType
  anchorDate: string
}

export interface TopicAnalysisRecord {
  id: number
  topic_tag_id: number
  analysis_type: TopicAnalysisType
  window_type: TopicGraphType
  anchor_date: string
  summary_count: number
  payload_json: string
  source: string
  version: number
  created_at: string
  updated_at: string
}

export interface RebuildAnalysisParams extends RebuildAnalysisRequest {
  tagID: number
  analysisType: TopicAnalysisType
}

export interface TopicAnalysisStatusRecord {
  status: AnalysisStatusResponse['status'] | 'missing' | 'ready'
  progress: number
  error: string | null
  result: any | null
}

export interface GetTopicArticlesParams {
  slug: string
  page?: number
  pageSize?: number
  dateRange?: TimelineFilters['dateRange']
  startDate?: string
  endDate?: string
  sources?: string[]
}

export interface TopicArticlesResponse {
  articles: Array<{
    id: string
    title: string
    summary: string
    content?: string
    pubDate: string
    feedName: string
    feedId: string
    tags: Array<{
      slug: string
      label: string
      category: TopicCategory
    }>
    imageUrl?: string
    link: string
  }>
  total: number
  page: number
  pageSize: number
}

function withQuery(endpoint: string, params: Record<string, string | undefined>) {
  const query = apiClient.buildQueryParams(params)
  return query ? `${endpoint}?${query}` : endpoint
}

export function useTopicGraphApi() {
  return {
    async getGraph(type: TopicGraphType, date?: string) {
      return apiClient.get<TopicGraphPayload>(withQuery(`/topic-graph/${type}`, { date }))
    },

    async getTopicDetail(slug: string, type: TopicGraphType, date?: string) {
      return apiClient.get<TopicGraphDetailPayload>(withQuery(`/topic-graph/topic/${slug}`, { type, date }))
    },

    async getTopicAnalysis(params: GetTopicAnalysisParams) {
      return apiClient.get<TopicAnalysisRecord>(withQuery('/topic-graph/analysis', {
        tag_id: String(params.tagID),
        analysis_type: params.analysisType,
        window_type: params.windowType,
        anchor_date: params.anchorDate,
      }))
    },

    async rebuildTopicAnalysis(params: RebuildAnalysisParams) {
      return apiClient.post<TopicAnalysisRecord>(withQuery('/topic-graph/analysis/rebuild', {
        tag_id: String(params.tagID),
        analysis_type: params.analysisType,
        window_type: params.windowType,
        anchor_date: params.anchorDate,
      }), {})
    },

    async getAnalysisStatus(params: GetTopicAnalysisParams) {
      return apiClient.get<TopicAnalysisStatusRecord>(withQuery('/topic-graph/analysis/status', {
        tag_id: String(params.tagID),
        analysis_type: params.analysisType,
        window_type: params.windowType,
        anchor_date: params.anchorDate,
      }))
    },

    async retryTopicAnalysis(params: RebuildAnalysisParams) {
      return apiClient.post<TopicAnalysisRecord>(withQuery('/topic-graph/analysis/retry', {
        tag_id: String(params.tagID),
        analysis_type: params.analysisType,
        window_type: params.windowType,
        anchor_date: params.anchorDate,
      }), {})
    },

    async getTopicsByCategory(type: TopicGraphType, date?: string) {
      return apiClient.get<TopicsByCategoryPayload>(withQuery('/topic-graph/by-category', { type, date }))
    },

    async getTopicArticles(params: GetTopicArticlesParams) {
      const queryParams: Record<string, string | undefined> = {
        page: params.page ? String(params.page) : undefined,
        page_size: params.pageSize ? String(params.pageSize) : undefined,
        date_range: params.dateRange || undefined,
        start_date: params.startDate,
        end_date: params.endDate,
      }

      if (params.sources && params.sources.length > 0) {
        queryParams.sources = params.sources.join(',')
      }

      return apiClient.get<TopicArticlesResponse>(
        withQuery(`/topic-graph/topic/${params.slug}/articles`, queryParams)
      )
    },
  }
}
