import { useCategoriesApi } from '~/composables/api'
import type {
  ApiResponse,
  Category,
  CreateCategoryData,
  UpdateCategoryData,
} from '~/types'

/**
 * 分类服务
 * 封装分类相关的业务逻辑
 */
export function useCategoryService() {
  const api = useCategoriesApi()

  /**
   * 加载所有分类
   * @returns API 响应
   */
  async function loadCategories(): Promise<ApiResponse<Category[]>> {
    return api.getCategories()
  }

  /**
   * 创建分类
   * @param data - 分类数据
   * @returns API 响应
   */
  async function createCategory(data: CreateCategoryData): Promise<ApiResponse<Category>> {
    return api.createCategory(data)
  }

  /**
   * 更新分类
   * @param id - 分类 ID
   * @param data - 更新数据
   * @returns API 响应
   */
  async function updateCategory(
    id: number,
    data: UpdateCategoryData
  ): Promise<ApiResponse<Category>> {
    return api.updateCategory(id, data)
  }

  /**
   * 删除分类
   * @param id - 分类 ID
   * @returns API 响应
   */
  async function deleteCategory(id: number): Promise<ApiResponse<void>> {
    return api.deleteCategory(id)
  }

  return {
    loadCategories,
    createCategory,
    updateCategory,
    deleteCategory,
  }
}
