import { apiClient } from './client'
import type { ApiResponse } from '~/types'

/**
 * OPML API
 */
export function useOpmlApi() {
  /**
   * 导入 OPML 文件
   */
  async function importOpml(file: File): Promise<ApiResponse<any>> {
    const formData = new FormData()
    formData.append('file', file)
    return apiClient.upload('/import-opml', formData)
  }

  /**
   * 导出 OPML 文件
   */
  async function exportOpml(): Promise<Blob> {
    return apiClient.download('/export-opml')
  }

  return {
    importOpml,
    exportOpml,
  }
}
