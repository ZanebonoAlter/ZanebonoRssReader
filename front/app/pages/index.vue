<script setup lang="ts">
import { Icon } from '@iconify/vue'
const feedsStore = useFeedsStore()
const articlesStore = useArticlesStore()

const latestArticles = computed(() => {
  return articlesStore.articles.slice(0, 10)
})

function handleArticleClick(article: any) {
  navigateTo(`/article/${article.id}`)
}

useHead({
  title: 'RSS Reader - 首页',
  meta: [
    { name: 'description', content: '简洁优雅的 RSS 订阅阅读器' }
  ]
})
</script>

<template>
  <div class="h-full flex flex-col bg-gray-50">
    <!-- Welcome Section -->
    <div class="flex-1 overflow-y-auto">
      <div class="max-w-4xl mx-auto px-6 py-8">
        <!-- Hero -->
        <div class="text-center mb-12">
          <div class="w-16 h-16 rounded-xl bg-gradient-to-br from-blue-500 to-purple-600 flex items-center justify-center mx-auto mb-4">
            <Icon icon="mdi:rss" class="text-white" width="32" height="32" />
          </div>
          <h1 class="text-3xl font-bold text-gray-900 mb-2">
            欢迎使用 RSS Reader
          </h1>
          <p class="text-gray-600">
            选择左侧的订阅源或分类开始阅读
          </p>
        </div>

        <!-- Stats -->
        <div class="grid grid-cols-2 md:grid-cols-4 gap-4 mb-12">
          <div class="bg-white rounded-xl p-6 text-center shadow-sm">
            <div class="text-3xl font-bold text-blue-600 mb-2">
              {{ feedsStore.feedCount }}
            </div>
            <div class="text-sm text-gray-500">订阅源</div>
          </div>
          <div class="bg-white rounded-xl p-6 text-center shadow-sm">
            <div class="text-3xl font-bold text-purple-600 mb-2">
              {{ feedsStore.categories.length }}
            </div>
            <div class="text-sm text-gray-500">分类</div>
          </div>
          <div class="bg-white rounded-xl p-6 text-center shadow-sm">
            <div class="text-3xl font-bold text-green-600 mb-2">
              {{ articlesStore.articles.length }}
            </div>
            <div class="text-sm text-gray-500">文章</div>
          </div>
          <div class="bg-white rounded-xl p-6 text-center shadow-sm">
            <div class="text-3xl font-bold text-yellow-600 mb-2">
              {{ articlesStore.unreadCount }}
            </div>
            <div class="text-sm text-gray-500">未读</div>
          </div>
        </div>

        <!-- Latest Articles -->
        <div v-if="latestArticles.length > 0">
          <div class="flex items-center justify-between mb-6">
            <h2 class="text-xl font-bold text-gray-900">最新文章</h2>
            <span class="text-sm text-gray-500">共 {{ latestArticles.length }} 篇</span>
          </div>
          <div class="space-y-3">
            <div
              v-for="article in latestArticles"
              :key="article.id"
              class="bg-white rounded-lg p-4 shadow-sm hover:shadow-md transition-shadow cursor-pointer"
              @click="handleArticleClick(article)"
            >
              <div class="flex items-start gap-3">
                <div
                  class="w-10 h-10 rounded-lg flex items-center justify-center flex-shrink-0"
                  :style="{ backgroundColor: feedsStore.feeds.find(f => f.id === article.feedId)?.color + '15' }"
                >
                  <FeedIcon
                    :icon="feedsStore.feeds.find(f => f.id === article.feedId)?.icon"
                    :color="feedsStore.feeds.find(f => f.id === article.feedId)?.color"
                    :size="20"
                  />
                </div>
                <div class="flex-1 min-w-0">
                  <h3 class="font-medium text-gray-900 mb-1 line-clamp-2">
                    {{ article.title }}
                  </h3>
                  <p class="text-sm text-gray-500 line-clamp-1 mb-2">
                    {{ article.description }}
                  </p>
                  <div class="flex items-center gap-3 text-xs text-gray-400">
                    <span>{{ $dayjs(article.pubDate).format('MM-DD HH:mm') }}</span>
                    <span v-if="article.favorite" class="text-yellow-500">
                      <Icon icon="mdi:star" width="14" height="14" />
                    </span>
                    <span v-if="article.read" class="text-green-600">已读</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="text-center py-16">
          <Icon icon="mdi:rss-off" width="64" height="64" class="text-gray-300 mb-4 mx-auto" />
          <h3 class="text-xl font-semibold text-gray-700 mb-2">
            暂无文章
          </h3>
          <p class="text-gray-500">
            添加一些 RSS 订阅源开始阅读吧
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.line-clamp-1 {
  overflow: hidden;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 1;
}

.line-clamp-2 {
  overflow: hidden;
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}
</style>
