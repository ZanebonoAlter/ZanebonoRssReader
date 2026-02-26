<script setup lang="ts">
import { Icon } from '@iconify/vue'
import type { RssFeed, Article } from '~/types'
import { useReadingTracker, useScrollDepthTracker } from '~/composables/useReadingTracker'

interface Props {
  article: Article | null
  articles?: Article[]
  onClose?: () => void
}

const props = withDefaults(defineProps<Props>(), {
  article: null,
  articles: () => [],
  onClose: () => {}
})

const emit = defineEmits<{
  favorite: [id: string]
  'navigate': [article: Article]
}>()

const feedsStore = useFeedsStore()
const { isAIEnabled } = useAI()

const viewMode = ref<'preview' | 'iframe'>('preview')
const iframeLoading = ref(true)
const showAISummary = ref(false)
const isFullscreen = ref(false)

const feed = computed(() => {
  if (!props.article) return null
  return feedsStore.feeds.find((f: RssFeed) => f.id === props.article?.feedId)
})

const currentIndex = computed(() => {
  if (!props.article || !props.articles.length) return -1
  return props.articles.findIndex(a => a.id === props.article?.id)
})

const hasPrev = computed(() => currentIndex.value > 0)
const hasNext = computed(() => currentIndex.value < props.articles.length - 1)

const contentContainer = ref<HTMLElement>()
let lastScrollDepth = 0

const { readingTime, trackEvent, uploadEvents } = useReadingTracker({
  article: computed(() => props.article),
})

useScrollDepthTracker(contentContainer, (depth) => {
  if (depth > lastScrollDepth && Math.abs(depth - lastScrollDepth) >= 10) {
    lastScrollDepth = depth
    trackEvent('scroll', depth, readingTime.value)
  }
})

watch(() => props.article, (newArticle) => {
  if (newArticle && !newArticle.read) {
    const apiStore = useApiStore()
    apiStore.markAsRead(newArticle.id)
  }
  iframeLoading.value = true
  viewMode.value = 'preview'
})

watch(() => props.article, (newArticle) => {
  if (newArticle) {
    useHead({
      title: `${newArticle.title} - RSS Reader`,
      meta: [
        { name: 'description', content: newArticle.description }
      ]
    })
  }
})

function toggleAISummary() {
  showAISummary.value = !showAISummary.value
}

function handleFavorite() {
  if (props.article) {
    const isFav = !props.article.favorite
    emit('favorite', props.article.id)
    trackEvent(isFav ? 'favorite' : 'unfavorite', lastScrollDepth, readingTime.value)
    uploadEvents()
  }
}

function openOriginal() {
  if (props.article?.link) {
    window.open(props.article.link, '_blank')
  }
}

function toggleViewMode() {
  viewMode.value = viewMode.value === 'preview' ? 'iframe' : 'preview'
  if (viewMode.value === 'iframe') {
    iframeLoading.value = true
  }
}

function toggleFullscreen() {
  isFullscreen.value = !isFullscreen.value
}

function handleIframeLoad() {
  iframeLoading.value = false
}

function handleIframeError() {
  iframeLoading.value = false
}

function navigatePrev() {
  if (hasPrev.value && props.article) {
    const prevArticle = props.articles[currentIndex.value - 1]
    if (prevArticle) {
      emit('navigate', prevArticle)
    }
  }
}

function navigateNext() {
  if (hasNext.value && props.article) {
    const nextArticle = props.articles[currentIndex.value + 1]
    if (nextArticle) {
      emit('navigate', nextArticle)
    }
  }
}

const aiEnabled = isAIEnabled

import './ArticleContent.css'
</script>

<template>
  <!-- 加载状态 -->
  <div v-if="!article" class="h-full flex items-center justify-center bg-white">
    <div class="text-center">
      <Icon icon="mdi:file-document-outline" width="64" height="64" class="text-gray-300 mx-auto mb-4" />
      <h3 class="text-xl font-semibold text-gray-700 mb-2">选择一篇文章开始阅读</h3>
      <p class="text-gray-500">点击左侧文章列表查看内容</p>
    </div>
  </div>

  <!-- 普通模式 -->
  <div
    v-else-if="!isFullscreen"
    class="article-content h-full flex flex-col"
  >
    <!-- 文章头部 -->
    <header class="article-header">
      <div class="header-left">
        <div
          v-if="feed"
          class="feed-badge"
        >
          <Icon :icon="feed.icon || 'mdi:rss'" :style="{ color: feed.color }" width="16" height="16" />
          <span class="text-sm font-medium" :style="{ color: feed.color }">
            {{ feed.title }}
          </span>
        </div>
        <span class="article-title">{{ article.title }}</span>
      </div>
      <div class="header-actions">
        <!-- 上一篇/下一篇 -->
        <template v-if="articles.length > 1">
          <button
            class="action-btn"
            :class="{ 'opacity-30 cursor-not-allowed': !hasPrev }"
            :disabled="!hasPrev"
            title="上一篇文章"
            @click="navigatePrev"
          >
            <Icon icon="mdi:chevron-up" width="20" height="20" />
          </button>
          <button
            class="action-btn"
            :class="{ 'opacity-30 cursor-not-allowed': !hasNext }"
            :disabled="!hasNext"
            title="下一篇文章"
            @click="navigateNext"
          >
            <Icon icon="mdi:chevron-down" width="20" height="20" />
          </button>
          <div class="w-px h-5 bg-ink-200 mx-1" />
        </template>

        <!-- AI 总结按钮 -->
        <button
          v-if="aiEnabled"
          class="action-btn"
          :class="{ active: showAISummary }"
          title="AI 总结分析"
          @click="toggleAISummary"
        >
          <Icon icon="mdi:brain" width="20" height="20" />
        </button>
        <!-- 视图切换 -->
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
        <!-- 收藏 -->
        <button
          class="action-btn"
          :class="{ active: article.favorite }"
          :title="article.favorite ? '取消收藏' : '收藏'"
          @click="handleFavorite"
        >
          <Icon
            :icon="article.favorite ? 'mdi:star' : 'mdi:star-outline'"
            width="20"
            height="20"
          />
        </button>
        <!-- 全屏 -->
        <button
          class="action-btn"
          title="全屏"
          @click="toggleFullscreen"
        >
          <Icon icon="mdi:fullscreen" width="20" height="20" />
        </button>
        <!-- 原文链接 -->
        <button
          class="action-btn"
          title="在新窗口打开原文"
          @click="openOriginal"
        >
          <Icon icon="mdi:external-link" width="20" height="20" />
        </button>
      </div>
    </header>

    <!-- 预览模式 -->
    <div ref="contentContainer" v-if="viewMode === 'preview'" class="preview-mode flex-1 overflow-y-auto">
      <!-- AI 总结 -->
      <AISummary
        v-if="showAISummary && article"
        :title="article.title"
        :content="article.content || article.description || ''"
        class="mb-6"
        @close="showAISummary = false"
      />

      <!-- 文章元数据 -->
      <div class="article-meta">
        <span>{{ $dayjs(article.pubDate).format('YYYY年MM月DD日 HH:mm') }}</span>
        <span v-if="article.author">作者：{{ article.author }}</span>
        <span
          v-if="article.read"
          class="read-badge"
        >
          <Icon icon="mdi:check-circle" width="14" height="14" />
          已读
        </span>
      </div>

      <!-- 文章标题 -->
      <h1 class="article-title-full">{{ article.title }}</h1>

      <!-- 文章描述 -->
      <div v-if="article.description" class="article-description">
        <div v-html="article.description" />
      </div>

      <!-- 特征图片 -->
      <div
        v-if="article.imageUrl"
        class="article-image"
      >
        <img
          :src="article.imageUrl"
          :alt="article.title"
          class="w-full"
        >
      </div>

      <!-- 文章内容 -->
      <div class="article-body">
        <div
          v-if="article.content"
          v-html="article.content"
        />
        <div v-else class="empty-content">
          <button
            class="btn btn-primary mt-4"
            @click="openOriginal"
          >
            前往原文阅读
          </button>
        </div>
      </div>
    </div>

    <!-- Iframe 模式 -->
    <div v-else class="iframe-mode flex-1 relative">
      <!-- 加载状态 -->
      <div
        v-if="iframeLoading"
        class="iframe-loading"
      >
        <div class="text-center">
          <Icon icon="mdi:loading" width="48" height="48" class="animate-spin text-blue-600 mx-auto mb-4" />
          <p class="text-gray-600">正在加载网页...</p>
        </div>
      </div>

      <!-- Iframe -->
      <iframe
        v-if="article.link"
        :src="article.link"
        class="w-full h-full border-0"
        title="Article Content"
        @load="handleIframeLoad"
        @error="handleIframeError"
        sandbox="allow-same-origin allow-scripts allow-forms allow-popups"
      />

      <!-- 错误状态 -->
      <div v-else class="iframe-error">
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

  <!-- 全屏模式 -->
  <Teleport v-else to="body">
    <div class="fullscreen-article fixed inset-0 z-50 bg-white flex flex-col">
      <!-- 文章头部 -->
      <header class="article-header">
        <div class="header-left">
          <button
            class="p-2 rounded-lg hover:bg-ink-50 transition-all duration-200 flex items-center gap-1 text-ink-medium hover:text-ink-dark"
            @click="toggleFullscreen"
          >
            <Icon icon="mdi:arrow-left" width="20" height="20" />
            <span class="text-sm">退出全屏</span>
          </button>
          <div
            v-if="feed"
            class="feed-badge"
          >
            <Icon :icon="feed.icon || 'mdi:rss'" :style="{ color: feed.color }" width="16" height="16" />
            <span class="text-sm font-medium" :style="{ color: feed.color }">
              {{ feed.title }}
            </span>
          </div>
        </div>
        <div class="header-actions">
          <!-- 上一篇/下一篇 -->
          <template v-if="articles.length > 1">
            <button
              class="action-btn"
              :class="{ 'opacity-30 cursor-not-allowed': !hasPrev }"
              :disabled="!hasPrev"
              title="上一篇文章"
              @click="navigatePrev"
            >
              <Icon icon="mdi:chevron-up" width="20" height="20" />
            </button>
            <button
              class="action-btn"
              :class="{ 'opacity-30 cursor-not-allowed': !hasNext }"
              :disabled="!hasNext"
              title="下一篇文章"
              @click="navigateNext"
            >
              <Icon icon="mdi:chevron-down" width="20" height="20" />
            </button>
            <div class="w-px h-5 bg-ink-200 mx-1" />
          </template>

          <!-- AI 总结按钮 -->
          <button
            v-if="aiEnabled"
            class="action-btn"
            :class="{ active: showAISummary }"
            title="AI 总结分析"
            @click="toggleAISummary"
          >
            <Icon icon="mdi:brain" width="20" height="20" />
          </button>
          <!-- 视图切换 -->
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
          <!-- 收藏 -->
          <button
            class="action-btn"
            :class="{ active: article.favorite }"
            :title="article.favorite ? '取消收藏' : '收藏'"
            @click="handleFavorite"
          >
            <Icon
              :icon="article.favorite ? 'mdi:star' : 'mdi:star-outline'"
              width="20"
              height="20"
            />
          </button>
          <!-- 退出全屏 -->
          <button
            class="action-btn"
            title="退出全屏"
            @click="toggleFullscreen"
          >
            <Icon icon="mdi:fullscreen-exit" width="20" height="20" />
          </button>
          <!-- 原文链接 -->
          <button
            class="action-btn"
            title="在新窗口打开原文"
            @click="openOriginal"
          >
            <Icon icon="mdi:external-link" width="20" height="20" />
          </button>
        </div>
      </header>

      <!-- 预览模式 -->
      <div v-if="viewMode === 'preview'" class="preview-mode flex-1 overflow-y-auto">
        <!-- AI 总结 -->
        <AISummary
          v-if="showAISummary && article"
          :title="article.title"
          :content="article.content || article.description || ''"
          class="mb-6"
          @close="showAISummary = false"
        />

        <!-- 文章元数据 -->
        <div class="article-meta">
          <span>{{ $dayjs(article.pubDate).format('YYYY年MM月DD日 HH:mm') }}</span>
          <span v-if="article.author">作者：{{ article.author }}</span>
          <span
            v-if="article.read"
            class="read-badge"
          >
            <Icon icon="mdi:check-circle" width="14" height="14" />
            已读
          </span>
        </div>

        <!-- 文章标题 -->
        <h1 class="article-title-full">{{ article.title }}</h1>

        <!-- 文章描述 -->
        <div v-if="article.description" class="article-description">
          <div v-html="article.description" />
        </div>

        <!-- 特征图片 -->
        <div
          v-if="article.imageUrl"
          class="article-image"
        >
          <img
            :src="article.imageUrl"
            :alt="article.title"
            class="w-full"
          >
        </div>

        <!-- 文章内容 -->
        <div class="article-body">
          <div
            v-if="article.content"
            v-html="article.content"
          />
          <div v-else class="empty-content">
            <button
              class="btn btn-primary mt-4"
              @click="openOriginal"
            >
              前往原文阅读
            </button>
          </div>
        </div>
      </div>

      <!-- Iframe 模式 -->
      <div v-else class="iframe-mode flex-1 relative">
        <!-- 加载状态 -->
        <div
          v-if="iframeLoading"
          class="iframe-loading"
        >
          <div class="text-center">
            <Icon icon="mdi:loading" width="48" height="48" class="animate-spin text-blue-600 mx-auto mb-4" />
            <p class="text-gray-600">正在加载网页...</p>
          </div>
        </div>

        <!-- Iframe -->
        <iframe
          v-if="article.link"
          :src="article.link"
          class="w-full h-full border-0"
          title="Article Content"
          @load="handleIframeLoad"
          @error="handleIframeError"
          sandbox="allow-same-origin allow-scripts allow-forms allow-popups"
        />

        <!-- 错误状态 -->
        <div v-else class="iframe-error">
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
  </Teleport>
</template>

<style scoped>
.fullscreen-article {
  background: white;
}

.fullscreen-article .article-header {
  flex-shrink: 0;
}

.fullscreen-article .preview-mode {
  flex: 1;
  overflow-y: auto;
}

.fullscreen-article .iframe-mode {
  flex: 1;
}
</style>
