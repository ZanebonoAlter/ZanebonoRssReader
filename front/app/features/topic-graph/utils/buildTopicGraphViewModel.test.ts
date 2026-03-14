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

  it('derives trunk node from top topic and classifies emphasis levels', () => {
    const viewModel = buildTopicGraphViewModel({
      type: 'daily',
      anchor_date: '2026-03-11',
      period_label: '2026-03-11 当日',
      topic_count: 3,
      summary_count: 2,
      feed_count: 2,
      top_topics: [
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
      ],
    })

    // Trunk node should be the hero topic
    expect(viewModel.trunkNode).not.toBeNull()
    expect(viewModel.trunkNode?.label).toBe('AI Agent')
    expect(viewModel.trunkNode?.id).toBe('ai-agent')

    // Branch nodes are directly connected to trunk
    expect(viewModel.branchNodeIds).toContain('openai')
    expect(viewModel.branchNodeIds).not.toContain('ai-agent')

    // Peripheral nodes are not connected to trunk
    expect(viewModel.peripheralNodeIds).toContain('feed-1')
    expect(viewModel.peripheralNodeIds).not.toContain('ai-agent')
    expect(viewModel.peripheralNodeIds).not.toContain('openai')

    // Emphasis levels are properly assigned
    expect(viewModel.emphasisLevels['ai-agent']).toBe('trunk')
    expect(viewModel.emphasisLevels['openai']).toBe('branch')
    expect(viewModel.emphasisLevels['feed-1']).toBe('peripheral')
  })

  it('sorts edges by weight for chronology presentation', () => {
    const viewModel = buildTopicGraphViewModel({
      type: 'daily',
      anchor_date: '2026-03-11',
      period_label: '2026-03-11 当日',
      topic_count: 3,
      summary_count: 2,
      feed_count: 2,
      top_topics: [
        { label: 'AI Agent', slug: 'ai-agent', kind: 'topic', score: 2.9 },
      ],
      nodes: [
        { id: 'ai-agent', label: 'AI Agent', slug: 'ai-agent', kind: 'topic', weight: 5.2, summary_count: 2 },
        { id: 'openai', label: 'OpenAI', slug: 'openai', kind: 'topic', weight: 4.1, summary_count: 2 },
      ],
      edges: [
        { id: 'weak', source: 'ai-agent', target: 'openai', kind: 'topic_topic', weight: 0.5 },
        { id: 'strong', source: 'ai-agent', target: 'openai', kind: 'topic_topic', weight: 3.5 },
      ],
    })

    // Edge chronology should be sorted by weight descending
    expect(viewModel.edgeChronology).toHaveLength(2)
    expect(viewModel.edgeChronology[0]?.weight).toBe(3.5)
    expect(viewModel.edgeChronology[1]?.weight).toBe(0.5)
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
    // New trunk/chronology fields must also be safe
    expect(viewModel.trunkNode).toBeNull()
    expect(viewModel.branchNodeIds).toEqual([])
    expect(viewModel.peripheralNodeIds).toEqual([])
    expect(viewModel.edgeChronology).toEqual([])
    expect(viewModel.emphasisLevels).toEqual({})
  })

  it('handles trunk derivation when hero does not match any node', () => {
    const viewModel = buildTopicGraphViewModel({
      type: 'daily',
      anchor_date: '2026-03-11',
      period_label: '2026-03-11 当日',
      topic_count: 2,
      summary_count: 1,
      feed_count: 1,
      top_topics: [
        { label: 'Unknown Topic', slug: 'unknown-topic', kind: 'topic', score: 2.9 },
      ],
      nodes: [
        { id: 'existing', label: 'Existing Node', slug: 'existing', kind: 'topic', weight: 3.0, summary_count: 1 },
      ],
      edges: [],
    })

    // Trunk should be null when no matching node exists
    expect(viewModel.trunkNode).toBeNull()
    // All nodes become peripheral when there's no trunk
    expect(viewModel.peripheralNodeIds).toContain('existing')
    expect(viewModel.branchNodeIds).toEqual([])
    expect(viewModel.emphasisLevels['existing']).toBe('peripheral')
  })
})
