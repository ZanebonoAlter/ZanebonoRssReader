export interface SchedulerTask {
  id: number
  name: string
  description: string
  check_interval: number
  last_execution_time: string | null
  next_execution_time: string | null
  status: 'idle' | 'running' | 'error'
  last_error: string
  last_error_time: string | null
  total_executions: number
  successful_executions: number
  failed_executions: number
  consecutive_failures: number
  last_execution_duration: number | null
  last_execution_result: string
  created_at: string
  updated_at: string
  success_rate: number
}

export interface SchedulerStatus {
  name: string
  description?: string
  running: boolean
  check_interval: string
  next_run: string
  ai_configured?: boolean
  is_executing?: boolean
  database_state?: SchedulerTask
  status?: string
  last_execution_time?: string
  last_error?: string
  concurrency?: number
}
