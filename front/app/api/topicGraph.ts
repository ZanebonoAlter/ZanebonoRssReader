import { apiClient } from './client'

export type TopicGraphType = 'daily' | 'weekly'

export interface TopicTag {
  label: string
  slug: string
  kind: 'topic' | 'entity'
  score: number
}

export interface TopicGraphNode {
  id: string
  label: string
  slug?: string
  kind: 'topic' | 'feed'
  weight: number
  summary_count?: number
  color?: string
  feed_name?: string
  category_name?: string
}

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
  nodes: TopicGraphNode[]
  edges: TopicGraphEdge[]
  topic_count: number
  summary_count: number
  feed_count: number
  top_topics: TopicTag[]
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

export interface TopicGraphDetailPayload {
  topic: TopicTag
  summaries: TopicGraphSummaryCard[]
  history: TopicHistoryPoint[]
  related_topics: TopicTag[]
  search_links: Record<string, string>
  app_links: Record<string, string>
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
  }
}
