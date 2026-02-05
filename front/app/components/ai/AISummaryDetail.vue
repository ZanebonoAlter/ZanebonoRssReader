<script setup lang="ts">
import { Icon } from "@iconify/vue"
import { marked } from 'marked'

interface AISummary {
  id: number
  category_id: number | null
  title: string
  summary: string
  key_points: string
  articles: string
  article_count: number
  time_range: number
  created_at: string
  updated_at: string
  category_name: string
}

const props = defineProps<{
  summary: AISummary | null
}>()

const emit = defineEmits<{
  'close': []
}>()

// Watch for summary changes to log for debugging
watch(() => props.summary, (newSummary) => {
  console.log('Summary changed:', newSummary?.id, newSummary?.title)
}, { immediate: true })

const renderedSummary = computed(() => {
  if (!props.summary) return ''
  return marked(props.summary.summary)
})

const formatDate = (dateString: string): string => {
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

const formatTimeRange = (minutes: number): string => {
  const hours = Math.floor(minutes / 60)
  if (hours >= 24) {
    const days = Math.floor(hours / 24)
    return `${days} 天`
  }
  return `${hours} 小时`
}
</script>

<template>
  <div
    v-if="summary"
    class="h-full flex flex-col"
  >
    <!-- Header -->
    <div class="flex-shrink-0 bg-white/95 backdrop-blur-sm border-b border-ink-200 px-6 py-4 flex items-center justify-between shadow-subtle">
      <div>
        <h1 class="text-xl font-bold text-ink-black flex items-center gap-2">
          <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-ink-500 to-ink-700 flex items-center justify-center shadow-md">
            <Icon icon="mdi:brain" width="18" height="18" class="text-white" />
          </div>
          {{ summary.title }}
        </h1>
        <div class="flex items-center gap-2 mt-2.5 flex-wrap">
          <span class="px-3 py-1 rounded-full text-xs font-medium bg-ink-100 text-ink-700">
            {{ summary.category_name }}
          </span>
          <span class="flex items-center gap-1 text-xs text-ink-medium">
            <Icon icon="mdi:file-document-multiple" width="14" height="14" />
            {{ summary.article_count }} 篇文章
          </span>
          <span class="flex items-center gap-1 text-xs text-ink-medium">
            <Icon icon="mdi:clock-outline" width="14" height="14" />
            {{ formatTimeRange(summary.time_range) }}
          </span>
          <span class="flex items-center gap-1 text-xs text-ink-medium">
            <Icon icon="mdi:calendar" width="14" height="14" />
            {{ formatDate(summary.created_at) }}
          </span>
        </div>
      </div>
      <button
        class="p-2.5 rounded-lg hover:bg-ink-50 transition-all duration-200"
        @click="emit('close')"
      >
        <Icon icon="mdi:close" width="20" height="20" class="text-ink-medium" />
      </button>
    </div>

    <!-- Content -->
    <div class="flex-1 overflow-y-auto p-6">
      <!-- Summary -->
      <div class="prose max-w-none">
        <div
          v-html="renderedSummary"
          class="ai-summary-content"
        />
      </div>
    </div>
  </div>

  <!-- Empty state -->
  <div
    v-else
    class="h-full flex items-center justify-center paper-card rounded-lg"
  >
    <div class="text-center p-8">
      <div class="w-20 h-20 mx-auto mb-4 rounded-lg bg-gradient-to-br from-ink-100 to-paper-warm flex items-center justify-center">
        <Icon icon="mdi:brain" width="40" height="40" class="text-ink-400" />
      </div>
      <h3 class="text-lg font-semibold text-ink-dark mb-1">选择 AI 总结</h3>
      <p class="text-sm text-ink-light">从左侧列表中选择一个总结查看详情</p>
    </div>
  </div>
</template>

<style scoped>
/* AI Summary Content Styling - Editorial Theme */
.ai-summary-content {
  color: var(--color-ink-dark);
  line-height: 1.75;
}

.ai-summary-content :deep(h1),
.ai-summary-content :deep(h2),
.ai-summary-content :deep(h3),
.ai-summary-content :deep(h4),
.ai-summary-content :deep(h5),
.ai-summary-content :deep(h6) {
  font-weight: 700;
  margin-top: 1.75em;
  margin-bottom: 0.75em;
  line-height: 1.3;
  color: var(--color-ink-black);
  letter-spacing: -0.01em;
}

.ai-summary-content :deep(h1) {
  font-size: 1.875em;
  padding-bottom: 0.5em;
  border-bottom: 2px solid var(--color-ink-300);
}

.ai-summary-content :deep(h2) {
  font-size: 1.5em;
  padding-bottom: 0.4em;
  border-bottom: 1px solid var(--color-ink-200);
}

.ai-summary-content :deep(h3) {
  font-size: 1.25em;
  color: var(--color-ink-dark);
}

.ai-summary-content :deep(p) {
  margin-top: 0;
  margin-bottom: 1.25em;
}

.ai-summary-content :deep(ul),
.ai-summary-content :deep(ol) {
  margin-top: 0;
  margin-bottom: 1.25em;
  padding-left: 1.75em;
}

.ai-summary-content :deep(li) {
  margin-bottom: 0.5em;
  position: relative;
}

.ai-summary-content :deep(li)::marker {
  color: var(--color-ink-500);
}

.ai-summary-content :deep(code) {
  padding: 0.2em 0.5em;
  margin: 0 0.1em;
  font-size: 0.875em;
  background: rgba(59, 107, 135, 0.08);
  border: 1px solid rgba(59, 107, 135, 0.15);
  border-radius: 4px;
  color: var(--color-ink-700);
}

.ai-summary-content :deep(pre) {
  padding: 1.25rem;
  overflow-x: auto;
  font-size: 0.875em;
  line-height: 1.6;
  background: var(--color-paper-warm);
  border: 1px solid var(--color-border-medium);
  border-radius: 0.5rem;
  margin-bottom: 1.5em;
  box-shadow: var(--shadow-subtle);
}

.ai-summary-content :deep(pre code) {
  padding: 0;
  margin: 0;
  font-size: 100%;
  background: transparent;
  border: none;
  color: inherit;
}

.ai-summary-content :deep(blockquote) {
  padding: 1em 1.25em;
  margin: 0 0 1.5em 0;
  color: var(--color-ink-medium);
  background: rgba(59, 107, 135, 0.04);
  border-left: 3px solid var(--color-ink-400);
  border-radius: 0 0.5rem 0.5rem 0;
  font-style: italic;
}

.ai-summary-content :deep(a) {
  color: var(--color-ink-500);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color 0.2s;
}

.ai-summary-content :deep(a:hover) {
  border-bottom-color: var(--color-ink-500);
}

.ai-summary-content :deep(strong) {
  font-weight: 700;
  color: var(--color-ink-900);
}

.ai-summary-content :deep(em) {
  font-style: italic;
  color: var(--color-ink-medium);
}

.ai-summary-content :deep(hr) {
  height: 2px;
  padding: 0;
  margin: 2.5em 0;
  background: linear-gradient(90deg, transparent, var(--color-ink-300), transparent);
  border: 0;
}

.ai-summary-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1.5em;
  background: white;
  border-radius: 0.5rem;
  overflow: hidden;
  box-shadow: var(--shadow-subtle);
}

.ai-summary-content :deep(th),
.ai-summary-content :deep(td) {
  padding: 0.75em 1em;
  text-align: left;
  border-bottom: 1px solid var(--color-border-subtle);
}

.ai-summary-content :deep(th) {
  background: rgba(59, 107, 135, 0.08);
  font-weight: 600;
  color: var(--color-ink-900);
}

.ai-summary-content :deep(tr:last-child td) {
  border-bottom: none;
}

.ai-summary-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 0.5rem;
  margin: 1.5em 0;
  box-shadow: var(--shadow-medium);
}
</style>
