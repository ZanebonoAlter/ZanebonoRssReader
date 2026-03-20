<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useArticlesApi } from '~/api/articles'
import {
  useTopicGraphApi,
  type HotspotDigestCard,
  type TopicsByCategoryPayload,
  type TopicCategory,
  type TopicGraphDetailPayload,
  type TopicGraphType,
} from '~/api/topicGraph'
import type { Article } from '~/types'
import type { TimelineDigest, TimelineDigestSelection, TimelineFilters } from '~/types/timeline'
import ArticleContentView from '~/features/articles/components/ArticleContentView.vue'
import TopicGraphCanvas from '~/features/topic-graph/components/TopicGraphCanvas.client.vue'
import TopicGraphFooterPanels from '~/features/topic-graph/components/TopicGraphFooterPanels.vue'
import TopicGraphHeader from '~/features/topic-graph/components/TopicGraphHeader.vue'
import TopicGraphSidebar from '~/features/topic-graph/components/TopicGraphSidebar.vue'
import TopicTimeline from '~/features/topic-graph/components/TopicTimeline.vue'
import { buildTopicGraphViewModel } from '~/features/topic-graph/utils/buildTopicGraphViewModel'
import { normalizeTopicCategory } from '~/features/topic-graph/utils/normalizeTopicCategory'

const topicGraphApi = useTopicGraphApi()
const articlesApi = useArticlesApi()

function formatDateInput(date = new Date()) {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

const selectedType = ref<TopicGraphType>('daily')
const selectedDate = ref(formatDateInput())
const graphPayload = ref<Awaited<ReturnType<typeof topicGraphApi.getGraph>>['data'] | null>(null)
const selectedTopicSlug = ref<string | null>(null)
const selectedCategory = ref<TopicCategory | null>(null)
const selectedKeywordSlug = ref<string | null>(null)
const selectedDigestId = ref<string | null>(null)
const previewDigestId = ref<string | null>(null)
const detail = ref<TopicGraphDetailPayload | null>(null)
const loadingGraph = ref(false)
const loadingDetail = ref(false)
const loadingPreviewArticle = ref(false)
const notice = ref<string | null>(null)
const selectedPreviewArticle = ref<Article | null>(null)
const previewArticles = ref<Article[]>([])

// Hotspot topics state (from getTopicsByCategory API)
const hotspotData = ref<TopicsByCategoryPayload | null>(null)
const loadingHotspots = ref(false)

// Hotspot digests state (reverse trace: tag -> articles -> digests)
const hotspotDigests = ref<HotspotDigestCard[]>([])
const loadingHotspotDigests = ref(false)
const selectedHotspotTag = ref<{ slug: string; label: string; category: TopicCategory } | null>(null)

// Timeline state
const timelineFilters = ref<TimelineFilters>({
  dateRange: null,
  sources: [],
})

// AI Analysis state
const aiAnalysisStatus = ref<'idle' | 'loading' | 'completed' | 'error'>('idle')
const aiAnalysisProgress = ref(0)
const aiAnalysisResult = ref<any>(null)
const aiAnalysisError = ref<string | null>(null)
let aiAnalysisPollTimer: ReturnType<typeof setTimeout> | null = null

const viewModel = computed(() => graphPayload.value
  ? buildTopicGraphViewModel(graphPayload.value)
  : buildTopicGraphViewModel({
      type: selectedType.value,
      anchor_date: selectedDate.value,
      period_label: '正在载入',
      topic_count: 0,
      article_count: 0,
      feed_count: 0,
      top_topics: [],
      nodes: [],
      edges: [],
    }))

const activeTopicNode = computed(() => {
  const focusSlug = selectedKeywordSlug.value || selectedTopicSlug.value
  return viewModel.value.graph.nodes.find(node => node.slug === focusSlug) || null
})
const highlightedNodeIds = computed(() => {
  const highlighted = new Set<string>()
  const focusSlug = selectedKeywordSlug.value || selectedTopicSlug.value
  if (!focusSlug) return []

  const focusNode = viewModel.value.graph.nodes.find(node => node.slug === focusSlug)
  if (!focusNode) return []

  highlighted.add(focusNode.id)

  for (const edge of viewModel.value.graph.edges) {
    if (edge.source === focusNode.id) {
      highlighted.add(edge.target)
    }
    if (edge.target === focusNode.id) {
      highlighted.add(edge.source)
    }
  }

  return Array.from(highlighted)
})
const relatedEdgeIds = computed(() => {
  const highlightedSet = new Set(highlightedNodeIds.value)
  if (!highlightedSet.size) return []

  return viewModel.value.graph.edges
    .filter(edge => {
      return highlightedSet.has(edge.source) && highlightedSet.has(edge.target)
    })
    .map(edge => edge.id)
})
const topTopicLabels = computed(() => viewModel.value.topTopics.slice(0, 12))

// Hotspot categories now use data from getTopicsByCategory API
// Hotspot search state
const hotspotSearchQueries = ref<Record<string, string>>({ event: '', person: '', keyword: '' })
const hotspotDropdownOpen = ref<Record<string, boolean>>({ event: false, person: false, keyword: false })
const hotspotShowAll = ref<Record<string, boolean>>({ event: false, person: false, keyword: false })

// Refs for hotspot search containers
const hotspotSearchRefs = ref<Record<string, HTMLDivElement | null>>({ event: null, person: null, keyword: null })

// Filter topics based on search query
function filterTopics(topics: any[], query: string) {
  if (!query.trim()) return topics
  const lowerQuery = query.toLowerCase()
  return topics.filter(topic =>
    topic.label.toLowerCase().includes(lowerQuery) ||
    topic.slug.toLowerCase().includes(lowerQuery)
  )
}

// Toggle show all topics
function toggleShowAll(categoryKey: string) {
  hotspotShowAll.value[categoryKey] = !hotspotShowAll.value[categoryKey]
}

// Close specific dropdown
function closeHotspotDropdown(categoryKey: string) {
  hotspotDropdownOpen.value[categoryKey] = false
}

// Close all dropdowns when clicking outside
function handleClickOutside(event: MouseEvent) {
  const target = event.target as Node
  
  Object.keys(hotspotSearchRefs.value).forEach((key) => {
    const container = hotspotSearchRefs.value[key]
    if (container && !container.contains(target)) {
      hotspotDropdownOpen.value[key] = false
    }
  })
}

// Add/remove click outside listener
watch(() => Object.values(hotspotDropdownOpen.value).some(Boolean), (isAnyOpen) => {
  if (isAnyOpen) {
    document.addEventListener('click', handleClickOutside, true)
  } else {
    document.removeEventListener('click', handleClickOutside, true)
  }
})

// Cleanup on unmount
onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside, true)
})

const hotspotCategories = computed(() => ([
  {
    key: 'event',
    label: '事件',
    icon: 'mdi:calendar-alert-outline',
    headerClass: 'topic-category-header--event',
    topics: hotspotData.value?.events || [],
    filteredTopics: filterTopics(hotspotData.value?.events || [], hotspotSearchQueries.value.event),
    displayTopics: hotspotShowAll.value.event 
      ? filterTopics(hotspotData.value?.events || [], hotspotSearchQueries.value.event)
      : filterTopics(hotspotData.value?.events || [], hotspotSearchQueries.value.event).slice(0, 8),
    hasMore: filterTopics(hotspotData.value?.events || [], hotspotSearchQueries.value.event).length > 8,
    showAll: hotspotShowAll.value.event,
  },
  {
    key: 'person',
    label: '人物',
    icon: 'mdi:account-voice-outline',
    headerClass: 'topic-category-header--person',
    topics: hotspotData.value?.people || [],
    filteredTopics: filterTopics(hotspotData.value?.people || [], hotspotSearchQueries.value.person),
    displayTopics: hotspotShowAll.value.person
      ? filterTopics(hotspotData.value?.people || [], hotspotSearchQueries.value.person)
      : filterTopics(hotspotData.value?.people || [], hotspotSearchQueries.value.person).slice(0, 8),
    hasMore: filterTopics(hotspotData.value?.people || [], hotspotSearchQueries.value.person).length > 8,
    showAll: hotspotShowAll.value.person,
  },
  {
    key: 'keyword',
    label: '关键词',
    icon: 'mdi:key-variant',
    headerClass: 'topic-category-header--keyword',
    topics: hotspotData.value?.keywords || [],
    filteredTopics: filterTopics(hotspotData.value?.keywords || [], hotspotSearchQueries.value.keyword),
    displayTopics: hotspotShowAll.value.keyword
      ? filterTopics(hotspotData.value?.keywords || [], hotspotSearchQueries.value.keyword)
      : filterTopics(hotspotData.value?.keywords || [], hotspotSearchQueries.value.keyword).slice(0, 8),
    hasMore: filterTopics(hotspotData.value?.keywords || [], hotspotSearchQueries.value.keyword).length > 8,
    showAll: hotspotShowAll.value.keyword,
  },
]))
const timelineItems = computed((): TimelineDigest[] => {
  const summaries = detail.value?.summaries || []

  return summaries
    .filter((summary) => matchesTimelineFilters(summary.created_at, timelineFilters.value))
    .map(summary => ({
      id: String(summary.id),
      title: summary.title,
      summary: summary.summary,
      createdAt: summary.created_at,
      feedName: summary.feed_name,
      categoryName: summary.category_name,
      articleCount: summary.article_count,
      tags: summary.topics.map(topic => ({
        slug: topic.slug,
        label: topic.label,
        category: normalizeTopicCategory(topic.category, topic.kind),
      })),
      articles: summary.articles.map(article => ({
        id: article.id,
        title: article.title,
        link: article.link,
      })),
    }))
})

// Hotspot digests converted to TimelineDigest format for display
const hotspotTimelineItems = computed((): TimelineDigest[] => {
  if (!hotspotDigests.value.length) return []

  return hotspotDigests.value
    .filter((digest) => matchesTimelineFilters(digest.created_at, timelineFilters.value))
    .map(digest => ({
      id: String(digest.id),
      title: digest.title,
      summary: digest.summary,
      createdAt: digest.created_at,
      feedName: digest.feed_name,
      categoryName: digest.category_name,
      articleCount: digest.article_count,
      tags: [], // Hotspot digests don't have tags in the response
      articles: digest.matched_articles?.map(article => ({
        id: article.id,
        title: article.title,
        link: '',
      })) || [],
    }))
})

// Effective timeline items: use hotspot digests when available, otherwise use topic detail summaries
// This allows hotspot tag clicks to show digests containing articles with that tag
const effectiveTimelineItems = computed((): TimelineDigest[] => {
  // When a hotspot tag is selected and we have hotspot digests, prioritize them
  if (selectedHotspotTag.value && hotspotDigests.value.length > 0) {
    return hotspotTimelineItems.value
  }
  // Otherwise use the topic detail summaries
  return timelineItems.value
})
const selectedDigest = computed<TimelineDigestSelection | null>(() => {
  if (!selectedDigestId.value) return null
  const digest = effectiveTimelineItems.value.find(item => item.id === selectedDigestId.value)
  if (!digest) return null

  // For hotspot digests, use hotspot tag to fetch matched articles
  if (selectedHotspotTag.value && hotspotDigests.value.length > 0) {
    const hotspotDigest = hotspotDigests.value.find(d => String(d.id) === selectedDigestId.value)
    if (hotspotDigest) {
      return {
        ...digest,
        matchedArticleIds: hotspotDigest.matched_articles?.map(a => a.id) || [],
      }
    }
  }

  // For topic detail digests, match against topic articles
  if (!detail.value) return null
  const topicArticleIds = new Set(detail.value.articles.map(article => article.id))
  const matchedArticleIds = digest.articles
    .map(article => article.id)
    .filter(id => topicArticleIds.has(id))

  return {
    ...digest,
    matchedArticleIds,
  }
})
const previewDigest = computed(() => {
  if (!previewDigestId.value) return null
  return effectiveTimelineItems.value.find(item => item.id === previewDigestId.value) || null
})
const statCards = computed(() => ([
  { label: '主题数', value: viewModel.value.stats.topicCount },
  { label: '文章数', value: viewModel.value.stats.articleCount },
  { label: 'Feed 数', value: viewModel.value.stats.feedCount },
]))

const pageState = computed(() => {
  if (loadingGraph.value) return 'loading'
  if (selectedPreviewArticle.value) return 'article-preview'
  if (detail.value) return 'detail'
  if (graphPayload.value) return 'graph-ready'
  return 'empty'
})

const selectedTopicInfo = computed(() => {
  if (detail.value?.topic) {
    return {
      id: detail.value.topic.id,
      slug: detail.value.topic.slug,
      label: detail.value.topic.label,
      category: normalizeTopicCategory(detail.value.topic.category, detail.value.topic.kind),
    }
  }

  if (!selectedTopicSlug.value) return null

  const topic = viewModel.value.topTopics.find(item => item.slug === selectedTopicSlug.value)
  if (!topic) return null

    return {
      id: topic.id ?? 0,
      slug: topic.slug,
      label: topic.label,
      category: normalizeTopicCategory(topic.category, topic.kind),
    }
})

async function loadHotspots() {
  loadingHotspots.value = true
  try {
    const response = await topicGraphApi.getTopicsByCategory(selectedType.value, selectedDate.value)
    if (response.success && response.data) {
      hotspotData.value = response.data
    } else {
      console.error('Failed to load hotspots:', response.error)
      hotspotData.value = null
    }
  } catch (error) {
    console.error('Failed to load hotspots:', error)
    hotspotData.value = null
  } finally {
    loadingHotspots.value = false
  }
}

async function loadGraph() {
  loadingGraph.value = true
  notice.value = null

  try {
    const response = await topicGraphApi.getGraph(selectedType.value, selectedDate.value)
    if (!response.success || !response.data) {
      notice.value = response.error || '主题图谱没拉下来'
      graphPayload.value = null
      detail.value = null
      return
    }

    graphPayload.value = response.data
    selectedTopicSlug.value = response.data.top_topics[0]?.slug || null
    selectedCategory.value = response.data.top_topics[0]
      ? normalizeTopicCategory(response.data.top_topics[0].category, response.data.top_topics[0].kind)
      : null
    selectedKeywordSlug.value = null
    selectedDigestId.value = null
    previewDigestId.value = null

    if (selectedTopicSlug.value) {
      void loadTopicDetail(selectedTopicSlug.value)
    } else {
      detail.value = null
    }

    // Load hotspot data in parallel
    void loadHotspots()
  } catch (error) {
    console.error('Failed to load topic graph:', error)
    notice.value = error instanceof Error ? error.message : '主题图谱加载失败'
  } finally {
    loadingGraph.value = false
  }
}

async function loadTopicDetail(slug: string) {
  selectedTopicSlug.value = slug
  selectedKeywordSlug.value = null
  selectedDigestId.value = null
  previewDigestId.value = null
  const topic = viewModel.value.topTopics.find(item => item.slug === slug) || null
  if (topic?.category) {
    selectedCategory.value = normalizeTopicCategory(topic.category, topic.kind)
  }
  loadingDetail.value = true

  try {
    const response = await topicGraphApi.getTopicDetail(slug, selectedType.value, selectedDate.value)
    if (response.success && response.data) {
      detail.value = response.data
      selectedCategory.value = normalizeTopicCategory(response.data.topic.category, response.data.topic.kind)
      selectedDigestId.value = null
      previewDigestId.value = null
      return
    }

    detail.value = null
    notice.value = response.error || '话题详情加载失败'
  } catch (error) {
    console.error('Failed to load topic detail:', error)
    detail.value = null
    notice.value = error instanceof Error ? error.message : '话题详情加载失败'
  } finally {
    loadingDetail.value = false
  }
}

async function handleTagSelect(slug: string, category: TopicCategory) {
  selectedCategory.value = category
  selectedTopicSlug.value = slug

  // Find the tag label from hotspot data
  let tagLabel = slug
  const allTags = [
    ...(hotspotData.value?.events || []),
    ...(hotspotData.value?.people || []),
    ...(hotspotData.value?.keywords || []),
  ]
  const foundTag = allTags.find(t => t.slug === slug)
  if (foundTag) {
    tagLabel = foundTag.label
  }

  // Update selected hotspot tag
  selectedHotspotTag.value = { slug, label: tagLabel, category }

  // Load digests for this tag (reverse trace: tag -> articles -> digests)
  await loadHotspotDigests(slug)

  // Also load topic detail for the sidebar
  void loadTopicDetail(slug)
}

async function loadHotspotDigests(tagSlug: string) {
  loadingHotspotDigests.value = true
  try {
    const response = await topicGraphApi.getDigestsByArticleTag(
      tagSlug,
      selectedType.value,
      selectedDate.value,
      20
    )
    if (response.success && response.data) {
      hotspotDigests.value = response.data.digests
    } else {
      hotspotDigests.value = []
      console.error('Failed to load hotspot digests:', response.error)
    }
  } catch (error) {
    console.error('Failed to load hotspot digests:', error)
    hotspotDigests.value = []
  } finally {
    loadingHotspotDigests.value = false
  }
}

function handleNodeClick(node: { slug?: string; kind: string; category?: TopicCategory }) {
  if (node.kind !== 'topic' || !node.slug) return

  if (node.category) {
    selectedCategory.value = node.category
  }

  void loadTopicDetail(node.slug)
}

function handleKeywordHighlight(keywordSlug: string | null) {
  if (!keywordSlug) {
    selectedKeywordSlug.value = null
    return
  }

  const existsInGraph = viewModel.value.graph.nodes.some(node => node.kind === 'topic' && node.slug === keywordSlug)
  selectedKeywordSlug.value = existsInGraph ? keywordSlug : null
}

function handleDigestSelect(digestId: string) {
  selectedDigestId.value = digestId
}

function handlePreviewDigest(digestId: string) {
  selectedDigestId.value = digestId
  previewDigestId.value = digestId
}

function closeDigestPreview() {
  previewDigestId.value = null
}

async function openArticlePreview(articleId: number) {
  loadingPreviewArticle.value = true

  try {
    const response = await articlesApi.getArticle(articleId)
    if (!response.success || !response.data) {
      notice.value = response.error || '文章预览加载失败'
      return
    }

    selectedPreviewArticle.value = normalizeArticle(response.data as any)

    if (detail.value) {
      const ids = detail.value.summaries.flatMap(summary => summary.articles.map(article => article.id))
      const uniqueIds = Array.from(new Set(ids))
      const articleResponses = await Promise.all(uniqueIds.slice(0, 12).map(id => articlesApi.getArticle(id)))
      previewArticles.value = articleResponses
        .filter(item => item.success && item.data)
        .map(item => normalizeArticle(item.data as any))
    }
  } catch (error) {
    console.error('Failed to open article preview:', error)
    notice.value = error instanceof Error ? error.message : '文章预览加载失败'
  } finally {
    loadingPreviewArticle.value = false
  }
}

function closeArticlePreview() {
  selectedPreviewArticle.value = null
}

function normalizeArticle(article: any): Article {
  return {
    id: String(article.id),
    feedId: String(article.feed_id),
    title: article.title,
    description: article.description || '',
    content: article.content || '',
    link: article.link,
    pubDate: article.pub_date || article.created_at || '',
    author: article.author,
    category: article.category_id ? String(article.category_id) : '',
    read: article.read || false,
    favorite: article.favorite || false,
    summaryStatus: article.summary_status,
    summaryGeneratedAt: article.summary_generated_at,
    completionAttempts: article.completion_attempts,
    completionError: article.completion_error,
    aiContentSummary: article.ai_content_summary,
    firecrawlStatus: article.firecrawl_status,
    firecrawlError: article.firecrawl_error,
    firecrawlContent: article.firecrawl_content,
    firecrawlCrawledAt: article.firecrawl_crawled_at,
    imageUrl: article.image_url,
  }
}

function handleTimelineFilterChange(filters: TimelineFilters) {
  timelineFilters.value = filters
}

function handleTimelineAIAnalysis() {
  // Trigger AI analysis start
  handleAIAnalysisStart()
}

function clearAIAnalysisPolling() {
  if (aiAnalysisPollTimer) {
    clearTimeout(aiAnalysisPollTimer)
    aiAnalysisPollTimer = null
  }
}

function normalizeTimelineAnalysisPayload(payload: Record<string, any>) {
  return {
    summary: payload.summary,
    timeline: Array.isArray(payload.timeline)
      ? payload.timeline.map((item: any) => ({
          date: item.date,
          title: item.title,
          summary: item.summary,
          sources: Array.isArray(item.sources)
            ? item.sources.map((source: any) => ({ articleId: source.articleId, title: source.title }))
            : Array.isArray(item.source_articles)
              ? item.source_articles.map((source: any) => ({
                  articleId: source.articleId ?? source.article_id,
                  title: source.title,
                }))
              : [],
        }))
      : undefined,
    keyMoments: payload.keyMoments ?? payload.key_moments,
    relatedEntities: payload.relatedEntities ?? payload.related_entities,
    profile: payload.profile,
    appearances: Array.isArray(payload.appearances)
      ? payload.appearances.map((item: any) => ({
          date: item.date,
          context: item.context ?? item.scene,
          quote: item.quote,
          articleId: item.articleId ?? item.article_id,
          articleTitle: item.articleTitle ?? item.article_title,
          articleLink: item.articleLink ?? item.article_link,
        }))
      : undefined,
    trend: payload.trend ?? payload.trend_data,
    relatedTopics: Array.isArray(payload.relatedTopics ?? payload.related_topics)
      ? (payload.relatedTopics ?? payload.related_topics).map((item: any) => typeof item === 'string' ? item : (item.topic ?? item.label ?? ''))
          .filter(Boolean)
      : undefined,
    coOccurrence: Array.isArray(payload.coOccurrence ?? payload.co_occurrence)
      ? (payload.coOccurrence ?? payload.co_occurrence).map((item: any) => ({
          term: item.term ?? item.keyword,
          count: item.count ?? item.score ?? 0,
        }))
      : undefined,
    contextExamples: Array.isArray(payload.contextExamples ?? payload.context_examples)
      ? (payload.contextExamples ?? payload.context_examples).map((item: any) => typeof item === 'string' ? item : item.text)
      : undefined,
  }
}

async function applyAIAnalysisResult() {
  if (!detail.value?.topic.id || !selectedCategory.value) return false

  const response = await topicGraphApi.getTopicAnalysis({
    tagID: detail.value.topic.id,
    analysisType: selectedCategory.value,
    windowType: selectedType.value,
    anchorDate: selectedDate.value,
  })

  if (!response.success || !response.data) {
    return false
  }

  const payload = typeof response.data.payload_json === 'string'
    ? JSON.parse(response.data.payload_json)
    : response.data.payload_json

  aiAnalysisResult.value = normalizeTimelineAnalysisPayload(payload || {})
  aiAnalysisProgress.value = 100
  aiAnalysisStatus.value = 'completed'
  aiAnalysisError.value = null
  clearAIAnalysisPolling()
  return true
}

async function pollAIAnalysisStatus() {
  if (!detail.value?.topic.id || !selectedCategory.value) return

  const params = {
    tagID: detail.value.topic.id,
    analysisType: selectedCategory.value,
    windowType: selectedType.value,
    anchorDate: selectedDate.value,
  }

  try {
    const response = await topicGraphApi.getAnalysisStatus(params)
    const status = response.data?.status
    const progress = response.data?.progress ?? 0

    if (!response.success || !status) {
      aiAnalysisStatus.value = 'error'
      aiAnalysisError.value = response.error || '分析状态获取失败'
      clearAIAnalysisPolling()
      return
    }

    if (status === 'ready' || status === 'completed') {
      const loaded = await applyAIAnalysisResult()
      if (!loaded) {
        aiAnalysisStatus.value = 'error'
        aiAnalysisError.value = '分析结果读取失败'
      }
      return
    }

    if (status === 'pending' || status === 'processing') {
      aiAnalysisStatus.value = 'loading'
      aiAnalysisProgress.value = Math.min(Math.max(Math.round(progress * 100), 1), 99)
      clearAIAnalysisPolling()
      aiAnalysisPollTimer = setTimeout(() => {
        void pollAIAnalysisStatus()
      }, 1800)
      return
    }

    aiAnalysisStatus.value = 'error'
    aiAnalysisError.value = status === 'failed' ? '分析任务失败' : '暂无分析结果'
    clearAIAnalysisPolling()
  } catch (error) {
    aiAnalysisStatus.value = 'error'
    aiAnalysisError.value = error instanceof Error ? error.message : '分析状态获取失败'
    clearAIAnalysisPolling()
  }
}

async function handleAIAnalysisStart() {
  if (!selectedTopicSlug.value || !selectedCategory.value) return

  aiAnalysisStatus.value = 'loading'
  aiAnalysisProgress.value = 0
  aiAnalysisError.value = null
  clearAIAnalysisPolling()

  try {
    const topicID = detail.value?.topic.id
    if (!topicID) {
      throw new Error('Topic not found')
    }

    const loaded = await applyAIAnalysisResult()
    if (loaded) {
      return
    }

    await topicGraphApi.rebuildTopicAnalysis({
      tagID: topicID,
      analysisType: selectedCategory.value,
      windowType: selectedType.value,
      anchorDate: selectedDate.value,
    })

    aiAnalysisProgress.value = 1
    await pollAIAnalysisStatus()
  } catch (error) {
    console.error('AI analysis error:', error)
    aiAnalysisError.value = error instanceof Error ? error.message : 'AI分析失败'
    aiAnalysisStatus.value = 'error'
    clearAIAnalysisPolling()
  }
}

async function handleAIAnalysisRetry() {
  await handleAIAnalysisStart()
}

function matchesTimelineFilters(createdAt: string, filters: TimelineFilters) {
  const parsed = createdAt ? new Date(createdAt) : null
  if (!parsed || Number.isNaN(parsed.getTime())) {
    return filters.dateRange === null
  }

  const current = new Date(parsed)
  current.setHours(0, 0, 0, 0)

  if (filters.dateRange === 'custom') {
    if (filters.startDate) {
      const start = new Date(filters.startDate)
      start.setHours(0, 0, 0, 0)
      if (current < start) return false
    }

    if (filters.endDate) {
      const end = new Date(filters.endDate)
      end.setHours(23, 59, 59, 999)
      if (parsed > end) return false
    }

    return true
  }

  if (filters.dateRange === 'today') {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    return current.getTime() === today.getTime()
  }

  if (filters.dateRange === 'week') {
    const weekStart = new Date()
    weekStart.setHours(0, 0, 0, 0)
    weekStart.setDate(weekStart.getDate() - 6)
    return current >= weekStart
  }

  if (filters.dateRange === 'month') {
    const monthStart = new Date()
    monthStart.setHours(0, 0, 0, 0)
    monthStart.setDate(monthStart.getDate() - 29)
    return current >= monthStart
  }

  return true
}

watch(selectedType, () => {
  void loadGraph()
})

watch(selectedDate, () => {
  void loadGraph()
})

watch(selectedTopicSlug, () => {
  aiAnalysisStatus.value = 'idle'
  aiAnalysisProgress.value = 0
  aiAnalysisError.value = null
  aiAnalysisResult.value = null
  clearAIAnalysisPolling()
})

watch(effectiveTimelineItems, (items) => {
  if (!items.length) {
    selectedDigestId.value = null
    previewDigestId.value = null
    return
  }

  const currentExists = selectedDigestId.value && items.some(item => item.id === selectedDigestId.value)
  if (!currentExists) {
    selectedDigestId.value = items[0]?.id || null
  }
}, { immediate: true })

onBeforeUnmount(() => {
  clearAIAnalysisPolling()
})

await loadGraph()
</script>

<template>
  <div
    class="topic-stage min-h-screen px-4 py-5 md:px-6 md:py-7"
    data-testid="topic-graph-page"
    :data-state="pageState"
  >
    <div class="topic-shell mx-auto w-full">
      <section class="topic-layout grid gap-5 2xl:grid-cols-[minmax(0,2.15fr)_minmax(430px,0.95fr)]">
        <div class="space-y-5">
          <article class="topic-canvas-shell rounded-[34px] p-4 md:p-5">
            <div class="topic-studio grid gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
              <aside class="topic-studio__rail rounded-[30px] p-4 md:p-5">
                <TopicGraphHeader
                  :selected-type="selectedType"
                  :selected-date="selectedDate"
                  :loading="loadingGraph"
                  :hero-label="viewModel.stats.heroLabel"
                  :hero-subline="viewModel.stats.heroSubline"
                  @update:type="selectedType = $event"
                  @update:date="selectedDate = $event"
                  @refresh="loadGraph"
                />

                <div class="mt-6">
                  <p class="text-xs uppercase tracking-[0.3em] text-white/42">Graph Field</p>
                  <h2 class="mt-2 font-serif text-2xl text-white md:text-[2.25rem]">{{ graphPayload?.period_label || '话题网络' }}</h2>
                  <p class="mt-3 text-sm leading-6 text-[rgba(255,255,255,0.68)]">
                    默认只保留重点标签常显，点中节点后再展开完整名字和一跳关系，减少视觉重叠。
                  </p>
                </div>

                <div class="mt-6 grid gap-3 sm:grid-cols-3 xl:grid-cols-1">
                  <article v-for="card in statCards" :key="card.label" class="topic-stat-card rounded-[24px] px-4 py-3">
                    <p class="topic-stat-card__label">{{ card.label }}</p>
                    <p class="topic-stat-card__value">{{ card.value }}</p>
                  </article>
                </div>

              </aside>

              <div class="space-y-4">
                <TopicGraphCanvas
                  :nodes="viewModel.graph.nodes"
                  :edges="viewModel.graph.edges"
                  :featured-node-ids="viewModel.graph.featuredNodeIds"
                  :active-node-id="activeTopicNode?.id || null"
                  :selected-category="selectedCategory"
                  :highlighted-node-ids="highlightedNodeIds"
                  :related-edge-ids="relatedEdgeIds"
                  @node-click="handleNodeClick"
                />

                <article class="topic-note rounded-[30px] px-5 py-4 text-sm leading-6 text-[rgba(255,255,255,0.78)]">
                  <div class="flex items-start gap-3">
                    <Icon icon="mdi:orbit-variant" width="20" height="20" class="mt-1 text-[rgba(240,138,75,0.92)]" />
                    <p>
                      先看结构，再读内容：亮色主节点是当前焦点，周边只保留一跳关系的高亮，更多细节放到右侧阅读栏。
                    </p>
                  </div>
                </article>

                <section class="topic-hotspot-strip rounded-[30px] p-4 md:p-5">
                  <div class="topic-hotspot-strip__header">
                    <div>
                      <p class="text-xs uppercase tracking-[0.24em] text-white/42">热点题材</p>
                      <h3 class="mt-2 font-serif text-xl text-white">把最热的话题平铺到底部，避免和左侧控制重复。</h3>
                    </div>
                  </div>

                  <div class="mt-4 grid gap-3 xl:grid-cols-3">
                    <section
                      v-for="category in hotspotCategories"
                      :key="category.key"
                      class="topic-category-column rounded-[22px] p-3"
                      :data-testid="`hotspot-category-${category.key}`"
                    >
                      <div class="topic-category-header" :class="category.headerClass">
                        <Icon :icon="category.icon" width="14" />
                        <span>{{ category.label }}</span>
                        <span class="topic-count">({{ category.topics.length }})</span>
                      </div>

                        <!-- Search Input -->
                      <div :ref="el => { if (el) hotspotSearchRefs[category.key] = el as HTMLDivElement }" class="topic-search-wrapper mt-3">
                        <div class="topic-search-input-wrapper" @click="hotspotDropdownOpen[category.key] = true">
                          <Icon icon="mdi:magnify" width="14" class="topic-search-icon" />
                          <input
                            v-model="hotspotSearchQueries[category.key]"
                            type="text"
                            class="topic-search-input"
                            placeholder="搜索..."
                            @focus="hotspotDropdownOpen[category.key] = true"
                          />
                          <button
                            v-if="hotspotSearchQueries[category.key]"
                            class="topic-search-clear"
                            @click.stop="hotspotSearchQueries[category.key] = ''"
                          >
                            <Icon icon="mdi:close" width="12" />
                          </button>
                        </div>

                        <!-- Dropdown -->
                        <div
                          v-if="hotspotDropdownOpen[category.key] && category.filteredTopics.length > 0"
                          class="topic-search-dropdown"
                          @mousedown.prevent
                        >
                          <div class="topic-dropdown-scroll">
                            <button
                              v-for="topic in category.displayTopics"
                              :key="topic.slug"
                              type="button"
                              class="topic-dropdown-item"
                              :class="{
                                'topic-dropdown-item--active': selectedTopicSlug === topic.slug,
                              }"
                              @click="handleTagSelect(topic.slug, normalizeTopicCategory(topic.category, topic.kind)); hotspotDropdownOpen[category.key] = false"
                            >
                              <Icon v-if="topic.icon" :icon="topic.icon" width="14" />
                              <span>{{ topic.label }}</span>
                            </button>
                          </div>
                          <button
                            v-if="category.hasMore"
                            class="topic-dropdown-toggle"
                            @click="toggleShowAll(category.key)"
                          >
                            <Icon :icon="category.showAll ? 'mdi:chevron-up' : 'mdi:chevron-down'" width="16" />
                            {{ category.showAll ? '收起' : `显示全部 (${category.filteredTopics.length})` }}
                          </button>
                        </div>

                        <!-- No Results -->
                        <div
                          v-if="hotspotDropdownOpen[category.key] && hotspotSearchQueries[category.key] && category.filteredTopics.length === 0"
                          class="topic-search-no-results"
                        >
                          未找到匹配的结果
                          <button
                            class="topic-dropdown-close"
                            @click.stop="closeHotspotDropdown(category.key)"
                          >
                            关闭
                          </button>
                        </div>
                      </div>

                      <!-- Quick Tags (show top 5 without search) -->
                      <div v-if="!hotspotSearchQueries[category.key]" class="topic-quick-tags mt-3">
                        <button
                          v-for="topic in category.topics.slice(0, 5)"
                          :key="topic.slug"
                          type="button"
                          class="topic-badge text-left"
                          :class="{
                            'topic-badge--event': normalizeTopicCategory(topic.category, topic.kind) === 'event',
                            'topic-badge--person': normalizeTopicCategory(topic.category, topic.kind) === 'person',
                            'topic-badge--keyword': normalizeTopicCategory(topic.category, topic.kind) === 'keyword',
                            'topic-badge--active': selectedTopicSlug === topic.slug,
                          }"
                          @click="handleTagSelect(topic.slug, normalizeTopicCategory(topic.category, topic.kind))"
                        >
                          <Icon v-if="topic.icon" :icon="topic.icon" width="14" />
                          {{ topic.label }}
                        </button>
                        <button
                          v-if="category.topics.length > 5"
                          class="topic-more-hint"
                          @click="hotspotSearchQueries[category.key] = ''; hotspotDropdownOpen[category.key] = true"
                        >
                          +{{ category.topics.length - 5 }} 更多
                        </button>
                      </div>
                    </section>
                  </div>
                </section>

                <TopicGraphFooterPanels :detail="detail" />
              </div>
            </div>
          </article>

          <p v-if="notice" class="rounded-[24px] border border-[rgba(240,138,75,0.28)] bg-[rgba(240,138,75,0.1)] px-4 py-3 text-sm text-[rgba(255,233,220,0.88)]">
            {{ notice }}
          </p>

          <!-- Timeline Section -->
          <article class="topic-timeline-shell rounded-[34px] p-4 md:p-5">
<TopicTimeline
                :selected-topic="selectedTopicInfo"
                :items="effectiveTimelineItems"
                :filters="timelineFilters"
                :active-digest-id="selectedDigestId"
                :ai-analysis-status="aiAnalysisStatus"
                :ai-analysis-progress="aiAnalysisProgress"
                :ai-analysis-result="aiAnalysisResult"
                :ai-analysis-error="aiAnalysisError"
                @filter-change="handleTimelineFilterChange"
                @select-digest="handleDigestSelect"
                @preview-digest="handlePreviewDigest"
                @ai-analysis="handleTimelineAIAnalysis"
                @ai-analysis-start="handleAIAnalysisStart"
                @ai-analysis-retry="handleAIAnalysisRetry"
                @open-article="openArticlePreview"
            />
          </article>
        </div>

        <div class="topic-reading-rail" data-testid="topic-graph-sidebar-region">
          <TopicGraphSidebar
            :detail="detail"
            :selected-digest="selectedDigest"
            :loading="loadingDetail"
            :error="notice"
            :data-state="detail ? 'detail' : (loadingDetail ? 'loading' : 'empty')"
            :selected-keyword="selectedKeywordSlug"
            :selected-tag-slug="selectedHotspotTag?.slug"
            @open-article="openArticlePreview"
            @highlight-keyword="handleKeywordHighlight"
          />
        </div>
      </section>
    </div>

    <Teleport to="body">
      <div
        v-if="previewDigest"
        class="topic-digest-modal"
        data-testid="topic-graph-digest-preview"
        @click.self="closeDigestPreview"
      >
        <div class="topic-digest-modal__panel">
          <header class="topic-digest-modal__header">
            <div>
              <p class="text-xs uppercase tracking-[0.24em] text-white/42">日报预览</p>
              <h3 class="mt-3 font-serif text-2xl text-white">{{ previewDigest.title }}</h3>
              <p class="mt-2 text-sm text-white/58">{{ previewDigest.feedName }} · {{ previewDigest.createdAt }}</p>
            </div>

            <button
              class="btn-ghost min-h-11 min-w-11 px-0"
              type="button"
              aria-label="关闭日报弹窗"
              @click="closeDigestPreview"
            >
              <Icon icon="mdi:close" width="18" />
            </button>
          </header>

          <div class="topic-digest-modal__body">
            <p class="topic-digest-modal__summary">{{ previewDigest.summary }}</p>

            <div v-if="previewDigest.tags.length" class="topic-digest-modal__tags">
              <span v-for="tag in previewDigest.tags" :key="tag.slug" class="topic-digest-modal__tag">
                {{ tag.label }}
              </span>
            </div>

            <div class="topic-digest-modal__sources">
              <p class="text-xs uppercase tracking-[0.22em] text-white/42">来源文章</p>
              <button
                v-for="article in previewDigest.articles"
                :key="article.id"
                type="button"
                class="topic-digest-modal__source"
                @click="openArticlePreview(article.id)"
              >
                {{ article.title }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div
        v-if="selectedPreviewArticle"
        class="topic-article-modal"
        data-testid="topic-graph-article-preview"
        @click.self="closeArticlePreview"
      >
        <div class="topic-article-modal__panel">
          <header class="topic-article-modal__header">
            <p class="truncate text-sm text-ink-medium">
              {{ loadingPreviewArticle ? '正在准备文章预览...' : '文章预览里保留项目已有的阅读、收藏和抓取动作。' }}
            </p>

            <button
              class="btn-ghost min-h-11 min-w-11 px-0"
              type="button"
              aria-label="关闭文章弹窗"
              data-testid="topic-graph-article-preview-close"
              @click="closeArticlePreview"
            >
              <Icon icon="mdi:close" width="18" />
            </button>
          </header>

          <div class="topic-article-modal__body">
            <ArticleContentView
              :article="selectedPreviewArticle"
              :articles="previewArticles"
              @navigate="selectedPreviewArticle = $event"
            />
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.topic-stage {
  background:
    radial-gradient(circle at top left, rgba(240, 138, 75, 0.18), transparent 24%),
    radial-gradient(circle at 85% 12%, rgba(63, 124, 255, 0.18), transparent 24%),
    linear-gradient(180deg, #0e161d 0%, #172733 54%, #10212e 100%);
}

.topic-shell {
  width: min(100%, calc(100vw - 1.5rem));
}

.topic-canvas-shell {
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(11, 18, 24, 0.4);
  box-shadow: 0 40px 120px rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(20px);
}

.topic-note {
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(11, 18, 24, 0.7);
  box-shadow: 0 24px 80px rgba(6, 10, 16, 0.24);
  backdrop-filter: blur(12px);
}

.topic-hotspot-strip {
  border: 1px solid rgba(255, 255, 255, 0.08);
  background:
    radial-gradient(circle at 12% 18%, rgba(240, 138, 75, 0.12), transparent 24%),
    linear-gradient(180deg, rgba(12, 19, 27, 0.86), rgba(8, 14, 22, 0.96));
  box-shadow: 0 24px 80px rgba(6, 10, 16, 0.22);
}

.topic-hotspot-strip__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.topic-timeline-shell {
  border: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(11, 18, 24, 0.4);
  box-shadow: 0 40px 120px rgba(0, 0, 0, 0.4);
  backdrop-filter: blur(20px);
}

.topic-layout {
  align-items: start;
}

.topic-studio__rail {
  display: flex;
  flex-direction: column;
  border: 1px solid rgba(255, 255, 255, 0.04);
  background: linear-gradient(180deg, rgba(15, 23, 31, 0.85), rgba(8, 14, 20, 0.95));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.05);
}

.topic-stat-card {
  border: 1px solid rgba(255, 255, 255, 0.04);
  background: rgba(0, 0, 0, 0.2);
}

.topic-stat-card__label {
  font-size: 0.7rem;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: rgba(255,255,255,0.46);
}

.topic-stat-card__value {
  margin-top: 0.55rem;
  font-size: 1.8rem;
  font-weight: 700;
  color: white;
}

.topic-reading-rail {
  position: sticky;
  top: 1rem;
}

.topic-digest-modal {
  position: fixed;
  inset: 0;
  z-index: 78;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  background: rgba(8, 12, 18, 0.7);
  backdrop-filter: blur(10px);
}

.topic-digest-modal__panel {
  width: min(760px, 100%);
  max-height: calc(100vh - 2rem);
  overflow: auto;
  border-radius: 1.75rem;
  background: linear-gradient(180deg, rgba(17, 27, 38, 0.98), rgba(9, 15, 23, 1));
  box-shadow: 0 30px 100px rgba(0, 0, 0, 0.32);
}

.topic-digest-modal__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  padding: 1.1rem 1.25rem 1rem;
}

.topic-digest-modal__body {
  display: grid;
  gap: 1rem;
  padding: 1.2rem 1.25rem 1.35rem;
}

.topic-digest-modal__summary {
  line-height: 1.8;
  color: rgba(236, 242, 248, 0.9);
  white-space: pre-wrap;
}

.topic-digest-modal__tags,
.topic-digest-modal__sources {
  display: flex;
  flex-wrap: wrap;
  gap: 0.6rem;
}

.topic-digest-modal__tag,
.topic-digest-modal__source {
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.04);
}

.topic-digest-modal__tag {
  padding: 0.32rem 0.7rem;
  font-size: 0.78rem;
  color: rgba(245, 227, 212, 0.88);
}

.topic-digest-modal__source {
  padding: 0.4rem 0.78rem;
  color: rgba(241, 246, 251, 0.84);
}

.topic-article-modal {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: flex;
  align-items: stretch;
  justify-content: center;
  background: rgba(8, 12, 18, 0.7);
  padding: 1rem;
  backdrop-filter: blur(10px);
}

.topic-article-modal__panel {
  display: flex;
  height: calc(100vh - 2rem);
  width: min(1500px, 100%);
  flex-direction: column;
  overflow: hidden;
  border-radius: 1.75rem;
  background: rgba(255, 252, 248, 0.98);
  box-shadow: 0 30px 100px rgba(0, 0, 0, 0.28);
}

.topic-article-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid rgba(20, 33, 44, 0.08);
  padding: 1rem 1.25rem;
}

.topic-article-modal__body {
  min-height: 0;
  flex: 1;
}

@media (min-width: 1280px) {
  .topic-shell {
    width: min(100%, calc(100vw - 2rem));
  }
}

@media (min-width: 1600px) {
  .topic-shell {
    width: min(100%, calc(100vw - 2.75rem));
  }
}

.topic-badge {
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 0.35rem;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 0.55rem 0.9rem;
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.78);
  background: rgba(255,255,255,0.04);
  transition:
    transform 0.15s ease,
    border-color 0.15s ease,
    background 0.15s ease,
    color 0.15s ease,
    box-shadow 0.15s ease;
}

.topic-badge:hover,
.topic-badge:focus-visible {
  transform: translateY(-1px);
  color: white;
}

.topic-badge--event {
  border-color: rgba(245, 158, 11, 0.72);
  background: rgba(245, 158, 11, 0.18);
}

.topic-badge--person {
  border-color: rgba(16, 185, 129, 0.72);
  background: rgba(16, 185, 129, 0.18);
}

.topic-badge--keyword {
  border-color: rgba(99, 102, 241, 0.72);
  background: rgba(99, 102, 241, 0.18);
}

.topic-badge--active {
  color: white;
  box-shadow: 0 12px 28px rgba(4, 8, 14, 0.24);
}

.topic-category-column {
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: linear-gradient(180deg, rgba(20, 29, 40, 0.74), rgba(10, 15, 23, 0.92));
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.topic-category-header {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.72rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.topic-category-header--button {
  border: 1px solid transparent;
  border-radius: 999px;
  padding: 0.28rem 0.62rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.topic-category-header--button:hover,
.topic-category-header--button:focus-visible {
  border-color: rgba(255, 255, 255, 0.28);
}

.topic-category-header--active {
  border-color: rgba(255, 255, 255, 0.44);
  background: rgba(255, 255, 255, 0.12);
}

.topic-category-header--event {
  color: rgba(252, 211, 77, 0.9);
}

.topic-category-header--person {
  color: rgba(110, 231, 183, 0.9);
}

.topic-category-header--keyword {
  color: rgba(165, 180, 252, 0.92);
}

.topic-category-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

/* Hotspot Search Styles */
.topic-search-wrapper {
  position: relative;
  width: 100%;
}

.topic-search-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 999px;
  padding: 0.35rem 0.75rem;
  transition: all 0.2s ease;
}

.topic-search-input-wrapper:focus-within {
  border-color: rgba(240, 138, 75, 0.5);
  background: rgba(0, 0, 0, 0.3);
}

.topic-search-icon {
  color: rgba(255, 255, 255, 0.4);
  flex-shrink: 0;
}

.topic-search-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: rgba(255, 255, 255, 0.9);
  font-size: 0.8rem;
  padding: 0 0.5rem;
  min-width: 0;
}

.topic-search-input::placeholder {
  color: rgba(255, 255, 255, 0.35);
}

.topic-search-clear {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.6);
  cursor: pointer;
  transition: all 0.15s ease;
  flex-shrink: 0;
}

.topic-search-clear:hover {
  background: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.9);
}

.topic-search-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  right: 0;
  background: rgba(22, 28, 38, 0.98);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 12px;
  padding: 0.5rem;
  max-height: 400px;
  display: flex;
  flex-direction: column;
  z-index: 50;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(20px);
}

.topic-dropdown-scroll {
  overflow-y: auto;
  max-height: 320px;
  padding-right: 0.25rem;
}

.topic-dropdown-scroll::-webkit-scrollbar {
  width: 4px;
}

.topic-dropdown-scroll::-webkit-scrollbar-track {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 2px;
}

.topic-dropdown-scroll::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.15);
  border-radius: 2px;
}

.topic-dropdown-scroll::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.25);
}

.topic-dropdown-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  width: 100%;
  padding: 0.6rem 0.75rem;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: rgba(255, 255, 255, 0.75);
  font-size: 0.82rem;
  text-align: left;
  cursor: pointer;
  transition: all 0.15s ease;
}

.topic-dropdown-item:hover {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.95);
}

.topic-dropdown-item--active {
  background: rgba(240, 138, 75, 0.2) !important;
  color: rgba(255, 235, 220, 0.95) !important;
}

.topic-dropdown-toggle {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.35rem;
  width: 100%;
  padding: 0.6rem 0.75rem;
  margin-top: 0.35rem;
  border-radius: 8px;
  border: none;
  background: rgba(240, 138, 75, 0.15);
  color: rgba(255, 220, 200, 0.85);
  font-size: 0.78rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.topic-dropdown-toggle:hover {
  background: rgba(240, 138, 75, 0.25);
  color: rgba(255, 235, 220, 0.95);
}

.topic-search-no-results {
  padding: 1rem;
  text-align: center;
  color: rgba(255, 255, 255, 0.45);
  font-size: 0.8rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.topic-dropdown-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.4rem 0.9rem;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.7);
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.15s ease;
  align-self: center;
}

.topic-dropdown-close:hover {
  background: rgba(255, 255, 255, 0.15);
  color: rgba(255, 255, 255, 0.9);
}

.topic-quick-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.topic-more-hint {
  display: inline-flex;
  align-items: center;
  padding: 0.4rem 0.75rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.4);
  font-size: 0.75rem;
  border: none;
  cursor: pointer;
  transition: all 0.15s ease;
}

.topic-more-hint:hover {
  background: rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.6);
}

.topic-count {
  margin-left: 0.5rem;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  color: rgba(255, 255, 255, 0.5);
  font-size: 0.7rem;
}
</style>
