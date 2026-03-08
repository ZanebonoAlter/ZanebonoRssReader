<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { onMounted, ref } from 'vue'
import { useDigestApi, type DigestConfig } from '~/api/digest'

const emit = defineEmits<{
  saved: []
  close: []
}>()

const digestApi = useDigestApi()

const loading = ref(false)
const saving = ref(false)
const testingFeishu = ref(false)
const testingObsidian = ref(false)
const notice = ref<{ type: 'success' | 'error' | 'info', text: string } | null>(null)

const weekdayOptions = [
  { label: '周一', value: 1 },
  { label: '周二', value: 2 },
  { label: '周三', value: 3 },
  { label: '周四', value: 4 },
  { label: '周五', value: 5 },
  { label: '周六', value: 6 },
  { label: '周日', value: 0 },
]

const config = ref<DigestConfig>({
  daily_enabled: false,
  daily_time: '09:00',
  weekly_enabled: false,
  weekly_day: 1,
  weekly_time: '09:00',
  feishu_enabled: false,
  feishu_webhook_url: '',
  feishu_push_summary: true,
  feishu_push_details: false,
  obsidian_enabled: false,
  obsidian_vault_path: '',
  obsidian_daily_digest: true,
  obsidian_weekly_digest: true,
})

function setNotice(type: 'success' | 'error' | 'info', text: string) {
  notice.value = { type, text }
}

async function loadConfig() {
  loading.value = true
  try {
    const response = await digestApi.getConfig()
    if (response.success && response.data) {
      config.value = response.data
    } else {
      setNotice('error', response.error || '配置没读出来')
    }
  } catch (error) {
    console.error('Failed to load digest config:', error)
    setNotice('error', '配置读取失败')
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const response = await digestApi.updateConfig(config.value)
    if (response.success && response.data) {
      config.value = response.data
      setNotice('success', response.message || '配置已保存')
      emit('saved')
      return
    }
    setNotice('error', response.error || '保存失败')
  } catch (error) {
    console.error('Failed to save digest config:', error)
    setNotice('error', '保存失败')
  } finally {
    saving.value = false
  }
}

async function testFeishu() {
  testingFeishu.value = true
  try {
    const response = await digestApi.testFeishu(config.value.feishu_webhook_url)
    if (response.success) {
      setNotice('success', response.message || '飞书测试消息已发出')
      return
    }
    setNotice('error', response.error || '飞书测试失败')
  } catch (error) {
    console.error('Failed to test feishu:', error)
    setNotice('error', '飞书测试失败')
  } finally {
    testingFeishu.value = false
  }
}

async function testObsidian() {
  testingObsidian.value = true
  try {
    const response = await digestApi.testObsidian(config.value.obsidian_vault_path)
    if (response.success) {
      setNotice('success', response.message || 'Obsidian 写入成功')
      return
    }
    setNotice('error', response.error || 'Obsidian 测试失败')
  } catch (error) {
    console.error('Failed to test obsidian:', error)
    setNotice('error', 'Obsidian 测试失败')
  } finally {
    testingObsidian.value = false
  }
}

onMounted(loadConfig)
</script>

<template>
  <section class="space-y-5 pb-4">
    <div class="flex items-start justify-between gap-4">
      <div>
        <p class="text-xs uppercase tracking-[0.28em] text-ink-light">Digest Setup</p>
        <h3 class="mt-2 text-2xl font-black text-ink-dark">排版后的设置区</h3>
        <p class="mt-2 text-sm leading-7 text-ink-medium">时间在上，推送在中，导出在下。别绕。</p>
      </div>
      <button class="btn-ghost min-h-11 min-w-11 px-0" type="button" aria-label="关闭设置" @click="emit('close')">
        <Icon icon="mdi:close" width="18" />
      </button>
    </div>

    <div v-if="notice" class="rounded-[22px] border px-4 py-3 text-sm"
      :class="{
        'border-[rgba(61,138,74,0.25)] bg-[rgba(61,138,74,0.08)] text-[var(--color-success)]': notice.type === 'success',
        'border-[rgba(196,47,60,0.25)] bg-[rgba(196,47,60,0.08)] text-[var(--color-error)]': notice.type === 'error',
        'border-[rgba(61,122,138,0.25)] bg-[rgba(61,122,138,0.08)] text-[var(--color-info)]': notice.type === 'info',
      }">
      {{ notice.text }}
    </div>

    <div v-if="loading" class="flex items-center justify-center py-16 text-ink-medium">
      <Icon icon="mdi:loading" width="32" class="animate-spin" />
    </div>

    <div v-else class="space-y-5">
      <section class="digest-settings-group">
        <div class="digest-settings-group__head">
          <p class="text-xs uppercase tracking-[0.26em] text-ink-light">Schedule</p>
          <h4 class="mt-2 text-lg font-black text-ink-dark">生成时间</h4>
        </div>

        <div class="space-y-4">
          <article class="digest-settings-card">
            <div class="flex items-center justify-between gap-3">
              <div>
                <h5 class="text-base font-bold text-ink-dark">日报</h5>
                <p class="mt-1 text-sm text-ink-medium">每天定时跑。</p>
              </div>
              <input v-model="config.daily_enabled" type="checkbox" class="h-5 w-5 accent-[var(--color-print-red-600)]">
            </div>
            <div v-if="config.daily_enabled" class="mt-4">
              <label class="mb-2 block text-sm font-medium text-ink-medium">日报时间</label>
              <input v-model="config.daily_time" type="time" class="input w-full">
            </div>
          </article>

          <article class="digest-settings-card">
            <div class="flex items-center justify-between gap-3">
              <div>
                <h5 class="text-base font-bold text-ink-dark">周报</h5>
                <p class="mt-1 text-sm text-ink-medium">选一天，打一版大的。</p>
              </div>
              <input v-model="config.weekly_enabled" type="checkbox" class="h-5 w-5 accent-[var(--color-ink-600)]">
            </div>
            <div v-if="config.weekly_enabled" class="mt-4 grid gap-3 sm:grid-cols-2">
              <div>
                <label class="mb-2 block text-sm font-medium text-ink-medium">星期</label>
                <select v-model="config.weekly_day" class="input w-full">
                  <option v-for="option in weekdayOptions" :key="option.value" :value="option.value">
                    {{ option.label }}
                  </option>
                </select>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-ink-medium">时间</label>
                <input v-model="config.weekly_time" type="time" class="input w-full">
              </div>
            </div>
          </article>
        </div>
      </section>

      <section class="digest-settings-group">
        <div class="digest-settings-group__head">
          <p class="text-xs uppercase tracking-[0.26em] text-ink-light">Feishu</p>
          <h4 class="mt-2 text-lg font-black text-ink-dark">飞书推送</h4>
        </div>

        <article class="digest-settings-card space-y-4">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h5 class="text-base font-bold text-ink-dark">自动推送</h5>
              <p class="mt-1 text-sm text-ink-medium">先测通，再打开。</p>
            </div>
            <input v-model="config.feishu_enabled" type="checkbox" class="h-5 w-5 accent-[var(--color-print-red-600)]">
          </div>

          <div v-if="config.feishu_enabled" class="space-y-4">
            <div>
              <label class="mb-2 block text-sm font-medium text-ink-medium">Webhook URL</label>
              <input v-model="config.feishu_webhook_url" type="text" class="input w-full" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/...">
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
              <label class="digest-toggle-row">
                <span>推送概要</span>
                <input v-model="config.feishu_push_summary" type="checkbox" class="h-4 w-4 accent-[var(--color-print-red-600)]">
              </label>
              <label class="digest-toggle-row">
                <span>推送明细</span>
                <input v-model="config.feishu_push_details" type="checkbox" class="h-4 w-4 accent-[var(--color-print-red-600)]">
              </label>
            </div>

            <button class="btn-secondary min-h-11 px-4" type="button" :disabled="testingFeishu" @click="testFeishu">
              {{ testingFeishu ? '飞书测试中...' : '测试飞书' }}
            </button>
          </div>
        </article>
      </section>

      <section class="digest-settings-group">
        <div class="digest-settings-group__head">
          <p class="text-xs uppercase tracking-[0.26em] text-ink-light">Obsidian</p>
          <h4 class="mt-2 text-lg font-black text-ink-dark">Obsidian 导出</h4>
        </div>

        <article class="digest-settings-card space-y-4">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h5 class="text-base font-bold text-ink-dark">自动写入</h5>
              <p class="mt-1 text-sm text-ink-medium">路径对了，它才会写。</p>
            </div>
            <input v-model="config.obsidian_enabled" type="checkbox" class="h-5 w-5 accent-[var(--color-ink-600)]">
          </div>

          <div v-if="config.obsidian_enabled" class="space-y-4">
            <div>
              <label class="mb-2 block text-sm font-medium text-ink-medium">Vault 路径</label>
              <input v-model="config.obsidian_vault_path" type="text" class="input w-full" placeholder="D:\\notes\\ObsidianVault">
            </div>

            <div class="grid gap-3 sm:grid-cols-2">
              <label class="digest-toggle-row">
                <span>导出日报</span>
                <input v-model="config.obsidian_daily_digest" type="checkbox" class="h-4 w-4 accent-[var(--color-ink-600)]">
              </label>
              <label class="digest-toggle-row">
                <span>导出周报</span>
                <input v-model="config.obsidian_weekly_digest" type="checkbox" class="h-4 w-4 accent-[var(--color-ink-600)]">
              </label>
            </div>

            <button class="btn-secondary min-h-11 px-4" type="button" :disabled="testingObsidian" @click="testObsidian">
              {{ testingObsidian ? '写入测试中...' : '测试 Obsidian' }}
            </button>
          </div>
        </article>
      </section>

      <div class="flex justify-end gap-3 border-t border-[var(--color-border-subtle)] pt-4">
        <button class="btn-ghost min-h-11 px-4" type="button" @click="emit('close')">
          先关掉
        </button>
        <button class="btn-primary min-h-11 px-4" type="button" :disabled="saving" @click="saveConfig">
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.digest-settings-group {
  display: grid;
  gap: 0.9rem;
}

.digest-settings-group__head {
  padding: 0 0.25rem;
}

.digest-settings-card {
  border-radius: 26px;
  border: 1px solid var(--color-border-subtle);
  background: rgba(255, 255, 255, 0.76);
  padding: 1rem;
}

.digest-toggle-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  min-height: 52px;
  border-radius: 18px;
  border: 1px solid var(--color-border-subtle);
  background: rgba(250, 247, 242, 0.9);
  padding: 0.85rem 1rem;
  font-size: 0.92rem;
  color: var(--color-ink-dark);
}
</style>
