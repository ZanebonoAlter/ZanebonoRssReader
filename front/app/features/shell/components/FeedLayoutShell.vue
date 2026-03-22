<script setup lang="ts">
import { Icon } from '@iconify/vue'
import LayoutAppHeader from '~/features/shell/components/AppHeaderShell.vue'
import LayoutAppSidebar from '~/features/shell/components/AppSidebarShell.vue'
import LayoutArticleListPanel from '~/features/shell/components/ArticleListPanelShell.vue'
import { useArticlesApi } from '~/api/articles'
import ArticleContent from '~/features/articles/components/ArticleContentView.vue'
import { normalizeArticle } from '../../articles/utils/normalizeArticle'
import { useGlobalAutoRefresh } from '~/features/feeds/composables/useAutoRefresh'
import AISummariesList from '~/features/summaries/components/AISummariesListView.vue'
import AISummaryDetail from '~/features/summaries/components/AISummaryDetailView.vue'
import { SIDEBAR_DEFAULT_WIDTH, MAX_POLLING_TIME, REFRESH_POLLING_INTERVAL } from '~/utils/constants'

const apiStore = useApiStore()
const feedsStore = useFeedsStore()
const articlesStore = useArticlesStore()
const articlesApi = useArticlesApi()

// 初始化全局自动刷新
useGlobalAutoRefresh()

// 侧边栏状态
const sidebarCollapsed = ref(false)
const sidebarWidth = ref(SIDEBAR_DEFAULT_WIDTH)
const selectedCategory = ref<string | null>(null)
const selectedFeed = ref<string | null>(null)
const selectedArticle = ref<any>(null)
const showAISummaries = ref(false)
const selectedSummary = ref<any>(null)

// 对话框状态
const showAddFeedDialog = ref(false)
const showAddCategoryDialog = ref(false)
const showImportDialog = ref(false)
const showGlobalSettings = ref(false)

// 编辑对话框状态
const editCategoryId = ref<string | null>(null)
const editFeedId = ref<string | null>(null)

// 刷新提示
const refreshMessage = ref('')
const refreshMessageType = ref<'success' | 'error' | 'info'>('info')
const showRefreshMessage = ref(false)

// 全局未读数
const globalUnreadCount = ref(0)

// 计算属性
const editingCategory = computed(() =>
  editCategoryId.value ? feedsStore.categories.find(c => c.id === editCategoryId.value) : null
)
const editingFeed = computed(() =>
  editFeedId.value ? feedsStore.feeds.find(f => f.id === editFeedId.value) : null
)

const filteredArticles = computed(() => {
  let articles = articlesStore.articles

  // 根据选中的订阅源筛选
  if (selectedFeed.value) {
    articles = articles.filter(a => a.feedId === selectedFeed.value)
  }
  // 根据收藏筛选
  else if (selectedCategory.value === 'favorites') {
    articles = articles.filter(a => a.favorite)
  }

  return articles
})

// 获取全局未读数
async function fetchGlobalUnreadCount() {
  const response = await apiStore.fetchArticlesStats()
  if (response.success && response.data) {
    globalUnreadCount.value = (response.data as any).unread || 0
  }
}

// 初始化
onMounted(async () => {
  await fetchGlobalUnreadCount()
})

// 清理轮询
onUnmounted(() => {
  stopPollingRefreshStatus()
})

async function hydrateSelectedArticle(article: any) {
  selectedArticle.value = article

  try {
    const response = await articlesApi.getArticle(Number(article.id))
    if (response.success && response.data && selectedArticle.value?.id === article.id) {
      selectedArticle.value = normalizeArticle(response.data)
    }
  } catch (error) {
    console.error('Failed to load article detail:', error)
  }
}

// 处理文章点击
function handleArticleClick(article: any) {
  void hydrateSelectedArticle(article)
  // 标记为已读
  if (!article.read) {
    apiStore.markAsRead(article.id)
  }
}

// 处理文章收藏
function handleArticleFavorite(articleId: string) {
  apiStore.toggleFavorite(articleId)
}

// 处理分类点击
const handleCategoryClick = async (categoryId: string) => {
  selectedCategory.value = categoryId
  selectedFeed.value = null
  showAISummaries.value = false
  selectedSummary.value = null

  if (categoryId === 'uncategorized') {
    await apiStore.fetchFeeds({ uncategorized: true, per_page: 10000 })
    await apiStore.fetchArticles({ uncategorized: true, per_page: 10000 })
  } else {
    await apiStore.fetchFeeds({
      category_id: parseInt(categoryId),
      per_page: 10000
    })
    await apiStore.fetchArticles({
      category_id: parseInt(categoryId),
      per_page: 10000
    })
  }
  await fetchGlobalUnreadCount()
}

// 处理订阅源点击
async function handleFeedClick(feedId: string) {
  selectedFeed.value = feedId
  showAISummaries.value = false
  selectedSummary.value = null

  // 获取该订阅源的文章
  await apiStore.fetchArticles({
    feed_id: parseInt(feedId),
    per_page: 10000
  })

  // 自动刷新该订阅源
  const response = await apiStore.refreshFeed(feedId)

  if (response.success) {
    refreshMessage.value = '正在刷新订阅源...'
    refreshMessageType.value = 'info'
    showRefreshMessage.value = true

    // 更新订阅源状态
    const feed = feedsStore.feeds.find(f => f.id === feedId)
    if (feed) {
      feed.refreshStatus = 'refreshing'
    }

    // 开始轮询刷新状态
    pollRefreshStatus()
  } else {
      refreshMessage.value = response.error || '刷新失败'
    refreshMessageType.value = 'error'
    showRefreshMessage.value = true
  }
}

// 处理收藏夹点击
async function handleFavoritesClick() {
  selectedCategory.value = 'favorites'
  selectedFeed.value = null
  showAISummaries.value = false
  selectedSummary.value = null

  await apiStore.fetchArticles({
    per_page: 10000,
    favorite: true
  })
}

// 处理全部文章点击
async function handleAllArticlesClick() {
  selectedCategory.value = null
  selectedFeed.value = null
  showAISummaries.value = false
  selectedSummary.value = null

  await apiStore.fetchFeeds({ per_page: 10000 })
  await apiStore.fetchArticles({ per_page: 10000 })
}

// 处理 AI 总结点击
function handleAISummariesClick() {
  selectedCategory.value = 'ai-summaries'
  selectedFeed.value = null
  showAISummaries.value = true
  selectedSummary.value = null
}

function handleDigestClick() {
  selectedCategory.value = 'digest'
  selectedFeed.value = null
  showAISummaries.value = false
  selectedSummary.value = null
  navigateTo('/digest')
}

function handleTopicGraphClick() {
  selectedCategory.value = 'topic-graph'
  selectedFeed.value = null
  showAISummaries.value = false
  selectedSummary.value = null
  navigateTo('/topics')
}

// 处理总结选择
function handleSummarySelect(summary: any) {
  selectedSummary.value = summary
}

// 轮询刷新状态
const refreshPollingInterval = ref<ReturnType<typeof setTimeout> | null>(null)

async function pollRefreshStatus() {
  const startTime = Date.now()

  const poll = async () => {
    // 超过最大轮询时间则停止
    if (Date.now() - startTime > MAX_POLLING_TIME) {
      stopPollingRefreshStatus()
      return
    }

    // 获取最新的订阅源状态
    if (selectedFeed.value) {
      await apiStore.fetchFeeds({ per_page: 10000 })
    } else if (selectedCategory.value === 'uncategorized') {
      await apiStore.fetchFeeds({ uncategorized: true, per_page: 10000 })
    } else if (selectedCategory.value && selectedCategory.value !== 'favorites' && selectedCategory.value !== 'ai-summaries') {
      await apiStore.fetchFeeds({
        category_id: parseInt(selectedCategory.value),
        per_page: 10000
      })
    } else {
      await apiStore.fetchFeeds({ per_page: 10000 })
    }

    // 检查是否还有订阅源仍在刷新
    const monitoredFeeds = selectedFeed.value
      ? feedsStore.feeds.filter(f => f.id === selectedFeed.value)
      : feedsStore.feeds

    const stillRefreshing = monitoredFeeds.some(f => f.refreshStatus === 'refreshing')

    if (stillRefreshing) {
      refreshPollingInterval.value = setTimeout(poll, REFRESH_POLLING_INTERVAL)
    } else {
      stopPollingRefreshStatus()

      // 刷新完成后重新获取文章
      const articleParams: any = { per_page: 10000 }
      if (selectedFeed.value) {
        articleParams.feed_id = parseInt(selectedFeed.value)
      } else if (selectedCategory.value === 'uncategorized') {
        articleParams.uncategorized = true
      } else if (selectedCategory.value && selectedCategory.value !== 'favorites' && selectedCategory.value !== 'ai-summaries') {
        articleParams.category_id = parseInt(selectedCategory.value)
      }

      await apiStore.fetchArticles(articleParams)
      await fetchGlobalUnreadCount()

      // 显示成功消息
      refreshMessage.value = '刷新完成'
      refreshMessageType.value = 'success'
      showRefreshMessage.value = true
      setTimeout(() => {
        showRefreshMessage.value = false
      }, 3000)
    }
  }

  poll()
}

function stopPollingRefreshStatus() {
  if (refreshPollingInterval.value) {
    clearTimeout(refreshPollingInterval.value)
    refreshPollingInterval.value = null
  }
}

// 刷新
async function handleRefresh() {
  stopPollingRefreshStatus()

  if (selectedFeed.value) {
    const response = await apiStore.refreshFeed(selectedFeed.value)
    if (response.success) {
      refreshMessage.value = response.message || '已开始后台刷新当前订阅源'
      refreshMessageType.value = 'info'
      await apiStore.fetchFeeds({ per_page: 10000 })
      pollRefreshStatus()
    } else {
      refreshMessage.value = response.error || '刷新失败'
      refreshMessageType.value = 'error'
    }
  } else {
    const response = await apiStore.refreshAllFeeds()
    if (response.success) {
      refreshMessage.value = response.message || '已开始后台刷新全部订阅源'
      refreshMessageType.value = 'info'
      await apiStore.fetchFeeds({ per_page: 10000 })
      pollRefreshStatus()
    } else {
      refreshMessage.value = response.error || '刷新失败'
      refreshMessageType.value = 'error'
    }
  }

  showRefreshMessage.value = true
}

// 全部标为已读
async function handleMarkAllRead() {
  if (selectedFeed.value) {
    await apiStore.markAllAsRead(selectedFeed.value)
  } else if (selectedCategory.value) {
    const feedIds = feedsStore.feeds
      .filter(f => f.category === selectedCategory.value)
      .map(f => f.id)
    for (const feedId of feedIds) {
      await apiStore.markAllAsRead(feedId)
    }
  } else {
    await apiStore.markAllAsRead()
  }
  await fetchGlobalUnreadCount()
}

// 导出 OPML
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
    console.error('导出失败:', error)
  }
}

// 切换侧边栏
function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

// 编辑分类
function handleEditCategory(categoryId: string) {
  editCategoryId.value = categoryId
}

// 编辑订阅源
function handleEditFeed(feedId: string) {
  editFeedId.value = feedId
}

// 删除分类
async function handleDeleteCategory(categoryId: string, categoryName: string) {
  if (confirm(`确定要删除分类 "${categoryName}" 吗？这个操作不会删除分类下的订阅源。`)) {
    const response = await apiStore.deleteCategory(categoryId)
    if (response.success) {
    } else {
      alert(response.error || '删除失败')
    }
  }
}

import '~/components/FeedLayout.css'
</script>

<template>
  <div class="feed-layout">
    <!-- 顶部工具栏 -->
    <LayoutAppHeader
      :show-refresh-message="showRefreshMessage"
      :refresh-message="refreshMessage"
      :refresh-message-type="refreshMessageType"
      @toggle-sidebar="toggleSidebar"
      @refresh="handleRefresh"
      @mark-all-read="handleMarkAllRead"
      @add-feed="showAddFeedDialog = true"
      @add-category="showAddCategoryDialog = true"
      @import-opml="showImportDialog = true"
      @settings="showGlobalSettings = true"
      @close-refresh-message="showRefreshMessage = false"
    />

      <!-- 主内容区 -->
    <div class="main-content">
      <!-- 侧边栏 -->
      <LayoutAppSidebar
        :sidebar-collapsed="sidebarCollapsed"
        :sidebar-width="sidebarWidth"
        :selected-category="selectedCategory"
        :selected-feed="selectedFeed"
        :global-unread-count="globalUnreadCount"
        @toggle-sidebar="toggleSidebar"
        @category-click="handleCategoryClick"
        @feed-click="handleFeedClick"
        @favorites-click="handleFavoritesClick"
        @ai-summaries-click="handleAISummariesClick"
        @digest-click="handleDigestClick"
        @topic-graph-click="handleTopicGraphClick"
        @all-articles-click="handleAllArticlesClick"
        @edit-category="handleEditCategory"
        @edit-feed="handleEditFeed"
        @delete-category="handleDeleteCategory"
      />

      <!-- 文章列表 -->
      <LayoutArticleListPanel
        v-if="!showAISummaries"
        :articles="filteredArticles"
        :selected-category="selectedCategory"
        :selected-feed="selectedFeed"
        :selected-article="selectedArticle"
        @article-click="handleArticleClick"
        @article-favorite="handleArticleFavorite"
      />

      <!-- AI 总结列表 -->
      <AISummariesList
        v-else
        :category-id="null"
        @select="handleSummarySelect"
      />

      <!-- 文章内容 / AI 总结详情 -->
      <div v-if="!showAISummaries && !selectedSummary" class="content-panel">
        <ArticleContent
          :article="selectedArticle"
          :articles="filteredArticles"
          @favorite="handleArticleFavorite"
          @navigate="handleArticleClick"
        />
      </div>

      <!-- AI 总结详情 -->
      <div v-else class="content-panel">
        <AISummaryDetail
          :key="selectedSummary?.id || 'empty'"
          :summary="selectedSummary"
          @close="selectedSummary = null"
        />
      </div>
    </div>

    <!-- 添加订阅源对话框 -->
    <DialogAddFeedDialog
      v-if="showAddFeedDialog"
      @close="showAddFeedDialog = false"
      @added="() => {}"
    />

    <!-- 添加分类对话框 -->
    <DialogAddCategoryDialog
      v-if="showAddCategoryDialog"
      @close="showAddCategoryDialog = false"
      @added="() => {}"
    />

    <!-- 编辑分类对话框 -->
    <DialogEditCategoryDialog
      v-if="editCategoryId && editingCategory"
      :category="editingCategory"
      @close="editCategoryId = null"
      @updated="() => {}"
    />

    <!-- 编辑订阅源对话框 -->
    <DialogEditFeedDialog
      v-if="editFeedId && editingFeed"
      :feed="editingFeed"
      @close="editFeedId = null"
      @updated="() => {}"
      @deleted="() => {}"
    />

    <!-- 导入 OPML 对话框 -->
    <DialogImportOpmlDialog
      v-if="showImportDialog"
      @close="showImportDialog = false"
      @imported="() => {}"
    />

    <!-- 全局设置对话框 -->
    <DialogGlobalSettingsDialog :show="showGlobalSettings" @update:show="showGlobalSettings = $event" />
  </div>
</template>




