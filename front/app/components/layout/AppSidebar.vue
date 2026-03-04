<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { useRefreshPolling } from '~/composables/useRefreshPolling'
import { SIDEBAR_MIN_WIDTH, SIDEBAR_MAX_WIDTH } from '~/utils/constants'
import AppTooltip from '~/components/common/AppTooltip.vue'
import FeedActionMenu from '~/components/feed/FeedActionMenu.vue'

interface Props {
  sidebarCollapsed?: boolean
  sidebarWidth?: number
  selectedCategory?: string | null
  selectedFeed?: string | null
  globalUnreadCount?: number
}

const props = withDefaults(defineProps<Props>(), {
  sidebarCollapsed: false,
  sidebarWidth: 256,
  selectedCategory: null,
  selectedFeed: null,
  globalUnreadCount: 0,
})

const emit = defineEmits<{
  toggleSidebar: []
  'update:sidebarWidth': [value: number]
  categoryClick: [categoryId: string]
  feedClick: [feedId: string]
  favoritesClick: []
  aiSummariesClick: []
  digestClick: []
  allArticlesClick: []
  editCategory: [categoryId: string]
  editFeed: [feedId: string]
  deleteCategory: [categoryId: string, categoryName: string]
  markFeedAsRead: [feedId: string]
  startResizing: [event: MouseEvent]
  stopResizing: []
}>()

const apiStore = useApiStore()
const feedsStore = useFeedsStore()
const articlesStore = useArticlesStore()

const isResizing = ref(false)

const { updateSelection } = useRefreshPolling()

// 未分类的订阅源
const uncategorizedFeeds = computed(() => {
  return feedsStore.feeds.filter((f) => !f.category)
})

// 开始调整侧边栏宽度
function startResizing(event: MouseEvent) {
  isResizing.value = true
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
  document.addEventListener('mousemove', onResizing)
  document.addEventListener('mouseup', stopResizing)
}

// 调整侧边栏宽度
function onResizing(event: MouseEvent) {
  if (!isResizing.value) return

  const newWidth = event.clientX
  if (newWidth >= SIDEBAR_MIN_WIDTH && newWidth <= SIDEBAR_MAX_WIDTH) {
    emit('update:sidebarWidth', newWidth)
  }
}

// 停止调整侧边栏宽度
function stopResizing() {
  isResizing.value = false
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  document.removeEventListener('mousemove', onResizing)
  document.removeEventListener('mouseup', stopResizing)
}

// 处理分类点击
function handleCategoryClick(categoryId: string) {
  updateSelection(categoryId, null)
  emit('categoryClick', categoryId)
}

// 处理订阅源点击
function handleFeedClick(feedId: string) {
  updateSelection(props.selectedCategory, feedId)
  emit('feedClick', feedId)
}

// 处理收藏夹点击
function handleFavoritesClick() {
  updateSelection('favorites', null)
  emit('favoritesClick')
}

// 处理 AI 摘要点击
function handleAISummariesClick() {
  updateSelection('ai-summaries', null)
  emit('aiSummariesClick')
}

// 处理日报周报点击
function handleDigestClick() {
  updateSelection('digest', null)
  emit('digestClick')
}

// 处理全部文章点击
function handleAllArticlesClick() {
  updateSelection(null, null)
  emit('allArticlesClick')
}

// 处理标记订阅源为已读
async function handleMarkFeedAsRead(feedId: string) {
  const response = await apiStore.markAllAsRead(feedId)
  if (response.success) {
    const feed = feedsStore.feeds.find(f => f.id === feedId)
    if (feed && feed.unreadCount) {
      feed.unreadCount = 0
    }
  }
}

import './AppSidebar.css'
</script>

<template>
  <aside
    class="app-sidebar"
    :style="sidebarCollapsed ? 'width: 48px' : `width: ${sidebarWidth}px`"
  >
    <!-- 调整手柄（仅当未折叠时显示） -->
    <div
      v-if="!sidebarCollapsed"
      class="resize-handle"
      :class="{ active: isResizing }"
      @mousedown="startResizing"
    />

    <div class="sidebar-content">
      <!-- 全部文章 -->
      <button
        class="sidebar-item"
        :class="{ active: !selectedCategory && !selectedFeed }"
        @click="handleAllArticlesClick"
      >
        <Icon icon="mdi:inbox" width="20" height="20" />
        <span v-if="!sidebarCollapsed" class="flex-1 text-left font-medium">
          全部文章
        </span>
        <span
          v-if="!sidebarCollapsed && globalUnreadCount > 0"
          class="badge"
        >
          {{ globalUnreadCount }}
        </span>
      </button>

      <!-- 收藏夹 -->
      <button
        class="sidebar-item"
        :class="{ active: selectedCategory === 'favorites' }"
        @click="handleFavoritesClick"
      >
        <Icon icon="mdi:star" width="20" height="20" />
        <span v-if="!sidebarCollapsed" class="flex-1 text-left font-medium">
          收藏夹
        </span>
        <span
          v-if="!sidebarCollapsed && articlesStore.favoriteCount > 0"
          class="badge badge-amber"
        >
          {{ articlesStore.favoriteCount }}
        </span>
      </button>

      <!-- AI 总结 -->
      <button
        class="sidebar-item"
        :class="{ active: selectedCategory === 'ai-summaries' }"
        @click="handleAISummariesClick"
      >
        <Icon icon="mdi:brain" width="20" height="20" class="text-ink-600" />
        <span v-if="!sidebarCollapsed" class="flex-1 text-left font-medium">
          AI 总结
        </span>
      </button>

      <!-- 日报周报 -->
      <button
        class="sidebar-item"
        :class="{ active: selectedCategory === 'digest' }"
        @click="handleDigestClick"
      >
        <Icon icon="mdi:newspaper-variant-multiple" width="20" height="20" class="text-ink-600" />
        <span v-if="!sidebarCollapsed" class="flex-1 text-left font-medium">
          日报周报
        </span>
      </button>

      <div v-if="!sidebarCollapsed" class="divider" />

      <!-- 分类列表 -->
      <div v-if="!sidebarCollapsed" class="categories">
        <div
          v-for="category in feedsStore.categories"
          :key="category.id"
          class="category-group"
        >
          <div
            class="category-item"
            :class="{ active: selectedCategory === category.id }"
          >
            <button
              class="category-btn"
              :class="{ 'text-ink-700': selectedCategory === category.id }"
              @click="handleCategoryClick(category.id)"
            >
              <Icon :icon="category.icon" width="18" height="18" />
              <span class="text-sm font-medium">{{ category.name }}</span>
              <span class="count">{{ category.feedCount }}</span>
            </button>
            <div class="category-actions">
              <button
                class="action-btn"
                title="编辑分类"
                @click.stop="$emit('editCategory', category.id)"
              >
                <Icon icon="mdi:pencil" width="15" height="15" class="text-gray-500" />
              </button>
              <button
                class="action-btn"
                title="删除分类"
                @click.stop="$emit('deleteCategory', category.id, category.name)"
              >
                <Icon icon="mdi:delete" width="15" height="15" class="text-gray-500" />
              </button>
            </div>
          </div>

           <!-- 分类下的订阅源 -->
           <div
             v-if="selectedCategory === category.id"
             class="feeds-list"
           >
             <div
               v-for="feed in feedsStore.getFeedsByCategory(category.id)"
               :key="feed.id"
               class="feed-item"
               :class="{ active: selectedFeed === feed.id }"
             >
               <AppTooltip
                 :content="`${feedsStore.unreadCountsByFeed[feed.id] || 0} 篇未读文章`"
                 :disabled="!(feedsStore.unreadCountsByFeed[feed.id] || 0) > 0"
               >
                 <span
                   v-if="(feedsStore.unreadCountsByFeed[feed.id] || 0) > 0"
                   class="badge badge-sm"
                 >
                   {{ feedsStore.unreadCountsByFeed[feed.id] || 0 }}
                 </span>
               </AppTooltip>
               <div class="feed-status-wrapper">
                 <AppTooltip :content="feed.refreshStatus === 'error' ? (feed.refreshError || '刷新失败') : (feed.refreshStatus === 'success' ? '刷新成功' : '正在刷新')">
                   <FeedRefreshStatusIcon :feed="feed" />
                 </AppTooltip>
               </div>
               <button
                 class="feed-btn"
                 @click="handleFeedClick(feed.id)"
               >
                 <FeedIcon
                   :icon="feed.icon"
                   :feed-id="feed.id"
                   :size="16"
                 />
                 <AppTooltip :content="feed.title">
                   <span class="truncate">{{ feed.title }}</span>
                 </AppTooltip>
               </button>
               <div class="feed-action-wrapper">
                 <FeedActionMenu
                   :feed-id="feed.id"
                   :feed-title="feed.title"
                   @mark-as-read="handleMarkFeedAsRead"
                   @edit="$emit('editFeed', $event)"
                 />
               </div>
             </div>
           </div>
        </div>

        <!-- 未分类部分 -->
        <div
          v-if="!sidebarCollapsed && uncategorizedFeeds.length > 0"
          class="uncategorized"
        >
          <div
            class="category-item"
            :class="{ active: selectedCategory === 'uncategorized' }"
          >
            <button
              class="category-btn"
              :class="{ 'text-ink-700': selectedCategory === 'uncategorized' }"
              @click="handleCategoryClick('uncategorized')"
            >
              <Icon icon="mdi:folder-off" width="18" height="18" />
              <span class="text-sm font-medium">未分类</span>
              <span class="count">{{ uncategorizedFeeds.length }}</span>
            </button>
          </div>

           <!-- 未分类的订阅源 -->
           <div
             v-if="selectedCategory === 'uncategorized'"
             class="feeds-list"
           >
             <div
               v-for="feed in uncategorizedFeeds"
               :key="feed.id"
               class="feed-item"
               :class="{ active: selectedFeed === feed.id }"
             >
               <AppTooltip
                 :content="`${feedsStore.unreadCountsByFeed[feed.id] || 0} 篇未读文章`"
                 :disabled="!(feedsStore.unreadCountsByFeed[feed.id] || 0) > 0"
               >
                 <span
                   v-if="(feedsStore.unreadCountsByFeed[feed.id] || 0) > 0"
                   class="badge badge-sm"
                 >
                   {{ feedsStore.unreadCountsByFeed[feed.id] || 0 }}
                 </span>
               </AppTooltip>
               <div class="feed-status-wrapper">
                 <AppTooltip :content="feed.refreshStatus === 'error' ? (feed.refreshError || '刷新失败') : (feed.refreshStatus === 'success' ? '刷新成功' : '正在刷新')">
                   <FeedRefreshStatusIcon :feed="feed" />
                 </AppTooltip>
               </div>
               <button
                 class="feed-btn"
                 @click="handleFeedClick(feed.id)"
               >
                 <FeedIcon
                   :icon="feed.icon"
                   :feed-id="feed.id"
                   :size="16"
                 />
                 <AppTooltip :content="feed.title">
                   <span class="truncate">{{ feed.title }}</span>
                 </AppTooltip>
               </button>
               <div class="feed-action-wrapper">
                 <FeedActionMenu
                   :feed-id="feed.id"
                   :feed-title="feed.title"
                   @mark-as-read="handleMarkFeedAsRead"
                   @edit="$emit('editFeed', $event)"
                 />
               </div>
             </div>
           </div>
        </div>
      </div>

      <!-- 折叠模式 - 只显示图标 -->
      <div v-else class="collapsed-view">
        <div
          v-for="category in feedsStore.categories"
          :key="category.id"
        >
          <button
            class="sidebar-item collapsed-item"
            :class="{ active: selectedCategory === category.id }"
            :title="category.name"
            @click="handleCategoryClick(category.id)"
          >
            <Icon :icon="category.icon" width="20" height="20" />
          </button>
        </div>
      </div>
    </div>
  </aside>
</template>
