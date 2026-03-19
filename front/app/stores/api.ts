import type { Category, RssFeed, Article } from '~/types'
import { useCategoriesApi } from '~/api/categories'
import { useFeedsApi } from '~/api/feeds'
import { useArticlesApi } from '~/api/articles'
import { useOpmlApi } from '~/api/opml'
import { useSummariesApi } from '~/api/summaries'

export const useApiStore = defineStore('api', () => {
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Categories
  const categories = ref<Category[]>([])

  async function fetchCategories() {
    loading.value = true
    error.value = null

    const categoriesApi = useCategoriesApi()
    const response = await categoriesApi.getCategories()

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
    const categoriesApi = useCategoriesApi()
    const response = await categoriesApi.createCategory(data)
    loading.value = false

    if (response.success) {
      await fetchCategories()
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
    const categoriesApi = useCategoriesApi()
    const response = await categoriesApi.updateCategory(Number(id), data)
    loading.value = false

    if (response.success) {
      await fetchCategories()
    }

    return response
  }

  async function deleteCategory(id: string) {
    loading.value = true
    const categoriesApi = useCategoriesApi()
    const response = await categoriesApi.deleteCategory(Number(id))
    loading.value = false

    if (response.success) {
      await fetchCategories()
    }

    return response
  }

  // Feeds
  const feeds = ref<RssFeed[]>([])
  const allFeeds = ref<RssFeed[]>([]) // Cache all feeds for sidebar display

  async function fetchFeeds(params: { page?: number; per_page?: number; category_id?: number; uncategorized?: boolean } = {}) {
    loading.value = true
    error.value = null

    const feedsApi = useFeedsApi()
    const response = await feedsApi.getFeeds(params)

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
        articleSummaryEnabled: feed.article_summary_enabled,
        completionOnRefresh: feed.completion_on_refresh,
        maxCompletionRetries: feed.max_completion_retries,
        firecrawlEnabled: feed.firecrawl_enabled,
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
    const feedsApi = useFeedsApi()
    const response = await feedsApi.createFeed(data)
    loading.value = false

    if (response.success) {
      await fetchFeeds({ per_page: 10000 })
      await fetchArticles({ per_page: 10000 })
    }

    return response
  }

  async function deleteFeed(id: string) {
    loading.value = true
    const feedsApi = useFeedsApi()
    const response = await feedsApi.deleteFeed(Number(id))
    loading.value = false

    if (response.success) {
      await fetchFeeds({ per_page: 10000 })
      await fetchArticles({ per_page: 10000 })
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
      max_articles?: number
      refresh_interval?: number
      ai_summary_enabled?: boolean
      article_summary_enabled?: boolean
      completion_on_refresh?: boolean
      max_completion_retries?: number
      firecrawl_enabled?: boolean
    }
  ) {
    loading.value = true
    const feedsApi = useFeedsApi()
    const response = await feedsApi.updateFeed(Number(id), data)
    loading.value = false

    if (response.success) {
      await fetchFeeds({ per_page: 10000 })
      await fetchArticles({ per_page: 10000 })
    }

    return response
  }

  async function refreshFeed(id: string) {
    loading.value = true
    const feedsApi = useFeedsApi()
    const response = await feedsApi.refreshFeed(Number(id))
    loading.value = false
    return response
  }

  async function refreshAllFeeds() {
    loading.value = true
    const feedsApi = useFeedsApi()
    const response = await feedsApi.refreshAllFeeds()
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
    start_date?: string
    end_date?: string
  } = {}) {
    loading.value = true
    error.value = null

    // Set a high default per_page to get all articles
    const params = { per_page: 10000, ...filters }

    const articlesApi = useArticlesApi()
    const response = await articlesApi.getArticles(params)

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
        category: article.category_id ? String(article.category_id) : '',
        read: article.read || false,
        favorite: article.favorite || false,
        summaryStatus: article.summary_status,
        summaryGeneratedAt: article.summary_generated_at,
        completionAttempts: article.completion_attempts,
        completionError: article.completion_error,
        aiContentSummary: article.ai_content_summary,
        firecrawlStatus: article.firecrawl_status,
        firecrawlError: article.firecrawl_error,
        firecrawlContent: article.firecrawl_content,
        firecrawlCrawledAt: article.firecrawl_crawled_at,
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
    const articlesApi = useArticlesApi()
    const article = articles.value.find((a) => a.id === id)
    const wasFavorite = article?.favorite

    const response = await articlesApi.updateArticle(Number(id), data)

    if (response.success) {
      if (article) {
        Object.assign(article, data)
      }

      if (data.read === true && article) {
        const sourceFeeds = allFeeds.value.length > 0 ? allFeeds.value : feeds.value
        const feed = sourceFeeds.find((f) => f.id === article.feedId)
        if (feed && feed.unreadCount && feed.unreadCount > 0) {
          feed.unreadCount--
        }
      }

      if (data.favorite !== undefined && wasFavorite !== undefined && article) {
        article.favorite = data.favorite
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

    const articlesApi = useArticlesApi()
    const response = await articlesApi.bulkUpdateArticles({
      ids,
      read: true,
    })

    if (response.success) {
      articles.value.forEach((a) => {
        if (!feedId || a.feedId === feedId) {
          a.read = true
        }
      })
    }

    return response
  }

  // OPML
  async function importOpml(file: File) {
    loading.value = true
    const opmlApi = useOpmlApi()
    const response = await opmlApi.importOpml(file)
    loading.value = false

    if (response.success) {
      await fetchFeeds({ per_page: 10000 })
      await fetchCategories()
    }

    return response
  }

  async function exportOpml() {
    const opmlApi = useOpmlApi()
    return opmlApi.exportOpml()
  }

  async function fetchArticlesStats() {
    const articlesApi = useArticlesApi()
    const response = await articlesApi.getArticlesStats()
    return response
  }

  // AI Summaries
  async function getSummaries(params: { category_id?: number; page?: number; per_page?: number } = {}) {
    loading.value = true
    error.value = null
    const summariesApi = useSummariesApi()
    const response = await summariesApi.getSummaries(params)
    loading.value = false
    return response
  }

  async function generateSummary(data: {
    category_id?: number | null
    time_range?: number
  }) {
    loading.value = true
    error.value = null
    const summariesApi = useSummariesApi()
    const response = await summariesApi.generateSummary(data)
    loading.value = false
    return response
  }

  async function deleteSummary(id: number) {
    loading.value = true
    error.value = null
    const summariesApi = useSummariesApi()
    const response = await summariesApi.deleteSummary(id)
    loading.value = false
    return response
  }

  // Queue Summary
  async function submitQueueSummary(data: {
    category_ids?: number[]
    feed_ids?: number[]
    time_range?: number
  }) {
    loading.value = true
    error.value = null
    const summariesApi = useSummariesApi()
    const response = await summariesApi.submitQueueSummary(data)
    loading.value = false
    return response
  }

  async function getQueueStatus() {
    const summariesApi = useSummariesApi()
    const response = await summariesApi.getQueueStatus()
    return response
  }

  async function getQueueJob(jobId: string) {
    const summariesApi = useSummariesApi()
    const response = await summariesApi.getQueueJob(jobId)
    return response
  }

  // Initialize
  async function initialize() {
    await Promise.all([
      fetchCategories(),
      fetchFeeds({ per_page: 10000 }),
      fetchArticles({ per_page: 10000 }),
    ])
  }

  return {
    loading,
    error,
    categories,
    feeds,
    allFeeds,
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
    submitQueueSummary,
    getQueueStatus,
    getQueueJob,
    initialize,
  }
})


