import { api } from '~/composables/useApi'
import type { Category, RssFeed, Article } from '~/types'

export const useApiStore = defineStore('api', () => {
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Categories
  const categories = ref<Category[]>([])

  async function fetchCategories() {
    loading.value = true
    error.value = null

    const response = await api.getCategories()

    if (response.success && response.data) {
      // Transform backend data to frontend format
      categories.value = (response.data as any[]).map((cat: any) => ({
        id: String(cat.id),
        name: cat.name,
        slug: cat.slug || cat.name.toLowerCase().replace(/\s+/g, '-'),
        icon: cat.icon || 'mdi:folder',
        color: cat.color || '#6b7280',
        description: cat.description || '',
        feedCount: cat.feed_count || 0,
      }))
    } else {
      error.value = response.error || 'Failed to fetch categories'
    }

    loading.value = false
    return response
  }

  async function createCategory(data: {
    name: string
    icon?: string
    color?: string
    description?: string
  }) {
    loading.value = true
    const response = await api.createCategory(data)
    loading.value = false

    if (response.success) {
      await fetchCategories()
      syncToLocalStores()
    }

    return response
  }

  async function updateCategory(
    id: string,
    data: {
      name?: string
      icon?: string
      color?: string
      description?: string
    }
  ) {
    loading.value = true
    const response = await api.updateCategory(Number(id), data)
    loading.value = false

    if (response.success) {
      await fetchCategories()
      syncToLocalStores()
    }

    return response
  }

  async function deleteCategory(id: string) {
    loading.value = true
    const response = await api.deleteCategory(Number(id))
    loading.value = false

    if (response.success) {
      await fetchCategories()
      syncToLocalStores()
    }

    return response
  }

  // Feeds
  const feeds = ref<RssFeed[]>([])
  const allFeeds = ref<RssFeed[]>([]) // Cache all feeds for sidebar display

  async function fetchFeeds(params: { page?: number; per_page?: number; category_id?: number; uncategorized?: boolean } = {}) {
    loading.value = true
    error.value = null

    const response = await api.getFeeds(params)

    if (response.success && response.data) {
      const data = response.data as any
      const items = data.items || data

      const mappedFeeds = items.map((feed: any) => ({
        id: String(feed.id),
        title: feed.title,
        description: feed.description || '',
        url: feed.url,
        category: feed.category_id ? String(feed.category_id) : '',
        icon: feed.icon || undefined, // Don't set default icon, let FeedIcon component handle fallback
        color: feed.color || '#6b7280',
        lastUpdated: feed.last_updated || new Date().toISOString(),
        articleCount: feed.article_count || 0,
        unreadCount: feed.unread_count || 0,
        maxArticles: feed.max_articles || 100,
        refreshInterval: feed.refresh_interval || 60,
        refreshStatus: feed.refresh_status || 'idle',
        refreshError: feed.refresh_error,
        lastRefreshAt: feed.last_refresh_at,
        aiSummaryEnabled: feed.ai_summary_enabled !== undefined ? feed.ai_summary_enabled : true, // Default to true if not set
      }))

      feeds.value = mappedFeeds

      // If fetching without filters (except per_page), cache all feeds
      const hasFilters = params.category_id !== undefined || params.uncategorized === true
      if (!hasFilters) {
        allFeeds.value = mappedFeeds
      }
    } else {
      error.value = response.error || 'Failed to fetch feeds'
    }

    loading.value = false
    return response
  }

  async function createFeed(data: {
    url: string
    category_id?: number
    title?: string
    description?: string
    icon?: string
    color?: string
  }) {
    loading.value = true
    const response = await api.createFeed(data)
    loading.value = false

    if (response.success) {
      await fetchFeeds({ per_page: 10000 })
      await fetchArticles({ per_page: 10000 })
      syncToLocalStores()
    }

    return response
  }

  async function deleteFeed(id: string) {
    loading.value = true
    const response = await api.deleteFeed(Number(id))
    loading.value = false

    if (response.success) {
      await fetchFeeds({ per_page: 10000 })
      await fetchArticles({ per_page: 10000 })
      syncToLocalStores()
    }

    return response
  }

  async function updateFeed(
    id: string,
    data: {
      url?: string
      title?: string
      description?: string
      category_id?: number
      icon?: string
      color?: string
    }
  ) {
    loading.value = true
    const response = await api.updateFeed(Number(id), data)
    loading.value = false

    if (response.success) {
      await fetchFeeds({ per_page: 10000 })
      await fetchArticles({ per_page: 10000 })
      syncToLocalStores()
    }

    return response
  }

  async function refreshFeed(id: string) {
    loading.value = true
    const response = await api.refreshFeed(Number(id))
    loading.value = false
    return response
  }

  async function refreshAllFeeds() {
    loading.value = true
    const response = await api.refreshAllFeeds()
    loading.value = false
    return response
  }

  // Articles
  const articles = ref<Article[]>([])
  const totalArticles = ref(0)

  async function fetchArticles(filters: {
    page?: number
    per_page?: number
    feed_id?: number
    category_id?: number
    uncategorized?: boolean
    read?: boolean
    favorite?: boolean
    search?: string
  } = {}) {
    loading.value = true
    error.value = null

    // Set a high default per_page to get all articles
    const params = { per_page: 10000, ...filters }

    const response = await api.getArticles(params)

    if (response.success && response.data) {
      const data = response.data as any
      const items = data.items || data

      articles.value = items.map((article: any) => ({
        id: String(article.id),
        feedId: String(article.feed_id),
        title: article.title,
        description: article.description || '',
        content: article.content || '',
        link: article.link,
        pubDate: article.pub_date || article.created_at,
        author: article.author,
        category: String(article.feed_id), // Will be mapped from feed
        read: article.read || false,
        favorite: article.favorite || false,
        imageUrl: article.image_url,
      }))

      totalArticles.value = data.total || items.length
    } else {
      error.value = response.error || 'Failed to fetch articles'
    }

    loading.value = false
    return response
  }

  async function updateArticle(
    id: string,
    data: { read?: boolean; favorite?: boolean }
  ) {
    const response = await api.updateArticle(Number(id), data)

    if (response.success) {
      // Update local article store
      const articlesStore = useArticlesStore()
      const article = articlesStore.articles.find((a) => a.id === id)
      if (article) {
        Object.assign(article, data)
      }

      // Update feed unread count when marking as read
      if (data.read === true && article) {
        const feedsStore = useFeedsStore()
        const feed = feedsStore.feeds.find((f) => f.id === article.feedId)
        if (feed && feed.unreadCount && feed.unreadCount > 0) {
          feed.unreadCount--
        }
      }
    }

    return response
  }

  async function toggleFavorite(id: string) {
    const article = articles.value.find((a) => a.id === id)
    if (article) {
      return updateArticle(id, { favorite: !article.favorite })
    }
    return { success: false, error: 'Article not found' }
  }

  async function markAsRead(id: string) {
    return updateArticle(id, { read: true })
  }

  async function markAllAsRead(feedId?: string) {
    const ids = feedId
      ? articles.value.filter((a) => a.feedId === feedId).map((a) => Number(a.id))
      : articles.value.map((a) => Number(a.id))

    const response = await api.bulkUpdateArticles({
      ids,
      read: true,
    })

    if (response.success) {
      // Update local article store
      const articlesStore = useArticlesStore()
      articles.value.forEach((a) => {
        const localArticle = articlesStore.articles.find((la) => la.id === a.id)
        if (localArticle) {
          localArticle.read = true
        }
      })
    }

    return response
  }

  // OPML
  async function importOpml(file: File) {
    loading.value = true
    const response = await api.importOpml(file)
    loading.value = false

    if (response.success) {
      await fetchFeeds({ per_page: 10000 })
      await fetchCategories()
      syncToLocalStores()
    }

    return response
  }

  async function exportOpml() {
    return api.exportOpml()
  }

  async function fetchArticlesStats() {
    const response = await api.getArticlesStats()
    return response
  }

  // AI Summaries
  async function getSummaries(params: { category_id?: number; page?: number; per_page?: number } = {}) {
    loading.value = true
    error.value = null
    const response = await api.getSummaries(params)
    loading.value = false
    return response
  }

  async function generateSummary(data: {
    category_id?: number | null
    time_range?: number
    base_url: string
    api_key: string
    model: string
  }) {
    loading.value = true
    error.value = null
    const response = await api.generateSummary(data)
    loading.value = false
    return response
  }

  async function deleteSummary(id: number) {
    loading.value = true
    error.value = null
    const response = await api.deleteSummary(id)
    loading.value = false
    return response
  }

  // Initialize
  async function initialize() {
    await Promise.all([
      fetchCategories(),
      fetchFeeds({ per_page: 10000 }), // Get all feeds
      fetchArticles({ per_page: 10000 }), // Get all articles
    ])
  }

  // Sync data to local stores
  function syncToLocalStores() {
    const feedsStore = useFeedsStore()
    const articlesStore = useArticlesStore()

    // Sync categories
    feedsStore.categories = categories.value.map(cat => ({
      id: cat.id,
      name: cat.name,
      slug: cat.slug,
      icon: cat.icon,
      color: cat.color,
      description: cat.description,
      feedCount: cat.feedCount
    }))

    // Sync feeds - use allFeeds for sidebar to ensure all feeds (including uncategorized) are shown
    feedsStore.feeds = allFeeds.value.length > 0 ? allFeeds.value : feeds.value

    // Sync articles
    articlesStore.articles = articles.value
  }

  return {
    loading,
    error,
    categories,
    feeds,
    articles,
    totalArticles,
    fetchCategories,
    createCategory,
    updateCategory,
    deleteCategory,
    fetchFeeds,
    createFeed,
    updateFeed,
    deleteFeed,
    refreshFeed,
    refreshAllFeeds,
    fetchArticles,
    updateArticle,
    toggleFavorite,
    markAsRead,
    markAllAsRead,
    importOpml,
    exportOpml,
    fetchArticlesStats,
    getSummaries,
    generateSummary,
    deleteSummary,
    initialize,
    syncToLocalStores,
  }
})
