const API_BASE = 'http://localhost:5000/api'

interface AISummaryRequest {
  base_url: string
  api_key: string
  model: string
  title: string
  content: string
  language?: string
}

interface AISummaryData {
  one_sentence: string
  key_points: string[]
  takeaways: string[]
  tags: string[]
}

interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
  error?: string
}

export const useAI = () => {
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Reactive AI settings
  const aiSettings = ref<any>(null)

  // Load settings on client side
  if (import.meta.client) {
    const loadSettings = () => {
      const settings = localStorage.getItem('aiSettings')
      if (settings) {
        try {
          aiSettings.value = JSON.parse(settings)
        } catch (e) {
          console.error('Failed to parse AI settings:', e)
          aiSettings.value = null
        }
      } else {
        aiSettings.value = null
      }
    }

    // Load initially
    loadSettings()

    // Listen for storage changes
    window.addEventListener('storage', (e) => {
      if (e.key === 'aiSettings') {
        loadSettings()
      }
    })
  }

  // Get AI settings from localStorage (non-reactive version for API calls)
  const getAISettings = () => {
    if (import.meta.client) {
      const settings = localStorage.getItem('aiSettings')
      if (settings) {
        return JSON.parse(settings)
      }
    }
    return null
  }

  // Check if AI is enabled (reactive computed)
  const isAIEnabled = computed(() => {
    return aiSettings.value?.summaryEnabled || false
  })

  // Summarize an article
  const summarizeArticle = async (
    title: string,
    content: string,
    language: string = 'zh'
  ): Promise<ApiResponse<AISummaryData>> => {
    loading.value = true
    error.value = null

    try {
      const settings = getAISettings()

      if (!settings || !settings.summaryEnabled) {
        throw new Error('AI 功能未启用')
      }

      if (!settings.baseURL || !settings.apiKey || !settings.model) {
        throw new Error('AI 配置不完整，请检查设置')
      }

      const requestData: AISummaryRequest = {
        base_url: settings.baseURL,
        api_key: settings.apiKey,
        model: settings.model,
        title,
        content,
        language
      }

      const response = await fetch(`${API_BASE}/ai/summarize`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestData),
      })

      const data = await response.json()

      if (!response.ok) {
        return {
          success: false,
          error: data.error || '总结生成失败',
        }
      }

      loading.value = false
      return {
        success: true,
        data: data.data,
      }
    } catch (err) {
      loading.value = false
      const errorMessage = err instanceof Error ? err.message : '未知错误'
      error.value = errorMessage
      return {
        success: false,
        error: errorMessage,
      }
    }
  }

  // Test AI connection
  const testConnection = async (): Promise<ApiResponse<void>> => {
    loading.value = true
    error.value = null

    try {
      const settings = getAISettings()

      if (!settings || !settings.baseURL || !settings.apiKey || !settings.model) {
        throw new Error('AI 配置不完整')
      }

      const response = await fetch(`${API_BASE}/ai/test`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          base_url: settings.baseURL,
          api_key: settings.apiKey,
          model: settings.model,
        }),
      })

      const data = await response.json()

      loading.value = false

      if (!response.ok) {
        return {
          success: false,
          error: data.error || '连接测试失败',
        }
      }

      return {
        success: true,
        message: data.message,
      }
    } catch (err) {
      loading.value = false
      const errorMessage = err instanceof Error ? err.message : '未知错误'
      error.value = errorMessage
      return {
        success: false,
        error: errorMessage,
      }
    }
  }

  return {
    loading,
    error,
    isAIEnabled,
    summarizeArticle,
    testConnection,
    getAISettings,
    aiSettings,
  }
}
