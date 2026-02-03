import { apiClient } from './client'
import type {
  ApiResponse,
  CreateCategoryData,
  UpdateCategoryData,
  Category,
} from '~/types'

/**
 * 分类 API
 */
export function useCategoriesApi() {
  /**
   * 获取所有分类
   */
  async function getCategories(): Promise<ApiResponse<Category[]>> {
    return apiClient.get<Category[]>('/categories')
  }

  /**
   * 创建分类
   */
  async function createCategory(data: CreateCategoryData): Promise<ApiResponse<Category>> {
    return apiClient.post<Category>('/categories', data)
  }

  /**
   * 更新分类
   */
  async function updateCategory(
    id: number,
    data: UpdateCategoryData
  ): Promise<ApiResponse<Category>> {
    return apiClient.put<Category>(`/categories/${id}`, data)
  }

  /**
   * 删除分类
   */
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
