import { apiClient } from './client'
import type { ApiResponse } from '~/types'
import type { MergePreviewResponse, MergeWithCustomNameRequest, MergeWithCustomNameResult } from '~/types/tagMerge'

export function useTagMergePreviewApi() {
  return {
    async scanMergePreview(params?: { limit?: number; includeArticles?: boolean }) {
      const queryParams = apiClient.buildQueryParams({
        limit: params?.limit ? String(params.limit) : undefined,
        include_articles: params?.includeArticles ? 'true' : undefined,
      })
      const endpoint = queryParams ? `/topic-tags/merge-preview?${queryParams}` : '/topic-tags/merge-preview'
      return apiClient.get<MergePreviewResponse>(endpoint)
    },

    async mergeTagsWithCustomName(request: MergeWithCustomNameRequest) {
      return apiClient.post<MergeWithCustomNameResult>('/topic-tags/merge-with-name', {
        source_tag_id: request.sourceTagId,
        target_tag_id: request.targetTagId,
        new_name: request.newName,
      })
    },
  }
}
