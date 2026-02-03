<script setup lang="ts">
import type { Article } from '~/types'
import { Icon } from '@iconify/vue'

interface Props {
  articles: Article[]
  selectedCategory?: string | null
  selectedFeed?: string | null
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  selectedCategory: null,
  selectedFeed: null,
  loading: false,
})

const emit = defineEmits<{
  articleClick: [article: Article]
  articleFavorite: [id: string]
}>()

const apiStore = useApiStore()
const feedsStore = useFeedsStore()

// 标题
const panelTitle = computed(() => {
  if (props.selectedFeed) return '订阅源文章'
  if (props.selectedCategory === 'favorites') return '收藏夹'
  if (props.selectedCategory === 'uncategorized') return '未分类'
  if (props.selectedCategory) return '分类文章'
  return '全部文章'
})

// 处理文章点击
function handleArticleClick(article: Article) {
  emit('articleClick', article)
}

// 处理收藏
function handleFavorite(id: string) {
  apiStore.toggleFavorite(id)
}

import './ArticleListPanel.css'
</script>

<template>
  <div class="article-list-panel">
    <!-- 文章列表头部 -->
    <div class="panel-header">
      <div class="header-content">
        <h2 class="header-title">{{ panelTitle }}</h2>
        <span class="article-count">{{ articles.length }}</span>
      </div>
    </div>

    <!-- 文章列表 -->
    <div class="panel-content">
      <!-- 加载状态 -->
      <div v-if="loading" class="loading-state">
        <div class="text-center">
          <Icon icon="mdi:loading" width="32" height="32" class="animate-spin text-blue-600 mx-auto mb-2" />
          <p class="text-sm text-gray-500">加载中...</p>
        </div>
      </div>

      <!-- 文章列表 -->
      <div v-else-if="articles.length > 0" class="articles-list">
        <ArticleCard
          v-for="article in articles.slice(0, 50)"
          :key="article.id"
          :article="article"
          compact
          @click="handleArticleClick"
          @favorite="handleFavorite"
        />
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="text-center">
          <Icon icon="mdi:file-document-outline" width="48" height="48" class="text-gray-300 mx-auto mb-2" />
          <h3 class="text-base font-semibold text-gray-700 mb-1">暂无文章</h3>
          <p class="text-sm text-gray-500">添加一些 RSS 订阅源开始阅读吧</p>
        </div>
      </div>
    </div>
  </div>
</template>
