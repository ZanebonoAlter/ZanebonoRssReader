import { apiClient } from './client'
import type { ApiResponse } from '~/types'
import type { TagHierarchyNode, TagHierarchyResponse } from '~/types/topicTag'

interface RawTagHierarchyNode {
  id: number
  label: string
  slug: string
  category: string
  icon: string
  feed_count: number
  similarity_score?: number
  children: RawTagHierarchyNode[]
}

interface RawHierarchyResponse {
  nodes: RawTagHierarchyNode[]
  total: number
}

function mapNode(raw: RawTagHierarchyNode): TagHierarchyNode {
  return {
    id: raw.id,
    label: raw.label,
    slug: raw.slug,
    category: raw.category,
    icon: raw.icon,
    feedCount: raw.feed_count,
    similarityScore: raw.similarity_score,
    children: raw.children ? raw.children.map(mapNode) : [],
  }
}

export function useAbstractTagApi() {
  return {
    async fetchHierarchy(category?: string): Promise<ApiResponse<TagHierarchyResponse>> {
      const query = category ? `?category=${encodeURIComponent(category)}` : ''
      const response = await apiClient.get<RawHierarchyResponse>(`/topic-tags/hierarchy${query}`)
      if (response.success && response.data) {
        return {
          ...response,
          data: {
            nodes: response.data.nodes.map(mapNode),
            total: response.data.total,
          },
        } as ApiResponse<TagHierarchyResponse>
      }
      return response as unknown as ApiResponse<TagHierarchyResponse>
    },

    async updateAbstractName(tagId: number, newName: string): Promise<ApiResponse<{ id: number; newName: string }>> {
      return apiClient.put('/topic-tags/' + tagId + '/abstract-name', { new_name: newName })
    },

    async detachChild(parentId: number, childId: number): Promise<ApiResponse<{ message: string }>> {
      return apiClient.post('/topic-tags/' + parentId + '/detach', { child_id: childId })
    },
  }
}
