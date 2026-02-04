<script setup lang="ts">
import type { Article } from '~/types'
import { Icon } from '@iconify/vue'

interface Props {
  articles: Article[]
  selectedCategory?: string | null
  selectedFeed?: string | null
  selectedArticle?: Article | null
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  selectedCategory: null,
  selectedFeed: null,
  selectedArticle: null,
  loading: false,
})

const emit = defineEmits<{
  articleClick: [article: Article]
  articleFavorite: [id: string]
}>()

const apiStore = useApiStore()
const feedsStore = useFeedsStore()

// Pagination
const currentPage = ref(1)
const pageSize = ref(20)

// Date filter
const startDate = ref<string>('')
const endDate = ref<string>('')
const showDateFilter = ref(false)

// Quick date filter
interface QuickDateOption {
  label: string
  days: number
}

const quickDateOptions: QuickDateOption[] = [
  { label: '1天内', days: 1 },
  { label: '3天内', days: 3 },
  { label: '7天内', days: 7 },
  { label: '30天内', days: 30 },
]

const selectedQuickDate = ref<number | null>(null)

// Filtered articles by date
const dateFilteredArticles = computed(() => {
  let result = props.articles

  if (startDate.value || endDate.value) {
    result = result.filter(article => {
      const articleDate = new Date(article.pubDate || '')
      if (startDate.value) {
        const start = new Date(startDate.value)
        start.setHours(0, 0, 0, 0)
        if (articleDate < start) return false
      }
      if (endDate.value) {
        const end = new Date(endDate.value)
        end.setHours(23, 59, 59, 999)
        if (articleDate > end) return false
      }
      return true
    })
  }

  return result
})

// Paginated articles
const totalItems = computed(() => dateFilteredArticles.value.length)
const totalPages = computed(() => Math.ceil(totalItems.value / pageSize.value))

const paginatedArticles = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return dateFilteredArticles.value.slice(start, end)
})

// Generate page numbers with ellipsis
const pageNumbers = computed(() => {
  const pages: (number | string)[] = []
  const current = currentPage.value
  const total = totalPages.value
  const delta = 1 // Number of pages to show on each side of current page

  if (total <= 7) {
    // Show all pages if total is small
    for (let i = 1; i <= total; i++) {
      pages.push(i)
    }
  } else {
    // Always show first page
    pages.push(1)

    // Calculate range around current page
    const rangeStart = Math.max(2, current - delta)
    const rangeEnd = Math.min(total - 1, current + delta)

    // Add ellipsis before range if needed
    if (rangeStart > 2) {
      pages.push('...')
    }

    // Add pages in range
    for (let i = rangeStart; i <= rangeEnd; i++) {
      pages.push(i)
    }

    // Add ellipsis after range if needed
    if (rangeEnd < total - 1) {
      pages.push('...')
    }

    // Always show last page
    pages.push(total)
  }

  return pages
})

// Reset page when articles or filters change
watch(() => props.articles, () => {
  currentPage.value = 1
}, { deep: true })

watch([startDate, endDate], () => {
  currentPage.value = 1
})

// Pagination functions
function changePage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
  }
}

function changePageSize(size: number) {
  pageSize.value = size
  currentPage.value = 1
}

// Quick date filter function
function applyQuickDateFilter(days: number) {
  if (selectedQuickDate.value === days) {
    // Toggle off if already selected
    selectedQuickDate.value = null
    clearDateFilter()
    return
  }

  selectedQuickDate.value = days
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - days)

  endDate.value = end.toISOString().split('T')[0]
  startDate.value = start.toISOString().split('T')[0]

  currentPage.value = 1
}

function clearDateFilter() {
  startDate.value = ''
  endDate.value = ''
  showDateFilter.value = false
  selectedQuickDate.value = null
}

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
        <span class="article-count">{{ totalItems }}</span>
      </div>
    </div>

    <!-- Filter Bar - Date Filter -->
    <div class="filter-bar">
      <div class="filter-left">
        <!-- Date Filter Toggle Button -->
        <button
          class="filter-toggle-btn"
          :class="{ active: showDateFilter || startDate || endDate || selectedQuickDate }"
          @click="showDateFilter = !showDateFilter"
        >
          <Icon
            :icon="showDateFilter ? 'mdi:chevron-up' : 'mdi:chevron-down'"
            width="14"
            height="14"
            class="toggle-icon"
          />
          <Icon icon="mdi:calendar-filter" width="14" height="14" />
          <span>日期筛选</span>
        </button>
      </div>

      <!-- Page Size Selector -->
      <div class="page-size-selector">
        <span class="text-xs text-gray-500">每页</span>
        <select
          v-model="pageSize"
          class="page-size-select"
          @change="changePageSize(Number($event.target.value))"
        >
          <option :value="10">10</option>
          <option :value="20">20</option>
          <option :value="50">50</option>
        </select>
        <span class="text-xs text-gray-500">条</span>
      </div>
    </div>

    <!-- Date Filter Expand Panel -->
    <div v-if="showDateFilter" class="date-filter-panel">
      <!-- Quick Date Options Row -->
      <div class="quick-options-row">
        <span class="row-label">快速选择：</span>
        <div class="quick-options">
          <button
            v-for="option in quickDateOptions"
            :key="option.days"
            class="quick-option-btn"
            :class="{ active: selectedQuickDate === option.days }"
            @click="applyQuickDateFilter(option.days)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>

      <!-- Custom Date Row -->
      <div class="custom-date-row-vertical">
        <div class="date-input-row">
          <span class="row-label">开始日期</span>
          <input
            v-model="startDate"
            type="date"
            class="date-input"
            @change="selectedQuickDate = null"
          />
        </div>
        <div class="date-input-row">
          <span class="row-label">结束日期</span>
          <input
            v-model="endDate"
            type="date"
            class="date-input"
            @change="selectedQuickDate = null"
          />
        </div>
      </div>

      <!-- Panel Actions -->
      <div class="panel-actions">
        <button class="btn-secondary-sm" @click="clearDateFilter">
          清除筛选
        </button>
        <button class="btn-secondary-sm" @click="showDateFilter = false">
          收起
        </button>
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
      <div v-else-if="paginatedArticles.length > 0" class="articles-list">
        <ArticleCard
          v-for="article in paginatedArticles"
          :key="article.id"
          :article="article"
          :selected="props.selectedArticle?.id === article.id"
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
          <p class="text-sm text-gray-500">
            {{ startDate || endDate ? '当前筛选条件下没有文章，请调整日期范围' : '添加一些 RSS 订阅源开始阅读吧' }}
          </p>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="!loading && paginatedArticles.length > 0 && totalPages > 1" class="pagination-bar">
      <span class="pagination-text">
        共 {{ totalItems }} 条
      </span>
      <div class="pagination-controls">
        <button
          class="pagination-btn"
          :disabled="currentPage <= 1"
          @click="changePage(currentPage - 1)"
        >
          <Icon icon="mdi:chevron-left" width="16" height="16" class="text-gray-600" />
        </button>

        <div class="pagination-pages">
          <button
            v-for="page in pageNumbers"
            :key="page"
            class="page-btn"
            :class="{ active: page === currentPage, ellipsis: page === '...' }"
            :disabled="page === '...'"
            @click="page !== '...' && changePage(page as number)"
          >
            {{ page }}
          </button>
        </div>

        <button
          class="pagination-btn"
          :disabled="currentPage >= totalPages"
          @click="changePage(currentPage + 1)"
        >
          <Icon icon="mdi:chevron-right" width="16" height="16" class="text-gray-600" />
        </button>
      </div>
    </div>
  </div>
</template>
