const API_BASE = 'http://localhost:5000/api'

interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
  error?: string
}

interface PaginationParams {
  page?: number
  per_page?: number
  category_id?: number
  uncategorized?: boolean
}

interface ArticleFilters extends PaginationParams {
  feed_id?: number
  read?: boolean
  favorite?: boolean
  search?: string
}

class ApiClient {
  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    try {
      const url = `${API_BASE}${endpoint}`
      const response = await fetch(url, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...options.headers,
        },
      })

      const data = await response.json()

      if (!response.ok) {
        return {
          success: false,
          error: data.error || data.message || 'Request failed',
          message: data.message,
        }
      }

      return {
        success: true,
        data: data.data,
        message: data.message,
      }
    } catch (error) {
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Network error',
      }
    }
  }

  // Categories
  async getCategories() {
    return this.request('/categories')
  }

  async createCategory(data: {
    name: string
    icon?: string
    color?: string
    description?: string
  }) {
    return this.request('/categories', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateCategory(id: number, data: Partial<{
    name: string
    icon: string
    color: string
    description: string
  }>) {
    return this.request(`/categories/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteCategory(id: number) {
    return this.request(`/categories/${id}`, {
      method: 'DELETE',
    })
  }

  // Feeds
  async getFeeds(params: PaginationParams = {}) {
    const query = new URLSearchParams(
      Object.entries(params).reduce((acc, [k, v]) => {
        if (v !== undefined) acc[k] = String(v)
        return acc
      }, {} as Record<string, string>)
    ).toString()

    return this.request(`/feeds${query ? `?${query}` : ''}`)
  }

  async createFeed(data: {
    url: string
    category_id?: number
    title?: string
    description?: string
    icon?: string
    color?: string
  }) {
    return this.request('/feeds', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async updateFeed(id: number, data: {
    url?: string
    category_id?: number
    title?: string
    description?: string
    icon?: string
    color?: string
    max_articles?: number
    refresh_interval?: number
    ai_summary_enabled?: boolean
  }) {
    return this.request(`/feeds/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async deleteFeed(id: number) {
    return this.request(`/feeds/${id}`, {
      method: 'DELETE',
    })
  }

  async fetchFeed(url: string) {
    return this.request('/feeds/fetch', {
      method: 'POST',
      body: JSON.stringify({ url }),
    })
  }

  async refreshFeed(id: number) {
    return this.request(`/feeds/${id}/refresh`, {
      method: 'POST',
    })
  }

  async refreshAllFeeds() {
    return this.request('/feeds/refresh-all', {
      method: 'POST',
    })
  }

  // Articles
  async getArticles(filters: ArticleFilters = {}) {
    const query = new URLSearchParams(
      Object.entries(filters).reduce((acc, [k, v]) => {
        if (v !== undefined) acc[k] = String(v)
        return acc
      }, {} as Record<string, string>)
    ).toString()

    return this.request(`/articles${query ? `?${query}` : ''}`)
  }

  async getArticle(id: number) {
    return this.request(`/articles/${id}`)
  }

  async updateArticle(
    id: number,
    data: {
      read?: boolean
      favorite?: boolean
    }
  ) {
    return this.request(`/articles/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  async bulkUpdateArticles(data: {
    ids: number[]
    read?: boolean
    favorite?: boolean
  }) {
    return this.request('/articles/bulk-update', {
      method: 'PUT',
      body: JSON.stringify(data),
    })
  }

  // OPML
  async importOpml(file: File) {
    console.log('importOpml called with file:', file.name, file.size, file.type)

    const formData = new FormData()
    formData.append('file', file)

    console.log('FormData entries:')
    for (const [key, value] of formData.entries()) {
      console.log(`  ${key}:`, value)
    }

    const url = `${API_BASE}/import-opml`
    console.log('Sending request to:', url)

    const response = await fetch(url, {
      method: 'POST',
      body: formData,
      // Don't set Content-Type - let browser set it with boundary
    })

    console.log('Response status:', response.status)

    const data = await response.json()
    console.log('Response data:', data)

    if (!response.ok) {
      return {
        success: false,
        error: data.error || data.message || 'Import failed',
      }
    }

    return {
      success: true,
      data: data.data || data,
      message: data.message,
    }
  }

  async exportOpml() {
    const url = `${API_BASE}/export-opml`
    const response = await fetch(url)
    if (!response.ok) {
      throw new Error('Export failed')
    }
    return response.blob()
  }

  // Articles stats
  async getArticlesStats() {
    return this.request('/articles/stats')
  }

  // Health check
  async healthCheck() {
    return this.request('/health')
  }

  // AI Summaries
  async getSummaries(params: { category_id?: number; page?: number; per_page?: number } = {}) {
    const query = new URLSearchParams(
      Object.entries(params).reduce((acc, [k, v]) => {
        if (v !== undefined) acc[k] = String(v)
        return acc
      }, {} as Record<string, string>)
    ).toString()

    return this.request(`/summaries${query ? `?${query}` : ''}`)
  }

  async getSummary(id: number) {
    return this.request(`/summaries/${id}`)
  }

  async generateSummary(data: {
    category_id?: number | null
    time_range?: number
    base_url: string
    api_key: string
    model: string
  }) {
    return this.request('/summaries/generate', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async autoGenerateSummary(data: {
    category_id?: number | null
    time_range?: number
    base_url: string
    api_key: string
    model: string
  }) {
    return this.request('/summaries/auto-generate', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }

  async deleteSummary(id: number) {
    return this.request(`/summaries/${id}`, {
      method: 'DELETE',
    })
  }

  // Auto-summary scheduler
  async getAutoSummaryStatus() {
    return this.request('/auto-summary/status')
  }

  async updateAutoSummaryConfig(data: {
    base_url: string
    api_key: string
    model: string
  }) {
    return this.request('/auto-summary/config', {
      method: 'POST',
      body: JSON.stringify(data),
    })
  }
}

export const api = new ApiClient()
