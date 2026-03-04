<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { ref, onMounted } from 'vue'
import { useDigestApi } from '~/composables/api/digest'

const digestApi = useDigestApi()
const loading = ref(false)
const digests = ref([])

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
        <button class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700">
          刷新
        </button>
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
