import { useApiStore } from '~/stores/api'

/**
 * 数据同步服务
 * 负责从 API Store 同步数据到本地 Store
 */
export function useSyncService() {
  const apiStore = useApiStore()

  /**
   * 同步所有数据
   * 从 API Store 同步分类、订阅源和文章到本地 Store
   */
  function syncAll() {
    syncCategories()
    syncFeeds()
    syncArticles()
  }

  /**
   * 同步分类数据
   */
  function syncCategories() {
    const feedsStore = useFeedsStore()
    feedsStore.categories = apiStore.categories.map((cat) => ({
      id: cat.id,
      name: cat.name,
      slug: cat.slug,
      icon: cat.icon,
      color: cat.color,
      description: cat.description,
      feedCount: cat.feedCount,
    }))
  }

  /**
   * 同步订阅源数据
   */
  function syncFeeds() {
    const feedsStore = useFeedsStore()
    feedsStore.feeds =
      apiStore.allFeeds.length > 0 ? apiStore.allFeeds : apiStore.feeds
  }

  /**
   * 同步文章数据
   */
  function syncArticles() {
    const articlesStore = useArticlesStore()
    articlesStore.articles = apiStore.articles
  }

  /**
   * 更新订阅源未读数量
   * @param feedId - 订阅源 ID
   * @param change - 数量变化（正数增加，负数减少）
   */
  function updateFeedUnreadCount(feedId: string, change: number) {
    const feedsStore = useFeedsStore()
    const feed = feedsStore.feeds.find((f) => f.id === feedId)
    if (feed && feed.unreadCount !== undefined) {
      feed.unreadCount = Math.max(0, feed.unreadCount + change)
    }
  }

  /**
   * 更新文章状态
   * @param articleId - 文章 ID
   * @param updates - 更新数据
   */
  function updateArticleStatus(
    articleId: string,
    updates: { read?: boolean; favorite?: boolean }
  ) {
    const articlesStore = useArticlesStore()
    const article = articlesStore.articles.find((a) => a.id === articleId)
    if (article) {
      Object.assign(article, updates)
    }
  }

  return {
    syncAll,
    syncCategories,
    syncFeeds,
    syncArticles,
    updateFeedUnreadCount,
    updateArticleStatus,
  }
}
