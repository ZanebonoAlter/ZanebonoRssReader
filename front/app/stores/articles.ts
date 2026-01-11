import { defineStore } from 'pinia'
import type { Article, FilterState } from '~/types'

export const useArticlesStore = defineStore('articles', () => {
  // State
  const articles = ref<Article[]>([])
  const filters = ref<FilterState>({
    sort: 'latest',
    filter: 'all',
    category: null,
    search: ''
  })
  const loading = ref(false)
  const currentArticle = ref<Article | null>(null)

  // Computed
  const filteredArticles = computed(() => {
    let result = [...articles.value]

    // Apply category filter
    if (filters.value.category) {
      result = result.filter(a => a.category === filters.value.category)
    }

    // Apply status filter
    if (filters.value.filter === 'unread') {
      result = result.filter(a => !a.read)
    } else if (filters.value.filter === 'favorites') {
      result = result.filter(a => a.favorite)
    }

    // Apply search filter
    if (filters.value.search) {
      const searchLower = filters.value.search.toLowerCase()
      result = result.filter(a =>
        a.title.toLowerCase().includes(searchLower) ||
        a.description.toLowerCase().includes(searchLower)
      )
    }

    // Apply sorting
    if (filters.value.sort === 'latest') {
      result.sort((a, b) => new Date(b.pubDate).getTime() - new Date(a.pubDate).getTime())
    } else if (filters.value.sort === 'popular') {
      result.sort((a, b) => b.title.localeCompare(a.title))
    } else if (filters.value.sort === 'unread') {
      result.sort((a, b) => Number(a.read) - Number(b.read))
    }

    return result
  })

  const unreadCount = computed(() =>
    articles.value.filter(a => !a.read).length
  )

  const favoriteCount = computed(() =>
    articles.value.filter(a => a.favorite).length
  )

  const articlesByFeed = computed(() => {
    const grouped: Record<string, Article[]> = {}
    articles.value.forEach((article) => {
      if (!grouped[article.feedId]) {
        grouped[article.feedId] = []
      }
      grouped[article.feedId]!.push(article)
    })
    return grouped
  })

  const unreadCountByFeed = computed(() => {
    const grouped: Record<string, number> = {}
    articles.value.forEach((article) => {
      if (!article.read) {
        if (!grouped[article.feedId]) {
          grouped[article.feedId] = 0
        }
        grouped[article.feedId]!++
      }
    })
    return grouped
  })

  // Actions
  function addArticles(newArticles: Article[]) {
    articles.value = [...newArticles, ...articles.value]
  }

  function addArticle(article: Article) {
    articles.value.unshift(article)
  }

  function markAsRead(id: string) {
    const article = articles.value.find(a => a.id === id)
    if (article) {
      article.read = true
    }
  }

  function markAllAsRead(feedId?: string) {
    if (feedId) {
      articles.value
        .filter(a => a.feedId === feedId)
        .forEach(a => a.read = true)
    } else {
      articles.value.forEach(a => a.read = true)
    }
  }

  function toggleFavorite(id: string) {
    const article = articles.value.find(a => a.id === id)
    if (article) {
      article.favorite = !article.favorite
    }
  }

  function updateFilters(newFilters: Partial<FilterState>) {
    filters.value = { ...filters.value, ...newFilters }
  }

  function resetFilters() {
    filters.value = {
      sort: 'latest',
      filter: 'all',
      category: null,
      search: ''
    }
  }

  function getArticleById(id: string) {
    return articles.value.find(a => a.id === id)
  }

  function getArticlesByFeed(feedId: string) {
    return articles.value.filter(a => a.feedId === feedId)
  }

  function setCurrentArticle(article: Article | null) {
    currentArticle.value = article
    if (article && !article.read) {
      markAsRead(article.id)
    }
  }

  // Initialize with sample articles
  function initSampleArticles() {
    const sampleArticles: Article[] = [
      {
        id: '1',
        feedId: '1',
        title: '科技爱好者周刊（第 300 期）',
        description: '本周话题：AI 时代的编程教育...',
        content: '完整的文章内容...',
        link: 'https://www.ruanyifeng.com/blog/2024/01/weekly-issue-300.html',
        pubDate: new Date().toISOString(),
        author: '阮一峰',
        category: 'tech',
        read: false,
        favorite: false,
        imageUrl: 'https://picsum.photos/800/400?random=1'
      },
      {
        id: '2',
        feedId: '2',
        title: '如何构建高效的工作流',
        description: '探索提升工作效率的工具和方法...',
        content: '完整的文章内容...',
        link: 'https://sspai.com/article/12345',
        pubDate: new Date(Date.now() - 3600000).toISOString(),
        author: '少数派',
        category: 'tech',
        read: false,
        favorite: true,
        imageUrl: 'https://picsum.photos/800/400?random=2'
      },
      {
        id: '3',
        feedId: '3',
        title: '2024 年科技行业趋势预测',
        description: '深入分析明年科技发展的关键方向...',
        content: '完整的文章内容...',
        link: 'https://36kr.com/p/123456',
        pubDate: new Date(Date.now() - 7200000).toISOString(),
        author: '36氪',
        category: 'news',
        read: true,
        favorite: false,
        imageUrl: 'https://picsum.photos/800/400?random=3'
      },
      {
        id: '4',
        feedId: '4',
        title: '微服务架构最佳实践',
        description: '分享微服务设计和实施的经验...',
        content: '完整的文章内容...',
        link: 'https://www.infoq.cn/article/12345',
        pubDate: new Date(Date.now() - 10800000).toISOString(),
        author: 'InfoQ',
        category: 'tech',
        read: false,
        favorite: false,
        imageUrl: 'https://picsum.photos/800/400?random=4'
      }
    ]

    articles.value = sampleArticles
  }

  return {
    articles,
    filters,
    loading,
    currentArticle,
    filteredArticles,
    unreadCount,
    favoriteCount,
    articlesByFeed,
    unreadCountByFeed,
    addArticles,
    addArticle,
    markAsRead,
    markAllAsRead,
    toggleFavorite,
    updateFilters,
    resetFilters,
    getArticleById,
    getArticlesByFeed,
    setCurrentArticle,
    initSampleArticles
  }
})

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useArticlesStore, import.meta.hot))
}
