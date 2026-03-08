import { ref } from 'vue'
import { apiClient } from '~/api/client'

export interface ContentCompletionStatus {
  contentStatus: 'complete' | 'incomplete' | 'pending' | 'failed'
  attempts: number
  error: string | null
  fetchedAt: string | null
  aiContentSummary?: string
  fullContent?: string
  firecrawlContent?: string
  firecrawlStatus?: 'pending' | 'processing' | 'completed' | 'failed'
  firecrawlError?: string | null
}

interface CompletionStatusPayload {
  content_status: 'complete' | 'incomplete' | 'pending' | 'failed'
  attempts: number
  error: string | null
  fetched_at: string | null
  ai_content_summary?: string
  full_content?: string
  firecrawl_content?: string
  firecrawl_status?: 'pending' | 'processing' | 'completed' | 'failed'
  firecrawl_error?: string | null
}

export function useContentCompletion() {
  const loading = ref(false)
  const error = ref<string | null>(null)

  const completeArticle = async (articleId: string, options: { force?: boolean } = {}) => {
    loading.value = true
    error.value = null

    try {
      const response = await apiClient.post<{ message?: string }>(
        `/content-completion/articles/${articleId}/complete`,
        options.force ? { force: true } : undefined,
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
    const response = await apiClient.get<CompletionStatusPayload>(
      `/content-completion/articles/${articleId}/status`
    )

    if (!response.success || !response.data) {
      throw new Error(response.error || 'Failed to get completion status')
    }

    return {
      contentStatus: response.data.content_status,
      attempts: response.data.attempts,
      error: response.data.error,
      fetchedAt: response.data.fetched_at,
      aiContentSummary: response.data.ai_content_summary,
      fullContent: response.data.full_content,
      firecrawlContent: response.data.firecrawl_content,
      firecrawlStatus: response.data.firecrawl_status,
      firecrawlError: response.data.firecrawl_error,
    }
  }

  const completeFeedArticles = async (feedId: string) => {
    loading.value = true
    error.value = null

    try {
      const response = await apiClient.post<{
        completed: number
        failed: number
        total: number
      }>(`/content-completion/feeds/${feedId}/complete-all`)

      if (!response.success || !response.data) {
        throw new Error(response.error || 'Failed to complete feed articles')
      }

      return {
        completed: response.data.completed,
        failed: response.data.failed,
        total: response.data.total,
      }
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
