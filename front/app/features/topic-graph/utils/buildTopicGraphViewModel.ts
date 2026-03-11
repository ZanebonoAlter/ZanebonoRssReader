import type { TopicGraphEdge, TopicGraphNode, TopicGraphPayload, TopicTag } from '~/api/topicGraph'

export interface TopicGraphSceneNode extends TopicGraphNode {
  size: number
  accent: string
  x?: number
  y?: number
  z?: number
}

export interface TopicGraphSceneEdge extends TopicGraphEdge {
  opacity: number
}

export interface TopicGraphViewModel {
  graph: {
    nodes: TopicGraphSceneNode[]
    edges: TopicGraphSceneEdge[]
    featuredNodeIds: string[]
  }
  stats: {
    heroLabel: string
    heroSubline: string
    topicCount: string
    summaryCount: string
    feedCount: string
  }
  topTopics: TopicTag[]
}

const TOPIC_COLOR = '#f08a4b'
const ENTITY_COLOR = '#3f7cff'
const FEED_COLOR = '#7b8a96'

export function buildTopicGraphViewModel(payload: TopicGraphPayload): TopicGraphViewModel {
  const topTopics = [...payload.top_topics].sort((left, right) => right.score - left.score)
  const nodes = payload.nodes.map((node) => ({
    ...node,
    size: buildNodeSize(node),
    accent: node.kind === 'feed'
      ? node.color || FEED_COLOR
      : node.label === node.label.toUpperCase()
        ? ENTITY_COLOR
        : TOPIC_COLOR,
  }))

  const edges = payload.edges
    .filter(edge => edge.weight >= 0.35)
    .map(edge => ({
      ...edge,
      opacity: edge.kind === 'topic_topic' ? 0.82 : 0.48,
    }))

  const hero = topTopics[0]

  return {
    graph: {
      nodes,
      edges,
      featuredNodeIds: nodes
        .filter(node => node.kind === 'topic')
        .sort((left, right) => right.weight - left.weight)
        .slice(0, 6)
        .map(node => node.id),
    },
    stats: {
      heroLabel: hero?.label || '还没有图谱',
      heroSubline: hero ? `${payload.period_label} 里最活跃的话题` : '先生成一些 AI 总结，图谱才会长出来。',
      topicCount: String(payload.topic_count ?? 0),
      summaryCount: String(payload.summary_count ?? 0),
      feedCount: String(payload.feed_count ?? 0),
    },
    topTopics,
  }
}

function buildNodeSize(node: TopicGraphNode) {
  const base = node.kind === 'feed' ? 5 : 8
  return Math.max(base, Math.round(base + node.weight * 2.2))
}
