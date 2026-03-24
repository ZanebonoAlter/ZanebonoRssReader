<script setup lang="ts">
import { Icon } from "@iconify/vue"
import { marked } from 'marked'
import type { Article } from '~/types'
import { useArticlesApi } from '~/api/articles'

interface AISummary {
  id: number
  feed_id: number | null
  category_id: number | null
  title: string
  summary: string
  key_points: string
  articles: string
  article_count: number
  time_range: number
  created_at: string
  updated_at: string
  category_name: string
  feed_name: string
  feed_icon: string
  feed_color: string
}

const props = defineProps<{
  summary: AISummary | null
}>()

const emit = defineEmits<{
  'close': []
}>()

const articlesApi = useArticlesApi()
const apiStore = useApiStore()
const feedsStore = useFeedsStore()

const expandedArticle = ref<Article | null>(null)
const summaryArticles = ref<Article[]>([])
const loadingArticles = ref(false)
const viewMode = ref<'preview' | 'iframe'>('preview')
const iframeLoading = ref(true)
const isFullscreen = ref(false)

watch(() => props.summary, async (newSummary) => {
  console.log('Summary changed:', newSummary?.id, newSummary?.feed_name)
  expandedArticle.value = null
  if (newSummary && newSummary.articles) {
    await loadSummaryArticles(newSummary.articles)
  }
}, { immediate: true })

async function loadSummaryArticles(articlesJson: string) {
  loadingArticles.value = true
  try {
    const articleIds: number[] = JSON.parse(articlesJson)
    const articles: Article[] = []
    for (const id of articleIds) {
      const response = await articlesApi.getArticle(id)
      if (response.success && response.data) {
        const art = response.data as any
        articles.push({
          id: String(art.id),
          feedId: String(art.feed_id),
          title: art.title,
          description: art.description || '',
          content: art.content || '',
          link: art.link,
          pubDate: art.pub_date,
          author: art.author,
          category: art.category || '',
          read: art.read || false,
          favorite: art.favorite || false,
          imageUrl: art.image_url,
        })
      }
    }
    summaryArticles.value = articles
  } catch (e) {
    console.error('Failed to load summary articles:', e)
  } finally {
    loadingArticles.value = false
  }
}

function findArticleByLink(link: string): Article | undefined {
  return summaryArticles.value.find(a => a.link === link)
}

function handleContentClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (target.tagName === 'A') {
    const href = target.getAttribute('href')
    if (href) {
      const article = findArticleByLink(href)
      if (article) {
        event.preventDefault()
        expandArticle(article)
      }
    }
  }
}

function expandArticle(article: Article) {
  expandedArticle.value = article
  viewMode.value = 'preview'
  iframeLoading.value = true
  if (!article.read) {
    updateArticleLocal(article.id, { read: true })
    apiStore.markAsRead(article.id)
  }
}

function closeExpandedArticle() {
  expandedArticle.value = null
}

function updateArticleLocal(articleId: string, updates: Partial<Article>) {
  const index = summaryArticles.value.findIndex(a => a.id === articleId)
  if (index !== -1 && summaryArticles.value[index]) {
    Object.assign(summaryArticles.value[index], updates)
  }
}

async function toggleFavorite() {
  if (expandedArticle.value) {
    const newFavorite = !expandedArticle.value.favorite
    updateArticleLocal(expandedArticle.value.id, { favorite: newFavorite })
    expandedArticle.value = {
      ...expandedArticle.value,
      favorite: newFavorite
    }
    await apiStore.toggleFavorite(expandedArticle.value.id)
  }
}

function openOriginal() {
  if (expandedArticle.value?.link) {
    window.open(expandedArticle.value.link, '_blank')
  }
}

function toggleViewMode() {
  viewMode.value = viewMode.value === 'preview' ? 'iframe' : 'preview'
  if (viewMode.value === 'iframe') {
    iframeLoading.value = true
  }
}

function handleIframeLoad() {
  iframeLoading.value = false
}

function handleIframeError() {
  iframeLoading.value = false
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
}

function navigatePrev() {
  if (hasPrevArticle.value && expandedArticle.value) {
    const prevArticle = summaryArticles.value[currentArticleIndex.value - 1]
    if (prevArticle) {
      expandArticle(prevArticle)
    }
  }
}

function navigateNext() {
  if (hasNextArticle.value && expandedArticle.value) {
    const nextArticle = summaryArticles.value[currentArticleIndex.value + 1]
    if (nextArticle) {
      expandArticle(nextArticle)
    }
  }
}

const renderedSummary = computed(() => {
  if (!props.summary) return ''
  return marked(props.summary.summary)
})

const expandedArticleFeed = computed(() => {
  if (!expandedArticle.value) return null
  return feedsStore.feeds.find((f: any) => f.id === expandedArticle.value?.feedId)
})

const currentArticleIndex = computed(() => {
  if (!expandedArticle.value) return -1
  return summaryArticles.value.findIndex(a => a.id === expandedArticle.value?.id)
})

const hasPrevArticle = computed(() => currentArticleIndex.value > 0)
const hasNextArticle = computed(() => currentArticleIndex.value < summaryArticles.value.length - 1)

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatTimeRange = (minutes: number): string => {
  const hours = Math.floor(minutes / 60)
  if (hours >= 24) {
    const days = Math.floor(hours / 24)
    return `${days} 天`
  }
  return `${hours} 小时`
}
</script>

<template>
  <div
    v-if="summary"
    class="h-full flex flex-col"
  >
    <!-- 展开的文章视图 - 普通模式 -->
    <div
      v-if="expandedArticle && !isFullscreen"
      class="h-full flex flex-col"
    >
      <div class="flex-shrink-0 bg-white/95 backdrop-blur-sm border-b border-ink-200 px-6 py-4 flex items-center justify-between shadow-subtle">
        <div class="flex items-center gap-3">
          <button
            class="p-2 rounded-lg hover:bg-ink-50 transition-all duration-200 flex items-center gap-1 text-ink-medium hover:text-ink-dark"
            @click="closeExpandedArticle"
          >
            <Icon icon="mdi:arrow-left" width="20" height="20" />
            <span class="text-sm">返回总结</span>
          </button>
          <div
            v-if="expandedArticleFeed"
            class="feed-badge"
          >
            <Icon :icon="expandedArticleFeed.icon || 'mdi:rss'" :style="{ color: expandedArticleFeed.color }" width="16" height="16" />
            <span class="text-sm font-medium" :style="{ color: expandedArticleFeed.color }">
              {{ expandedArticleFeed.title }}
            </span>
          </div>
        </div>
        <div class="flex items-center gap-2">
          <!-- 上一篇/下一篇 -->
          <template v-if="summaryArticles.length > 1">
            <button
              class="action-btn"
              :class="{ 'opacity-30 cursor-not-allowed': !hasPrevArticle }"
              :disabled="!hasPrevArticle"
              title="上一篇文章"
              @click="navigatePrev"
            >
              <Icon icon="mdi:chevron-up" width="20" height="20" />
            </button>
            <button
              class="action-btn"
              :class="{ 'opacity-30 cursor-not-allowed': !hasNextArticle }"
              :disabled="!hasNextArticle"
              title="下一篇文章"
              @click="navigateNext"
            >
              <Icon icon="mdi:chevron-down" width="20" height="20" />
            </button>
            <div class="w-px h-5 bg-ink-200 mx-1" />
          </template>
          <button
            class="action-btn"
            :title="viewMode === 'preview' ? '切换到内嵌网页' : '切换到内容预览'"
            @click="toggleViewMode"
          >
            <Icon
              :icon="viewMode === 'preview' ? 'mdi:web' : 'mdi:file-document-outline'"
              width="20"
              height="20"
            />
          </button>
          <button
            class="action-btn"
            :class="{ active: expandedArticle.favorite }"
            :title="expandedArticle.favorite ? '取消收藏' : '收藏'"
            @click="toggleFavorite"
          >
            <Icon
              :icon="expandedArticle.favorite ? 'mdi:star' : 'mdi:star-outline'"
              width="20"
              height="20"
            />
          </button>
          <button
            class="action-btn"
            title="全屏"
            @click="toggleFullscreen"
          >
            <Icon icon="mdi:fullscreen" width="20" height="20" />
          </button>
          <button
            class="action-btn"
            title="在新窗口打开原文"
            @click="openOriginal"
          >
            <Icon icon="mdi:external-link" width="20" height="20" />
          </button>
        </div>
      </div>

      <div class="flex-1 overflow-y-auto">
        <!-- 预览模式 -->
        <div v-if="viewMode === 'preview'" class="p-6">
          <article class="expanded-article">
            <div class="article-meta">
              <span>{{ $dayjs(expandedArticle.pubDate).format('YYYY年MM月DD日 HH:mm') }}</span>
              <span v-if="expandedArticle.author">作者：{{ expandedArticle.author }}</span>
              <span
                v-if="expandedArticle.read"
                class="read-badge"
              >
                <Icon icon="mdi:check-circle" width="14" height="14" />
                已读
              </span>
            </div>

            <h1 class="article-title-full">{{ expandedArticle.title }}</h1>

            <div
              v-if="expandedArticle.description"
              class="article-description"
            >
              <div v-html="expandedArticle.description" />
            </div>

            <div
              v-if="expandedArticle.imageUrl"
              class="article-image"
            >
              <img
                :src="expandedArticle.imageUrl"
                :alt="expandedArticle.title"
                class="w-full"
              >
            </div>

            <div class="article-body">
              <div
                v-if="expandedArticle.content"
                v-html="expandedArticle.content"
              />
              <div
                v-else
                class="empty-content"
              >
                <p class="text-ink-light mb-4">暂无正文内容</p>
                <button
                  class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700 transition-colors"
                  @click="openOriginal"
                >
                  前往原文阅读
                </button>
              </div>
            </div>
          </article>
        </div>

        <!-- Iframe 模式 -->
        <div v-else class="h-full relative">
          <div
            v-if="iframeLoading"
            class="absolute inset-0 flex items-center justify-center bg-white"
          >
            <div class="text-center">
              <Icon icon="mdi:loading" width="48" height="48" class="animate-spin text-ink-500 mx-auto mb-4" />
              <p class="text-ink-medium">正在加载网页...</p>
            </div>
          </div>

          <iframe
            v-if="expandedArticle.link"
            :src="expandedArticle.link"
            class="w-full h-full border-0"
            title="Article Content"
            @load="handleIframeLoad"
            @error="handleIframeError"
            sandbox="allow-same-origin allow-scripts allow-forms allow-popups"
          />

          <div
            v-else
            class="h-full flex items-center justify-center"
          >
            <div class="text-center text-ink-400">
              <Icon icon="mdi:web-off" width="64" height="64" class="mb-4 mx-auto" />
              <p>无法加载网页</p>
              <button
                class="mt-4 px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700 transition-colors"
                @click="openOriginal"
              >
                在新窗口打开
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 展开的文章视图 - 全屏模式 -->
    <Teleport v-else-if="expandedArticle && isFullscreen" to="body">
      <div class="fullscreen-article fixed inset-0 z-50 bg-white flex flex-col">
        <div class="flex-shrink-0 bg-white/95 backdrop-blur-sm border-b border-ink-200 px-6 py-4 flex items-center justify-between shadow-subtle">
          <div class="flex items-center gap-3">
            <button
              class="p-2 rounded-lg hover:bg-ink-50 transition-all duration-200 flex items-center gap-1 text-ink-medium hover:text-ink-dark"
              @click="toggleFullscreen"
            >
              <Icon icon="mdi:arrow-left" width="20" height="20" />
              <span class="text-sm">退出全屏</span>
            </button>
            <div
              v-if="expandedArticleFeed"
              class="feed-badge"
            >
              <Icon :icon="expandedArticleFeed.icon || 'mdi:rss'" :style="{ color: expandedArticleFeed.color }" width="16" height="16" />
              <span class="text-sm font-medium" :style="{ color: expandedArticleFeed.color }">
                {{ expandedArticleFeed.title }}
              </span>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <!-- 上一篇/下一篇 -->
            <template v-if="summaryArticles.length > 1">
              <button
                class="action-btn"
                :class="{ 'opacity-30 cursor-not-allowed': !hasPrevArticle }"
                :disabled="!hasPrevArticle"
                title="上一篇文章"
                @click="navigatePrev"
              >
                <Icon icon="mdi:chevron-up" width="20" height="20" />
              </button>
              <button
                class="action-btn"
                :class="{ 'opacity-30 cursor-not-allowed': !hasNextArticle }"
                :disabled="!hasNextArticle"
                title="下一篇文章"
                @click="navigateNext"
              >
                <Icon icon="mdi:chevron-down" width="20" height="20" />
              </button>
              <div class="w-px h-5 bg-ink-200 mx-1" />
            </template>
            <button
              class="action-btn"
              :title="viewMode === 'preview' ? '切换到内嵌网页' : '切换到内容预览'"
              @click="toggleViewMode"
            >
              <Icon
                :icon="viewMode === 'preview' ? 'mdi:web' : 'mdi:file-document-outline'"
                width="20"
                height="20"
              />
            </button>
            <button
              class="action-btn"
              :class="{ active: expandedArticle.favorite }"
              :title="expandedArticle.favorite ? '取消收藏' : '收藏'"
              @click="toggleFavorite"
            >
              <Icon
                :icon="expandedArticle.favorite ? 'mdi:star' : 'mdi:star-outline'"
                width="20"
                height="20"
              />
            </button>
            <button
              class="action-btn"
              title="退出全屏"
              @click="toggleFullscreen"
            >
              <Icon icon="mdi:fullscreen-exit" width="20" height="20" />
            </button>
            <button
              class="action-btn"
              title="在新窗口打开原文"
              @click="openOriginal"
            >
              <Icon icon="mdi:external-link" width="20" height="20" />
            </button>
          </div>
        </div>

        <div class="flex-1 overflow-y-auto">
          <!-- 预览模式 -->
          <div v-if="viewMode === 'preview'" class="p-6">
            <article class="expanded-article">
              <div class="article-meta">
                <span>{{ $dayjs(expandedArticle.pubDate).format('YYYY年MM月DD日 HH:mm') }}</span>
                <span v-if="expandedArticle.author">作者：{{ expandedArticle.author }}</span>
                <span
                  v-if="expandedArticle.read"
                  class="read-badge"
                >
                  <Icon icon="mdi:check-circle" width="14" height="14" />
                  已读
                </span>
              </div>

              <h1 class="article-title-full">{{ expandedArticle.title }}</h1>

              <div
                v-if="expandedArticle.description"
                class="article-description"
              >
                <div v-html="expandedArticle.description" />
              </div>

              <div
                v-if="expandedArticle.imageUrl"
                class="article-image"
              >
                <img
                  :src="expandedArticle.imageUrl"
                  :alt="expandedArticle.title"
                  class="w-full"
                >
              </div>

              <div class="article-body">
                <div
                  v-if="expandedArticle.content"
                  v-html="expandedArticle.content"
                />
                <div
                  v-else
                  class="empty-content"
                >
                  <p class="text-ink-light mb-4">暂无正文内容</p>
                  <button
                    class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700 transition-colors"
                    @click="openOriginal"
                  >
                    前往原文阅读
                  </button>
                </div>
              </div>
            </article>
          </div>

          <!-- Iframe 模式 -->
          <div v-else class="h-full relative">
            <div
              v-if="iframeLoading"
              class="absolute inset-0 flex items-center justify-center bg-white"
            >
              <div class="text-center">
                <Icon icon="mdi:loading" width="48" height="48" class="animate-spin text-ink-500 mx-auto mb-4" />
                <p class="text-ink-medium">正在加载网页...</p>
              </div>
            </div>

            <iframe
              v-if="expandedArticle.link"
              :src="expandedArticle.link"
              class="w-full h-full border-0"
              title="Article Content"
              @load="handleIframeLoad"
              @error="handleIframeError"
              sandbox="allow-same-origin allow-scripts allow-forms allow-popups"
            />

            <div
              v-else
              class="h-full flex items-center justify-center"
            >
              <div class="text-center text-ink-400">
                <Icon icon="mdi:web-off" width="64" height="64" class="mb-4 mx-auto" />
                <p>无法加载网页</p>
                <button
                  class="mt-4 px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700 transition-colors"
                  @click="openOriginal"
                >
                  在新窗口打开
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- AI 总结视图 -->
    <template v-else>
      <div class="flex-shrink-0 bg-white/95 backdrop-blur-sm border-b border-ink-200 px-6 py-4 flex items-center justify-between shadow-subtle">
        <div>
          <h1 class="text-xl font-bold text-ink-black flex items-center gap-2">
            <div
              class="w-8 h-8 rounded-lg flex items-center justify-center shadow-md"
              :style="{ backgroundColor: summary.feed_color || '#3b6b87' }"
            >
              <Icon
                :icon="summary.feed_icon || 'mdi:rss'"
                width="18"
                height="18"
                class="text-white"
              />
            </div>
            {{ summary.feed_name || summary.title }}
          </h1>
          <div class="flex items-center gap-2 mt-2.5 flex-wrap">
            <span
              v-if="summary.category_name"
              class="px-3 py-1 rounded-full text-xs font-medium bg-ink-100 text-ink-700"
            >
              {{ summary.category_name }}
            </span>
            <span class="flex items-center gap-1 text-xs text-ink-medium">
              <Icon icon="mdi:file-document-multiple" width="14" height="14" />
              {{ summary.article_count }} 篇文章
            </span>
            <span class="flex items-center gap-1 text-xs text-ink-medium">
              <Icon icon="mdi:clock-outline" width="14" height="14" />
              {{ formatTimeRange(summary.time_range) }}
            </span>
            <span class="flex items-center gap-1 text-xs text-ink-medium">
              <Icon icon="mdi:calendar" width="14" height="14" />
              {{ formatDate(summary.created_at) }}
            </span>
          </div>
        </div>
        <button
          class="p-2.5 rounded-lg hover:bg-ink-50 transition-all duration-200"
          @click="emit('close')"
        >
          <Icon icon="mdi:close" width="20" height="20" class="text-ink-medium" />
        </button>
      </div>

      <div class="flex-1 overflow-y-auto p-6">
        <div class="prose max-w-none">
          <div
            v-html="renderedSummary"
            class="ai-summary-content"
            @click="handleContentClick"
          />
        </div>
        <div
          v-if="loadingArticles"
          class="mt-6 text-center text-ink-light"
        >
          <Icon icon="mdi:loading" width="24" height="24" class="animate-spin inline" />
          <span class="ml-2">加载相关文章...</span>
        </div>
      </div>
    </template>
  </div>

  <div
    v-else
    class="h-full flex items-center justify-center paper-card rounded-lg"
  >
    <div class="text-center p-8">
      <div class="w-20 h-20 mx-auto mb-4 rounded-lg bg-gradient-to-br from-ink-100 to-paper-warm flex items-center justify-center">
        <Icon icon="mdi:brain" width="40" height="40" class="text-ink-400" />
      </div>
      <h3 class="text-lg font-semibold text-ink-dark mb-1">选择 AI 总结</h3>
      <p class="text-sm text-ink-light">从左侧列表中选择一个总结查看详情</p>
    </div>
  </div>
</template>

<style scoped>
.action-btn {
  padding: 0.5rem;
  border-radius: 0.5rem;
  color: var(--color-ink-medium);
  transition: all 0.2s;
}

.action-btn:hover {
  background: var(--color-ink-50);
  color: var(--color-ink-dark);
}

.action-btn.active {
  color: #f59e0b;
}

.feed-badge {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.75rem;
  background: var(--color-ink-50);
  border-radius: 9999px;
}

.expanded-article {
  max-width: 720px;
  margin: 0 auto;
}

.expanded-article .article-meta {
  display: flex;
  align-items: center;
  gap: 1rem;
  font-size: 0.875rem;
  color: var(--color-ink-medium);
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
}

.expanded-article .read-badge {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  padding: 0.25rem 0.5rem;
  background: rgba(59, 107, 135, 0.1);
  border-radius: 0.375rem;
  font-size: 0.75rem;
  color: var(--color-ink-600);
}

.expanded-article .article-title-full {
  font-size: 1.75rem;
  font-weight: 700;
  color: var(--color-ink-black);
  line-height: 1.3;
  margin-bottom: 1.5rem;
}

.expanded-article .article-description {
  padding: 1rem 1.25rem;
  margin-bottom: 1.5rem;
  background: var(--color-paper-warm);
  border-left: 3px solid var(--color-ink-400);
  border-radius: 0 0.5rem 0.5rem 0;
  color: var(--color-ink-medium);
  font-style: italic;
}

.expanded-article .article-image {
  margin-bottom: 2rem;
  border-radius: 0.5rem;
  overflow: hidden;
  box-shadow: var(--shadow-medium);
}

.expanded-article .article-body {
  color: var(--color-ink-dark);
  line-height: 1.75;
}

.expanded-article .article-body :deep(h1),
.expanded-article .article-body :deep(h2),
.expanded-article .article-body :deep(h3) {
  font-weight: 600;
  margin-top: 1.5em;
  margin-bottom: 0.75em;
  color: var(--color-ink-black);
}

.expanded-article .article-body :deep(p) {
  margin-bottom: 1.25em;
}

.expanded-article .article-body :deep(img) {
  max-width: 100%;
  border-radius: 0.5rem;
  margin: 1.5em 0;
}

.expanded-article .article-body :deep(a) {
  color: var(--color-ink-500);
  border-bottom: 1px solid transparent;
  transition: border-color 0.2s;
}

.expanded-article .article-body :deep(a:hover) {
  border-bottom-color: var(--color-ink-500);
}

.expanded-article .empty-content {
  text-align: center;
  padding: 2rem;
  background: var(--color-paper-warm);
  border-radius: 0.5rem;
}

.ai-summary-content {
  color: var(--color-ink-dark);
  line-height: 1.75;
}

.ai-summary-content :deep(h1),
.ai-summary-content :deep(h2),
.ai-summary-content :deep(h3),
.ai-summary-content :deep(h4),
.ai-summary-content :deep(h5),
.ai-summary-content :deep(h6) {
  font-weight: 700;
  margin-top: 1.75em;
  margin-bottom: 0.75em;
  line-height: 1.3;
  color: var(--color-ink-black);
  letter-spacing: -0.01em;
}

.ai-summary-content :deep(h1) {
  font-size: 1.875em;
  padding-bottom: 0.5em;
  border-bottom: 2px solid var(--color-ink-300);
}

.ai-summary-content :deep(h2) {
  font-size: 1.5em;
  padding-bottom: 0.4em;
  border-bottom: 1px solid var(--color-ink-200);
}

.ai-summary-content :deep(h3) {
  font-size: 1.25em;
  color: var(--color-ink-dark);
}

.ai-summary-content :deep(p) {
  margin-top: 0;
  margin-bottom: 1.25em;
}

.ai-summary-content :deep(ul),
.ai-summary-content :deep(ol) {
  margin-top: 0;
  margin-bottom: 1.25em;
  padding-left: 1.75em;
}

.ai-summary-content :deep(li) {
  margin-bottom: 0.5em;
  position: relative;
}

.ai-summary-content :deep(li)::marker {
  color: var(--color-ink-500);
}

.ai-summary-content :deep(code) {
  padding: 0.2em 0.5em;
  margin: 0 0.1em;
  font-size: 0.875em;
  background: rgba(59, 107, 135, 0.08);
  border: 1px solid rgba(59, 107, 135, 0.15);
  border-radius: 4px;
  color: var(--color-ink-700);
}

.ai-summary-content :deep(pre) {
  padding: 1.25rem;
  overflow-x: auto;
  font-size: 0.875em;
  line-height: 1.6;
  background: var(--color-paper-warm);
  border: 1px solid var(--color-border-medium);
  border-radius: 0.5rem;
  margin-bottom: 1.5em;
  box-shadow: var(--shadow-subtle);
}

.ai-summary-content :deep(pre code) {
  padding: 0;
  margin: 0;
  font-size: 100%;
  background: transparent;
  border: none;
  color: inherit;
}

.ai-summary-content :deep(blockquote) {
  padding: 1em 1.25em;
  margin: 0 0 1.5em 0;
  color: var(--color-ink-medium);
  background: rgba(59, 107, 135, 0.04);
  border-left: 3px solid var(--color-ink-400);
  border-radius: 0 0.5rem 0.5rem 0;
  font-style: italic;
}

.ai-summary-content :deep(a) {
  color: var(--color-ink-500);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color 0.2s;
}

.ai-summary-content :deep(a:hover) {
  border-bottom-color: var(--color-ink-500);
}

.ai-summary-content :deep(strong) {
  font-weight: 700;
  color: var(--color-ink-900);
}

.ai-summary-content :deep(em) {
  font-style: italic;
  color: var(--color-ink-medium);
}

.ai-summary-content :deep(hr) {
  height: 2px;
  padding: 0;
  margin: 2.5em 0;
  background: linear-gradient(90deg, transparent, var(--color-ink-300), transparent);
  border: 0;
}

.ai-summary-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1.5em;
  background: white;
  border-radius: 0.5rem;
  overflow: hidden;
  box-shadow: var(--shadow-subtle);
}

.ai-summary-content :deep(th),
.ai-summary-content :deep(td) {
  padding: 0.75em 1em;
  text-align: left;
  border-bottom: 1px solid var(--color-border-subtle);
}

.ai-summary-content :deep(th) {
  background: rgba(59, 107, 135, 0.08);
  font-weight: 600;
  color: var(--color-ink-900);
}

.ai-summary-content :deep(tr:last-child td) {
  border-bottom: none;
}

.ai-summary-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 0.5rem;
  margin: 1.5em 0;
  box-shadow: var(--shadow-medium);
}

.fullscreen-article {
  background: white;
}

.fullscreen-article .feed-badge {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.25rem 0.75rem;
  background: var(--color-ink-50);
  border-radius: 9999px;
}
</style>
