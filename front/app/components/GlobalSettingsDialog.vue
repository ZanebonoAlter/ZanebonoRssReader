<script setup lang="ts">
import { Icon } from "@iconify/vue";
import type { RssFeed } from '~/types'

interface Props {
  show: boolean
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:show': [value: boolean]
}>()

const apiStore = useApiStore()
const feedsStore = useFeedsStore()

const activeTab = ref<'feeds' | 'categories' | 'general'>('feeds')
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

  const response = await api.updateFeed(Number(feedId), {
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
  return 'text-purple-600'
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
      <div class="flex-1 overflow-y-auto p-6">
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
                        <Icon icon="mdi:brain" width="16" height="16" class="text-purple-500" />
                        <span class="text-xs font-medium text-gray-700">AI 总结</span>
                      </div>
                      <button
                        class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors"
                        :class="feed.aiSummaryEnabled !== false ? 'bg-purple-600' : 'bg-gray-300'"
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

        <!-- General Settings Tab -->
        <div v-if="activeTab === 'general'" class="space-y-6">
          <!-- AI Summary Settings -->
          <div class="bg-gradient-to-br from-purple-50 to-blue-50 rounded-xl p-6 border border-purple-100">
            <div class="flex items-start justify-between mb-4">
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-purple-500 to-blue-600 flex items-center justify-center">
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
              <div class="pt-4 border-t border-purple-200/50 mt-4">
                <div class="flex items-center justify-between">
                  <div>
                    <h4 class="text-sm font-medium text-gray-900">自动生成总结</h4>
                    <p class="text-xs text-gray-500 mt-0.5">每小时自动为每个分类生成 AI 总结</p>
                  </div>
                  <button
                    class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors"
                    :class="autoSummaryEnabled ? 'bg-purple-600' : 'bg-gray-300'"
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
