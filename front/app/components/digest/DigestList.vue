<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { ref, onMounted } from 'vue'
import { useDigestApi } from '~/composables/api/digest'
import DigestSettings from './DigestSettings.vue'

const digestApi = useDigestApi()
const loading = ref(false)
const digests = ref([])
const showSettings = ref(false)

onMounted(async () => {
  loading.value = true
  // TODO: 获取日报周报列表
  loading.value = false
})
</script>

<template>
  <div class="digest-list">
    <div class="flex items-center justify-between mb-6">
      <h2 class="text-xl font-bold">日报周报</h2>
      <div class="flex gap-2">
        <button
          @click="showSettings = true"
          class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700"
        >
          设置
        </button>
      </div>
    </div>

    <!-- 对话框 -->
    <div
      v-if="showSettings"
      class="fixed inset-0 bg-black/50 flex items-center justify-center z-50"
      @click.self="showSettings = false"
    >
      <div class="bg-white rounded-lg p-6 max-w-2xl w-full max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-bold">设置</h3>
          <button @click="showSettings = false" class="text-ink-medium hover:text-ink-dark">
            <Icon icon="mdi:close" width="24" height="24" />
          </button>
        </div>
        <DigestSettings />
      </div>
    </div>

    <div v-if="loading" class="text-center py-12">
      <Icon icon="mdi:loading" width="48" class="animate-spin" />
    </div>

    <div v-else class="space-y-4">
      <!-- TODO: 显示日报周报列表 -->
      <div class="text-center text-ink-light py-12">
        暂无日报周报
      </div>
    </div>
  </div>
</template>
