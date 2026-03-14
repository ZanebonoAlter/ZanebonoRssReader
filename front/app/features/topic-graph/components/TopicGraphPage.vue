<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { computed, ref, watch } from 'vue'
import { useArticlesApi } from '~/api/articles'
import { useTopicGraphApi, type TopicGraphDetailPayload, type TopicGraphType } from '~/api/topicGraph'
import type { Article } from '~/types'
import ArticleContentView from '~/features/articles/components/ArticleContentView.vue'
import TopicGraphCanvas from '~/features/topic-graph/components/TopicGraphCanvas.client.vue'
import TopicGraphFooterPanels from '~/features/topic-graph/components/TopicGraphFooterPanels.vue'
import TopicGraphHeader from '~/features/topic-graph/components/TopicGraphHeader.vue'
import TopicGraphSidebar from '~/features/topic-graph/components/TopicGraphSidebar.vue'
import { buildTopicGraphViewModel } from '~/features/topic-graph/utils/buildTopicGraphViewModel'

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
const detail = ref<TopicGraphDetailPayload | null>(null)
const loadingGraph = ref(false)
const loadingDetail = ref(false)
const loadingPreviewArticle = ref(false)
const notice = ref<string | null>(null)
const selectedPreviewArticle = ref<Article | null>(null)
const previewArticles = ref<Article[]>([])

const viewModel = computed(() => graphPayload.value
  ? buildTopicGraphViewModel(graphPayload.value)
  : buildTopicGraphViewModel({
      type: selectedType.value,
      anchor_date: selectedDate.value,
      period_label: '正在载入',
      topic_count: 0,
      summary_count: 0,
      feed_count: 0,
      top_topics: [],
      nodes: [],
      edges: [],
    }))

const activeTopicNode = computed(() => viewModel.value.graph.nodes.find(node => node.slug === selectedTopicSlug.value) || null)
const topTopicLabels = computed(() => viewModel.value.topTopics.slice(0, 6))
const statCards = computed(() => ([
  { label: '主题数', value: viewModel.value.stats.topicCount },
  { label: '总结数', value: viewModel.value.stats.summaryCount },
  { label: 'Feed 数', value: viewModel.value.stats.feedCount },
]))

const pageState = computed(() => {
  if (loadingGraph.value) return 'loading'
  if (selectedPreviewArticle.value) return 'article-preview'
  if (detail.value) return 'detail'
  if (graphPayload.value) return 'graph-ready'
  return 'empty'
})

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

    if (selectedTopicSlug.value) {
      void loadTopicDetail(selectedTopicSlug.value)
    } else {
      detail.value = null
    }
  } catch (error) {
    console.error('Failed to load topic graph:', error)
    notice.value = error instanceof Error ? error.message : '主题图谱加载失败'
  } finally {
    loadingGraph.value = false
  }
}

async function loadTopicDetail(slug: string) {
  selectedTopicSlug.value = slug
  loadingDetail.value = true

  try {
    const response = await topicGraphApi.getTopicDetail(slug, selectedType.value, selectedDate.value)
    if (response.success && response.data) {
      detail.value = response.data
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

function handleNodeClick(node: { slug?: string; kind: string }) {
  if (node.kind !== 'topic' || !node.slug) return
  void loadTopicDetail(node.slug)
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
    contentStatus: article.content_status,
    fullContent: article.full_content,
    contentFetchedAt: article.content_fetched_at,
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

watch(selectedType, () => {
  void loadGraph()
})

watch(selectedDate, () => {
  void loadGraph()
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

                <div class="mt-6">
                  <p class="text-xs uppercase tracking-[0.24em] text-white/42">热点题材</p>
                  <div class="mt-3 flex flex-wrap gap-2 xl:flex-col xl:items-stretch">
                    <button
                      v-for="topic in topTopicLabels"
                      :key="topic.slug"
                      type="button"
                      class="topic-badge text-left"
                      :class="{ 'topic-badge--active': selectedTopicSlug === topic.slug }"
                      @click="loadTopicDetail(topic.slug)"
                    >
                      {{ topic.label }}
                    </button>
                  </div>
                </div>
              </aside>

              <div class="space-y-4">
                <TopicGraphCanvas
                  :nodes="viewModel.graph.nodes"
                  :edges="viewModel.graph.edges"
                  :featured-node-ids="viewModel.graph.featuredNodeIds"
                  :active-node-id="activeTopicNode?.id || null"
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

                <TopicGraphFooterPanels :detail="detail" />
              </div>
            </div>
          </article>

          <p v-if="notice" class="rounded-[24px] border border-[rgba(240,138,75,0.28)] bg-[rgba(240,138,75,0.1)] px-4 py-3 text-sm text-[rgba(255,233,220,0.88)]">
            {{ notice }}
          </p>
        </div>

        <div class="topic-reading-rail" data-testid="topic-graph-sidebar-region">
          <TopicGraphSidebar
            :detail="detail"
            :loading="loadingDetail"
            :error="notice"
            :data-state="detail ? 'detail' : (loadingDetail ? 'loading' : 'empty')"
            @open-article="openArticlePreview"
          />
        </div>
      </section>
    </div>

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
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  padding: 0.55rem 0.9rem;
  font-size: 0.78rem;
  color: rgba(255, 255, 255, 0.78);
  background: rgba(255,255,255,0.04);
}

.topic-badge--active {
  border-color: rgba(240, 138, 75, 0.72);
  background: rgba(240, 138, 75, 0.18);
  color: white;
}
</style>
