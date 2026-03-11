import { describe, expect, it } from 'vitest'

import { buildTopicGraphViewModel } from './buildTopicGraphViewModel'

describe('buildTopicGraphViewModel', () => {
  it('sorts spotlight topics by weight and filters weak edges', () => {
    const viewModel = buildTopicGraphViewModel({
      type: 'daily',
      anchor_date: '2026-03-11',
      period_label: '2026-03-11 当日',
      topic_count: 3,
      summary_count: 2,
      feed_count: 2,
      top_topics: [
        { label: 'OpenAI', slug: 'openai', kind: 'entity', score: 2.4 },
        { label: 'AI Agent', slug: 'ai-agent', kind: 'topic', score: 2.9 },
      ],
      nodes: [
        { id: 'ai-agent', label: 'AI Agent', slug: 'ai-agent', kind: 'topic', weight: 5.2, summary_count: 2 },
        { id: 'openai', label: 'OpenAI', slug: 'openai', kind: 'topic', weight: 4.1, summary_count: 2 },
        { id: 'feed-1', label: 'OpenAI Blog', kind: 'feed', weight: 1.8, color: '#3b6b87', feed_name: 'OpenAI Blog', category_name: 'AI' },
      ],
      edges: [
        { id: 'ai-agent::openai', source: 'ai-agent', target: 'openai', kind: 'topic_topic', weight: 2.2 },
        { id: 'openai::feed-1', source: 'openai', target: 'feed-1', kind: 'topic_feed', weight: 1.1 },
        { id: 'too-weak', source: 'feed-1', target: 'ai-agent', kind: 'topic_feed', weight: 0.1 },
      ],
    })

    expect(viewModel.stats.heroLabel).toBe('AI Agent')
    expect(viewModel.graph.edges).toHaveLength(2)
    expect(viewModel.graph.nodes[0]!.size).toBeGreaterThan(viewModel.graph.nodes[2]!.size)
    expect(viewModel.graph.featuredNodeIds).toContain('ai-agent')
    expect(viewModel.graph.featuredNodeIds).toContain('openai')
  })

  it('creates a safe empty state when the graph payload is empty', () => {
    const viewModel = buildTopicGraphViewModel({
      type: 'weekly',
      anchor_date: '2026-03-11',
      period_label: '03-10 - 03-16',
      topic_count: 0,
      summary_count: 0,
      feed_count: 0,
      top_topics: [],
      nodes: [],
      edges: [],
    })

    expect(viewModel.stats.heroLabel).toBe('还没有图谱')
    expect(viewModel.graph.nodes).toEqual([])
    expect(viewModel.graph.edges).toEqual([])
    expect(viewModel.graph.featuredNodeIds).toEqual([])
  })
})
