import { apiClient } from './client'

export interface WatchedTag {
  id: number
  slug: string
  label: string
  category: string
  watchedAt: string | null
  isAbstract: boolean
  childSlugs: string[]
}

export function useWatchedTagsApi() {
  return {
    async listWatchedTags() {
      const res = await apiClient.get<any>('/topic-tags/watched')
      if (!res.success) return res
      const data = (res.data || []).map((t: any) => ({
        id: t.id,
        slug: t.slug,
        label: t.label,
        category: t.category,
        watchedAt: t.watched_at,
        isAbstract: t.is_abstract || false,
        childSlugs: t.child_slugs || [],
      }))
      return { ...res, data } as any
    },
    async watchTag(tagId: number) {
      return apiClient.post(`/topic-tags/${tagId}/watch`, {})
    },
    async unwatchTag(tagId: number) {
      return apiClient.post(`/topic-tags/${tagId}/unwatch`, {})
    },
  }
}
