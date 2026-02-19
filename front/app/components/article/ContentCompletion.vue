<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { useContentCompletion, type ContentCompletionStatus } from '~/composables/useContentCompletion'

interface Props {
  articleId: string
  contentStatus?: string
  fullContent?: string
  aiSummary?: string
  readonly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  contentStatus: 'complete',
  fullContent: '',
  aiSummary: '',
  readonly: false,
})

const emit = defineEmits<{
  completed: [fullContent: string, aiSummary: string]
}>()

const { loading, error, completeArticle, getCompletionStatus } = useContentCompletion()

const status = ref<ContentCompletionStatus>({
  content_status: props.contentStatus as any,
  attempts: 0,
  error: null,
  fetched_at: null,
})

const isIncomplete = computed(() => status.value.content_status === 'incomplete')
const isPending = computed(() => status.value.content_status === 'pending')
const isFailed = computed(() => status.value.content_status === 'failed')
const isComplete = computed(() => status.value.content_status === 'complete' || props.fullContent)

const statusText = computed(() => {
  if (isPending.value) return '正在补全...'
  if (isFailed.value) return '补全失败'
  if (isComplete.value && props.fullContent) return '内容已补全'
  return '内容不完整'
})

const statusIcon = computed(() => {
  if (isPending.value) return 'mdi:loading'
  if (isFailed.value) return 'mdi:alert-circle'
  if (isComplete.value && props.fullContent) return 'mdi:check-circle'
  return 'mdi:alert'
})

const statusColor = computed(() => {
  if (isPending.value) return 'text-amber-500'
  if (isFailed.value) return 'text-red-500'
  if (isComplete.value && props.fullContent) return 'text-emerald-500'
  return 'text-amber-500'
})

const canComplete = computed(() => 
  !props.readonly && (isIncomplete.value || isFailed.value) && !loading.value
)

const handleComplete = async () => {
  try {
    await completeArticle(props.articleId)
    
    const newStatus = await getCompletionStatus(props.articleId)
    status.value = newStatus

    if (newStatus.content_status === 'complete') {
      emit('completed', props.fullContent || '', props.aiSummary || '')
    }
  } catch (e) {
    console.error('Failed to complete article:', e)
  }
}

onMounted(async () => {
  try {
    status.value = await getCompletionStatus(props.articleId)
  } catch (e) {
    console.error('Failed to fetch completion status:', e)
  }
})
</script>

<template>
  <div class="content-completion">
    <div v-if="isComplete && fullContent" class="completion-status complete">
      <Icon :icon="statusIcon" :class="statusColor" class="w-4 h-4" />
      <span :class="statusColor" class="text-sm">{{ statusText }}</span>
    </div>

    <div v-else-if="isPending" class="completion-status pending">
      <Icon :icon="statusIcon" class="w-4 h-4 animate-spin" :class="statusColor" />
      <span :class="statusColor" class="text-sm">{{ statusText }}</span>
    </div>

    <div v-else-if="isFailed" class="completion-status failed">
      <Icon :icon="statusIcon" :class="statusColor" class="w-4 h-4" />
      <div class="flex-1">
        <span :class="statusColor" class="text-sm">{{ statusText }}</span>
        <span v-if="status.error" class="text-xs text-gray-500 block mt-1">
          {{ status.error }}
        </span>
      </div>
      <button
        v-if="canComplete"
        @click="handleComplete"
        class="ml-2 px-3 py-1 text-xs bg-amber-500 hover:bg-amber-600 text-white rounded"
      >
        重试
      </button>
    </div>

    <div v-else class="completion-status incomplete">
      <Icon :icon="statusIcon" :class="statusColor" class="w-4 h-4" />
      <span :class="statusColor" class="text-sm">{{ statusText }}</span>
      <button
        v-if="canComplete"
        @click="handleComplete"
        :disabled="loading"
        class="ml-2 px-3 py-1 text-xs bg-blue-500 hover:bg-blue-600 text-white rounded disabled:opacity-50"
      >
        {{ loading ? '处理中...' : '补全内容' }}
      </button>
    </div>

    <div v-if="aiSummary" class="ai-summary mt-4 p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
      <h3 class="text-sm font-semibold text-purple-700 dark:text-purple-300 mb-2">AI 总结</h3>
      <div class="text-sm prose dark:prose-invert max-w-none" v-html="aiSummary" />
    </div>
  </div>
</template>

<style scoped>
.content-completion {
  @apply flex flex-col gap-2;
}

.completion-status {
  @apply flex items-center gap-2 p-3 rounded-lg;
}

.completion-status.complete {
  @apply bg-emerald-50 dark:bg-emerald-900/20;
}

.completion-status.pending {
  @apply bg-amber-50 dark:bg-amber-900/20;
}

.completion-status.failed {
  @apply bg-red-50 dark:bg-red-900/20;
}

.completion-status.incomplete {
  @apply bg-amber-50 dark:bg-amber-900/20;
}
</style>
