<script setup lang="ts">
import { Icon } from "@iconify/vue"

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

// Generation status tracking
interface GenerationStatus {
  categoryId: string
  categoryName: string
  status: 'pending' | 'generating' | 'success' | 'failed' | 'timeout'
  error?: string
}

const generationStatus = ref<GenerationStatus[]>([])
const totalToGenerate = ref(0)
const generatedCount = ref(0)

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

async function fetchSummaries() {
  loading.value = true
  error.value = null

  const params: any = {
    per_page: 50
  }

  if (props.categoryId && props.categoryId !== 'all') {
    params.category_id = parseInt(props.categoryId)
  }

  const response = await apiStore.getSummaries(params)
  loading.value = false

  if (response.success && response.data) {
    summaries.value = response.data as AISummary[]
  } else {
    error.value = response.error || '加载失败'
  }
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

async function generateSummary() {
  if (!aiSettings.value.apiKey) {
    error.value = '请先在设置中配置 AI'
    setTimeout(() => error.value = null, 3000)
    return
  }

  generating.value = true
  error.value = null

  try {
    // Get all categories
    const categories = feedsStore.categories || []

    if (categories.length === 0) {
      error.value = '没有找到分类'
      setTimeout(() => error.value = null, 3000)
      generating.value = false
      return
    }

    // Initialize generation status
    totalToGenerate.value = categories.length
    generatedCount.value = 0
    generationStatus.value = categories.map(cat => ({
      categoryId: cat.id,
      categoryName: cat.name,
      status: 'pending' as const
    }))

    // Generate summaries for all categories in parallel with individual error handling
    const promises = categories.map(async (category, index) => {
      // Update status to generating
      const statusItem = generationStatus.value[index]
      if (!statusItem) return

      statusItem.status = 'generating'

      try {
        const result = await generateSummaryWithTimeout(
          parseInt(category.id),
          category.name,
          120000 // 2 minute timeout per category
        )

        if (result.success) {
          statusItem.status = 'success'
          generatedCount.value++
        } else if (result.error === 'timeout') {
          statusItem.status = 'timeout'
          statusItem.error = '请求超时'
        } else {
          statusItem.status = 'failed'
          statusItem.error = result.error || '生成失败'
        }
      } catch (err) {
        statusItem.status = 'failed'
        statusItem.error = (err as Error).message
      }
    })

    // Wait for all promises to settle (both resolved and rejected)
    await Promise.allSettled(promises)

    generating.value = false

    // Refresh the list to show newly generated summaries
    await fetchSummaries()

    // Count results
    const successCount = generationStatus.value.filter(s => s.status === 'success').length
    const failedCount = generationStatus.value.filter(s => s.status === 'failed' || s.status === 'timeout').length

    // Show result message
    if (successCount > 0) {
      error.value = `成功生成 ${successCount}/${totalToGenerate.value} 个分类总结${failedCount > 0 ? `，${failedCount} 个失败` : ''}`
      setTimeout(() => error.value = null, 5000)
    } else {
      error.value = '生成失败'
      setTimeout(() => error.value = null, 5000)
    }
  } catch (err) {
    generating.value = false
    error.value = '生成失败：' + (err as Error).message
    setTimeout(() => error.value = null, 5000)
  }
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
  emit('select', summary)
}

// Expose fetchSummaries for parent component to refresh
defineExpose({
  fetchSummaries
})
</script>

<template>
  <div class="h-full flex flex-col bg-white/30 backdrop-blur-sm">
    <!-- Header -->
    <div class="px-4 py-3 border-b border-white/20 flex items-center justify-between bg-white/40 flex-shrink-0">
      <h2 class="font-semibold text-gray-800 flex items-center gap-2">
        <Icon icon="mdi:brain" width="18" height="18" class="text-purple-600" />
        AI 总结
      </h2>
      <div class="flex items-center gap-2">
        <select
          v-model="selectedTimeRange"
          class="text-xs px-2 py-1 border border-gray-200 rounded-lg focus:ring-2 focus:ring-purple-500 focus:border-transparent"
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
          class="px-3 py-1.5 text-xs font-medium bg-purple-600 text-white rounded-lg hover:bg-purple-700 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1"
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

    <!-- Error/Status message -->
    <div
      v-if="error"
      class="mx-4 mt-3 p-2 rounded-lg text-xs flex-shrink-0"
      :class="error.includes('成功') ? 'bg-green-50 border border-green-200 text-green-600' : 'bg-red-50 border border-red-200 text-red-600'"
    >
      {{ error }}
    </div>

    <!-- Generation progress -->
    <div
      v-if="generating && generationStatus.length > 0"
      class="mx-4 mt-3 p-3 bg-blue-50 border border-blue-200 rounded-lg flex-shrink-0"
    >
      <div class="flex items-center justify-between mb-2">
        <span class="text-xs font-medium text-blue-700">
          正在生成总结... ({{ generatedCount }}/{{ totalToGenerate }})
        </span>
        <Icon icon="mdi:loading" width="14" height="14" class="animate-spin text-blue-600" />
      </div>
      <div class="space-y-1">
        <div
          v-for="(status, index) in generationStatus"
          :key="index"
          class="flex items-center gap-2 text-xs"
        >
          <Icon
            :icon="status.status === 'success' ? 'mdi:check-circle' : status.status === 'generating' ? 'mdi:loading' : status.status === 'timeout' ? 'mdi:clock-alert' : status.status === 'failed' ? 'mdi:alert-circle' : 'mdi:circle-outline'"
            :class="{
              'text-green-500': status.status === 'success',
              'text-blue-500 animate-spin': status.status === 'generating',
              'text-orange-500': status.status === 'timeout',
              'text-red-500': status.status === 'failed',
              'text-gray-400': status.status === 'pending'
            }"
            width="12"
            height="12"
          />
          <span class="flex-1 text-gray-700">{{ status.categoryName }}</span>
          <span
            v-if="status.error"
            class="text-red-500 text-right"
            :title="status.error"
          >
            {{ status.status === 'timeout' ? '超时' : '失败' }}
          </span>
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
        <Icon icon="mdi:loading" width="32" height="32" class="animate-spin text-purple-600 mx-auto mb-2" />
        <p class="text-sm text-gray-500">加载中...</p>
      </div>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="summaries.length === 0"
      class="flex-1 flex items-center justify-center"
    >
      <div class="text-center">
        <Icon icon="mdi:brain" width="48" height="48" class="text-gray-300 mx-auto mb-2" />
        <p class="text-sm text-gray-500">还没有 AI 总结</p>
        <p class="text-xs text-gray-400 mt-1">点击"生成总结"开始使用</p>
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
          class="bg-white rounded-lg p-3 hover:shadow-md transition-shadow cursor-pointer border border-gray-100"
          @click="selectSummary(summary)"
        >
          <div class="flex items-start justify-between mb-2">
            <div class="flex-1 min-w-0">
              <h3 class="font-medium text-gray-900 text-sm truncate">
                {{ summary.title }}
              </h3>
              <div class="flex items-center gap-2 mt-1 text-xs text-gray-500">
                <span class="bg-purple-50 text-purple-700 px-2 py-0.5 rounded-full">
                  {{ summary.category_name }}
                </span>
                <span>{{ summary.article_count }} 篇文章</span>
                <span>{{ formatDate(summary.created_at) }}</span>
              </div>
            </div>
            <button
              class="p-1 hover:bg-red-50 rounded-lg transition-colors text-gray-400 hover:text-red-500 flex-shrink-0 ml-2"
              @click.stop="deleteSummary(summary.id)"
            >
              <Icon icon="mdi:delete" width="16" height="16" />
            </button>
          </div>
          <p class="text-xs text-gray-600 line-clamp-2">
            {{ truncateSummary(summary.summary) }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
