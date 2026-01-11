<script setup lang="ts">
import { Icon } from "@iconify/vue";

const apiStore = useApiStore()
const feedsStore = useFeedsStore()
const articlesStore = useArticlesStore()

// Initialize global auto-refresh (singleton pattern)
useGlobalAutoRefresh()

const sidebarCollapsed = ref(false)
const selectedCategory = ref<string | null>(null)
const selectedFeed = ref<string | null>(null)
const showAISummaries = ref(false)
const selectedSummary = ref<any>(null)
const showAddFeedDialog = ref(false)
const showAddCategoryDialog = ref(false)
const showImportDialog = ref(false)
const showGlobalSettings = ref(false)

// Sidebar width (resizable)
const sidebarWidth = ref(256) // Default width in pixels (w-64)
const minSidebarWidth = 200
const maxSidebarWidth = 500
const isResizing = ref(false)

// Refresh feedback
const refreshMessage = ref('')
const refreshMessageType = ref<'success' | 'error' | 'info'>('info')
const showRefreshMessage = ref(false)

// Edit dialogs
const editCategoryId = ref<string | null>(null)
const editFeedId = ref<string | null>(null)

// Computed
const editingCategory = computed(() =>
  editCategoryId.value ? feedsStore.categories.find(c => c.id === editCategoryId.value) : null
)
const editingFeed = computed(() =>
  editFeedId.value ? feedsStore.feeds.find(f => f.id === editFeedId.value) : null
)

// Uncategorized feeds
const uncategorizedFeeds = computed(() => {
  return feedsStore.feeds.filter(f => !f.category)
})

const filteredArticles = computed(() => {
  let articles = articlesStore.articles

  // Filter by selected feed (still needed since feed selection uses local filtering)
  if (selectedFeed.value) {
    articles = articles.filter((a) => a.feedId === selectedFeed.value)
  }
  // Filter by favorites (still needed since we fetch all articles for favorites)
  else if (selectedCategory.value === 'favorites') {
    articles = articles.filter((a) => a.favorite)
  }
  // For categories and uncategorized, data is already filtered by API
  // but we keep the filter as a safety check

  return articles
})

// Global unread count - tracks total unread articles across all categories
const globalUnreadCount = ref(0)

// Fetch global unread count from stats API
async function fetchGlobalUnreadCount() {
  const response = await apiStore.fetchArticlesStats()
  if (response.success && response.data) {
    globalUnreadCount.value = (response.data as any).unread || 0
  }
}

// Initialize global unread count on mount
onMounted(() => {
  fetchGlobalUnreadCount()
})

// Clean up polling on unmount
onUnmounted(() => {
  stopPollingRefreshStatus()
})

// Functions
async function handleCategoryClick(categoryId: string) {
  selectedCategory.value = categoryId
  selectedFeed.value = null
  showAISummaries.value = false
  selectedSummary.value = null

  // Only load feeds and articles for this category
  if (categoryId === 'uncategorized') {
    // For uncategorized, use backend filter
    await apiStore.fetchFeeds({ uncategorized: true, per_page: 10000 })
    await apiStore.fetchArticles({ uncategorized: true, per_page: 10000 })
  } else {
    // Fetch feeds for this category
    await apiStore.fetchFeeds({
      category_id: parseInt(categoryId),
      per_page: 10000
    })
    // Fetch articles for this category
    await apiStore.fetchArticles({
      category_id: parseInt(categoryId),
      per_page: 10000
    })
  }
  apiStore.syncToLocalStores()
}

async function handleFavoritesClick() {
  selectedCategory.value = 'favorites'
  selectedFeed.value = null
  showAISummaries.value = false
  selectedSummary.value = null

  // Fetch ALL articles (not filtered by category), then filter locally for favorites
  // This ensures favorites are shown from all feeds, not just current category
  await apiStore.fetchArticles({
    per_page: 10000,
    favorite: true  // Let backend filter for favorites
  })
  apiStore.syncToLocalStores()
}

async function handleFeedClick(feedId: string) {
  selectedFeed.value = feedId
  showAISummaries.value = false
  selectedSummary.value = null

  // 获取该订阅源的文章（显示当前已有文章）
  await apiStore.fetchArticles({
    feed_id: parseInt(feedId),
    per_page: 10000
  })
  apiStore.syncToLocalStores()

  // 自动刷新该订阅源（后台进行，不影响当前显示）
  const response = await apiStore.refreshFeed(feedId)

  if (response.success) {
    // 刷新成功，显示刷新状态
    refreshMessage.value = '正在刷新订阅源...'
    refreshMessageType.value = 'info'
    showRefreshMessage.value = true

    // 只更新当前 feed 的状态为 refreshing，不重新获取所有 feeds
    const feed = feedsStore.feeds.find(f => f.id === feedId)
    if (feed) {
      feed.refreshStatus = 'refreshing'
    }

    // 开始轮询刷新状态（复用现有的轮询机制）
    pollRefreshStatus()
  } else {
    // 刷新失败，显示错误
    refreshMessage.value = response.error || '刷新失败'
    refreshMessageType.value = 'error'
    showRefreshMessage.value = true
  }
}

// Function to reset to all articles view
async function handleAllArticlesClick() {
  selectedCategory.value = null
  selectedFeed.value = null
  showAISummaries.value = false

  // Fetch all feeds and articles
  await apiStore.fetchFeeds({ per_page: 10000 })
  await apiStore.fetchArticles({ per_page: 10000 })
  apiStore.syncToLocalStores()
}

async function handleAISummariesClick() {
  selectedCategory.value = 'ai-summaries'
  selectedFeed.value = null
  showAISummaries.value = true
  selectedSummary.value = null
}

function handleSummarySelect(summary: any) {
  selectedSummary.value = summary
}

// Poll for refresh status updates
const refreshPollingInterval = ref<ReturnType<typeof setTimeout> | null>(null)
const MAX_POLLING_TIME = 60000 // Max 60 seconds of polling
const POLLING_INTERVAL = 2000 // Check every 2 seconds

async function pollRefreshStatus() {
  const startTime = Date.now()

  const poll = async () => {
    // Stop polling if exceeded max time
    if (Date.now() - startTime > MAX_POLLING_TIME) {
      stopPollingRefreshStatus()
      return
    }

    // Fetch latest feeds status based on current selection
    if (selectedFeed.value) {
      // Single feed selected - fetch all and find the specific one
      await apiStore.fetchFeeds({ per_page: 10000 })
    } else if (selectedCategory.value === 'uncategorized') {
      await apiStore.fetchFeeds({ uncategorized: true, per_page: 10000 })
    } else if (selectedCategory.value && selectedCategory.value !== 'favorites' && selectedCategory.value !== 'ai-summaries') {
      await apiStore.fetchFeeds({ category_id: parseInt(selectedCategory.value), per_page: 10000 })
    } else {
      await apiStore.fetchFeeds({ per_page: 10000 })
    }

    apiStore.syncToLocalStores()

    // Get the feeds we're monitoring for refresh status
    const monitoredFeeds = selectedFeed.value
      ? feedsStore.feeds.filter(f => f.id === selectedFeed.value)
      : feedsStore.feeds

    // Check if any feed is still refreshing
    const stillRefreshing = monitoredFeeds.some(
      f => f.refreshStatus === 'refreshing'
    )

    if (stillRefreshing) {
      // Continue polling
      refreshPollingInterval.value = setTimeout(poll, POLLING_INTERVAL)
    } else {
      // All feeds finished refreshing
      stopPollingRefreshStatus()

      // Fetch articles based on current selection
      const articleParams: any = { per_page: 10000 }
      if (selectedFeed.value) {
        articleParams.feed_id = parseInt(selectedFeed.value)
      } else if (selectedCategory.value === 'uncategorized') {
        articleParams.uncategorized = true
      } else if (selectedCategory.value && selectedCategory.value !== 'favorites' && selectedCategory.value !== 'ai-summaries') {
        articleParams.category_id = parseInt(selectedCategory.value)
      }

      await apiStore.fetchArticles(articleParams)
      apiStore.syncToLocalStores()
      await fetchGlobalUnreadCount()

      // Show success message
      refreshMessage.value = '刷新完成'
      refreshMessageType.value = 'success'
      showRefreshMessage.value = true
      setTimeout(() => {
        showRefreshMessage.value = false
      }, 3000)
    }
  }

  // Start polling
  poll()
}

function stopPollingRefreshStatus() {
  if (refreshPollingInterval.value) {
    clearTimeout(refreshPollingInterval.value)
    refreshPollingInterval.value = null
  }
}

async function handleRefresh() {
  // Stop any existing polling
  stopPollingRefreshStatus()

  if (selectedFeed.value) {
    const response = await apiStore.refreshFeed(selectedFeed.value)
    if (response.success) {
      refreshMessage.value = response.message || '已开始后台刷新'
      refreshMessageType.value = 'info'
      // Update local store to show refreshing status
      await apiStore.fetchFeeds({ per_page: 10000 })
      apiStore.syncToLocalStores()
      // Start polling for status updates
      pollRefreshStatus()
    } else {
      refreshMessage.value = response.error || '刷新失败'
      refreshMessageType.value = 'error'
    }
  } else {
    const response = await apiStore.refreshAllFeeds()
    if (response.success) {
      refreshMessage.value = response.message || '已开始后台刷新所有订阅源'
      refreshMessageType.value = 'info'
      // Update local store to show refreshing status
      await apiStore.fetchFeeds({ per_page: 10000 })
      apiStore.syncToLocalStores()
      // Start polling for status updates
      pollRefreshStatus()
    } else {
      refreshMessage.value = response.error || '刷新失败'
      refreshMessageType.value = 'error'
    }
  }

  showRefreshMessage.value = true
}

async function handleMarkAllRead() {
  if (selectedFeed.value) {
    await apiStore.markAllAsRead(selectedFeed.value)
  } else if (selectedCategory.value) {
    // Mark all articles in category as read
    const feedIds = feedsStore.feeds
      .filter((f) => f.category === selectedCategory.value)
      .map((f) => f.id)
    for (const feedId of feedIds) {
      await apiStore.markAllAsRead(feedId)
    }
  } else {
    await apiStore.markAllAsRead()
  }
  // Update global unread count after marking as read
  await fetchGlobalUnreadCount()
}

async function handleExportOpml() {
  try {
    const blob = await apiStore.exportOpml()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `feeds-export-${new Date().toISOString().split('T')[0]}.opml`
    a.click()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    console.error('Export failed:', error)
  }
}

function handleAddCategory() {
  showAddCategoryDialog.value = true
}

function handleEditCategory(categoryId: string) {
  editCategoryId.value = categoryId
}

function handleEditFeed(feedId: string) {
  editFeedId.value = feedId
}

async function handleDeleteCategory(categoryId: string, categoryName: string) {
  if (confirm(`确定要删除分类"${categoryName}"吗？此操作不会删除该分类下的订阅源。`)) {
    const response = await apiStore.deleteCategory(categoryId)
    if (response.success) {
      apiStore.syncToLocalStores()
    } else {
      alert(response.error || '删除失败')
    }
  }
}

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

// Sidebar resizing
function startResizing(event: MouseEvent) {
  isResizing.value = true
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', onResizing)
  document.addEventListener('mouseup', stopResizing)
}

function onResizing(event: MouseEvent) {
  if (!isResizing.value) return

  const newWidth = event.clientX
  if (newWidth >= minSidebarWidth && newWidth <= maxSidebarWidth) {
    sidebarWidth.value = newWidth
  }
}

function stopResizing() {
  isResizing.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  document.removeEventListener('mousemove', onResizing)
  document.removeEventListener('mouseup', stopResizing)
}
</script>

<template>
  <div class="h-screen flex flex-col overflow-hidden gradient-bg">
    <!-- Top Toolbar -->
    <header class="glass-strong border-b border-white/20 px-4 py-2.5 flex items-center justify-between flex-shrink-0">
      <div class="flex items-center gap-3">
        <button
          class="p-2.5 rounded-xl hover:bg-white/60 transition-all duration-200"
          @click="toggleSidebar"
        >
          <Icon icon="mdi:menu" width="20" height="20" class="text-gray-600" />
        </button>
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl bg-linear-to-br from-primary-500 to-primary-700 flex items-center justify-center shadow-lg">
            <Icon icon="mdi:rss" class="text-white" width="20" height="20" />
          </div>
          <span class="font-bold text-lg text-gray-800">RSS Reader</span>
        </div>
      </div>

      <div class="flex items-center gap-1.5">
        <button
          class="p-2.5 rounded-xl hover:bg-white/60 transition-all duration-200"
          title="刷新"
          @click="handleRefresh"
        >
          <Icon icon="mdi:refresh" width="20" height="20" class="text-gray-600" />
        </button>
        <button
          class="p-2.5 rounded-xl hover:bg-white/60 transition-all duration-200"
          title="全部标为已读"
          @click="handleMarkAllRead"
        >
          <Icon icon="mdi:email-open-multiple" width="20" height="20" class="text-gray-600" />
        </button>
        <button
          class="p-2.5 rounded-xl hover:bg-white/60 transition-all duration-200"
          title="添加订阅"
          @click="showAddFeedDialog = true"
        >
          <Icon icon="mdi:plus" width="20" height="20" class="text-gray-600" />
        </button>
        <button
          class="p-2.5 rounded-xl hover:bg-white/60 transition-all duration-200"
          title="添加分类"
          @click="handleAddCategory"
        >
          <Icon icon="mdi:folder-plus" width="20" height="20" class="text-gray-600" />
        </button>
        <button
          class="p-2.5 rounded-xl hover:bg-white/60 transition-all duration-200"
          title="导入"
          @click="showImportDialog = true"
        >
          <Icon icon="mdi:import" width="20" height="20" class="text-gray-600" />
        </button>
        <div class="w-px h-6 bg-gray-200/60 mx-1" />
        <button
          class="p-2.5 rounded-xl hover:bg-white/60 transition-all duration-200"
          title="设置"
          @click="showGlobalSettings = true"
        >
          <Icon icon="mdi:cog" width="20" height="20" class="text-gray-600" />
        </button>
      </div>
    </header>

    <!-- Refresh Message Toast -->
    <transition
      enter-active-class="transition ease-out duration-300"
      enter-from-class="transform opacity-0 translate-y-2"
      enter-to-class="transform opacity-100 translate-y-0"
      leave-active-class="transition ease-in duration-200"
      leave-from-class="transform opacity-100 translate-y-0"
      leave-to-class="transform opacity-0 translate-y-2"
    >
      <div
        v-if="showRefreshMessage"
        class="fixed top-20 left-1/2 transform -translate-x-1/2 z-50 max-w-md"
      >
        <div
          class="glass-strong px-4 py-3 rounded-2xl shadow-xl flex items-center gap-3"
          :class="{
            'border-l-4 border-l-emerald-500': refreshMessageType === 'success',
            'border-l-4 border-l-red-500': refreshMessageType === 'error',
            'border-l-4 border-l-primary-500': refreshMessageType === 'info'
          }"
        >
          <Icon
            :icon="refreshMessageType === 'success' ? 'mdi:check-circle' : refreshMessageType === 'error' ? 'mdi:alert-circle' : 'mdi:information'"
            :width="22"
            :height="22"
            :class="{
              'text-emerald-600': refreshMessageType === 'success',
              'text-red-500': refreshMessageType === 'error',
              'text-primary-600': refreshMessageType === 'info'
            }"
          />
          <span class="text-sm font-medium text-gray-700">{{ refreshMessage }}</span>
          <button
            class="ml-auto p-1 rounded-lg hover:bg-black/5 transition-colors"
            @click="showRefreshMessage = false"
          >
            <Icon icon="mdi:close" width="18" height="18" class="text-gray-400" />
          </button>
        </div>
      </div>
    </transition>

    <!-- Main Content -->
    <div class="flex-1 flex overflow-hidden">
      <!-- Sidebar - Categories & Feeds -->
      <aside
        class="glass-card border-r border-white/20 flex-shrink-0 relative m-2 rounded-2xl overflow-hidden"
        :style="sidebarCollapsed ? 'width: 48px' : `width: ${sidebarWidth}px`"
      >
        <!-- Resize handle (only show when not collapsed) -->
        <div
          v-if="!sidebarCollapsed"
          class="absolute top-0 right-0 w-1.5 h-full cursor-col-resize hover:bg-primary-400/50 transition-colors z-10 after:content-[''] after:absolute after:left-1/2 after:top-1/2 after:-translate-x-1/2 after:-translate-y-1/2 after:w-1 after:h-8 after:bg-primary-400 after:rounded-full after:opacity-0 hover:after:opacity-100 after:transition-opacity"
          :class="{ 'bg-primary-400/50': isResizing }"
          @mousedown="startResizing"
        />

        <div class="h-full overflow-y-auto glass-scroll p-2">
          <!-- All Items -->
          <button
            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-white/50 transition-all duration-200 mb-1"
            :class="{ 'bg-primary-100/80 text-primary-700 shadow-sm': !selectedCategory && !selectedFeed }"
            @click="handleAllArticlesClick"
          >
            <Icon icon="mdi:inbox" width="20" height="20" />
            <span v-if="!sidebarCollapsed" class="flex-1 text-left font-medium">
              全部文章
            </span>
            <span
              v-if="!sidebarCollapsed && globalUnreadCount > 0"
              class="text-xs bg-primary-200/80 text-primary-700 px-2.5 py-1 rounded-full font-semibold"
            >
              {{ globalUnreadCount }}
            </span>
          </button>

          <!-- Favorites -->
          <button
            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-white/50 transition-all duration-200 mb-1"
            :class="{ 'bg-primary-100/80 text-primary-700 shadow-sm': selectedCategory === 'favorites' }"
            @click="handleFavoritesClick"
          >
            <Icon icon="mdi:star" width="20" height="20" />
            <span v-if="!sidebarCollapsed" class="flex-1 text-left font-medium">
              收藏夹
            </span>
            <span
              v-if="!sidebarCollapsed && articlesStore.favoriteCount > 0"
              class="text-xs bg-amber-200/80 text-amber-700 px-2.5 py-1 rounded-full font-semibold"
            >
              {{ articlesStore.favoriteCount }}
            </span>
          </button>

          <!-- AI Summaries -->
          <button
            class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-white/50 transition-all duration-200 mb-1"
            :class="{ 'bg-linear-to-r from-purple-100/80 to-blue-100/80 text-purple-700 shadow-sm': selectedCategory === 'ai-summaries' }"
            @click="handleAISummariesClick"
          >
            <Icon icon="mdi:brain" width="20" height="20" class="text-purple-600" />
            <span v-if="!sidebarCollapsed" class="flex-1 text-left font-medium">
              AI 总结
            </span>
          </button>

          <div v-if="!sidebarCollapsed" class="my-2.5 border-t border-gray-200/50" />

          <!-- Categories -->
          <div v-if="!sidebarCollapsed" class="space-y-1">
            <div
              v-for="category in feedsStore.categories"
              :key="category.id"
              class="category-group"
            >
              <div
                class="flex items-center gap-1 px-3 py-2 rounded-xl hover:bg-white/50 transition-all duration-200 group"
                :class="{ 'bg-primary-100/80': selectedCategory === category.id }"
              >
                <button
                  class="flex-1 flex items-center gap-2 text-left"
                  :class="{ 'text-primary-700': selectedCategory === category.id }"
                  @click="handleCategoryClick(category.id)"
                >
                  <Icon :icon="category.icon" width="18" height="18" />
                  <span class="text-sm font-medium">
                    {{ category.name }}
                  </span>
                  <span class="text-xs opacity-50">
                    {{ category.feedCount }}
                  </span>
                </button>
                <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                  <button
                    class="p-1.5 rounded-lg hover:bg-white/60 transition-colors"
                    title="编辑分类"
                    @click.stop="handleEditCategory(category.id)"
                  >
                    <Icon icon="mdi:pencil" width="15" height="15" class="text-gray-500" />
                  </button>
                  <button
                    class="p-1.5 rounded-lg hover:bg-white/60 transition-colors"
                    title="删除分类"
                    @click.stop="handleDeleteCategory(category.id, category.name)"
                  >
                    <Icon icon="mdi:delete" width="15" height="15" class="text-gray-500" />
                  </button>
                </div>
              </div>

              <!-- Feeds under category -->
              <div
                v-if="selectedCategory === category.id"
                class="ml-4 mt-1 space-y-0.5"
              >
                <div
                  v-for="feed in feedsStore.getFeedsByCategory(category.id)"
                  :key="feed.id"
                  class="flex items-center gap-1 px-3 py-2 rounded-xl hover:bg-white/50 transition-all duration-200 text-sm group"
                  :class="{ 'bg-primary-50/80 text-primary-700': selectedFeed === feed.id }"
                >
                  <button
                    class="flex-1 flex items-center gap-2 text-left truncate"
                    @click="handleFeedClick(feed.id)"
                  >
                    <FeedIcon :icon="feed.icon" :feed-id="feed.id" :size="16" />
                    <span class="truncate">{{ feed.title }}</span>
                  </button>
                  <span
                    v-if="(feedsStore.unreadCountsByFeed[feed.id] || 0) > 0"
                    class="text-xs bg-primary-200/80 text-primary-700 px-2 py-0.5 rounded-full font-semibold"
                  >
                    {{ feedsStore.unreadCountsByFeed[feed.id] || 0 }}
                  </span>
                  <!-- Refresh status indicator -->
                  <Icon
                    v-if="feed.refreshStatus === 'refreshing'"
                    icon="mdi:loading"
                    width="14"
                    height="14"
                    class="animate-spin text-primary-500"
                  />
                  <Icon
                    v-else-if="feed.refreshStatus === 'error'"
                    icon="mdi:alert-circle"
                    width="14"
                    height="14"
                    class="text-red-500"
                    :title="feed.refreshError || '刷新失败'"
                  />
                  <Icon
                    v-else-if="feed.refreshStatus === 'success'"
                    icon="mdi:check-circle"
                    width="14"
                    height="14"
                    class="text-emerald-500"
                  />
                  <button
                    class="opacity-0 group-hover:opacity-100 p-1.5 rounded-lg hover:bg-white/60 transition-all"
                    title="编辑订阅源"
                    @click.stop="handleEditFeed(feed.id)"
                  >
                    <Icon icon="mdi:pencil" width="13" height="13" class="text-gray-500" />
                  </button>
                </div>
              </div>
            </div>

            <!-- Uncategorized Section -->
            <div
              v-if="!sidebarCollapsed && uncategorizedFeeds.length > 0"
              class="mt-2"
            >
              <div
                class="flex items-center gap-1 px-3 py-2 rounded-xl hover:bg-white/50 transition-all duration-200 group"
                :class="{ 'bg-primary-100/80': selectedCategory === 'uncategorized' }"
              >
                <button
                  class="flex-1 flex items-center gap-2 text-left"
                  :class="{ 'text-primary-700': selectedCategory === 'uncategorized' }"
                  @click="handleCategoryClick('uncategorized')"
                >
                  <Icon icon="mdi:folder-off" width="18" height="18" />
                  <span class="text-sm font-medium">
                    未分类
                  </span>
                  <span class="text-xs opacity-50">
                    {{ uncategorizedFeeds.length }}
                  </span>
                </button>
              </div>

              <!-- Uncategorized feeds -->
              <div
                v-if="selectedCategory === 'uncategorized'"
                class="ml-4 mt-1 space-y-0.5"
              >
                <div
                  v-for="feed in uncategorizedFeeds"
                  :key="feed.id"
                  class="flex items-center gap-1 px-3 py-2 rounded-xl hover:bg-white/50 transition-all duration-200 text-sm group"
                  :class="{ 'bg-primary-50/80 text-primary-700': selectedFeed === feed.id }"
                >
                  <button
                    class="flex-1 flex items-center gap-2 text-left truncate"
                    @click="handleFeedClick(feed.id)"
                  >
                    <FeedIcon :icon="feed.icon" :feed-id="feed.id" :size="16" />
                    <span class="truncate">{{ feed.title }}</span>
                  </button>
                  <span
                    v-if="(feedsStore.unreadCountsByFeed[feed.id] || 0) > 0"
                    class="text-xs bg-primary-200/80 text-primary-700 px-2 py-0.5 rounded-full font-semibold"
                  >
                    {{ feedsStore.unreadCountsByFeed[feed.id] || 0 }}
                  </span>
                  <!-- Refresh status indicator -->
                  <Icon
                    v-if="feed.refreshStatus === 'refreshing'"
                    icon="mdi:loading"
                    width="14"
                    height="14"
                    class="animate-spin text-primary-500"
                  />
                  <Icon
                    v-else-if="feed.refreshStatus === 'error'"
                    icon="mdi:alert-circle"
                    width="14"
                    height="14"
                    class="text-red-500"
                    :title="feed.refreshError || '刷新失败'"
                  />
                  <Icon
                    v-else-if="feed.refreshStatus === 'success'"
                    icon="mdi:check-circle"
                    width="14"
                    height="14"
                    class="text-emerald-500"
                  />
                  <button
                    class="opacity-0 group-hover:opacity-100 p-1.5 rounded-lg hover:bg-white/60 transition-all"
                    title="编辑订阅源"
                    @click.stop="handleEditFeed(feed.id)"
                  >
                    <Icon icon="mdi:pencil" width="13" height="13" class="text-gray-500" />
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Collapse mode - just show icons -->
          <div v-else class="space-y-1">
            <div
              v-for="category in feedsStore.categories"
              :key="category.id"
            >
              <button
                class="w-full flex items-center justify-center px-3 py-2.5 rounded-xl hover:bg-white/50 transition-all duration-200"
                :class="{ 'bg-primary-100/80 text-primary-700': selectedCategory === category.id }"
                :title="category.name"
                @click="handleCategoryClick(category.id)"
              >
                <Icon :icon="category.icon" width="20" height="20" />
              </button>
            </div>
          </div>
        </div>
      </aside>

      <!-- Article List -->
      <div v-if="!showAISummaries" class="w-80 glass-card border-r border-white/20 flex-shrink-0 flex flex-col m-2 mt-2 mb-2 ml-0 rounded-2xl overflow-hidden">
        <!-- Article List Header -->
        <div class="px-4 py-3 border-b border-white/20 flex items-center justify-between bg-white/40">
          <div class="flex items-center gap-2">
            <h2 class="font-semibold text-gray-800">
              {{ selectedFeed ? '订阅源文章' : selectedCategory === 'favorites' ? '收藏夹' : selectedCategory === 'uncategorized' ? '未分类' : selectedCategory ? '分类文章' : '全部文章' }}
            </h2>
            <span class="text-sm text-gray-500">
              {{ filteredArticles.length }}
            </span>
          </div>
        </div>

        <!-- Article List -->
        <div class="flex-1 overflow-y-auto glass-scroll">
          <ArticleCard
            v-for="article in filteredArticles.slice(0, 50)"
            :key="article.id"
            :article="article"
            compact
            @click="(art) => $router.push(`/article/${art.id}`)"
            @favorite="(id) => apiStore.toggleFavorite(id)"
          />
        </div>
      </div>

      <!-- AI Summaries List -->
      <SummaryAISummariesList
        v-else
        :category-id="null"
        @select="handleSummarySelect"
      />

      <!-- Article Content / Category News Summary -->
      <div v-if="!showAISummaries || !selectedSummary" class="flex-1 overflow-y-auto m-2 mt-2 mb-2 ml-0 rounded-2xl">
        <!-- Normal article content view -->
        <div class="h-full overflow-y-auto bg-white/30 backdrop-blur-sm rounded-2xl">
          <NuxtPage />
        </div>
      </div>

      <!-- AI Summary Detail -->
      <SummaryAISummaryDetail
        v-else
        :key="selectedSummary?.id || 'empty'"
        :summary="selectedSummary"
        @close="selectedSummary = null"
      />
    </div>

    <!-- Add Feed Dialog -->
    <CrudAddFeedDialog
      v-if="showAddFeedDialog"
      @close="showAddFeedDialog = false"
      @added="() => { apiStore.syncToLocalStores() }"
    />

    <!-- Add Category Dialog -->
    <CrudAddCategoryDialog
      v-if="showAddCategoryDialog"
      @close="showAddCategoryDialog = false"
      @added="() => { apiStore.syncToLocalStores() }"
    />

    <!-- Edit Category Dialog -->
    <CrudEditCategoryDialog
      v-if="editCategoryId && editingCategory"
      :category="editingCategory"
      @close="editCategoryId = null"
      @updated="() => { apiStore.syncToLocalStores() }"
    />

    <!-- Edit Feed Dialog -->
    <CrudEditFeedDialog
      v-if="editFeedId && editingFeed"
      :feed="editingFeed"
      @close="editFeedId = null"
      @updated="() => { apiStore.syncToLocalStores() }"
      @deleted="() => { apiStore.syncToLocalStores() }"
    />

    <!-- Import OPML Dialog -->
    <ImportOpmlDialog
      v-if="showImportDialog"
      @close="showImportDialog = false"
      @imported="() => { apiStore.syncToLocalStores() }"
    />

    <!-- Global Settings Dialog -->
    <GlobalSettingsDialog :show="showGlobalSettings" @update:show="showGlobalSettings = $event" />
  </div>
</template>
