<script setup lang="ts">
import { computed, onBeforeUnmount, reactive, watch } from 'vue'
import {
  useTopicGraphApi,
  type TopicAnalysisType,
  type TopicGraphDetailPayload,
  type TopicGraphType,
} from '~/api/topicGraph'
import TopicAnalysisPanel from '~/features/topic-graph/components/TopicAnalysisPanel.vue'
import { normalizeTopicCategory } from '~/features/topic-graph/utils/normalizeTopicCategory'

interface Props {
  detail: TopicGraphDetailPayload | null
}

const props = defineProps<Props>()

const topicGraphApi = useTopicGraphApi()

type ParsedAnalysisData = Record<string, unknown>
type AnalysisState = 'idle' | 'pending' | 'processing' | 'completed' | 'failed' | 'missing'

const analysisDataByType = reactive<Record<TopicAnalysisType, ParsedAnalysisData | null>>({
  event: null,
  person: null,
  keyword: null,
})

const loadingByType = reactive<Record<TopicAnalysisType, boolean>>({
  event: false,
  person: false,
  keyword: false,
})

const errorByType = reactive<Record<TopicAnalysisType, string | null>>({
  event: null,
  person: null,
  keyword: null,
})

const statusByType = reactive<Record<TopicAnalysisType, AnalysisState>>({
  event: 'idle',
  person: 'idle',
  keyword: 'idle',
})

const progressByType = reactive<Record<TopicAnalysisType, number>>({
  event: 0,
  person: 0,
  keyword: 0,
})

const pollingTimers = new Map<TopicAnalysisType, ReturnType<typeof setTimeout>>()

const selectedAnalysisType = computed<TopicAnalysisType>(() => props.detail
  ? normalizeTopicCategory(props.detail.topic.category, props.detail.topic.kind)
  : 'event')
const currentAnalysisData = computed(() => analysisDataByType[selectedAnalysisType.value])
const loadingAnalysis = computed(() => loadingByType[selectedAnalysisType.value])
const analysisError = computed(() => errorByType[selectedAnalysisType.value])
const analysisStatus = computed(() => statusByType[selectedAnalysisType.value])
const analysisProgress = computed(() => progressByType[selectedAnalysisType.value])
const panelTitle = computed(() => {
  switch (selectedAnalysisType.value) {
    case 'person':
      return '人物分析'
    case 'keyword':
      return '关键词分析'
    default:
      return '事件分析'
  }
})

function stopPolling(analysisType: TopicAnalysisType) {
  const timer = pollingTimers.get(analysisType)
  if (timer) {
    clearTimeout(timer)
    pollingTimers.delete(analysisType)
  }
}

function schedulePolling(analysisType: TopicAnalysisType) {
  stopPolling(analysisType)
  const timer = setTimeout(() => {
    void pollAnalysisStatus(analysisType)
  }, 1800)
  pollingTimers.set(analysisType, timer)
}

onBeforeUnmount(() => {
  for (const analysisType of ['event', 'person', 'keyword'] as TopicAnalysisType[]) {
    stopPolling(analysisType)
  }
})

function resetAnalysisState() {
  analysisDataByType.event = null
  analysisDataByType.person = null
  analysisDataByType.keyword = null
  errorByType.event = null
  errorByType.person = null
  errorByType.keyword = null
  statusByType.event = 'idle'
  statusByType.person = 'idle'
  statusByType.keyword = 'idle'
  progressByType.event = 0
  progressByType.person = 0
  progressByType.keyword = 0
  stopPolling('event')
  stopPolling('person')
  stopPolling('keyword')
}

function resolveAnchorDate() {
  const points = props.detail?.history || []
  if (points.length) {
    const latest = [...points]
      .sort((left, right) => new Date(right.anchor_date).getTime() - new Date(left.anchor_date).getTime())[0]
    if (latest?.anchor_date) {
      return latest.anchor_date
    }
  }

  const now = new Date()
  const year = now.getFullYear()
  const month = `${now.getMonth() + 1}`.padStart(2, '0')
  const day = `${now.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

function parsePayload(payloadJSON: string) {
  try {
    const parsed = JSON.parse(payloadJSON)
    if (parsed && typeof parsed === 'object') {
      return parsed as ParsedAnalysisData
    }
    return null
  } catch {
    return null
  }
}

async function loadAnalysis(analysisType: TopicAnalysisType, force = false) {
  if (!props.detail) return
  if (!force && (analysisDataByType[analysisType] || loadingByType[analysisType])) return

  const tagID = props.detail.topic.id
  if (!tagID) {
    errorByType[analysisType] = '当前话题缺少标签 ID，暂时无法拉取分析。'
    return
  }

  loadingByType[analysisType] = true
  errorByType[analysisType] = null
  if (statusByType[analysisType] === 'idle') {
    statusByType[analysisType] = 'pending'
  }

  try {
    const windowType: TopicGraphType = 'daily'
    const anchorDate = resolveAnchorDate()
    const response = await topicGraphApi.getTopicAnalysis({
      tagID,
      analysisType,
      windowType,
      anchorDate,
    })

    if (response.success && response.data) {
      const payload = parsePayload(response.data.payload_json)
      if (!payload) {
        analysisDataByType[analysisType] = null
        statusByType[analysisType] = 'failed'
        errorByType[analysisType] = '分析结果格式异常'
        return
      }

      analysisDataByType[analysisType] = payload
      statusByType[analysisType] = 'completed'
      progressByType[analysisType] = 100
      stopPolling(analysisType)
      return
    }

    await pollAnalysisStatus(analysisType, true)
  } catch (error) {
    analysisDataByType[analysisType] = null
    statusByType[analysisType] = 'failed'
    errorByType[analysisType] = error instanceof Error ? error.message : '分析数据加载失败'
  } finally {
    loadingByType[analysisType] = false
  }
}

async function pollAnalysisStatus(analysisType: TopicAnalysisType, enqueueIfMissing = false) {
  if (!props.detail?.topic.id) return

  const params = {
    tagID: props.detail.topic.id,
    analysisType,
    windowType: 'daily' as TopicGraphType,
    anchorDate: resolveAnchorDate(),
  }

  try {
    const response = await topicGraphApi.getAnalysisStatus(params)
    const status = response.data?.status
    const progress = response.data?.progress || 0

    if (!response.success || !status) {
      statusByType[analysisType] = 'failed'
      errorByType[analysisType] = response.error || '获取分析状态失败'
      stopPolling(analysisType)
      return
    }

    if (status === 'ready') {
      await loadAnalysis(analysisType, true)
      return
    }

    if (status === 'pending' || status === 'processing') {
      statusByType[analysisType] = status
      progressByType[analysisType] = Math.min(Math.max(Math.round(progress * 100), 1), 99)
      schedulePolling(analysisType)
      return
    }

    if (status === 'missing' && enqueueIfMissing) {
      await topicGraphApi.rebuildTopicAnalysis(params)
      statusByType[analysisType] = 'pending'
      progressByType[analysisType] = 1
      schedulePolling(analysisType)
      return
    }

    if (status === 'failed') {
      statusByType[analysisType] = 'failed'
      errorByType[analysisType] = '分析任务失败，请重试'
      stopPolling(analysisType)
      return
    }

    statusByType[analysisType] = 'missing'
    stopPolling(analysisType)
  } catch (error) {
    statusByType[analysisType] = 'failed'
    errorByType[analysisType] = error instanceof Error ? error.message : '状态查询失败'
    stopPolling(analysisType)
  }
}

async function refreshCurrentAnalysis() {
  if (!props.detail?.topic.id) return

  const analysisType = selectedAnalysisType.value
  const params = {
    tagID: props.detail.topic.id,
    analysisType,
    windowType: 'daily' as TopicGraphType,
    anchorDate: resolveAnchorDate(),
  }

  stopPolling(analysisType)
  statusByType[analysisType] = 'pending'
  progressByType[analysisType] = 0
  errorByType[analysisType] = null
  await topicGraphApi.retryTopicAnalysis(params)
  await pollAnalysisStatus(analysisType, true)
}

watch(
  () => props.detail?.topic.slug,
  async () => {
    resetAnalysisState()
    if (!props.detail) return
    await loadAnalysis(selectedAnalysisType.value)
  },
  { immediate: true },
)

watch(
  () => selectedAnalysisType.value,
  async (value) => {
    if (!props.detail) return
    await loadAnalysis(value)
  },
)
</script>

<template>
  <section class="topic-footer-grid" data-testid="topic-graph-footer">
    <div class="topic-footer-intro">
      <p class="topic-footer-intro__eyebrow">{{ panelTitle }}</p>
      <h3 class="topic-footer-intro__title">跟着当前焦点切换，不再额外开一排重复控制。</h3>
    </div>

    <TopicAnalysisPanel
      :analysis-type="selectedAnalysisType"
      :data="currentAnalysisData"
      :loading="loadingAnalysis"
      :error="analysisError"
      :analysis-status="analysisStatus"
      :analysis-progress="analysisProgress"
      @retry="refreshCurrentAnalysis"
      data-testid="topic-graph-history-region"
    />
  </section>
</template>

<style scoped>
.topic-footer-grid {
  display: grid;
  gap: 0.9rem;
}

.topic-footer-intro {
  display: grid;
  gap: 0.28rem;
  border-left: 2px solid rgba(240, 138, 75, 0.42);
  padding-left: 0.9rem;
}

.topic-footer-intro__eyebrow {
  font-size: 0.72rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(186, 206, 226, 0.72);
}

.topic-footer-intro__title {
  font-size: 0.95rem;
  line-height: 1.55;
  color: rgba(241, 247, 252, 0.9);
}
</style>
