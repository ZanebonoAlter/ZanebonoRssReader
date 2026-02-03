import { REFRESH_POLLING_INTERVAL, MAX_POLLING_TIME } from '~/utils/constants'

/**
 * 刷新轮询 Composable
 * 封装订阅源刷新状态的轮询逻辑
 */
export function useRefreshPolling() {
  const apiStore = useApiStore()
  const feedsStore = useFeedsStore()

  const selectedCategory = ref<string | null>(null)
  const selectedFeed = ref<string | null>(null)

  const refreshPollingInterval = ref<ReturnType<typeof setTimeout> | null>(null)
  const isPolling = ref(false)

  /**
   * 开始轮询刷新状态
   */
  function startPolling() {
    const startTime = Date.now()

    const poll = async () => {
      // 超过最大轮询时间则停止
      if (Date.now() - startTime > MAX_POLLING_TIME) {
        stopPolling()
        return
      }

      // 根据当前选择获取最新的订阅源状态
      if (selectedFeed.value) {
        // 单个订阅源选中 - 获取所有并找到特定订阅源
        await apiStore.fetchFeeds({ per_page: 10000 })
      } else if (selectedCategory.value === 'uncategorized') {
        await apiStore.fetchFeeds({ uncategorized: true, per_page: 10000 })
      } else if (
        selectedCategory.value &&
        selectedCategory.value !== 'favorites' &&
        selectedCategory.value !== 'ai-summaries'
      ) {
        await apiStore.fetchFeeds({
          category_id: parseInt(selectedCategory.value),
          per_page: 10000,
        })
      } else {
        await apiStore.fetchFeeds({ per_page: 10000 })
      }

      apiStore.syncToLocalStores()

      // 获取正在监控刷新状态的订阅源
      const monitoredFeeds = selectedFeed.value
        ? feedsStore.feeds.filter((f) => f.id === selectedFeed.value)
        : feedsStore.feeds

      // 检查是否有订阅源仍在刷新
      const stillRefreshing = monitoredFeeds.some(
        (f) => f.refreshStatus === 'refreshing'
      )

      if (stillRefreshing) {
        // 继续轮询
        refreshPollingInterval.value = setTimeout(poll, REFRESH_POLLING_INTERVAL)
      } else {
        // 所有订阅源刷新完成
        stopPolling()
      }
    }

    // 开始轮询
    isPolling.value = true
    poll()
  }

  /**
   * 停止轮询
   */
  function stopPolling() {
    if (refreshPollingInterval.value) {
      clearTimeout(refreshPollingInterval.value)
      refreshPollingInterval.value = null
    }
    isPolling.value = false
  }

  /**
   * 更新选择状态
   * @param category - 选中的分类
   * @param feed - 选中的订阅源
   */
  function updateSelection(category: string | null, feed: string | null) {
    selectedCategory.value = category
    selectedFeed.value = feed
  }

  return {
    isPolling,
    startPolling,
    stopPolling,
    updateSelection,
  }
}
