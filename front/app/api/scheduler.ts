import type { ApiResponse } from '~/types'
import type { SchedulerStatus, SchedulerTriggerResult } from '~/types/scheduler'
import { getApiBaseUrl } from '~/utils/api'
import { apiClient } from './client'

async function triggerSchedulerRequest(name: string): Promise<ApiResponse<SchedulerTriggerResult>> {
  try {
    const response = await fetch(`${getApiBaseUrl()}/schedulers/${name}/trigger`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    const body = await response.json()

    if (!response.ok) {
      return {
        success: false,
        error: body.error || body.message || '触发失败',
        message: body.message,
        data: body.data,
      }
    }

    return {
      success: true,
      data: body.data,
      message: body.message,
    }
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : '网络错误',
    }
  }
}

export function useSchedulerApi() {
  return {
    async getSchedulersStatus() {
      return apiClient.get<SchedulerStatus[]>('/schedulers/status')
    },

    async getSchedulerStatus(name: string) {
      return apiClient.get<SchedulerStatus>(`/schedulers/${name}/status`)
    },

    async triggerScheduler(name: string) {
      return triggerSchedulerRequest(name)
    },

    async resetSchedulerStats(name: string) {
      return apiClient.post<{ message: string }>(`/schedulers/${name}/reset-stats`)
    },
  }
}
