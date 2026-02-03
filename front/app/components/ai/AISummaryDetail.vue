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
    class="h-full overflow-y-auto glass-card rounded-2xl"
  >
    <!-- Header -->
    <div class="sticky top-0 z-10 glass-strong border-b border-white/20 px-6 py-4 flex items-center justify-between">
      <div>
        <h1 class="text-xl font-bold text-gray-800 flex items-center gap-2">
          <div class="w-8 h-8 rounded-xl bg-linear-to-br from-purple-500 to-blue-600 flex items-center justify-center shadow-lg">
            <Icon icon="mdi:brain" width="18" height="18" class="text-white" />
          </div>
          {{ summary.title }}
        </h1>
        <div class="flex items-center gap-2 mt-2.5 flex-wrap">
          <span class="px-3 py-1 rounded-full text-xs font-medium bg-purple-100/80 text-purple-700">
            {{ summary.category_name }}
          </span>
          <span class="flex items-center gap-1 text-xs text-gray-500">
            <Icon icon="mdi:file-document-multiple" width="14" height="14" />
            {{ summary.article_count }} 篇文章
          </span>
          <span class="flex items-center gap-1 text-xs text-gray-500">
            <Icon icon="mdi:clock-outline" width="14" height="14" />
            {{ formatTimeRange(summary.time_range) }}
          </span>
          <span class="flex items-center gap-1 text-xs text-gray-500">
            <Icon icon="mdi:calendar" width="14" height="14" />
            {{ formatDate(summary.created_at) }}
          </span>
        </div>
      </div>
      <button
        class="p-2.5 rounded-xl hover:bg-white/60 transition-all duration-200"
        @click="emit('close')"
      >
        <Icon icon="mdi:close" width="20" height="20" class="text-gray-500" />
      </button>
    </div>

    <!-- Content -->
    <div class="p-6">
      <!-- Summary -->
      <div class="prose prose-purple max-w-none">
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
    class="h-full flex items-center justify-center glass-card rounded-2xl"
  >
    <div class="text-center p-8">
      <div class="w-20 h-20 mx-auto mb-4 rounded-2xl bg-linear-to-br from-purple-100 to-blue-100 flex items-center justify-center">
        <Icon icon="mdi:brain" width="40" height="40" class="text-purple-400" />
      </div>
      <h3 class="text-lg font-semibold text-gray-700 mb-1">选择 AI 总结</h3>
      <p class="text-sm text-gray-400">从左侧列表中选择一个总结查看详情</p>
    </div>
  </div>
</template>

<style scoped>
/* AI Summary Content Styling */
.ai-summary-content {
  color: #374151;
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
  color: #1f2937;
}

.ai-summary-content :deep(h1) {
  font-size: 1.875em;
  padding-bottom: 0.5em;
  border-bottom: 2px solid rgba(139, 92, 246, 0.2);
}

.ai-summary-content :deep(h2) {
  font-size: 1.5em;
  padding-bottom: 0.4em;
  border-bottom: 1px solid rgba(139, 92, 246, 0.15);
}

.ai-summary-content :deep(h3) {
  font-size: 1.25em;
  color: #4b5563;
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
  color: #8b5cf6;
}

.ai-summary-content :deep(code) {
  padding: 0.2em 0.5em;
  margin: 0 0.1em;
  font-size: 0.875em;
  background: linear-gradient(135deg, rgba(139, 92, 246, 0.1), rgba(59, 130, 246, 0.1));
  border: 1px solid rgba(139, 92, 246, 0.2);
  border-radius: 6px;
  color: #7c3aed;
}

.ai-summary-content :deep(pre) {
  padding: 1.25rem;
  overflow-x: auto;
  font-size: 0.875em;
  line-height: 1.6;
  background: linear-gradient(135deg, #f8fafc, #f1f5f9);
  border: 1px solid rgba(139, 92, 246, 0.15);
  border-radius: 12px;
  margin-bottom: 1.5em;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
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
  color: #6b7280;
  background: linear-gradient(135deg, rgba(139, 92, 246, 0.05), rgba(59, 130, 246, 0.05));
  border-left: 4px solid #8b5cf6;
  border-radius: 0 12px 12px 0;
  font-style: italic;
}

.ai-summary-content :deep(a) {
  color: #7c3aed;
  text-decoration: none;
  border-bottom: 1px solid transparent;
  transition: border-color 0.2s;
}

.ai-summary-content :deep(a:hover) {
  border-bottom-color: #7c3aed;
}

.ai-summary-content :deep(strong) {
  font-weight: 700;
  color: #4c1d95;
}

.ai-summary-content :deep(em) {
  font-style: italic;
  color: #6b7280;
}

.ai-summary-content :deep(hr) {
  height: 2px;
  padding: 0;
  margin: 2.5em 0;
  background: linear-gradient(90deg, transparent, rgba(139, 92, 246, 0.3), transparent);
  border: 0;
}

.ai-summary-content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1.5em;
  background: white;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.ai-summary-content :deep(th),
.ai-summary-content :deep(td) {
  padding: 0.75em 1em;
  text-align: left;
  border-bottom: 1px solid rgba(139, 92, 246, 0.1);
}

.ai-summary-content :deep(th) {
  background: linear-gradient(135deg, rgba(139, 92, 246, 0.1), rgba(59, 130, 246, 0.1));
  font-weight: 600;
  color: #4c1d95;
}

.ai-summary-content :deep(tr:last-child td) {
  border-bottom: none;
}

.ai-summary-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 12px;
  margin: 1.5em 0;
  box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
}
</style>
