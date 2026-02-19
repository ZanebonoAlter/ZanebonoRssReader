<script setup lang="ts">
import { Icon } from "@iconify/vue"
import type { SummaryBatch, SummaryJob } from '~/types'

interface AISummary {
  id: number
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
}

const props = defineProps<{
  categoryId?: string | null
}>()

// Get feeds store to access categories
const feedsStore = useFeedsStore()

const emit = defineEmits<{
  'select': [summary: AISummary]
}>()

const apiStore = useApiStore()
const loading = ref(false)
const summaries = ref<AISummary[]>([])
const error = ref<string | null>(null)
const generating = ref(false)
const selectedSummaryId = ref<number | null>(null)

// 队列相关状态
const showCategoryDialog = ref(false)
const currentBatch = ref<SummaryBatch | null>(null)
const queueStatusPolling = ref<number | null>(null)
const expandedErrors = ref<Set<string>>(new Set())

// WebSocket
const ws = useSummaryWebSocket()

// Pagination
const currentPage = ref(1)
const pageSize = ref(20)
const totalItems = ref(0)
const totalPages = ref(0)
const paginationData = ref<any>(null)

// Generate page numbers with ellipsis
const pageNumbers = computed(() => {
  const pages: (number | string)[] = []
  const current = currentPage.value
  const total = totalPages.value
  const delta = 1

  if (total <= 7) {
    for (let i = 1; i <= total; i++) {
      pages.push(i)
    }
  } else {
    pages.push(1)

    const rangeStart = Math.max(2, current - delta)
    const rangeEnd = Math.min(total - 1, current + delta)

    if (rangeStart > 2) {
      pages.push('...')
    }

    for (let i = rangeStart; i <= rangeEnd; i++) {
      pages.push(i)
    }

    if (rangeEnd < total - 1) {
      pages.push('...')
    }

    pages.push(total)
  }

  return pages
})

// Date filter
const startDate = ref<string | undefined>('')
const endDate = ref<string | undefined>('')
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

// Generation status tracking (旧版兼容)
interface GenerationStatus {
  categoryId: string
  categoryName: string
  status: 'pending' | 'generating' | 'success' | 'failed' | 'timeout'
  error?: string
}

const generationStatus = ref<GenerationStatus[]>([])
const totalToGenerate = ref(0)
const generatedCount = ref(0)

// 错误代码映射为友好提示
const errorCodeMap: Record<string, string> = {
  'NO_ARTICLES': '该分类下没有找到文章',
  'REQUEST_FAILED': '创建请求失败',
  'API_ERROR': 'AI API调用失败',
  'PARSE_ERROR': '解析响应失败',
  'AI_ERROR': 'AI服务返回错误',
  'NO_RESPONSE': 'AI未返回响应',
  'DB_ERROR': '保存数据失败',
  'UNKNOWN': '未知错误'
}

// Load settings from localStorage
const aiSettings = ref({
  baseURL: '',
  apiKey: '',
  model: ''
})

// Time range options
const timeRangeOptions = [
  { label: '最近 1 小时', value: 60 },
  { label: '最近 3 小时', value: 180 },
  { label: '最近 6 小时', value: 360 },
  { label: '最近 12 小时', value: 720 },
  { label: '最近 24 小时', value: 1440 },
]

const selectedTimeRange = ref(180)

onMounted(() => {
  loadAISettings()
  fetchSummaries()
})

onUnmounted(() => {
  stopQueuePolling()
  ws.disconnect()
})

// 监听WebSocket消息
watch(() => ws.lastMessage.value, (message) => {
  if (message && generating.value) {
    const batch = ws.toBatchData(message)
    currentBatch.value = batch

    // 如果已完成，更新状态
    if (batch.status === 'completed') {
      generating.value = false
      ws.disconnect()
      fetchSummaries()

      // 显示结果
      const completed = batch.completed_jobs || 0
      const failed = batch.failed_jobs || 0
      const total = batch.total_jobs || 0

      if (completed > 0) {
        error.value = `完成！成功 ${completed}/${total} 个${failed > 0 ? `，失败 ${failed} 个` : ''}`
      } else {
        error.value = '所有任务都失败了'
      }
      setTimeout(() => error.value = null, 5000)
    }
  }
}, { deep: true })

// Watch for category changes
watch(() => props.categoryId, () => {
  // Reset to first page when category changes
  fetchSummaries(true)
})

// Reset page when date filters change
watch([startDate, endDate], () => {
  currentPage.value = 1
})

function loadAISettings() {
  const settings = localStorage.getItem('aiSettings')
  if (settings) {
    const parsed = JSON.parse(settings)
    aiSettings.value = {
      baseURL: parsed.baseURL || 'https://api.openai.com/v1',
      apiKey: parsed.apiKey || '',
      model: parsed.model || 'gpt-4o-mini'
    }
  }
}

async function fetchSummaries(resetPage = false) {
  if (resetPage) {
    currentPage.value = 1
  }

  loading.value = true
  error.value = null

  const params: any = {
    page: currentPage.value,
    per_page: pageSize.value
  }

  if (props.categoryId && props.categoryId !== 'all') {
    params.category_id = parseInt(props.categoryId)
  }

  // Add date filter if set
  if (startDate.value) {
    params.start_date = startDate.value
  }
  if (endDate.value) {
    params.end_date = endDate.value
  }

  const response = await apiStore.getSummaries(params)
  loading.value = false

  if (response.success && response.data && response.pagination) {
    summaries.value = response.data as AISummary[]
    totalItems.value = response.pagination.total || 0
    totalPages.value = response.pagination.pages || 0
    paginationData.value = response.pagination
  } else if (response.success && Array.isArray(response.data)) {
    summaries.value = response.data as AISummary[]
    totalItems.value = response.data.length
    totalPages.value = Math.ceil(totalItems.value / pageSize.value)
    paginationData.value = null
  } else {
    error.value = response.error || '加载失败'
  }
}

function changePage(page: number) {
  if (page >= 1 && page <= totalPages.value) {
    currentPage.value = page
    fetchSummaries()
  }
}

function changePageSize(size: number) {
  pageSize.value = size
  currentPage.value = 1
  fetchSummaries()
}

// Quick date filter function
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

  endDate.value = end.toISOString().split('T')[0]
  startDate.value = start.toISOString().split('T')[0]

  fetchSummaries(true)
}

function clearDateFilter() {
  startDate.value = ''
  endDate.value = ''
  showDateFilter.value = false
  selectedQuickDate.value = null
  fetchSummaries(true)
}

// Helper function to generate summary with timeout
async function generateSummaryWithTimeout(
  categoryId: number,
  categoryName: string,
  timeoutMs = 120000 // 2 minutes timeout
): Promise<{ success: boolean; error?: string; categoryName: string }> {
  return Promise.race([
    apiStore.generateSummary({
      category_id: categoryId,
      time_range: selectedTimeRange.value,
      base_url: aiSettings.value.baseURL,
      api_key: aiSettings.value.apiKey,
      model: aiSettings.value.model
    }).then(response => ({
      success: response.success,
      error: response.error,
      categoryName
    })),
    new Promise<{ success: boolean; error: string; categoryName: string }>((_, reject) =>
      setTimeout(() => reject(new Error('timeout')), timeoutMs)
    ).catch(() => ({
      success: false,
      error: 'timeout',
      categoryName
    }))
  ])
}

// 打开分类选择对话框
function openCategorySelect() {
  if (!aiSettings.value.apiKey) {
    error.value = '请先在设置中配置 AI'
    setTimeout(() => error.value = null, 3000)
    return
  }
  showCategoryDialog.value = true
}

// 提交队列任务（多分类）
async function submitQueueSummary(selectedCategoryIds: string[]) {
  if (selectedCategoryIds.length === 0) return

  generating.value = true
  error.value = null
  showCategoryDialog.value = false

  try {
    const categoryIds = selectedCategoryIds.map(id => parseInt(id))

    // 先连接WebSocket
    ws.connect()

    const response = await apiStore.submitQueueSummary({
      category_ids: categoryIds,
      time_range: selectedTimeRange.value,
      base_url: aiSettings.value.baseURL,
      api_key: aiSettings.value.apiKey,
      model: aiSettings.value.model
    })

    if (response.success && response.data) {
      currentBatch.value = response.data
      // WebSocket会自动接收进度更新
    } else {
      error.value = response.error || '提交任务失败'
      generating.value = false
      ws.disconnect()
      setTimeout(() => error.value = null, 3000)
    }
  } catch (err) {
    error.value = '提交失败：' + (err as Error).message
    generating.value = false
    ws.disconnect()
    setTimeout(() => error.value = null, 3000)
  }
}

// 开始轮询队列状态（已弃用，使用WebSocket）
function startQueuePolling() {
  // 现在使用WebSocket实时推送，不需要轮询
  console.log('[AISummariesList] Using WebSocket instead of polling')
}

// 轮询队列状态（已弃用，使用WebSocket）
async function pollQueueStatus() {
  // 现在使用WebSocket实时推送，不需要轮询
}

// 停止轮询和WebSocket
function stopQueuePolling() {
  if (queueStatusPolling.value) {
    clearInterval(queueStatusPolling.value)
    queueStatusPolling.value = null
  }
  ws.disconnect()
}

// 切换错误详情展开
function toggleError(jobId: string) {
  if (expandedErrors.value.has(jobId)) {
    expandedErrors.value.delete(jobId)
  } else {
    expandedErrors.value.add(jobId)
  }
}

// 获取友好错误提示
function getErrorMessage(job: SummaryJob): string {
  const code = job.error_code
  if (code && errorCodeMap[code]) {
    return errorCodeMap[code]
  }
  return job.error_message || '未知错误'
}

// 旧版生成函数（保留兼容）
async function generateSummary() {
  if (!aiSettings.value.apiKey) {
    error.value = '请先在设置中配置 AI'
    setTimeout(() => error.value = null, 3000)
    return
  }

  // 默认打开分类选择对话框
  openCategorySelect()
}

async function deleteSummary(summaryId: number) {
  if (!confirm('确定要删除这个总结吗？')) return

  const response = await apiStore.deleteSummary(summaryId)
  if (response.success) {
    summaries.value = summaries.value.filter(s => s.id !== summaryId)
  }
}

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  if (hours < 24) return `${hours} 小时前`
  return `${days} 天前`
}

function formatTimeRange(minutes: number): string {
  const option = timeRangeOptions.find(opt => opt.value === minutes)
  return option?.label || `${minutes} 分钟`
}

function truncateSummary(summary: string, maxLength = 150): string {
  if (summary.length <= maxLength) return summary
  return summary.substring(0, maxLength) + '...'
}

function selectSummary(summary: AISummary) {
  selectedSummaryId.value = summary.id
  emit('select', summary)
}

// Expose methods for parent component
defineExpose({
  fetchSummaries,
  clearDateFilter
})
</script>

<template>
  <div class="ai-summaries-list-panel h-full flex flex-col w-300">
    <!-- Header -->
    <div class="px-4 py-3 border-b border-white/20 bg-white/40 flex-shrink-0 space-y-2">
      <div class="flex items-center justify-between">
        <h2 class="font-semibold text-ink-black flex items-center gap-2">
          <Icon icon="mdi:brain" width="18" height="18" class="text-ink-600" />
          AI 总结
        </h2>
        <div class="flex items-center gap-2">
          <select
            v-model="selectedTimeRange"
            class="text-xs px-2 py-1 border border-gray-200 rounded-lg focus:ring-2 focus:ring-ink-400 focus:border-transparent"
          >
            <option
              v-for="option in timeRangeOptions"
              :key="option.value"
              :value="option.value"
            >
              {{ option.label }}
            </option>
          </select>
          <button
            class="px-3 py-1.5 text-xs font-medium bg-ink-600 text-white rounded-lg hover:bg-ink-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
            :disabled="generating || !aiSettings.apiKey"
            @click="generateSummary"
          >
            <Icon
              :icon="generating ? 'mdi:loading' : 'mdi:plus'"
              :class="{ 'animate-spin': generating }"
              width="14"
              height="14"
            />
            生成总结
          </button>
        </div>
      </div>

      <!-- Filter Bar - First Row -->
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
          <span class="text-xs text-ink-light">每页</span>
          <select
            v-model="pageSize"
            class="page-size-select"
            @change="changePageSize(Number(($event.target as HTMLSelectElement).value))"
          >
            <option :value="10">10</option>
            <option :value="20">20</option>
            <option :value="50">50</option>
          </select>
          <span class="text-xs text-ink-light">条</span>
        </div>
      </div>

      <!-- Date Filter Expand Panel -->
      <div
        v-if="showDateFilter"
        class="date-filter-panel"
      >
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
              @change="selectedQuickDate = null; fetchSummaries(true)"
            />
          </div>
          <div class="date-input-row">
            <span class="row-label">结束日期</span>
            <input
              v-model="endDate"
              type="date"
              class="date-input"
              @change="selectedQuickDate = null; fetchSummaries(true)"
            />
          </div>
        </div>

        <!-- Panel Actions -->
        <div class="panel-actions">
          <button
            class="btn-secondary-sm"
            @click="clearDateFilter"
          >
            清除筛选
          </button>
          <button
            class="btn-secondary-sm"
            @click="showDateFilter = false"
          >
            收起
          </button>
        </div>
      </div>
    </div>

    <!-- Error/Status message -->
    <div
      v-if="error"
      class="mx-4 mt-3 p-2 rounded-lg text-xs flex-shrink-0"
      :class="error.includes('成功') ? 'bg-green-50 border border-green-200 text-green-600' : 'bg-red-50 border border-red-200 text-red-600'"
    >
      {{ error }}
    </div>

    <!-- 队列状态展示 -->
    <div
      v-if="currentBatch"
      class="mx-4 mt-3 p-3 bg-blue-50 border border-blue-200 rounded-lg flex-shrink-0"
    >
      <div class="flex items-center justify-between mb-2">
        <span class="text-xs font-medium text-blue-700">
          {{ currentBatch.status === 'completed' ? '处理完成' : '正在处理队列' }} 
          ({{ currentBatch.completed_jobs }}/{{ currentBatch.total_jobs }})
        </span>
        <div class="flex items-center gap-2">
          <Icon 
            :icon="currentBatch.status === 'completed' ? 'mdi:check-circle' : 'mdi:loading'" 
            width="14" 
            height="14" 
            :class="currentBatch.status === 'completed' ? 'text-green-600' : 'animate-spin text-blue-600'" 
          />
          <button
            v-if="currentBatch.status === 'completed'"
            class="p-1 hover:bg-blue-100 rounded transition-colors"
            @click="currentBatch = null"
            title="关闭"
          >
            <Icon icon="mdi:close" width="14" height="14" class="text-blue-600" />
          </button>
        </div>
      </div>
      <div class="space-y-1 max-h-40 overflow-y-auto">
        <div
          v-for="job in currentBatch.jobs"
          :key="job.id"
          class="flex flex-col gap-1"
        >
          <div class="flex items-center gap-2 text-xs">
            <Icon
              :icon="job.status === 'completed' ? 'mdi:check-circle' : job.status === 'processing' ? 'mdi:loading' : job.status === 'failed' ? 'mdi:alert-circle' : 'mdi:circle-outline'"
              :class="{
                'text-green-500': job.status === 'completed',
                'text-blue-500 animate-spin': job.status === 'processing',
                'text-red-500': job.status === 'failed',
                'text-ink-muted': job.status === 'pending'
              }"
              width="12"
              height="12"
            />
            <span class="flex-1 text-ink-dark">{{ job.category_name }}</span>
            <span
              v-if="job.status === 'failed'"
              class="text-red-500 text-right flex items-center gap-1 cursor-pointer hover:underline"
              @click="toggleError(job.id)"
            >
              <span>失败</span>
              <Icon 
                :icon="expandedErrors.has(job.id) ? 'mdi:chevron-up' : 'mdi:chevron-down'" 
                width="12" 
                height="12" 
              />
            </span>
          </div>
          <!-- 错误详情 -->
          <div
            v-if="job.status === 'failed' && expandedErrors.has(job.id)"
            class="ml-5 p-2 bg-red-50 rounded text-xs text-red-600 border border-red-100"
          >
            <div class="font-medium mb-0.5">{{ getErrorMessage(job) }}</div>
            <div v-if="job.error_message" class="text-red-400 text-[10px] truncate" :title="job.error_message">
              {{ job.error_message }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- AI Key warning -->
    <div
      v-if="!aiSettings.apiKey"
      class="mx-4 mt-3 p-2 bg-amber-50 border border-amber-200 rounded-lg text-xs text-amber-600 flex-shrink-0"
    >
      请先在设置中配置 AI API 密钥
    </div>

    <!-- Loading -->
    <div
      v-if="loading"
      class="flex-1 flex items-center justify-center"
    >
      <div class="text-center">
        <Icon icon="mdi:loading" width="32" height="32" class="animate-spin text-ink-600 mx-auto mb-2" />
        <p class="text-sm text-ink-light">加载中...</p>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="summaries.length === 0"
      class="flex-1 flex items-center justify-center"
    >
      <div class="text-center">
        <Icon icon="mdi:brain" width="48" height="48" class="text-ink-light mx-auto mb-2" />
        <p class="text-sm text-ink-light">还没有 AI 总结</p>
        <p class="text-xs text-ink-muted mt-1">点击"生成总结"开始使用</p>
      </div>
    </div>

    <!-- Summaries list -->
    <div
      v-else
      class="flex-1 overflow-y-auto"
    >
      <div class="p-2 space-y-2">
        <div
          v-for="summary in summaries"
          :key="summary.id"
          class="rounded-lg p-3 hover:shadow-md transition-all cursor-pointer border"
          :class="selectedSummaryId === summary.id
            ? 'bg-ink-50 border-ink-300 shadow-sm'
            : 'bg-white border-gray-100'"
          @click="selectSummary(summary)"
        >
          <div class="flex items-start justify-between mb-2">
            <div class="flex-1 min-w-0">
              <h3 class="font-medium text-ink-black text-sm truncate">
                {{ summary.title }}
              </h3>
              <div class="flex items-center gap-2 mt-1 text-xs text-ink-light">
                <span class="bg-ink-50 text-ink-700 px-2 py-0.5 rounded-full">
                  {{ summary.category_name }}
                </span>
                <span>{{ summary.article_count }} 篇文章</span>
                <span>{{ formatDate(summary.created_at) }}</span>
              </div>
            </div>
            <button
              class="p-1 hover:bg-red-50/80 rounded-lg transition-colors text-ink-muted hover:text-red-500 flex-shrink-0 ml-2"
              @click.stop="deleteSummary(summary.id)"
            >
              <Icon icon="mdi:delete" width="16" height="16" />
            </button>
          </div>
          <p class="text-xs text-ink-medium line-clamp-2">
            {{ truncateSummary(summary.summary) }}
          </p>
        </div>
      </div>
    </div>

    <!-- Pagination -->
    <div
      v-if="!loading && summaries.length > 0 && totalPages > 1"
      class="pagination-bar"
    >
      <span class="pagination-text">
        共 {{ totalItems }} 条
      </span>
      <div class="pagination-controls">
        <button
          class="pagination-btn"
          :disabled="currentPage <= 1"
          @click="changePage(currentPage - 1)"
        >
          <Icon icon="mdi:chevron-left" width="16" height="16" class="text-ink-medium" />
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
          <Icon icon="mdi:chevron-right" width="16" height="16" class="text-ink-medium" />
        </button>
      </div>
    </div>

    <!-- 分类选择对话框 -->
    <DialogCategorySelectDialog
      v-model:visible="showCategoryDialog"
      :categories="feedsStore.categories || []"
      :loading="generating"
      @confirm="submitQueueSummary"
    />
  </div>
</template>

<style scoped>
/* Filter Bar - First Row */
.filter-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.4);
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
}

.filter-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.filter-toggle-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-ink-dark);
  cursor: pointer;
  transition: all 0.2s ease;
}

.filter-toggle-btn:hover {
  background: rgba(59, 107, 135, 0.1);
  border-color: rgba(59, 107, 135, 0.3);
  color: #25465c;
}

.filter-toggle-btn.active {
  background: linear-gradient(135deg, #3b6b87 0%, #25465c 100%);
  color: white;
  border-color: transparent;
  box-shadow: 0 2px 8px rgba(59, 107, 135, 0.3);
}

.toggle-icon {
  transition: transform 0.2s ease;
}

/* Date Filter Expand Panel */
.date-filter-panel {
  padding: 16px;
  background: rgba(249, 250, 251, 0.9);
  border-bottom: 1px solid rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Quick Options Row */
.quick-options-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.row-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-ink-medium);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  white-space: nowrap;
  min-width: 60px;
}

.quick-options {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.quick-option-btn {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-ink-dark);
  cursor: pointer;
  transition: all 0.2s ease;
}

.quick-option-btn:hover {
  background: rgba(59, 107, 135, 0.1);
  border-color: rgba(59, 107, 135, 0.3);
  color: #25465c;
}

.quick-option-btn.active {
  background: linear-gradient(135deg, #3b6b87 0%, #25465c 100%);
  color: white;
  border-color: transparent;
  box-shadow: 0 2px 8px rgba(59, 107, 135, 0.3);
}

/* Custom Date Row - Vertical Layout */
.custom-date-row-vertical {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.date-input-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.date-input-row .row-label {
  min-width: 80px;
}

.input-label {
  font-size: 13px;
  color: var(--color-ink-dark);
  font-weight: 500;
}

.date-input {
  padding: 8px 12px;
  font-size: 13px;
  color: #1f2937;
  background: white;
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  transition: all 0.2s ease;
  min-width: 120px;
}

.date-input:focus {
  outline: none;
  border-color: #3b6b87;
  box-shadow: 0 0 0 3px rgba(59, 107, 135, 0.1);
}

/* Panel Actions */
.panel-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  padding-top: 12px;
  border-top: 1px solid rgba(0, 0, 0, 0.05);
}

.btn-secondary-sm {
  padding: 8px 16px;
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid rgba(0, 0, 0, 0.1);
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-ink-dark);
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn-secondary-sm:hover {
  background: rgba(0, 0, 0, 0.05);
}

.btn-primary-sm {
  padding: 8px 16px;
  background: linear-gradient(135deg, #3b6b87 0%, #25465c 100%);
  border: none;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 600;
  color: white;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 2px 8px rgba(59, 107, 135, 0.3);
}

.btn-primary-sm:hover {
  box-shadow: 0 4px 12px rgba(59, 107, 135, 0.4);
  transform: translateY(-1px);
}

/* Pagination */
.pagination-bar {
  padding: 0.75rem 1rem;
  background: rgba(255, 255, 255, 0.5);
  border-top: 1px solid rgba(26, 26, 26, 0.06);
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.pagination-text {
  font-size: 0.75rem;
  color: var(--color-ink-medium);
  white-space: nowrap;
}

.pagination-controls {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.pagination-btn {
  padding: 0.25rem;
  border: none;
  border-radius: 0.375rem;
  background: transparent;
  cursor: pointer;
  transition: background 0.2s;
  color: var(--color-ink-medium);
}

.pagination-btn:hover:not(:disabled) {
  background: rgba(26, 26, 26, 0.04);
  color: var(--color-ink-dark);
}

.pagination-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.pagination-pages {
  display: flex;
  align-items: center;
  gap: 0.125rem;
}

.page-btn {
  min-width: 1.75rem;
  padding: 0.25rem 0.375rem;
  font-size: 0.75rem;
  border: none;
  border-radius: 0.375rem;
  background: transparent;
  color: var(--color-ink-dark);
  cursor: pointer;
  transition: all 0.2s;
}

.page-btn:hover:not(:disabled):not(.ellipsis) {
  background: rgba(26, 26, 26, 0.04);
}

.page-btn.active {
  background: #3b6b87;
  color: white;
}

.page-btn.ellipsis {
  cursor: default;
  color: var(--color-ink-light);
}

.page-btn:disabled {
  cursor: default;
}

/* Page Size Selector */
.page-size-selector {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.page-size-select {
  background: rgba(255, 255, 255, 0.6);
  color: var(--color-ink-dark);
  font-size: 0.75rem;
  padding: 0.125rem 0.25rem;
  border: 1px solid rgba(26, 26, 26, 0.15);
  border-radius: 0.25rem;
}

/* Animation */
@keyframes pulse {
  0%, 100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}

/* Scrollbar for summaries list */
.overflow-y-auto::-webkit-scrollbar {
  width: 6px;
}

.overflow-y-auto::-webkit-scrollbar-track {
  background: transparent;
}

.overflow-y-auto::-webkit-scrollbar-thumb {
  background: rgba(59, 107, 135, 0.2);
  border-radius: 3px;
}

.overflow-y-auto::-webkit-scrollbar-thumb:hover {
  background: rgba(59, 107, 135, 0.4);
}
</style>
