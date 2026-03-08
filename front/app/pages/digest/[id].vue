<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import DigestListView from '~/features/digest/components/DigestListView.vue'
import type { DigestType } from '~/api/digest'

const route = useRoute()

const digestType = computed<DigestType | null>(() => {
  const value = route.params.id
  return value === 'daily' || value === 'weekly' ? value : null
})
</script>

<template>
  <div v-if="digestType" class="h-full">
    <DigestListView :locked-type="digestType" show-digest-back />
  </div>

  <div v-else class="digest-stage flex min-h-screen items-center justify-center px-4 py-8">
    <div class="paper-card max-w-lg rounded-[32px] px-8 py-10 text-center">
      <p class="text-xs uppercase tracking-[0.32em] text-ink-light">Invalid Digest</p>
      <h1 class="mt-4 text-4xl font-black text-ink-dark">只支持 daily 或 weekly</h1>
      <p class="mt-4 text-sm leading-7 text-ink-medium">这个 digest 路径不对，回总览再选一次。</p>
      <div class="mt-8 flex justify-center gap-3">
        <button class="btn-secondary min-h-11 px-5" type="button" @click="navigateTo('/digest')">回总览</button>
        <button class="btn-ghost min-h-11 px-5" type="button" @click="navigateTo('/')">返回主界面</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.digest-stage {
  background:
    radial-gradient(circle at top left, rgba(193, 47, 47, 0.08), transparent 28%),
    radial-gradient(circle at top right, rgba(61, 107, 135, 0.1), transparent 32%),
    linear-gradient(135deg, var(--color-paper-ivory) 0%, #f4edde 42%, #faf7f2 100%);
}
</style>
