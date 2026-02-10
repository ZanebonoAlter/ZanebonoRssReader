<script setup lang="ts">
import { Icon } from '@iconify/vue'
import type { RssFeed, Article } from '~/types'
import { useReadingTracker, useScrollDepthTracker } from '~/composables/useReadingTracker'

interface Props {
  article: Article | null
  onClose?: () => void
}

const props = withDefaults(defineProps<Props>(), {
  onClose: () => {}
})

const emit = defineEmits<{
  favorite: [id: string]
}>()

const feedsStore = useFeedsStore()
const { isAIEnabled } = useAI()

const viewMode = ref<'preview' | 'iframe'>('preview')
const iframeLoading = ref(true)
const showAISummary = ref(false)

const feed = computed(() => {
  if (!props.article) return null
  return feedsStore.feeds.find((f: RssFeed) => f.id === props.article?.feedId)
})

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

// 标记为已读
watch(() => props.article, (newArticle) => {
  if (newArticle && !newArticle.read) {
    const apiStore = useApiStore()
    apiStore.markAsRead(newArticle.id)
  }
})

// 更新页面标题
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
}

function handleIframeLoad() {
  iframeLoading.value = false
}

function handleIframeError() {
  iframeLoading.value = false
}

// 重置 iframe 加载
watch(viewMode, () => {
  if (viewMode.value === 'iframe') {
    iframeLoading.value = true
  }
})

// 重置滚动位置到顶部
watch(() => props.article, (newArticle, oldArticle) => {
  if (newArticle && oldArticle && newArticle.id !== oldArticle.id) {
    nextTick(() => {
      if (contentContainer.value) {
        contentContainer.value.scrollTop = 0
      }
      lastScrollDepth = 0
    })
  }
})

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

  <!-- 文章内容 -->
  <div v-else class="article-content">
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
    <div ref="contentContainer" v-if="viewMode === 'preview'" class="preview-mode">
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
    <div v-else class="iframe-mode">
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
</template>
