import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TopicTimeline from './TopicTimeline.vue'
import type { TopicCategory } from '~/api/topicGraph'
import type { TimelineDigest, TimelineFilters } from '~/types/timeline'

function createMockItem(overrides: Partial<TimelineDigest> = {}): TimelineDigest {
  return {
    id: 'digest-1',
    title: 'AI Agent 日报',
    summary: '整理当天话题与来源文章。',
    createdAt: '2026-03-14T08:30:00+08:00',
    feedName: 'OpenAI Blog',
    categoryName: 'AI',
    articleCount: 2,
    tags: [
      { slug: 'ai-agent', label: 'AI Agent', category: 'keyword' },
    ],
    articles: [
      { id: 1, title: 'Source 1', link: 'https://example.com/1' },
    ],
    ...overrides,
  }
}

function createMockTopic(overrides: { slug?: string; label?: string; category?: TopicCategory } = {}) {
  return {
    slug: 'ai-agent',
    label: 'AI Agent',
    category: 'keyword' as TopicCategory,
    ...overrides,
  }
}

const defaultFilters: TimelineFilters = {
  dateRange: null,
  sources: [],
}

describe('TopicTimeline', () => {
  it('renders digest items', () => {
    const wrapper = mount(TopicTimeline, {
      global: {
        stubs: {
          TimelineHeader: true,
          TimelineItem: true,
          AIAnalysisPanel: true,
        },
      },
      props: {
        selectedTopic: createMockTopic(),
        items: [createMockItem(), createMockItem({ id: 'digest-2' })],
        filters: defaultFilters,
      },
    })

    expect(wrapper.find('.topic-timeline').exists()).toBe(true)
    expect(wrapper.find('.timeline-list').exists()).toBe(true)
    expect(wrapper.findAllComponents({ name: 'TimelineItem' })).toHaveLength(2)
  })

  it('shows empty state without selected topic', () => {
    const wrapper = mount(TopicTimeline, {
      global: {
        stubs: {
          TimelineHeader: true,
          AIAnalysisPanel: true,
        },
      },
      props: {
        selectedTopic: null,
        items: [],
        filters: defaultFilters,
      },
    })

    expect(wrapper.find('.timeline-empty').text()).toContain('请先选择')
  })

  it('shows empty state when no digest items', () => {
    const wrapper = mount(TopicTimeline, {
      global: {
        stubs: {
          TimelineHeader: true,
          AIAnalysisPanel: true,
        },
      },
      props: {
        selectedTopic: createMockTopic(),
        items: [],
        filters: defaultFilters,
      },
    })

    expect(wrapper.find('.timeline-empty').text()).toContain('没有日报')
  })

  it('emits filter-change from header', async () => {
    const TimelineHeaderStub = {
      template: '<button @click="$emit(\'filter-change\', { dateRange: \'today\', sources: [] })">Change</button>',
      emits: ['filter-change'],
    }

    const wrapper = mount(TopicTimeline, {
      global: {
        stubs: {
          TimelineHeader: TimelineHeaderStub,
          TimelineItem: true,
          AIAnalysisPanel: true,
        },
      },
      props: {
        selectedTopic: createMockTopic(),
        items: [createMockItem()],
        filters: defaultFilters,
      },
    })

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('filter-change')).toEqual([[{ dateRange: 'today', sources: [] }]])
  })

  it('emits open-article from timeline item', async () => {
    const TimelineItemStub = {
      template: '<button @click="$emit(\'open-article\', 12)">Open</button>',
      props: ['item', 'isFirst', 'isLast'],
      emits: ['open-article'],
    }

    const wrapper = mount(TopicTimeline, {
      global: {
        stubs: {
          TimelineHeader: true,
          TimelineItem: TimelineItemStub,
          AIAnalysisPanel: true,
        },
      },
      props: {
        selectedTopic: createMockTopic(),
        items: [createMockItem()],
        filters: defaultFilters,
      },
    })

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('open-article')).toEqual([[12]])
  })

  it('emits select-digest from timeline item', async () => {
    const TimelineItemStub = {
      template: '<button @click="$emit(\'select\', \'digest-1\')">Select</button>',
      props: ['item', 'isFirst', 'isLast', 'isActive'],
      emits: ['select'],
    }

    const wrapper = mount(TopicTimeline, {
      global: {
        stubs: {
          TimelineHeader: true,
          TimelineItem: TimelineItemStub,
          AIAnalysisPanel: true,
        },
      },
      props: {
        selectedTopic: createMockTopic(),
        items: [createMockItem()],
        filters: defaultFilters,
        activeDigestId: 'digest-1',
      },
    })

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('select-digest')).toEqual([['digest-1']])
  })

  it('emits preview-digest from timeline item', async () => {
    const TimelineItemStub = {
      template: '<button @click="$emit(\'preview-digest\', \'digest-1\')">Preview</button>',
      props: ['item', 'isFirst', 'isLast', 'isActive'],
      emits: ['preview-digest'],
    }

    const wrapper = mount(TopicTimeline, {
      global: {
        stubs: {
          TimelineHeader: true,
          TimelineItem: TimelineItemStub,
          AIAnalysisPanel: true,
        },
      },
      props: {
        selectedTopic: createMockTopic(),
        items: [createMockItem()],
        filters: defaultFilters,
        activeDigestId: 'digest-1',
      },
    })

    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('preview-digest')).toEqual([['digest-1']])
  })
})
