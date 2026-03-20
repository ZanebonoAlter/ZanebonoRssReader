import type { SummaryBatch, SummaryJob } from '~/types'

interface WSJob {
  id: string
  feed_id: number | null
  feed_name: string
  feed_icon: string
  feed_color: string
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

type WSStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

export function useSummaryWebSocket() {
  const config = useRuntimeConfig()
  const ws = ref<WebSocket | null>(null)
  const status = ref<WSStatus>('disconnected')
  const lastMessage = ref<WSMessage | null>(null)
  const reconnectAttempts = ref(0)
  const maxReconnectAttempts = 5
  const reconnectDelay = 3000

  const getWsUrl = () => {
    const apiBase = (config.public.apiBase as string) || 'http://localhost:5000'
    const wsBase = apiBase.replace(/^http/, 'ws')
    return `${wsBase}/ws`
  }

  const connect = () => {
    if (ws.value?.readyState === WebSocket.OPEN) return Promise.resolve()
    if (ws.value?.readyState === WebSocket.CONNECTING) {
      return new Promise<void>((resolve, reject) => {
        const startedAt = Date.now()
        const timer = setInterval(() => {
          if (ws.value?.readyState === WebSocket.OPEN) {
            clearInterval(timer)
            resolve()
            return
          }
          if (status.value === 'error' || Date.now() - startedAt > 5000) {
            clearInterval(timer)
            reject(new Error('WebSocket connect timeout'))
          }
        }, 100)
      })
    }

    status.value = 'connecting'

    return new Promise<void>((resolve, reject) => {
      try {
      const url = getWsUrl()
      console.log('[WebSocket] Connecting to:', url)

      ws.value = new WebSocket(url)

      ws.value.onopen = () => {
        console.log('[WebSocket] Connected')
        status.value = 'connected'
        reconnectAttempts.value = 0
        resolve()
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

        if (!event.wasClean && reconnectAttempts.value < maxReconnectAttempts) {
          reconnectAttempts.value++
          console.log(`[WebSocket] Reconnecting in ${reconnectDelay}ms (attempt ${reconnectAttempts.value})`)
          setTimeout(connect, reconnectDelay)
        }
      }

      ws.value.onerror = (error) => {
        console.error('[WebSocket] Error:', error)
        status.value = 'error'
        reject(new Error('WebSocket error'))
      }
      } catch (err) {
        console.error('[WebSocket] Failed to connect:', err)
        status.value = 'error'
        reject(err instanceof Error ? err : new Error('WebSocket connect failed'))
      }
    })
  }

  const disconnect = () => {
    if (ws.value) {
      ws.value.close(1000, 'Manual disconnect')
      ws.value = null
      status.value = 'disconnected'
    }
  }

  const clearLastMessage = () => {
    lastMessage.value = null
  }

  const toBatchData = (message: WSMessage): SummaryBatch => {
    const jobs: SummaryJob[] = []

    if (message.jobs && message.jobs.length > 0) {
      for (const job of message.jobs) {
        jobs.push({
          id: job.id,
          batch_id: message.batch_id,
          feed_id: job.feed_id,
          feed_name: job.feed_name,
          feed_icon: job.feed_icon,
          feed_color: job.feed_color,
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
      jobs.push({
        id: message.current_job.id,
        batch_id: message.batch_id,
        feed_id: message.current_job.feed_id,
        feed_name: message.current_job.feed_name,
        feed_icon: message.current_job.feed_icon,
        feed_color: message.current_job.feed_color,
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

  onUnmounted(() => {
    disconnect()
  })

  return {
    ws,
    status,
    lastMessage,
    connect,
    disconnect,
    clearLastMessage,
    toBatchData,
  }
}
