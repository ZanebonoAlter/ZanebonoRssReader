import { apiClient } from './client'
import type { ApiResponse, CreateCategoryData, UpdateCategoryData, Category } from '~/types'

export function useCategoriesApi() {
  async function getCategories(): Promise<ApiResponse<Category[]>> {
    return apiClient.get<Category[]>('/categories')
  }

  async function createCategory(data: CreateCategoryData): Promise<ApiResponse<Category>> {
    return apiClient.post<Category>('/categories', data)
  }

  async function updateCategory(id: number, data: UpdateCategoryData): Promise<ApiResponse<Category>> {
    return apiClient.put<Category>(`/categories/${id}`, data)
  }

  async function deleteCategory(id: number): Promise<ApiResponse<void>> {
    return apiClient.delete<void>(`/categories/${id}`)
  }

  return {
    getCategories,
    createCategory,
    updateCategory,
    deleteCategory,
  }
}
