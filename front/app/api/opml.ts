import { apiClient } from './client'
import type { ApiResponse } from '~/types'

export function useOpmlApi() {
  async function importOpml(file: File): Promise<ApiResponse<any>> {
    const formData = new FormData()
    formData.append('file', file)
    return apiClient.upload('/import-opml', formData)
  }

  async function exportOpml(): Promise<Blob> {
    return apiClient.download('/export-opml')
  }

  return {
    importOpml,
    exportOpml,
  }
}
