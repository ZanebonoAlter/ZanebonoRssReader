import type { SummaryBatch, SummaryJob } from '~/types'

// WebSocket消息类型
interface WSJob {
  id: string
  category_id: number | null
  category_name: string
  status: 'pending' | 'processing' | 'completed' | 'failed'
  error_message?: string
  error_code?: string
  result_id?: number
}

interface WSMessage {
  type: 'progress'
  batch_id: string
  status: 'pending' | 'processing' | 'completed'
  total_jobs: number
  completed_jobs: number
  failed_jobs: number
  current_job?: WSJob
  jobs?: WSJob[]
}

// WebSocket连接状态
type WSStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

export function useSummaryWebSocket() {
  const config = useRuntimeConfig()
  const ws = ref<WebSocket | null>(null)
  const status = ref<WSStatus>('disconnected')
  const lastMessage = ref<WSMessage | null>(null)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5
  const reconnectDelay = 3000 // 3秒

  // 获取WebSocket URL
  const getWsUrl = () => {
    const apiBase = (config.public.apiBase as string) || 'http://localhost:5000'
    const wsBase = apiBase.replace(/^http/, 'ws')
    return `${wsBase}/ws`
  }

  // 连接WebSocket
  const connect = () => {
    if (ws.value?.readyState === WebSocket.OPEN) {
      return
    }

    status.value = 'connecting'

    try {
      const url = getWsUrl()
      console.log('[WebSocket] Connecting to:', url)
      
      ws.value = new WebSocket(url)

      ws.value.onopen = () => {
        console.log('[WebSocket] Connected')
        status.value = 'connected'
        reconnectAttempts.value = 0
      }

      ws.value.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data)
          console.log('[WebSocket] Received:', message)
          lastMessage.value = message
        } catch (err) {
          console.error('[WebSocket] Failed to parse message:', err)
        }
      }

      ws.value.onclose = (event) => {
        console.log('[WebSocket] Closed:', event.code, event.reason)
        status.value = 'disconnected'
        ws.value = null

        // 非手动关闭，尝试重连
        if (!event.wasClean && reconnectAttempts.value < maxReconnectAttempts) {
          reconnectAttempts.value++
          console.log(`[WebSocket] Reconnecting in ${reconnectDelay}ms (attempt ${reconnectAttempts.value})`)
          setTimeout(connect, reconnectDelay)
        }
      }

      ws.value.onerror = (error) => {
        console.error('[WebSocket] Error:', error)
        status.value = 'error'
      }
    } catch (err) {
      console.error('[WebSocket] Failed to connect:', err)
      status.value = 'error'
    }
  }

  // 断开连接
  const disconnect = () => {
    if (ws.value) {
      ws.value.close(1000, 'Manual disconnect')
      ws.value = null
      status.value = 'disconnected'
    }
  }

  // 将WS消息转换为批次数据
  const toBatchData = (message: WSMessage): SummaryBatch => {
    // 优先使用完整的jobs列表，如果没有则使用current_job
    const jobs: SummaryJob[] = []
    
    if (message.jobs && message.jobs.length > 0) {
      // 使用完整的任务列表
      for (const job of message.jobs) {
        jobs.push({
          id: job.id,
          batch_id: message.batch_id,
          category_id: job.category_id,
          category_name: job.category_name,
          status: job.status,
          error_message: job.error_message,
          error_code: job.error_code,
          result_id: job.result_id,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        })
      }
    } else if (message.current_job) {
      // 兼容旧格式：只使用当前任务
      jobs.push({
        id: message.current_job.id,
        batch_id: message.batch_id,
        category_id: message.current_job.category_id,
        category_name: message.current_job.category_name,
        status: message.current_job.status,
        error_message: message.current_job.error_message,
        error_code: message.current_job.error_code,
        result_id: message.current_job.result_id,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
      })
    }

    return {
      id: message.batch_id,
      status: message.status,
      total_jobs: message.total_jobs,
      completed_jobs: message.completed_jobs,
      failed_jobs: message.failed_jobs,
      created_at: new Date().toISOString(),
      jobs,
    }
  }

  // 在组件卸载时断开连接
  onUnmounted(() => {
    disconnect()
  })

  return {
    ws,
    status,
    lastMessage,
    connect,
    disconnect,
    toBatchData,
  }
}
