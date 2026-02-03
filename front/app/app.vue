<script setup lang="ts">
import { Icon } from '@iconify/vue'

const apiStore = useApiStore()
const loading = ref(true)
const error = ref<string | null>(null)

// 初始化 API Store
onMounted(async () => {
  try {
    await apiStore.initialize()
    // 同步 API 数据到本地 Store
    apiStore.syncToLocalStores()
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载数据失败'
    console.error('初始化错误:', e)
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div v-if="loading" class="h-screen flex items-center justify-center">
    <div class="text-center">
      <Icon icon="mdi:loading" width="48" height="48" class="animate-spin text-blue-500 mx-auto mb-4" />
      <p class="text-gray-600">正在加载...</p>
    </div>
  </div>

  <div v-else-if="error" class="h-screen flex items-center justify-center">
    <div class="text-center max-w-md">
      <Icon icon="mdi:alert-circle" width="48" height="48" class="text-red-500 mx-auto mb-4" />
      <h2 class="text-xl font-bold text-gray-900 mb-2">加载失败</h2>
      <p class="text-gray-600 mb-4">{{ error }}</p>
      <button class="btn btn-primary" @click="$router.go(0)">
        重新加载
      </button>
    </div>
  </div>

  <FeedLayout v-else />
</template>
