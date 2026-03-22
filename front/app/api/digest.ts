import { apiClient } from './client'

export type DigestType = 'daily' | 'weekly'

export interface DigestConfig {
  id?: number
  daily_enabled: boolean
  daily_time: string
  weekly_enabled: boolean
  weekly_day: number
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

export interface DigestPreviewTopicTag {
  slug: string
  label: string
  category: string
  icon?: string
  aliases?: string[]
  score: number
}

export interface DigestAggregatedTag {
  slug: string
  label: string
  category: string
  kind?: string
  icon?: string
  score: number
  article_count: number
}

export interface DigestPreviewSummary {
  id: number
  feed_id: number | null
  feed_name: string
  feed_icon: string
  feed_color: string
  category_id: number
  category_name: string
  summary_text: string
  article_count: number
  article_ids: number[]
  topics: DigestPreviewTopicTag[]
  aggregated_tags: DigestAggregatedTag[]
  created_at: string
}

export interface DigestPreviewCategory {
  id: number
  name: string
  feed_count: number
  summary_count: number
  summaries: DigestPreviewSummary[]
}

export interface DigestPreview {
  type: DigestType
  title: string
  period_label: string
  generated_at: string
  anchor_date: string
  category_count: number
  summary_count: number
  markdown: string
  categories: DigestPreviewCategory[]
  default_category_id?: number | null
  default_summary_id?: number | null
}

export interface DigestStatus {
  running: boolean
  daily_enabled: boolean
  weekly_enabled: boolean
  daily_time: string
  weekly_day: number
  weekly_time: string
  next_runs: string[]
  active_jobs: number
}

export interface DigestRunResult {
  preview: DigestPreview
  sent_to_feishu: boolean
  exported_to_obsidian: boolean
  sent_to_open_notebook?: boolean
}

export interface OpenNotebookConfig {
  enabled: boolean
  base_url: string
  api_key: string
  model: string
  target_notebook: string
  prompt_mode: 'digest_summary'
  auto_send_daily: boolean
  auto_send_weekly: boolean
  export_back_to_obsidian: boolean
}

export interface OpenNotebookRunResult {
  digest_type: DigestType
  anchor_date: string
  source_markdown: string
  summary_markdown: string
  remote_id?: string
  remote_url?: string
}

function withDateQuery(endpoint: string, date?: string) {
  const query = apiClient.buildQueryParams({ date })
  return query ? `${endpoint}?${query}` : endpoint
}

export function useDigestApi() {
  return {
    async getConfig() {
      return apiClient.get<DigestConfig>('/digest/config')
    },

    async getStatus() {
      return apiClient.get<DigestStatus>('/digest/status')
    },

    async getPreview(type: DigestType, date?: string) {
      return apiClient.get<DigestPreview>(withDateQuery(`/digest/preview/${type}`, date))
    },

    async updateConfig(config: DigestConfig) {
      return apiClient.put<DigestConfig>('/digest/config', config)
    },

    async runNow(type: DigestType, date?: string) {
      return apiClient.post<DigestRunResult>(withDateQuery(`/digest/run/${type}`, date), {})
    },

    async getOpenNotebookConfig() {
      return apiClient.get<OpenNotebookConfig>('/digest/open-notebook/config')
    },

    async updateOpenNotebookConfig(config: OpenNotebookConfig) {
      return apiClient.put<OpenNotebookConfig>('/digest/open-notebook/config', config)
    },

    async sendToOpenNotebook(type: DigestType, date?: string) {
      return apiClient.post<OpenNotebookRunResult>(withDateQuery(`/digest/open-notebook/${type}`, date), {})
    },

    async testFeishu(webhookURL?: string) {
      return apiClient.post('/digest/test-feishu', {
        webhook_url: webhookURL,
      })
    },

    async testObsidian(vaultPath?: string) {
      return apiClient.post('/digest/test-obsidian', {
        vault_path: vaultPath,
      })
    },
  }
}
