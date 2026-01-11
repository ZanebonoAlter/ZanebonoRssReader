import { defineStore } from 'pinia'
import type { RssFeed, Category } from '~/types'

export const useFeedsStore = defineStore('feeds', () => {
  // State
  const feeds = ref<RssFeed[]>([])
  const categories = ref<Category[]>([
    {
      id: 'tech',
      name: '技术',
      slug: 'tech',
      icon: 'mdi:code-tags',
      color: '#3b82f6',
      description: '技术开发、编程、架构',
      feedCount: 0
    },
    {
      id: 'news',
      name: '新闻',
      slug: 'news',
      icon: 'mdi:newspaper',
      color: '#ef4444',
      description: '时事新闻、热点追踪',
      feedCount: 0
    },
    {
      id: 'design',
      name: '设计',
      slug: 'design',
      icon: 'mdi:palette',
      color: '#8b5cf6',
      description: 'UI/UX、平面设计、创意',
      feedCount: 0
    },
    {
      id: 'blog',
      name: '博客',
      slug: 'blog',
      icon: 'mdi:post',
      color: '#10b981',
      description: '个人博客、随笔',
      feedCount: 0
    },
    {
      id: 'ai',
      name: '人工智能',
      slug: 'ai',
      icon: 'mdi:brain',
      color: '#f59e0b',
      description: 'AI、机器学习、深度学习',
      feedCount: 0
    },
    {
      id: 'product',
      name: '产品',
      slug: 'product',
      icon: 'mdi:cube-outline',
      color: '#ec4899',
      description: '产品设计、产品管理',
      feedCount: 0
    }
  ])

  // Computed
  const feedCount = computed(() => feeds.value.length)
  const categorizedFeeds = computed(() => {
    const grouped: Record<string, RssFeed[]> = {}
    feeds.value.forEach((feed) => {
      if (!grouped[feed.category]) {
        grouped[feed.category] = []
      }
      grouped[feed.category]?.push(feed)
    })
    return grouped
  })

  // Get unread count for a specific feed
  const getFeedUnreadCount = (feedId: string) => {
    const feed = feeds.value.find(f => f.id === feedId)
    return feed?.unreadCount || 0
  }

  // Get all unread counts as a record
  const unreadCountsByFeed = computed(() => {
    const counts: Record<string, number> = {}
    feeds.value.forEach((feed) => {
      counts[feed.id] = feed.unreadCount || 0
    })
    return counts
  })

  // Actions
  function addFeed(feed: RssFeed) {
    feeds.value.push(feed)
    updateCategoryCount(feed.category)
  }

  function removeFeed(id: string) {
    const feed = feeds.value.find(f => f.id === id)
    if (feed) {
      feeds.value = feeds.value.filter(f => f.id !== id)
      updateCategoryCount(feed.category)
    }
  }

  function updateFeed(id: string, updates: Partial<RssFeed>) {
    const index = feeds.value.findIndex(f => f.id === id)
    if (index !== -1) {
      feeds.value[index] = { ...feeds.value[index], ...updates } as RssFeed
    }
  }

  function updateCategoryCount(categoryId: string) {
    const category = categories.value.find(c => c.id === categoryId)
    if (category) {
      category.feedCount = feeds.value.filter(f => f.category === categoryId).length
    }
  }

  function getFeedsByCategory(categoryId: string) {
    return feeds.value.filter(f => f.category === categoryId)
  }

  function getCategoryBySlug(slug: string) {
    return categories.value.find(c => c.slug === slug)
  }

  function addCategory(category: Category) {
    categories.value.push(category)
  }

  function deleteCategory(categoryId: string) {
    const index = categories.value.findIndex(c => c.id === categoryId)
    if (index !== -1) {
      categories.value.splice(index, 1)
    }
  }

  function updateCategory(categoryId: string, updates: Partial<Category>) {
    const category = categories.value.find(c => c.id === categoryId)
    if (category) {
      Object.assign(category, updates)
    }
  }

  // Initialize with sample feeds
  function initSampleFeeds() {
    const sampleFeeds: RssFeed[] = [
      {
        id: '1',
        title: '阮一峰的网络日志',
        description: '知名技术博客，每周科技爱好者',
        url: 'https://www.ruanyifeng.com/blog/atom.xml',
        category: 'tech',
        icon: 'mdi:rss',
        color: '#3b82f6',
        lastUpdated: new Date().toISOString(),
        articleCount: 0
      },
      {
        id: '2',
        title: '少数派',
        description: '高品质数字生活指南',
        url: 'https://sspai.com/feed',
        category: 'tech',
        icon: 'mdi:tablet-ipad',
        color: '#3b82f6',
        lastUpdated: new Date().toISOString(),
        articleCount: 0
      },
      {
        id: '3',
        title: '36氪',
        description: '领先的科技创投媒体',
        url: 'https://36kr.com/feed',
        category: 'news',
        icon: 'mdi:alpha-k-circle-outline',
        color: '#ef4444',
        lastUpdated: new Date().toISOString(),
        articleCount: 0
      },
      {
        id: '4',
        title: 'InfoQ',
        description: '促进软件开发及相关领域知识与创新的传播',
        url: 'https://www.infoq.cn/feed',
        category: 'tech',
        icon: 'mdi:alpha-q',
        color: '#3b82f6',
        lastUpdated: new Date().toISOString(),
        articleCount: 0
      }
    ]

    sampleFeeds.forEach(feed => addFeed(feed))
  }

  return {
    feeds,
    categories,
    feedCount,
    categorizedFeeds,
    getFeedUnreadCount,
    unreadCountsByFeed,
    addFeed,
    removeFeed,
    updateFeed,
    getFeedsByCategory,
    getCategoryBySlug,
    addCategory,
    deleteCategory,
    updateCategory,
    initSampleFeeds
  }
})

if (import.meta.hot) {
  import.meta.hot.accept(acceptHMRUpdate(useFeedsStore, import.meta.hot))
}
