import { apiClient } from './client'

export interface DigestConfig {
  id?: number
  daily_enabled: boolean
  daily_time: string
  weekly_enabled: boolean
  weekly_day: string
  weekly_time: string
  feishu_enabled: boolean
  feishu_webhook_url: string
  feishu_push_summary: boolean
  feishu_push_details: boolean
  obsidian_enabled: boolean
  obsidian_vault_path: string
  obsidian_daily_digest: boolean
  obsidian_weekly_digest: boolean
}

export function useDigestApi() {
  return {
    async getConfig() {
      return apiClient.get<DigestConfig>('/digest/config')
    },

    async updateConfig(config: DigestConfig) {
      return apiClient.put<DigestConfig>('/digest/config', config)
    },

    async testFeishu() {
      return apiClient.post('/digest/test-feishu', {})
    },

    async testObsidian() {
      return apiClient.post('/digest/test-obsidian', {})
    }
  }
}
