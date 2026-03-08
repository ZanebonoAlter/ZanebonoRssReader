<script setup lang="ts">
import { Icon } from '@iconify/vue'
import type { Article } from '~/types'

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

const currentPage = ref(1)
const pageSize = ref(20)
const startDate = ref<string>('')
const endDate = ref<string>('')
const showDateFilter = ref(false)
const selectedQuickDate = ref<number | null>(null)

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

const currentFeed = computed(() => props.selectedFeed ? feedsStore.feeds.find(feed => feed.id === props.selectedFeed) ?? null : null)

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

const totalItems = computed(() => dateFilteredArticles.value.length)
const totalPages = computed(() => Math.ceil(totalItems.value / pageSize.value))

const paginatedArticles = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return dateFilteredArticles.value.slice(start, end)
})

const pageNumbers = computed(() => {
  const pages: (number | string)[] = []
  const current = currentPage.value
  const total = totalPages.value
  const delta = 1

  if (total <= 7) {
    for (let i = 1; i <= total; i++) pages.push(i)
    return pages
  }

  pages.push(1)
  const rangeStart = Math.max(2, current - delta)
  const rangeEnd = Math.min(total - 1, current + delta)

  if (rangeStart > 2) pages.push('...')
  for (let i = rangeStart; i <= rangeEnd; i++) pages.push(i)
  if (rangeEnd < total - 1) pages.push('...')
  pages.push(total)

  return pages
})

const panelTitle = computed(() => {
  if (currentFeed.value) return currentFeed.value.title
  if (props.selectedCategory === 'favorites') return '收藏夹'
  if (props.selectedCategory === 'uncategorized') return '未分类文章'
  if (props.selectedCategory) return '分类文章'
  return '全部文章'
})

const feedStatusItems = computed(() => {
  if (!currentFeed.value) return []

  return [
    {
      label: '刷新',
      value: currentFeed.value.refreshStatus === 'refreshing'
        ? '进行中'
        : currentFeed.value.refreshStatus === 'success'
          ? '正常'
          : currentFeed.value.refreshStatus === 'error'
            ? '失败'
            : '空闲',
      tone: currentFeed.value.refreshStatus === 'error'
        ? 'rose'
        : currentFeed.value.refreshStatus === 'success'
          ? 'emerald'
          : currentFeed.value.refreshStatus === 'refreshing'
            ? 'sky'
            : 'stone',
      icon: currentFeed.value.refreshStatus === 'refreshing' ? 'mdi:loading' : 'mdi:refresh',
      spinning: currentFeed.value.refreshStatus === 'refreshing',
    },
    {
      label: '总结',
      value: currentFeed.value.contentCompletionEnabled ? '开启' : '关闭',
      tone: currentFeed.value.contentCompletionEnabled ? 'emerald' : 'stone',
      icon: 'mdi:brain',
      spinning: false,
    },
    {
      label: '抓取',
      value: currentFeed.value.firecrawlEnabled ? '开启' : '关闭',
      tone: currentFeed.value.firecrawlEnabled ? 'sky' : 'stone',
      icon: 'mdi:spider-web',
      spinning: false,
    },
  ]
})

watch(() => props.articles, () => {
  currentPage.value = 1
}, { deep: true })

watch([startDate, endDate], () => {
  currentPage.value = 1
})

function changePage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
  }
}

function changePageSize(size: number) {
  pageSize.value = size
  currentPage.value = 1
}

function applyQuickDateFilter(days: number) {
  if (selectedQuickDate.value === days) {
    selectedQuickDate.value = null
    clearDateFilter()
    return
  }

  selectedQuickDate.value = days
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - days)

  endDate.value = end.toISOString().split('T')[0] || ''
  startDate.value = start.toISOString().split('T')[0] || ''
  currentPage.value = 1
}

function clearDateFilter() {
  startDate.value = ''
  endDate.value = ''
  showDateFilter.value = false
  selectedQuickDate.value = null
}

function handleArticleClick(article: Article) {
  emit('articleClick', article)
}

function handleFavorite(id: string) {
  apiStore.toggleFavorite(id)
}

function statusToneClasses(tone: string) {
  if (tone === 'rose') return 'border-rose-200 bg-rose-50 text-rose-700'
  if (tone === 'emerald') return 'border-emerald-200 bg-emerald-50 text-emerald-700'
  if (tone === 'sky') return 'border-sky-200 bg-sky-50 text-sky-700'
  return 'border-stone-200 bg-stone-100 text-stone-700'
}

import './ArticleListPanel.css'
</script>

<template>
  <div class="article-list-panel">
    <div class="panel-header">
      <div class="header-content">
        <h2 class="header-title">{{ panelTitle }}</h2>
        <span class="article-count">{{ totalItems }}</span>
      </div>
    </div>

    <div v-if="currentFeed" class="mx-4 mt-4 rounded-2xl border border-ink-200 bg-white/75 p-3 shadow-subtle">
      <div class="flex flex-wrap items-center gap-2">
        <div class="mr-1 text-sm font-semibold text-ink-black">订阅源状态</div>
        <div v-for="item in feedStatusItems" :key="item.label" class="inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold" :class="statusToneClasses(item.tone)">
          <Icon :icon="item.icon" width="14" height="14" :class="{ 'animate-spin': item.spinning }" />
          <span>{{ item.label }}</span>
          <span>{{ item.value }}</span>
        </div>
      </div>
    </div>

    <div class="filter-bar">
      <div class="filter-left">
        <button class="filter-toggle-btn" :class="{ active: showDateFilter || startDate || endDate || selectedQuickDate }" @click="showDateFilter = !showDateFilter">
          <Icon :icon="showDateFilter ? 'mdi:chevron-up' : 'mdi:chevron-down'" width="14" height="14" class="toggle-icon" />
          <Icon icon="mdi:calendar-filter" width="14" height="14" />
          <span>日期筛选</span>
        </button>
      </div>

      <div class="page-size-selector">
        <span class="text-xs text-gray-500">每页</span>
        <select v-model="pageSize" class="page-size-select" @change="changePageSize(Number((($event.target as HTMLSelectElement).value)))">
          <option :value="10">10</option>
          <option :value="20">20</option>
          <option :value="50">50</option>
        </select>
        <span class="text-xs text-gray-500">条</span>
      </div>
    </div>

    <div v-if="showDateFilter" class="date-filter-panel">
      <div class="quick-options-row">
        <span class="row-label">快速选择：</span>
        <div class="quick-options">
          <button v-for="option in quickDateOptions" :key="option.days" class="quick-option-btn" :class="{ active: selectedQuickDate === option.days }" @click="applyQuickDateFilter(option.days)">
            {{ option.label }}
          </button>
        </div>
      </div>

      <div class="custom-date-row-vertical">
        <div class="date-input-row">
          <span class="row-label">开始日期</span>
          <input v-model="startDate" type="date" class="date-input" @change="selectedQuickDate = null" />
        </div>
        <div class="date-input-row">
          <span class="row-label">结束日期</span>
          <input v-model="endDate" type="date" class="date-input" @change="selectedQuickDate = null" />
        </div>
      </div>

      <div class="panel-actions">
        <button class="btn-secondary-sm" @click="clearDateFilter">清除筛选</button>
        <button class="btn-secondary-sm" @click="showDateFilter = false">收起</button>
      </div>
    </div>

    <div class="panel-content">
      <div v-if="loading" class="loading-state">
        <div class="text-center">
          <Icon icon="mdi:loading" width="32" height="32" class="animate-spin text-blue-600 mx-auto mb-2" />
          <p class="text-sm text-gray-500">加载中...</p>
        </div>
      </div>

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

      <div v-else class="empty-state">
        <div class="text-center">
          <Icon icon="mdi:file-document-outline" width="48" height="48" class="text-gray-300 mx-auto mb-2" />
          <h3 class="mb-1 text-base font-semibold text-gray-700">暂无文章</h3>
          <p class="text-sm text-gray-500">
            {{ startDate || endDate ? '当前筛选条件下没有文章，请调整日期范围。' : '添加一些 RSS 订阅源开始阅读吧。' }}
          </p>
        </div>
      </div>
    </div>

    <div v-if="!loading && paginatedArticles.length > 0 && totalPages > 1" class="pagination-bar">
      <span class="pagination-text">共 {{ totalItems }} 条</span>
      <div class="pagination-controls">
        <button class="pagination-btn" :disabled="currentPage <= 1" @click="changePage(currentPage - 1)">
          <Icon icon="mdi:chevron-left" width="16" height="16" class="text-gray-600" />
        </button>

        <div class="pagination-pages">
          <button v-for="page in pageNumbers" :key="page" class="page-btn" :class="{ active: page === currentPage, ellipsis: page === '...' }" :disabled="page === '...'" @click="page !== '...' && changePage(page as number)">
            {{ page }}
          </button>
        </div>

        <button class="pagination-btn" :disabled="currentPage >= totalPages" @click="changePage(currentPage + 1)">
          <Icon icon="mdi:chevron-right" width="16" height="16" class="text-gray-600" />
        </button>
      </div>
    </div>
  </div>
</template>
