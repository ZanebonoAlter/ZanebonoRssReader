<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { ref, onMounted } from 'vue'
import { useDigestApi, type DigestConfig } from '~/composables/api/digest'

const digestApi = useDigestApi()
const loading = ref(false)
const saving = ref(false)
const config = ref<DigestConfig>({
  daily_enabled: true,
  daily_time: '09:00',
  weekly_enabled: true,
  weekly_day: 'Monday',
  weekly_time: '09:00',
  feishu_enabled: true,
  feishu_webhook_url: '',
  feishu_push_summary: true,
  feishu_push_details: true,
  obsidian_enabled: true,
  obsidian_vault_path: '',
  obsidian_daily_digest: true,
  obsidian_weekly_digest: true,
})

onMounted(async () => {
  await loadConfig()
})

async function loadConfig() {
  loading.value = true
  try {
    const response = await digestApi.getConfig()
    if (response.success && response.data) {
      config.value = response.data
    }
  } catch (error) {
    console.error('Failed to load config:', error)
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const response = await digestApi.updateConfig(config.value)
    if (response.success) {
      alert('配置已保存')
    }
  } catch (error) {
    console.error('Failed to save config:', error)
    alert('保存失败')
  } finally {
    saving.value = false
  }
}

async function testFeishu() {
  try {
    const response = await digestApi.testFeishu()
    if (response.success) {
      alert('测试消息已发送，请检查飞书')
    }
  } catch (error) {
    console.error('Failed to test Feishu:', error)
    alert('测试失败')
  }
}

async function testObsidian() {
  try {
    const response = await digestApi.testObsidian()
    if (response.success) {
      alert('测试文件已写入，请检查Obsidian vault')
    }
  } catch (error) {
    console.error('Failed to test Obsidian:', error)
    alert('测试失败')
  }
}
</script>

<template>
  <div class="digest-settings">
    <h3 class="text-lg font-bold mb-4">日报周报设置</h3>

    <div v-if="loading" class="text-center py-12">
      <Icon icon="mdi:loading" width="48" class="animate-spin" />
    </div>

    <div v-else class="space-y-6">
      <!-- 基础设置 -->
      <div class="space-y-4">
        <h4 class="font-semibold">基础设置</h4>

        <div class="flex items-center justify-between">
          <label>启用日报</label>
          <input
            v-model="config.daily_enabled"
            type="checkbox"
            class="w-5 h-5"
          >
        </div>

        <div v-if="config.daily_enabled" class="pl-4">
          <label class="block text-sm mb-1">日报时间</label>
          <input
            v-model="config.daily_time"
            type="time"
            class="border rounded px-3 py-2"
          >
        </div>

        <div class="flex items-center justify-between">
          <label>启用周报</label>
          <input
            v-model="config.weekly_enabled"
            type="checkbox"
            class="w-5 h-5"
          >
        </div>

        <div v-if="config.weekly_enabled" class="pl-4 space-y-2">
          <div>
            <label class="block text-sm mb-1">周报星期</label>
            <select v-model="config.weekly_day" class="border rounded px-3 py-2">
              <option value="Monday">周一</option>
              <option value="Tuesday">周二</option>
              <option value="Wednesday">周三</option>
              <option value="Thursday">周四</option>
              <option value="Friday">周五</option>
              <option value="Saturday">周六</option>
              <option value="Sunday">周日</option>
            </select>
          </div>
          <div>
            <label class="block text-sm mb-1">周报时间</label>
            <input
              v-model="config.weekly_time"
              type="time"
              class="border rounded px-3 py-2"
            >
          </div>
        </div>
      </div>

      <!-- 飞书设置 -->
      <div class="space-y-4">
        <h4 class="font-semibold">飞书推送</h4>

        <div class="flex items-center justify-between">
          <label>启用飞书推送</label>
          <input
            v-model="config.feishu_enabled"
            type="checkbox"
            class="w-5 h-5"
          >
        </div>

        <div v-if="config.feishu_enabled" class="pl-4 space-y-2">
          <div>
            <label class="block text-sm mb-1">Webhook URL</label>
            <input
              v-model="config.feishu_webhook_url"
              type="text"
              placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..."
              class="w-full border rounded px-3 py-2"
            >
          </div>

          <div class="flex items-center justify-between">
            <label>推送汇总通知</label>
            <input
              v-model="config.feishu_push_summary"
              type="checkbox"
              class="w-5 h-5"
            >
          </div>

          <div class="flex items-center justify-between">
            <label>推送细节通知</label>
            <input
              v-model="config.feishu_push_details"
              type="checkbox"
              class="w-5 h-5"
            >
          </div>

          <button
            @click="testFeishu"
            class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700"
          >
            测试推送
          </button>
        </div>
      </div>

      <!-- Obsidian设置 -->
      <div class="space-y-4">
        <h4 class="font-semibold">Obsidian导出</h4>

        <div class="flex items-center justify-between">
          <label>启用Obsidian导出</label>
          <input
            v-model="config.obsidian_enabled"
            type="checkbox"
            class="w-5 h-5"
          >
        </div>

        <div v-if="config.obsidian_enabled" class="pl-4 space-y-2">
          <div>
            <label class="block text-sm mb-1">Vault路径</label>
            <input
              v-model="config.obsidian_vault_path"
              type="text"
              placeholder="/path/to/ObsidianVault"
              class="w-full border rounded px-3 py-2"
            >
          </div>

          <div class="flex items-center justify-between">
            <label>导出日报</label>
            <input
              v-model="config.obsidian_daily_digest"
              type="checkbox"
              class="w-5 h-5"
            >
          </div>

          <div class="flex items-center justify-between">
            <label>导出周报</label>
            <input
              v-model="config.obsidian_weekly_digest"
              type="checkbox"
              class="w-5 h-5"
            >
          </div>

          <button
            @click="testObsidian"
            class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700"
          >
            测试写入
          </button>
        </div>
      </div>

      <!-- 保存按钮 -->
      <div class="pt-4 border-t">
        <button
          @click="saveConfig"
          :disabled="saving"
          class="w-full px-4 py-3 bg-amber-600 text-white rounded-lg hover:bg-amber-700 disabled:opacity-50"
        >
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>
    </div>
  </div>
</template>
