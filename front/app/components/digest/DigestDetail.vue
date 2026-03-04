<script setup lang="ts">
import { ref, computed } from 'vue'
import { marked } from 'marked'

const props = defineProps<{
  digest: any
}>()

const renderedContent = computed(() => {
  if (!props.digest) return ''
  return marked(props.digest.content)
})
</script>

<template>
  <div class="digest-detail">
    <div class="prose max-w-none">
      <div v-html="renderedContent" class="markdown-content" />
    </div>
  </div>
</template>

<style scoped>
.markdown-content {
  color: var(--color-ink-dark);
  line-height: 1.75;
}

.markdown-content :deep(h1),
.markdown-content :deep(h2),
.markdown-content :deep(h3) {
  font-weight: 700;
  margin-top: 1.75em;
  margin-bottom: 0.75em;
}

.markdown-content :deep(p) {
  margin-bottom: 1.25em;
}

.markdown-content :deep(a) {
  color: var(--color-ink-500);
  text-decoration: none;
  border-bottom: 1px solid transparent;
}

.markdown-content :deep(a:hover) {
  border-bottom-color: var(--color-ink-500);
}
</style>
