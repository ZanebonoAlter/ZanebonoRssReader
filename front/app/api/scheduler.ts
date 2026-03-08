import type { SchedulerStatus } from '~/types/scheduler'
import { apiClient } from './client'

export function useSchedulerApi() {
  return {
    async getSchedulersStatus() {
      return apiClient.get<SchedulerStatus[]>('/schedulers/status')
    },

    async getSchedulerStatus(name: string) {
      return apiClient.get<SchedulerStatus>(`/schedulers/${name}/status`)
    },

    async triggerScheduler(name: string) {
      return apiClient.post<{ name: string; status: string }>(`/schedulers/${name}/trigger`)
    },

    async resetSchedulerStats(name: string) {
      return apiClient.post<{ message: string }>(`/schedulers/${name}/reset-stats`)
    },
  }
}
