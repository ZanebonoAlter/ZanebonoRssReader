<script setup lang="ts">
import { Icon } from "@iconify/vue";
import type { RssFeed } from '~/types'
import type { ReadingStats, UserPreference } from '~/types/reading_behavior'

interface Props {
  show: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

const apiStore = useApiStore()
const feedsStore = useFeedsStore()
const preferencesStore = usePreferencesStore()

const activeTab = ref<'feeds' | 'categories' | 'general' | 'preferences'>('feeds')
const loading = ref(false)
const error = ref<string | null>(null)
const success = ref<string | null>(null)

// AI Summary Settings
const aiSummaryEnabled = ref(false)
const aiBaseURL = ref('')
const aiAPIKey = ref('')
const aiModel = ref('')
const showApiKey = ref(false)
const autoSummaryEnabled = ref(false)

// AI Podcast Settings
const aiPodcastEnabled = ref(false)

// Reading preferences
const preferenceType = ref<'feed' | 'category'>('feed')
const readingStats = ref<ReadingStats | null>(null)
const userPreferences = ref<UserPreference[]>([])
const preferencesLoading = ref(false)
const preferencesUpdating = ref(false)

// Get feeds grouped by category
const feedsByCategory = computed(() => {
  const grouped: Record<string, RssFeed[]> = {}
  apiStore.feeds.forEach((feed: RssFeed) => {
    const categoryName = feedsStore.categories.find(c => c.id === feed.category)?.name || '未分类'
    if (!grouped[categoryName]) {
      grouped[categoryName] = []
    }
    grouped[categoryName].push(feed)
  })
  return grouped
})

// Refresh interval options
const refreshOptions = [
  { label: '手动刷新', value: 0 },
  { label: '每 15 分钟', value: 15 },
  { label: '每 30 分钟', value: 30 },
  { label: '每小时', value: 60 },
  { label: '每 2 小时', value: 120 },
  { label: '每 6 小时', value: 360 },
  { label: '每天', value: 1440 },
]

// Max articles options
const maxArticlesOptions = [
  { label: '50 篇', value: 50 },
  { label: '100 篇', value: 100 },
  { label: '200 篇', value: 200 },
  { label: '500 篇', value: 500 },
  { label: '1000 篇', value: 1000 },
  { label: '无限制', value: 9999 },
]

async function updateFeedSetting(
  feedId: string,
  setting: 'refresh_interval' | 'max_articles' | 'ai_summary_enabled',
  value: number | boolean
) {
  loading.value = true
  error.value = null
  success.value = null

  const response = await apiStore.updateFeed(feedId, {
    [setting]: value,
  })

  loading.value = false

  if (response.success) {
    await apiStore.fetchFeeds({ per_page: 10000 })
    success.value = '设置已更新'

    // Update auto-refresh if refresh interval changed
    if (setting === 'refresh_interval') {
      const autoRefresh = useGlobalAutoRefresh()
      autoRefresh.updateFeedRefresh(feedId, value as number)
    }

    setTimeout(() => {
      success.value = null
    }, 2000)
  } else {
    error.value = response.error || '更新失败'
  }
}

function formatInterval(minutes: number): string {
  const option = refreshOptions.find(opt => opt.value === minutes)
  return option?.label || `${minutes} 分钟`
}

function formatMaxArticles(count: number): string {
  if (count >= 9999) return '无限制'
  const option = maxArticlesOptions.find(opt => opt.value === count)
  return option?.label || `${count} 篇`
}

function getIntervalColor(minutes: number): string {
  if (minutes === 0) return 'text-gray-500'
  if (minutes <= 30) return 'text-green-600'
  if (minutes <= 120) return 'text-blue-600'
  return 'text-ink-600'
}

async function refreshFeed(feedId: string) {
  loading.value = true
  await apiStore.refreshFeed(feedId)
  await apiStore.fetchFeeds({ per_page: 10000 })
  loading.value = false
  success.value = '订阅源已刷新'
  setTimeout(() => {
    success.value = null
  }, 2000)
}

function close() {
  // Reset to actual saved values to avoid confusion
  loadAISettings()
  emit('update:show', false)
}

// Load settings from localStorage
function loadAISettings() {
  const aiSettings = localStorage.getItem('aiSettings')
  if (aiSettings) {
    const settings = JSON.parse(aiSettings)
    aiSummaryEnabled.value = settings.summaryEnabled || false
    aiBaseURL.value = settings.baseURL || ''
    aiAPIKey.value = settings.apiKey || ''
    aiModel.value = settings.model || ''
    aiPodcastEnabled.value = settings.podcastEnabled || false
    autoSummaryEnabled.value = settings.autoSummaryEnabled || false
  } else {
    // Reset to defaults if no settings exist
    aiSummaryEnabled.value = false
    aiBaseURL.value = ''
    aiAPIKey.value = ''
    aiModel.value = ''
    aiPodcastEnabled.value = false
    autoSummaryEnabled.value = false
  }
}

// Load settings from localStorage
onMounted(() => {
  loadAISettings()
})

// Watch active tab changes
watch(activeTab, async (newTab) => {
  if (newTab === 'preferences') {
    await loadPreferencesData()
  }
})

// Load reading preferences data
async function loadPreferencesData() {
  preferencesLoading.value = true
  try {
    await Promise.all([
      preferencesStore.fetchStats(),
      preferencesStore.fetchPreferences(preferenceType.value)
    ])
    readingStats.value = preferencesStore.stats
    userPreferences.value = preferencesStore.preferences
  } catch (e) {
    console.error('Failed to load preferences:', e)
  } finally {
    preferencesLoading.value = false
  }
}

// Trigger preference update
async function triggerPreferenceUpdate() {
  preferencesUpdating.value = true
  try {
    await preferencesStore.triggerUpdate()
    await loadPreferencesData()
  } finally {
    preferencesUpdating.value = false
  }
}

// Get score color
function getScoreColor(score: number): string {
  if (score >= 0.7) return 'bg-green-500'
  if (score >= 0.4) return 'bg-yellow-500'
  return 'bg-gray-400'
}

// Save AI summary settings
async function saveAISummarySettings() {
  const settings = {
    summaryEnabled: aiSummaryEnabled.value,
    baseURL: aiBaseURL.value,
    apiKey: aiAPIKey.value,
    model: aiModel.value,
    podcastEnabled: aiPodcastEnabled.value,
    autoSummaryEnabled: autoSummaryEnabled.value,
  }
  localStorage.setItem('aiSettings', JSON.stringify(settings))

  // Update auto-summary scheduler config on backend
  if (autoSummaryEnabled.value && aiAPIKey.value) {
    try {
      await api.updateAutoSummaryConfig({
        base_url: aiBaseURL.value || 'https://api.openai.com/v1',
        api_key: aiAPIKey.value,
        model: aiModel.value || 'gpt-4o-mini'
      })
    } catch (e) {
      console.error('Failed to update auto-summary config:', e)
    }
  }

  success.value = 'AI 设置已保存'
  setTimeout(() => {
    success.value = null
  }, 2000)
}

// Test AI connection
async function testAIConnection() {
  loading.value = true
  error.value = null

  try {
    const { testConnection } = useAI()
    const result = await testConnection()

    if (result.success) {
      success.value = result.message || '连接测试成功'
      setTimeout(() => {
        success.value = null
      }, 2000)
    } else {
      error.value = result.error || '连接测试失败'
    }
  } catch (e) {
    error.value = '连接测试失败，请检查配置'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div
    v-if="props.show"
    class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
    @click.self="close"
  >
    <div
      class="bg-white rounded-xl shadow-xl w-full max-w-4xl mx-4 overflow-hidden max-h-[90vh] flex flex-col"
      style="max-height: calc(90vh - 2rem);"
      @click.stop
    >
      <!-- Header -->
      <div class="px-6 py-4 border-b border-gray-200 flex items-center justify-between flex-shrink-0">
        <h2 class="text-xl font-bold text-gray-900">RSS 阅读器设置</h2>
        <button
          class="p-2 hover:bg-gray-100 rounded-lg transition-colors"
          @click="close"
        >
          <Icon icon="mdi:close" width="20" height="20" />
        </button>
      </div>

      <!-- Tabs -->
      <div class="flex border-b border-gray-200 flex-shrink-0">
        <button
          class="px-6 py-3 text-sm font-medium transition-colors"
          :class="activeTab === 'feeds' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-gray-500 hover:text-gray-700'"
          @click="activeTab = 'feeds'"
        >
          订阅源配置
        </button>
        <button
          class="px-6 py-3 text-sm font-medium transition-colors"
          :class="activeTab === 'categories' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-gray-500 hover:text-gray-700'"
          @click="activeTab = 'categories'"
        >
          分类管理
        </button>
        <button
          class="px-6 py-3 text-sm font-medium transition-colors"
          :class="activeTab === 'general' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-gray-500 hover:text-gray-700'"
          @click="activeTab = 'general'"
        >
          通用设置
        </button>
        <button
          class="px-6 py-3 text-sm font-medium transition-colors"
          :class="activeTab === 'preferences' ? 'text-blue-600 border-b-2 border-blue-600' : 'text-gray-500 hover:text-gray-700'"
          @click="activeTab = 'preferences'"
        >
          阅读偏好
        </button>
      </div>

      <!-- Success/Error messages -->
      <div
        v-if="success"
        class="mx-6 mt-4 p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-600 flex-shrink-0"
      >
        {{ success }}
      </div>
      <div
        v-if="error"
        class="mx-6 mt-4 p-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-600 flex-shrink-0"
      >
        {{ error }}
      </div>

      <!-- Content -->
      <div class="flex-1 overflow-y-auto p-6 min-h-0">
        <!-- Feeds Configuration Tab -->
        <div v-if="activeTab === 'feeds'" class="space-y-6">
          <div v-if="Object.keys(feedsByCategory).length === 0" class="text-center text-gray-500 py-8">
            <Icon icon="mdi:rss-off" width="48" height="48" class="mx-auto mb-2 opacity-50" />
            <p>还没有订阅源</p>
          </div>

          <div
            v-for="(feeds, categoryName) in feedsByCategory"
            :key="categoryName"
            class="space-y-3"
          >
            <h3 class="text-sm font-semibold text-gray-700 flex items-center gap-2">
              <Icon icon="mdi:folder" width="16" height="16" />
              {{ categoryName }}
              <span class="text-xs font-normal text-gray-400">({{ feeds.length }})</span>
            </h3>

            <div class="space-y-2">
              <div
                v-for="feed in feeds"
                :key="feed.id"
                class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors"
              >
                <div class="flex items-start gap-3">
                  <div
                    class="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0"
                    :style="{ backgroundColor: feed.color + '15' }"
                  >
                    <FeedIcon
                      :icon="feed.icon"
                      :feed-id="feed.id"
                      :color="feed.color"
                      :size="20"
                    />
                  </div>

                  <div class="flex-1 min-w-0">
                    <div class="flex items-start justify-between gap-2 mb-3">
                      <div>
                        <h4 class="font-medium text-gray-900 truncate">{{ feed.title }}</h4>
                        <p class="text-xs text-gray-500 truncate">{{ feed.url }}</p>
                      </div>
                      <button
                        class="p-1.5 hover:bg-gray-100 rounded-lg transition-colors"
                        :title="'立即刷新'"
                        :disabled="loading"
                        @click="refreshFeed(feed.id)"
                      >
                        <Icon
                          :icon="loading ? 'mdi:loading' : 'mdi:refresh'"
                          :class="{ 'animate-spin': loading }"
                          width="16"
                          height="16"
                          class="text-gray-500"
                        />
                      </button>
                    </div>

                    <div class="grid grid-cols-2 gap-3">
                      <!-- Refresh Interval -->
                      <div>
                        <label class="block text-xs font-medium text-gray-600 mb-1">
                          自动刷新
                        </label>
                        <select
                          :value="feed.refreshInterval"
                          class="w-full text-xs px-2 py-1.5 border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          @change="updateFeedSetting(feed.id, 'refresh_interval', Number(($event.target as HTMLSelectElement).value))"
                        >
                          <option
                            v-for="option in refreshOptions"
                            :key="option.value"
                            :value="option.value"
                          >
                            {{ option.label }}
                          </option>
                        </select>
                      </div>

                      <!-- Max Articles -->
                      <div>
                        <label class="block text-xs font-medium text-gray-600 mb-1">
                          最大文章数
                        </label>
                        <select
                          :value="feed.maxArticles"
                          class="w-full text-xs px-2 py-1.5 border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                          @change="updateFeedSetting(feed.id, 'max_articles', Number(($event.target as HTMLSelectElement).value))"
                        >
                          <option
                            v-for="option in maxArticlesOptions"
                            :key="option.value"
                            :value="option.value"
                          >
                            {{ option.label }}
                          </option>
                        </select>
                      </div>
                    </div>

                    <!-- AI Summary Toggle -->
                    <div class="flex items-center justify-between mt-3 pt-3 border-t border-gray-100">
                      <div class="flex items-center gap-2">
                        <Icon icon="mdi:brain" width="16" height="16" class="text-ink-500" />
                        <span class="text-xs font-medium text-gray-700">AI 总结</span>
                      </div>
                      <button
                        class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors"
                        :class="feed.aiSummaryEnabled !== false ? 'bg-ink-600' : 'bg-gray-300'"
                        @click="updateFeedSetting(feed.id, 'ai_summary_enabled', !(feed.aiSummaryEnabled !== false))"
                      >
                        <span
                          class="inline-block h-3 w-3 transform rounded-full bg-white transition-transform"
                          :class="feed.aiSummaryEnabled !== false ? 'translate-x-5' : 'translate-x-1'"
                        />
                      </button>
                    </div>

                    <!-- Current Settings Summary -->
                    <div class="flex items-center gap-3 mt-2 text-xs">
                      <span :class="getIntervalColor(feed.refreshInterval || 0)">
                        <Icon icon="mdi:clock" width="12" height="12" class="inline-block mr-1" />
                        {{ formatInterval(feed.refreshInterval || 0) }}
                      </span>
                      <span class="text-gray-500">
                        <Icon icon="mdi:file-document-multiple" width="12" height="12" class="inline-block mr-1" />
                        {{ formatMaxArticles(feed.maxArticles || 100) }}
                      </span>
                      <span class="text-gray-500">
                        <Icon icon="mdi:article" width="12" height="12" class="inline-block mr-1" />
                        {{ feed.articleCount }} 篇
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Categories Management Tab -->
        <div v-if="activeTab === 'categories'" class="space-y-4">
          <p class="text-sm text-gray-500">分类管理功能开发中...</p>
        </div>

        <!-- Reading Preferences Tab -->
        <div v-if="activeTab === 'preferences'" class="space-y-6">
          <!-- Loading State -->
          <div v-if="preferencesLoading" class="flex items-center justify-center py-12">
            <div class="text-center">
              <Icon icon="mdi:loading" width="48" height="48" class="animate-spin text-blue-500 mx-auto mb-3" />
              <p class="text-sm text-gray-500">加载偏好数据...</p>
            </div>
          </div>

          <!-- Preferences Content -->
          <div v-else class="space-y-6">
            <!-- Quick Stats -->
            <div class="grid grid-cols-4 gap-4">
              <div class="bg-gradient-to-br from-blue-50 to-blue-100 rounded-xl p-4 border border-blue-200">
                <div class="flex items-center gap-2 mb-2">
                  <Icon icon="mdi:file-document" width="18" height="18" class="text-blue-600" />
                  <span class="text-xs font-medium text-blue-900">总文章数</span>
                </div>
                <div class="text-2xl font-bold text-blue-700">
                  {{ readingStats?.total_articles || 0 }}
                </div>
              </div>

              <div class="bg-gradient-to-br from-green-50 to-green-100 rounded-xl p-4 border border-green-200">
                <div class="flex items-center gap-2 mb-2">
                  <Icon icon="mdi:clock" width="18" height="18" class="text-green-600" />
                  <span class="text-xs font-medium text-green-900">总阅读时长</span>
                </div>
                <div class="text-2xl font-bold text-green-700">
                  {{ readingStats?.total_reading_time || 0 }}s
                </div>
              </div>

              <div class="bg-gradient-to-br from-ink-50 to-paper-cream rounded-xl p-4 border border-ink-200">
                <div class="flex items-center gap-2 mb-2">
                  <Icon icon="mdi:timer" width="18" height="18" class="text-ink-600" />
                  <span class="text-xs font-medium text-ink-900">平均时长</span>
                </div>
                <div class="text-2xl font-bold text-ink-700">
                  {{ Math.round(readingStats?.avg_reading_time || 0) }}s
                </div>
              </div>

              <div class="bg-gradient-to-br from-orange-50 to-orange-100 rounded-xl p-4 border border-orange-200">
                <div class="flex items-center gap-2 mb-2">
                  <Icon icon="mdi:arrow-down-bold" width="18" height="18" class="text-orange-600" />
                  <span class="text-xs font-medium text-orange-900">平均深度</span>
                </div>
                <div class="text-2xl font-bold text-orange-700">
                  {{ Math.round(readingStats?.avg_scroll_depth || 0) }}%
                </div>
              </div>
            </div>

            <!-- Controls -->
            <div class="flex items-center justify-between bg-gray-50 rounded-lg p-4">
              <div class="flex items-center gap-3">
                <div class="text-sm">
                  <div class="font-medium text-gray-900">查看偏好</div>
                  <div class="text-xs text-gray-500">按订阅源或分类查看</div>
                </div>
                <div class="flex gap-2">
                  <button
                    class="px-3 py-1.5 text-xs font-medium rounded-lg transition-colors"
                    :class="preferenceType === 'feed' ? 'bg-blue-600 text-white' : 'bg-white text-gray-700 border border-gray-200'"
                    @click="preferenceType = 'feed'; loadPreferencesData()"
                  >
                    订阅源
                  </button>
                  <button
                    class="px-3 py-1.5 text-xs font-medium rounded-lg transition-colors"
                    :class="preferenceType === 'category' ? 'bg-blue-600 text-white' : 'bg-white text-gray-700 border border-gray-200'"
                    @click="preferenceType = 'category'; loadPreferencesData()"
                  >
                    分类
                  </button>
                </div>
              </div>
              <button
                class="px-4 py-2 text-sm font-medium text-white bg-orange-600 rounded-lg hover:bg-orange-700 transition-colors flex items-center gap-2"
                :disabled="preferencesUpdating"
                @click="triggerPreferenceUpdate"
              >
                <Icon
                  :icon="preferencesUpdating ? 'mdi:loading' : 'mdi:refresh'"
                  :class="{ 'animate-spin': preferencesUpdating }"
                  width="16"
                  height="16"
                />
                更新偏好
              </button>
            </div>

            <!-- Preferences List -->
            <div class="bg-white rounded-xl border border-gray-200 overflow-hidden">
              <div v-if="userPreferences.length === 0" class="text-center py-12">
                <Icon icon="mdi:chart-line" width="48" height="48" class="text-gray-300 mx-auto mb-3" />
                <p class="text-sm text-gray-500">暂无偏好数据</p>
                <p class="text-xs text-gray-400 mt-1">阅读文章后将自动生成偏好分析</p>
              </div>

              <div v-else class="divide-y divide-gray-100">
                <div
                  v-for="pref in userPreferences"
                  :key="pref.id"
                  class="p-4 hover:bg-gray-50 transition-colors"
                >
                  <div class="flex items-center justify-between">
                    <div class="flex-1">
                      <div class="flex items-center gap-2 mb-1">
                        <h4 class="text-sm font-medium text-gray-900">
                          {{ pref.feed_title || pref.category_name }}
                        </h4>
                        <span class="px-2 py-0.5 text-xs font-medium rounded-full text-white"
                              :class="getScoreColor(pref.preference_score)">
                          {{ (pref.preference_score * 100).toFixed(0) }}%
                        </span>
                      </div>
                      <div class="flex items-center gap-4 text-xs text-gray-500">
                        <span class="flex items-center gap-1">
                          <Icon icon="mdi:cursor-default-click" width="12" height="12" />
                          {{ pref.interaction_count }} 次互动
                        </span>
                        <span class="flex items-center gap-1">
                          <Icon icon="mdi:clock" width="12" height="12" />
                          平均 {{ pref.avg_reading_time }} 秒
                        </span>
                        <span class="flex items-center gap-1">
                          <Icon icon="mdi:arrow-down-bold" width="12" height="12" />
                          {{ Math.round(pref.scroll_depth_avg) }}% 深度
                        </span>
                      </div>
                    </div>

                    <!-- Preference Score Bar -->
                    <div class="ml-4 w-24">
                      <div class="h-2 bg-gray-200 rounded-full overflow-hidden">
                        <div
                          class="h-full rounded-full transition-all"
                          :class="getScoreColor(pref.preference_score)"
                          :style="{ width: (pref.preference_score * 100) + '%' }"
                        />
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Info Box -->
            <div class="bg-blue-50 rounded-lg p-4 flex items-start gap-3">
              <Icon icon="mdi:information" width="20" height="20" class="text-blue-600 flex-shrink-0 mt-0.5" />
              <div class="text-sm text-blue-900">
                <div class="font-medium mb-1">关于阅读偏好</div>
                <p class="text-blue-700 text-xs">
                  系统会根据您的阅读行为（滚动深度、阅读时长、互动频率）自动计算偏好分数。
                  分数范围 0-100%，越高表示您对该订阅源或分类越感兴趣。
                  偏好数据每 30 分钟自动更新，也可手动触发更新。
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- General Settings Tab -->
        <div v-if="activeTab === 'general'" class="space-y-6">
          <!-- AI Summary Settings -->
          <div class="bg-gradient-to-br from-ink-50 to-paper-cream rounded-xl p-6 border border-ink-100">
            <div class="flex items-start justify-between mb-4">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-ink-500 to-ink-700 flex items-center justify-center">
                  <Icon icon="mdi:brain" width="20" height="20" class="text-white" />
                </div>
                <div>
                  <h3 class="font-semibold text-gray-900">AI 总结分析</h3>
                  <p class="text-xs text-gray-500">使用 AI 模型对文章进行智能总结和分析</p>
                </div>
              </div>
              <button
                class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors"
                :class="aiSummaryEnabled ? 'bg-blue-600' : 'bg-gray-300'"
                @click="aiSummaryEnabled = !aiSummaryEnabled"
              >
                <span
                  class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                  :class="aiSummaryEnabled ? 'translate-x-6' : 'translate-x-1'"
                />
              </button>
            </div>

            <div v-if="aiSummaryEnabled" class="space-y-4 mt-4">
              <!-- Base URL -->
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1.5">
                  Base URL
                </label>
                <input
                  v-model="aiBaseURL"
                  type="text"
                  placeholder="https://api.openai.com/v1"
                  class="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <!-- API Key -->
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1.5">
                  API Key
                </label>
                <div class="relative">
                  <input
                    v-model="aiAPIKey"
                    :type="showApiKey ? 'text' : 'password'"
                    placeholder="sk-..."
                    class="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent pr-20"
                  />
                  <div class="absolute right-2 top-1/2 -translate-y-1/2 flex gap-1">
                    <button
                      class="p-1 hover:bg-gray-100 rounded text-gray-400 hover:text-gray-600"
                      @click="showApiKey = !showApiKey"
                    >
                      <Icon :icon="showApiKey ? 'mdi:eye-off' : 'mdi:eye'" width="16" height="16" />
                    </button>
                  </div>
                </div>
              </div>

              <!-- Model -->
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1.5">
                  模型
                </label>
                <input
                  v-model="aiModel"
                  type="text"
                  placeholder="gpt-4o-mini"
                  class="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                />
              </div>

              <!-- Actions -->
              <div class="flex gap-2 pt-2">
                <button
                  class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
                  :disabled="loading"
                  @click="saveAISummarySettings"
                >
                  保存配置
                </button>
                <button
                  class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-200 rounded-lg hover:bg-gray-50 transition-colors"
                  :disabled="loading"
                  @click="testAIConnection"
                >
                  <Icon v-if="loading" icon="mdi:loading" width="14" height="14" class="animate-spin inline-block mr-1" />
                  测试连接
                </button>
              </div>

              <!-- Auto Summary Toggle -->
              <div class="pt-4 border-t border-ink-200/50 mt-4">
                <div class="flex items-center justify-between">
                  <div>
                    <h4 class="text-sm font-medium text-gray-900">自动生成总结</h4>
                    <p class="text-xs text-gray-500 mt-0.5">每小时自动为每个分类生成 AI 总结</p>
                  </div>
                  <button
                    class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors"
                    :class="autoSummaryEnabled ? 'bg-ink-600' : 'bg-gray-300'"
                    @click="autoSummaryEnabled = !autoSummaryEnabled"
                  >
                    <span
                      class="inline-block h-3 w-3 transform rounded-full bg-white transition-transform"
                      :class="autoSummaryEnabled ? 'translate-x-5' : 'translate-x-1'"
                    />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- AI Podcast Settings -->
          <div class="bg-gradient-to-br from-green-50 to-teal-50 rounded-xl p-6 border border-green-100">
            <div class="flex items-start justify-between mb-4">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-green-500 to-teal-600 flex items-center justify-center">
                  <Icon icon="mdi:podium" width="20" height="20" class="text-white" />
                </div>
                <div>
                  <h3 class="font-semibold text-gray-900">AI 播客</h3>
                  <p class="text-xs text-gray-500">将文章转换为播客音频（即将推出）</p>
                </div>
              </div>
              <button
                class="relative inline-flex h-6 w-11 items-center rounded-full transition-colors cursor-not-allowed opacity-50"
                :class="aiPodcastEnabled ? 'bg-green-600' : 'bg-gray-300'"
                @click="aiPodcastEnabled = !aiPodcastEnabled"
              >
                <span
                  class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                  :class="aiPodcastEnabled ? 'translate-x-6' : 'translate-x-1'"
                />
              </button>
            </div>

            <div class="flex items-center gap-2 text-sm text-gray-500 bg-white/50 rounded-lg p-3">
              <Icon icon="mdi:information" width="16" height="16" class="text-green-600" />
              <span>AI 播客功能正在开发中，敬请期待...</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-6 py-4 bg-gray-50 border-t border-gray-200 flex justify-end flex-shrink-0">
        <button class="btn btn-primary" @click="close">
          完成
        </button>
      </div>
    </div>
  </div>
</template>
