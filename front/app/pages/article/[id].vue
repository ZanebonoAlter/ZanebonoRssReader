<script setup lang="ts">
import { Icon } from '@iconify/vue'
import type { RssFeed } from '~/types'
const route = useRoute()
const router = useRouter()
const apiStore = useApiStore()
const articlesStore = useArticlesStore()
const feedsStore = useFeedsStore()
const { isAIEnabled } = useAI()

// View mode: 'preview' | 'iframe'
const viewMode = ref<'preview' | 'iframe'>('preview')
const iframeLoading = ref(true)
const loading = ref(true)
const notFound = ref(false)

// AI Summary
const showAISummary = ref(false)

const article = computed(() => articlesStore.getArticleById(route.params.id as string))
const feed = computed(() => article.value ? feedsStore.feeds.find((f: RssFeed) => f.id === article.value?.feedId) : null)

// Try to fetch article from API if not in store
onMounted(async () => {
  // If article already exists, no need to load
  if (article.value) {
    loading.value = false
    return
  }

  // Try to fetch from API
  const articleId = Number(route.params.id)
  if (!isNaN(articleId)) {
    const response = await apiStore.fetchArticles({ per_page: 10000 })
    if (response.success) {
      // Check again after fetching
      if (!articlesStore.getArticleById(route.params.id as string)) {
        notFound.value = true
      }
    } else {
      notFound.value = true
    }
  } else {
    notFound.value = true
  }
  loading.value = false
})

// Check if AI is enabled - isAIEnabled is now a computed property
const aiEnabled = isAIEnabled

function toggleAISummary() {
  showAISummary.value = !showAISummary.value
}

function handleFavorite() {
  if (article.value) {
    apiStore.toggleFavorite(article.value.id)
  }
}

function openOriginal() {
  if (article.value?.link) {
    window.open(article.value.link, '_blank')
  }
}

function handleClose() {
  // Navigate back to root
  router.push('/')
}

function toggleViewMode() {
  viewMode.value = viewMode.value === 'preview' ? 'iframe' : 'preview'
}

function handleIframeLoad() {
  iframeLoading.value = false
}

function handleIframeError() {
  iframeLoading.value = false
}

// Reset iframe loading when view mode changes
watch(viewMode, () => {
  if (viewMode.value === 'iframe') {
    iframeLoading.value = true
  }
})

// Mark as read when viewing
watchEffect(() => {
  if (article.value && !article.value.read) {
    apiStore.markAsRead(article.value.id)
  }
})

// Set head only when article is available
watchEffect(() => {
  if (article.value) {
    useHead({
      title: `${article.value.title} - RSS Reader`,
      meta: [
        { name: 'description', content: article.value.description }
      ]
    })
  }
})
</script>

<template>
  <!-- Loading State -->
  <div v-if="loading" class="h-full flex items-center justify-center bg-white">
    <div class="text-center">
      <Icon icon="mdi:loading" width="48" height="48" class="animate-spin text-blue-500 mx-auto mb-4" />
      <p class="text-gray-600">加载中...</p>
    </div>
  </div>

  <!-- 404 State -->
  <div v-else-if="notFound || !article" class="h-full flex items-center justify-center bg-white">
    <div class="text-center">
      <Icon icon="mdi:file-remove" width="64" height="64" class="text-gray-400 mx-auto mb-4" />
      <h1 class="text-2xl font-bold text-gray-900 mb-2">文章未找到</h1>
      <p class="text-gray-600 mb-6">抱歉，找不到这篇文章</p>
      <button
        class="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        @click="handleClose"
      >
        返回首页
      </button>
    </div>
  </div>

  <!-- Article Content -->
  <div v-else class="h-full flex flex-col bg-white">
    <!-- Article Header with close button -->
    <header class="px-6 py-4 border-b border-gray-200 flex items-center justify-between flex-shrink-0">
      <div class="flex items-center gap-3 min-w-0">
        <div
          v-if="feed"
          class="flex items-center gap-2 px-3 py-1.5 rounded-lg flex-shrink-0"
          :style="{ backgroundColor: feed.color + '15' }"
        >
          <Icon :icon="feed.icon || 'mdi:rss'" :style="{ color: feed.color }" width="16" height="16" />
          <span class="text-sm font-medium" :style="{ color: feed.color }">
            {{ feed.title }}
          </span>
        </div>
        <span class="text-sm text-gray-500 truncate">
          {{ article?.title }}
        </span>
      </div>
      <div class="flex items-center gap-2 flex-shrink-0">
        <!-- AI Summary Button (only show if AI is enabled) -->
        <button
          v-if="aiEnabled"
          class="p-2 hover:bg-gray-100 rounded-lg transition-colors relative"
          :class="{ 'text-purple-600': showAISummary }"
          title="AI 总结分析"
          @click="toggleAISummary"
        >
          <Icon icon="mdi:brain" width="20" height="20" />
        </button>
        <!-- View Mode Toggle -->
        <button
          class="p-2 hover:bg-gray-100 rounded-lg transition-colors relative"
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
          class="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          :class="{ 'text-yellow-600': article?.favorite }"
          :title="article?.favorite ? '取消收藏' : '收藏'"
          @click="handleFavorite"
        >
          <Icon
            :icon="article?.favorite ? 'mdi:star' : 'mdi:star-outline'"
            width="20"
            height="20"
          />
        </button>
        <button
          class="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          title="在新窗口打开原文"
          @click="openOriginal"
        >
          <Icon icon="mdi:external-link" width="20" height="20" />
        </button>
        <button
          class="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          title="关闭"
          @click="handleClose"
        >
          <Icon icon="mdi:close" width="20" height="20" />
        </button>
      </div>
    </header>

    <!-- Preview Mode -->
    <div v-if="viewMode === 'preview'" class="flex-1 overflow-y-auto">
      <div class="max-w-4xl mx-auto px-6 py-8">
        <!-- AI Summary (shown when enabled and toggled on) -->
        <SummaryAISummary
          v-if="showAISummary && article"
          :title="article.title"
          :content="article.content || article.description || ''"
          class="mb-6"
          @close="showAISummary = false"
        />

        <!-- Article Meta -->
        <div class="flex flex-wrap items-center gap-3 mb-6 text-sm text-gray-500">
          <span>
            {{ $dayjs(article?.pubDate).format('YYYY年MM月DD日 HH:mm') }}
          </span>
          <span v-if="article?.author">
            作者：{{ article?.author }}
          </span>
          <span
            v-if="article?.read"
            class="flex items-center gap-1 text-green-600"
          >
            <Icon icon="mdi:check-circle" width="14" height="14" />
            已读
          </span>
        </div>

        <!-- Article Title -->
        <h1 class="text-3xl font-bold text-gray-900 mb-6">
          {{ article?.title }}
        </h1>

        <!-- Article Description -->
        <div v-if="article?.description" class="text-lg text-gray-600 mb-8 prose">
          <div v-html="article?.description" />
        </div>

        <!-- Featured Image -->
        <div
          v-if="article?.imageUrl"
          class="mb-8 rounded-xl overflow-hidden"
        >
          <img
            :src="article?.imageUrl"
            :alt="article?.title"
            class="w-full"
          >
        </div>

        <!-- Article Content -->
        <div class="prose prose-lg max-w-none">
          <div
            v-if="article?.content"
            v-html="article?.content"
          />
          <div v-else class="text-center py-12 text-gray-400">
            <button
              class="btn btn-primary mt-4"
              @click="openOriginal"
            >
              前往原文阅读
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Iframe Mode -->
    <div v-else class="flex-1 relative bg-gray-50">
      <!-- Loading State -->
      <div
        v-if="iframeLoading"
        class="absolute inset-0 flex items-center justify-center bg-gray-50 z-10"
      >
        <div class="text-center">
          <Icon icon="mdi:loading" width="48" height="48" class="animate-spin text-blue-500 mx-auto mb-4" />
          <p class="text-gray-600">正在加载网页...</p>
        </div>
      </div>

      <!-- Iframe -->
      <iframe
        v-if="article?.link"
        :src="article.link"
        class="w-full h-full border-0"
        title="Article Content"
        @load="handleIframeLoad"
        @error="handleIframeError"
        sandbox="allow-same-origin allow-scripts allow-forms allow-popups"
      />

      <!-- Error State -->
      <div v-else class="absolute inset-0 flex items-center justify-center">
        <div class="text-center text-gray-400">
          <Icon icon="mdi:web-off" width="64" height="64" class="mb-4 mx-auto" />
          <p>无法加载网页</p>
          <button
            class="btn btn-primary mt-4"
            @click="openOriginal"
          >
            在新窗口打开
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.prose {
  color: #374151;
  line-height: 1.75;
}

.prose :deep(h1) {
  font-size: 2em;
  margin-top: 1.5em;
  margin-bottom: 0.5em;
  font-weight: 700;
}

.prose :deep(h2) {
  font-size: 1.5em;
  margin-top: 1.5em;
  margin-bottom: 0.5em;
  font-weight: 600;
}

.prose :deep(p) {
  margin-top: 1em;
  margin-bottom: 1em;
}

.prose :deep(a) {
  color: #3b82f6;
  text-decoration: underline;
}

.prose :deep(a:hover) {
  color: #2563eb;
}

.prose :deep(img) {
  margin-top: 1.5em;
  margin-bottom: 1.5em;
  border-radius: 0.5rem;
}

.prose :deep(code) {
  background-color: #f3f4f6;
  padding: 0.2em 0.4em;
  border-radius: 0.25rem;
  font-size: 0.875em;
}

.prose :deep(pre) {
  background-color: #1f2937;
  color: #f9fafb;
  padding: 1em;
  border-radius: 0.5rem;
  overflow-x: auto;
  margin-top: 1.5em;
  margin-bottom: 1.5em;
}

.prose :deep(pre code) {
  background-color: transparent;
  padding: 0;
  color: inherit;
}

.prose :deep(ul), .prose :deep(ol) {
  margin-top: 1em;
  margin-bottom: 1em;
  padding-left: 1.5em;
}

.prose :deep(li) {
  margin-top: 0.5em;
  margin-bottom: 0.5em;
}

.prose :deep(blockquote) {
  border-left: 4px solid #e5e7eb;
  padding-left: 1em;
  margin-top: 1.5em;
  margin-bottom: 1.5em;
  font-style: italic;
  color: #6b7280;
}

/* Iframe styles */
iframe {
  display: block;
}
</style>
