<script setup lang="ts">
import { Icon } from '@iconify/vue'

interface Props {
  title: string
  content: string
  language?: string
}

const props = withDefaults(defineProps<Props>(), {
  language: 'zh'
})

const emit = defineEmits<{
  close: []
}>()

const { loading, error, summarizeArticle } = useAI()

interface SummaryData {
  one_sentence: string
  key_points: string[]
  takeaways: string[]
  tags: string[]
}

const summary = ref<SummaryData | null>(null)
const isGenerating = ref(false)
const showFullContent = ref(false)

// Generate summary
async function generateSummary() {
  isGenerating.value = true
  summary.value = null
  error.value = null

  const result = await summarizeArticle(
    props.title,
    props.content,
    props.language
  )

  isGenerating.value = false

  if (result.success && result.data) {
    summary.value = result.data
  } else {
    error.value = result.error || '生成总结失败'
  }
}

// Auto-generate on mount if content is available
onMounted(() => {
  if (props.content) {
    generateSummary()
  }
})

// Close summary
function close() {
  emit('close')
}

// Get color for tags
function getTagColor(index: number): string {
  const colors = [
    'bg-ink-100 text-ink-700',
    'bg-teal-50 text-accent-teal',
    'bg-amber-50 text-accent-amber',
    'bg-rose-50 text-print-red-600',
    'bg-emerald-50 text-success',
    'bg-indigo-50 text-accent-indigo',
  ]
  return colors[index % colors.length] || 'bg-gray-100 text-gray-700'
}
</script>

<template>
  <div class="ai-summary bg-gradient-to-br from-ink-50 to-paper-cream rounded-xl border border-ink-200 overflow-hidden">
    <!-- Header -->
    <div class="px-4 py-3 bg-gradient-to-r from-ink-500 to-ink-700 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Icon icon="mdi:brain" width="18" height="18" class="text-white" />
        <span class="font-semibold text-white">AI 总结分析</span>
      </div>
      <button
        class="p-1 hover:bg-white/20 rounded-lg transition-colors"
        @click="close"
      >
        <Icon icon="mdi:close" width="18" height="18" class="text-white" />
      </button>
    </div>

    <!-- Loading State -->
    <div v-if="isGenerating" class="p-6 flex flex-col items-center justify-center text-center">
      <Icon
        icon="mdi:loading"
        width="48"
        height="48"
        class="animate-spin text-ink-500 mb-4"
      />
      <p class="text-gray-600 font-medium">AI 正在分析文章...</p>
      <p class="text-sm text-gray-500 mt-1">这可能需要几秒钟</p>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="p-6 text-center">
      <Icon
        icon="mdi:alert-circle"
        width="48"
        height="48"
        class="text-red-500 mx-auto mb-4"
      />
      <p class="text-red-600 font-medium mb-2">生成失败</p>
      <p class="text-sm text-gray-600 mb-4">{{ error }}</p>
      <button
        class="px-4 py-2 bg-ink-600 text-white rounded-lg hover:bg-ink-700 transition-colors"
        @click="generateSummary"
      >
        重试
      </button>
    </div>

    <!-- Summary Content -->
    <div v-else-if="summary" class="p-5 space-y-5">
      <!-- One Sentence Summary -->
      <div class="bg-white/80 rounded-lg p-4 shadow-subtle">
        <div class="flex items-start gap-2 mb-2">
          <Icon icon="mdi:lightning-bolt" width="16" height="16" class="text-accent-amber mt-0.5 flex-shrink-0" />
          <h4 class="font-semibold text-ink-black text-sm">一句话总结</h4>
        </div>
        <p class="text-ink-dark leading-relaxed">
          {{ summary.one_sentence }}
        </p>
      </div>

      <!-- Key Points -->
      <div v-if="summary.key_points.length > 0" class="bg-white/80 rounded-lg p-4 shadow-subtle">
        <div class="flex items-start gap-2 mb-3">
          <Icon icon="mdi:lightbulb" width="16" height="16" class="text-accent-amber mt-0.5 flex-shrink-0" />
          <h4 class="font-semibold text-ink-black text-sm">核心观点</h4>
        </div>
        <ul class="space-y-2">
          <li
            v-for="(point, index) in summary.key_points"
            :key="index"
            class="flex items-start gap-2 text-ink-dark text-sm"
          >
            <span class="text-ink-500 mt-1">•</span>
            <span>{{ point }}</span>
          </li>
        </ul>
      </div>

      <!-- Main Takeaways -->
      <div v-if="summary.takeaways.length > 0" class="bg-white/80 rounded-lg p-4 shadow-subtle">
        <div class="flex items-start gap-2 mb-3">
          <Icon icon="mdi:check-circle" width="16" height="16" class="text-success mt-0.5 flex-shrink-0" />
          <h4 class="font-semibold text-ink-black text-sm">关键要点</h4>
        </div>
        <ol class="space-y-2">
          <li
            v-for="(takeaway, index) in summary.takeaways"
            :key="index"
            class="flex items-start gap-2 text-ink-dark text-sm"
          >
            <span class="text-ink-500 font-semibold mt-0.5">{{ index + 1 }}.</span>
            <span>{{ takeaway }}</span>
          </li>
        </ol>
      </div>

      <!-- Tags -->
      <div v-if="summary.tags.length > 0" class="flex flex-wrap gap-2">
        <span
          v-for="(tag, index) in summary.tags"
          :key="index"
          class="px-3 py-1 rounded-full text-xs font-medium"
          :class="getTagColor(index)"
        >
          #{{ tag }}
        </span>
      </div>

      <!-- Regenerate Button -->
      <button
        class="w-full px-4 py-2 bg-white/90 border border-ink-300 text-ink-600 rounded-lg hover:bg-white hover:border-ink-400 transition-colors flex items-center justify-center gap-2 text-sm"
        @click="generateSummary"
      >
        <Icon icon="mdi:refresh" width="16" height="16" />
        重新生成
      </button>
    </div>

    <!-- Empty State -->
    <div v-else class="p-6 text-center text-ink-light">
      <Icon icon="mdi:brain" width="48" height="48" class="mx-auto mb-3 opacity-50" />
      <p>暂无总结内容</p>
    </div>
  </div>
</template>

<style scoped>
.ai-summary {
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}
</style>
