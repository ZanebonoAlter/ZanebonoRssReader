import { useFeedsApi } from './api'

interface RefreshTimer {
  feedId: string
  interval: number
  timer: ReturnType<typeof setInterval> | null
}

export const useAutoRefresh = () => {
  const timers = ref<Map<string, RefreshTimer>>(new Map())
  const apiStore = useApiStore()
  const isRefreshing = ref(false)

  // Clear existing timer for a feed
  function clearTimer(feedId: string) {
    const timerData = timers.value.get(feedId)
    if (timerData?.timer) {
      clearInterval(timerData.timer)
      timers.value.delete(feedId)
    }
  }

  // Setup auto-refresh for a feed
  function setupAutoRefresh(feedId: string, intervalMinutes: number) {
    // Clear existing timer first
    clearTimer(feedId)

    // Don't setup if interval is 0 (manual refresh only)
    if (intervalMinutes <= 0) {
      return
    }

    // Convert minutes to milliseconds
    const intervalMs = intervalMinutes * 60 * 1000

    // Create new timer
    const api = useFeedsApi()

    const timer = setInterval(async () => {
      try {
        await api.refreshFeed(Number(feedId))
        await apiStore.fetchFeeds({ per_page: 10000 })
        await apiStore.fetchArticles({ per_page: 10000 })
      } catch (error) {
        console.error(`Auto-refresh failed for feed ${feedId}:`, error)
      }
    }, intervalMs)

    timers.value.set(feedId, {
      feedId,
      interval: intervalMinutes,
      timer,
    })
  }

  // Initialize auto-refresh for all feeds
  function initialize() {
    // Clear all existing timers
    timers.value.forEach((timerData) => {
      if (timerData.timer) {
        clearInterval(timerData.timer)
      }
    })
    timers.value.clear()

    // Setup timers for feeds with auto-refresh enabled
    apiStore.feeds.forEach((feed) => {
      const interval = feed.refreshInterval || 0
      if (interval > 0) {
        setupAutoRefresh(feed.id, interval)
      }
    })
  }

  // Refresh a specific feed's settings
  function updateFeedRefresh(feedId: string, intervalMinutes: number) {
    setupAutoRefresh(feedId, intervalMinutes)
  }

  // Remove a feed's timer
  function removeFeed(feedId: string) {
    clearTimer(feedId)
  }

  // Cleanup all timers
  function cleanup() {
    timers.value.forEach((timerData) => {
      if (timerData.timer) {
        clearInterval(timerData.timer)
      }
    })
    timers.value.clear()
  }

  // Get active timers count
  const activeCount = computed(() => timers.value.size)

  return {
    timers,
    isRefreshing,
    activeCount,
    setupAutoRefresh,
    initialize,
    updateFeedRefresh,
    removeFeed,
    cleanup,
  }
}

// Global auto-refresh manager (singleton pattern)
let globalAutoRefresh: ReturnType<typeof useAutoRefresh> | null = null

export function useGlobalAutoRefresh() {
  if (!globalAutoRefresh) {
    globalAutoRefresh = useAutoRefresh()
  }

  // Initialize on first use
  onMounted(() => {
    const apiStore = useApiStore()
    // Wait for feeds to be loaded
    watch(
      () => apiStore.feeds.length,
      (length) => {
        if (length > 0) {
          globalAutoRefresh?.initialize()
        }
      },
      { immediate: true }
    )
  })

  // Cleanup on unmount
  onUnmounted(() => {
    globalAutoRefresh?.cleanup()
  })

  return globalAutoRefresh
}
