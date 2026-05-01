import { apiClient } from './client'
import type { ApiResponse } from '~/types'

export interface BoardConcept {
  id: number
  name: string
  description: string
  scope_type: string
  scope_category_id: number | null
  is_system: boolean
  is_active: boolean
  display_order: number
  created_at: string
  updated_at: string
}

export interface ConceptSuggestion {
  name: string
  description: string
}

export function useBoardConceptsApi() {
  async function getBoardConcepts(): Promise<ApiResponse<BoardConcept[]>> {
    return apiClient.get<BoardConcept[]>('/narratives/board-concepts')
  }

  async function createBoardConcept(data: {
    name: string
    description: string
    scope_type?: string
    scope_category_id?: number
  }): Promise<ApiResponse<BoardConcept>> {
    return apiClient.post<BoardConcept>('/narratives/board-concepts', data)
  }

  async function updateBoardConcept(
    id: number,
    data: { name: string; description: string },
  ): Promise<ApiResponse<BoardConcept>> {
    return apiClient.put<BoardConcept>(`/narratives/board-concepts/${id}`, data)
  }

  async function deleteBoardConcept(id: number): Promise<ApiResponse<void>> {
    return apiClient.delete<void>(`/narratives/board-concepts/${id}`)
  }

  async function suggestConcepts(): Promise<ApiResponse<ConceptSuggestion[]>> {
    return apiClient.post<ConceptSuggestion[]>('/narratives/board-concepts/suggest')
  }

  return {
    getBoardConcepts,
    createBoardConcept,
    updateBoardConcept,
    deleteBoardConcept,
    suggestConcepts,
  }
}
