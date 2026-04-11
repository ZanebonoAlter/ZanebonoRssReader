import type { ApiResponse } from '~/types'
import type { SchedulerStatus, SchedulerTriggerResult } from '~/types/scheduler'
import { apiClient } from './client'

async function triggerSchedulerRequest(name: string): Promise<ApiResponse<SchedulerTriggerResult>> {
  return apiClient.post<SchedulerTriggerResult>(`/schedulers/${name}/trigger`, {})
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
