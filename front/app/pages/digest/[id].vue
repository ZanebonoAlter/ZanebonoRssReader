<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import DigestDetail from '~/components/digest/DigestDetail.vue'

const route = useRoute()
const digest = ref(null)
const loading = ref(true)

onMounted(async () => {
  const id = route.params.id
  // TODO: 获取digest详情
  loading.value = false
})
</script>

<template>
  <div class="h-full flex flex-col">
    <div v-if="loading" class="flex-1 flex items-center justify-center">
      <Icon icon="mdi:loading" width="48" class="animate-spin" />
    </div>
    <DigestDetail v-else-if="digest" :digest="digest" />
    <div v-else class="flex-1 flex items-center justify-center">
      <div class="text-center">
        <Icon icon="mdi:file-document-outline" width="64" class="text-ink-light" />
        <p class="mt-4 text-ink-light">未找到该日报周报</p>
      </div>
    </div>
  </div>
</template>
