import { ref } from 'vue'

export interface ContentCompletionStatus {
  content_status: 'complete' | 'incomplete' | 'pending' | 'failed'
  attempts: number
  error: string | null
  fetched_at: string | null
}

export function useContentCompletion() {
  const loading = ref(false)
  const error = ref<string | null>(null)

  const completeArticle = async (articleId: string) => {
    loading.value = true
    error.value = null

    try {
      const response = await $fetch<{ success: boolean; message?: string; error?: string }>(
        `/api/content-completion/articles/${articleId}/complete`,
        { method: 'POST' }
      )

      if (!response.success) {
        throw new Error(response.error || 'Failed to complete article')
      }

      return response
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  const getCompletionStatus = async (articleId: string): Promise<ContentCompletionStatus> => {
    try {
      const response = await $fetch<{ success: boolean; data: ContentCompletionStatus }>(
        `/api/content-completion/articles/${articleId}/status`
      )

      if (!response.success) {
        throw new Error('Failed to get completion status')
      }

      return response.data
    } catch (e) {
      throw e
    }
  }

  const completeFeedArticles = async (feedId: string) => {
    loading.value = true
    error.value = null

    try {
      const response = await $fetch<{
        success: boolean
        completed: number
        failed: number
        total: number
      }>(
        `/api/content-completion/feeds/${feedId}/complete-all`,
        { method: 'POST' }
      )

      if (!response.success) {
        throw new Error('Failed to complete feed articles')
      }

      return response.data
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Unknown error'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    completeArticle,
    getCompletionStatus,
    completeFeedArticles,
  }
}
