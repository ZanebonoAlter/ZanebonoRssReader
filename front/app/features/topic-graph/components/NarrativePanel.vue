<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { ref, computed, watch } from 'vue'
import { useNarrativeApi, type NarrativeItem, type NarrativeTimelineDay } from '~/api/topicGraph'
import { useSchedulerApi } from '~/api/scheduler'
import NarrativeCanvas from './NarrativeCanvas.client.vue'

interface NarrativeTag {
  id: number
  slug: string
  label: string
  category: string
  kind?: string
}

interface Props {
  date: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'select-tag': [tag: NarrativeTag]
}>()

const narrativeApi = useNarrativeApi()
const schedulerApi = useSchedulerApi()
const timelineDays = ref<NarrativeTimelineDay[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const expandedIds = ref<Set<number>>(new Set())
const selectedId = ref<number | null>(null)
const hoveredId = ref<number | null>(null)
const triggering = ref(false)
const triggerMessage = ref<string | null>(null)

const statusStyle: Record<string, { label: string; dot: string; ring: string; bg: string; border: string }> = {
  emerging:   { label: '新兴', dot: '#34d399', ring: 'rgba(52,211,153,0.25)',  bg: 'rgba(52,211,153,0.08)',  border: 'rgba(52,211,153,0.35)' },
  continuing: { label: '持续', dot: '#60a5fa', ring: 'rgba(96,165,250,0.25)',  bg: 'rgba(96,165,250,0.08)',  border: 'rgba(96,165,250,0.35)' },
  splitting:  { label: '分化', dot: '#fb923c', ring: 'rgba(251,146,60,0.25)', bg: 'rgba(251,146,60,0.08)', border: 'rgba(251,146,60,0.35)' },
  merging:    { label: '融合', dot: '#c084fc', ring: 'rgba(192,132,252,0.25)', bg: 'rgba(192,132,252,0.08)', border: 'rgba(192,132,252,0.35)' },
  ending:     { label: '终结', dot: '#6b7280', ring: 'rgba(107,114,128,0.25)', bg: 'rgba(107,114,128,0.08)', border: 'rgba(107,114,128,0.35)' },
}

const daysWithData = computed(() => timelineDays.value.filter(d => d.narratives.length > 0))

const allNarratives = computed(() => {
  const all: NarrativeItem[] = []
  for (const day of timelineDays.value) all.push(...day.narratives)
  return all
})

const selectedNarrative = computed(() => {
  if (selectedId.value === null) return null
  return allNarratives.value.find(n => n.id === selectedId.value) ?? null
})

const totalCount = computed(() => allNarratives.value.length)

function toggleExpand(id: number) {
  const next = new Set(expandedIds.value)
  if (next.has(id)) next.delete(id); else next.add(id)
  expandedIds.value = next
}

function handleCanvasSelect(id: number) {
  selectedId.value = selectedId.value === id ? null : id
}

function handleCanvasHover(id: number | null) {
  hoveredId.value = id
}

async function loadTimeline() {
  loading.value = true
  error.value = null
  try {
    const response = await narrativeApi.getNarrativeTimeline(props.date, 7)
    if (response.success && response.data) {
      timelineDays.value = response.data
    } else {
      error.value = response.error || '叙事数据加载失败'
      timelineDays.value = []
    }
  } catch (err) {
    console.error('Failed to load narrative timeline:', err)
    error.value = '叙事数据加载失败'
    timelineDays.value = []
  } finally {
    loading.value = false
  }
}

async function triggerGeneration() {
  triggering.value = true
  triggerMessage.value = null
  try {
    const delResp = await narrativeApi.deleteNarratives(props.date)
    if (!delResp.success) {
      triggerMessage.value = delResp.error || '删除旧叙事失败'
      setTimeout(() => { triggerMessage.value = null }, 5000)
      return
    }
    const response = await schedulerApi.triggerScheduler('narrative_summary', { date: props.date })
    if (response.success && response.data?.accepted) {
      triggerMessage.value = '叙事重新整理已启动，完成后将自动刷新'
      setTimeout(() => { triggerMessage.value = null }, 5000)
      setTimeout(() => { void loadTimeline() }, 15000)
    } else {
      triggerMessage.value = response.data?.message || response.error || '触发失败'
      setTimeout(() => { triggerMessage.value = null }, 5000)
    }
  } catch (err) {
    console.error('Failed to trigger narrative generation:', err)
    triggerMessage.value = '触发失败'
    setTimeout(() => { triggerMessage.value = null }, 5000)
  } finally {
    triggering.value = false
  }
}

watch(() => props.date, () => {
  timelineDays.value = []
  expandedIds.value = new Set()
  selectedId.value = null
  hoveredId.value = null
  void loadTimeline()
}, { immediate: true })
</script>

<template>
  <section class="narrative-panel">
    <!-- Header -->
    <div class="narrative-panel__header">
      <div>
        <p class="narrative-panel__eyebrow">叙事脉络</p>
        <h3 class="narrative-panel__title">话题演化时间线</h3>
      </div>
      <div class="narrative-panel__actions">
        <span v-if="totalCount" class="narrative-panel__count">{{ totalCount }} 条叙事</span>
        <button
          type="button"
          class="narrative-panel__trigger"
          :disabled="triggering"
          @click="triggerGeneration"
        >
          <Icon v-if="triggering" icon="mdi:loading" width="14" class="animate-spin" />
          <Icon v-else icon="mdi:auto-fix" width="14" />
          {{ triggering ? '整理中...' : '重新整理' }}
        </button>
      </div>
    </div>

    <div v-if="triggerMessage" class="narrative-panel__msg">
      {{ triggerMessage }}
    </div>

    <!-- Loading -->
    <div v-if="loading" class="narrative-panel__centered">
      <Icon icon="mdi:loading" width="20" class="animate-spin text-white/40" />
      <span>正在加载叙事数据...</span>
    </div>

    <!-- Error -->
    <div v-else-if="error" class="narrative-panel__centered narrative-panel__centered--error">
      <Icon icon="mdi:alert-circle-outline" width="18" />
      <span>{{ error }}</span>
    </div>

    <!-- Empty -->
    <div v-else-if="!daysWithData.length" class="narrative-panel__centered">
      <Icon icon="mdi:text-box-search-outline" width="28" class="text-white/20" />
      <p>近 7 天暂无叙事记录</p>
      <p class="narrative-panel__empty-hint">叙事会在话题分析过程中自动生成</p>
    </div>

    <!-- Canvas + Detail -->
    <div v-else class="narrative-panel__body">
      <ClientOnly>
        <NarrativeCanvas
          :days="daysWithData"
          :selected-id="selectedId"
          @select="handleCanvasSelect"
          @hover="handleCanvasHover"
        />
      </ClientOnly>

      <!-- Floating detail card -->
      <Transition name="detail-slide">
        <div v-if="selectedNarrative" class="narrative-detail">
          <div class="narrative-detail__head">
            <h4 class="narrative-detail__title">{{ selectedNarrative.title }}</h4>
            <span
              class="narrative-detail__status"
              :style="{
                color: statusStyle[selectedNarrative.status]?.dot,
                background: statusStyle[selectedNarrative.status]?.bg,
                borderColor: statusStyle[selectedNarrative.status]?.border,
              }"
            >
              {{ statusStyle[selectedNarrative.status]?.label }}
            </span>
            <button type="button" class="narrative-detail__close" @click.stop="selectedId = null">
              <Icon icon="mdi:close" width="14" />
            </button>
          </div>

          <div class="narrative-detail__summary">
            <p v-if="!expandedIds.has(selectedNarrative.id) && selectedNarrative.summary.length > 240" class="narrative-detail__text">
              {{ selectedNarrative.summary.slice(0, 240) }}...
            </p>
            <p v-else class="narrative-detail__text">{{ selectedNarrative.summary }}</p>
            <button
              v-if="selectedNarrative.summary.length > 240"
              type="button"
              class="narrative-detail__expand"
              @click="toggleExpand(selectedNarrative.id)"
            >
              {{ expandedIds.has(selectedNarrative.id) ? '收起' : '展开全文' }}
            </button>
          </div>

          <div v-if="selectedNarrative.related_tags.length" class="narrative-detail__tags">
            <button
              v-for="tag in selectedNarrative.related_tags"
              :key="tag.id"
              type="button"
              class="narrative-detail__tag"
              @click="emit('select-tag', tag)"
            >
              {{ tag.label }}
            </button>
          </div>

          <div class="narrative-detail__meta">
            <span v-if="selectedNarrative.generation > 0" class="narrative-detail__meta-item">
              <Icon icon="mdi:source-branch" width="12" />
              第 {{ selectedNarrative.generation }} 代
            </span>
            <span class="narrative-detail__meta-item">{{ selectedNarrative.period_date }}</span>
            <span v-if="selectedNarrative.parent_ids.length" class="narrative-detail__meta-item">
              <Icon icon="mdi:arrow-left-top" width="12" />
              继承 {{ selectedNarrative.parent_ids.length }} 条
            </span>
            <span v-if="selectedNarrative.child_ids.length" class="narrative-detail__meta-item">
              <Icon icon="mdi:arrow-right-bottom" width="12" />
              衍生 {{ selectedNarrative.child_ids.length }} 条
            </span>
          </div>
        </div>
      </Transition>
    </div>
  </section>
</template>

<style scoped>
.narrative-panel {
  display: grid;
  gap: 1rem;
}

/* ── Header ── */
.narrative-panel__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.narrative-panel__eyebrow {
  font-size: 0.72rem;
  letter-spacing: 0.24em;
  text-transform: uppercase;
  color: rgba(186, 206, 226, 0.72);
}

.narrative-panel__title {
  margin-top: 0.35rem;
  font-size: 0.95rem;
  line-height: 1.55;
  color: rgba(241, 247, 252, 0.9);
}

.narrative-panel__actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}

.narrative-panel__count {
  padding: 0.2rem 0.6rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(255, 255, 255, 0.5);
  font-size: 0.72rem;
}

.narrative-panel__trigger {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
  padding: 0.3rem 0.7rem;
  border-radius: 999px;
  border: 1px solid rgba(240, 138, 75, 0.3);
  background: rgba(240, 138, 75, 0.08);
  color: rgba(240, 180, 140, 0.9);
  font-size: 0.72rem;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.narrative-panel__trigger:hover:not(:disabled) {
  border-color: rgba(240, 138, 75, 0.5);
  background: rgba(240, 138, 75, 0.15);
}

.narrative-panel__trigger:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.narrative-panel__msg {
  padding: 0.45rem 0.8rem;
  border-radius: 10px;
  border: 1px solid rgba(240, 138, 75, 0.2);
  background: rgba(240, 138, 75, 0.06);
  color: rgba(240, 180, 140, 0.8);
  font-size: 0.78rem;
  text-align: center;
}

/* ── Centered states ── */
.narrative-panel__centered {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.6rem;
  padding: 2.5rem 1rem;
  border-radius: 16px;
  border: 1px dashed rgba(186, 206, 226, 0.2);
  background: rgba(255, 255, 255, 0.02);
  color: rgba(186, 206, 226, 0.55);
  font-size: 0.85rem;
  text-align: center;
}

.narrative-panel__centered--error {
  border-color: rgba(240, 138, 75, 0.25);
  color: rgba(240, 180, 140, 0.7);
}

.narrative-panel__empty-hint {
  font-size: 0.75rem;
  color: rgba(186, 206, 226, 0.35);
}

/* ── Body ── */
.narrative-panel__body {
  position: relative;
}

/* ── Detail card ── */
.narrative-detail {
  margin-top: 0.75rem;
  border-radius: 14px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: linear-gradient(180deg, rgba(20, 29, 40, 0.96), rgba(12, 18, 26, 0.98));
  padding: 1rem 1.1rem;
  backdrop-filter: blur(14px);
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.45);
}

.narrative-detail__head {
  display: flex;
  align-items: flex-start;
  gap: 0.5rem;
}

.narrative-detail__title {
  flex: 1;
  font-size: 0.9rem;
  font-weight: 600;
  line-height: 1.5;
  color: rgba(241, 247, 252, 0.92);
}

.narrative-detail__status {
  flex-shrink: 0;
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  border: 1px solid;
  font-size: 0.66rem;
  font-weight: 500;
}

.narrative-detail__close {
  flex-shrink: 0;
  border: none;
  background: none;
  color: rgba(255, 255, 255, 0.35);
  cursor: pointer;
  padding: 0.15rem;
  transition: color 0.15s ease;
}

.narrative-detail__close:hover {
  color: rgba(255, 255, 255, 0.7);
}

.narrative-detail__summary {
  margin-top: 0.6rem;
}

.narrative-detail__text {
  font-size: 0.82rem;
  line-height: 1.7;
  color: rgba(186, 206, 226, 0.72);
}

.narrative-detail__expand {
  display: inline-flex;
  align-items: center;
  margin-top: 0.2rem;
  border: none;
  background: none;
  color: rgba(240, 138, 75, 0.72);
  font-size: 0.75rem;
  cursor: pointer;
  padding: 0;
  transition: color 0.15s ease;
}

.narrative-detail__expand:hover {
  color: rgba(240, 138, 75, 1);
}

.narrative-detail__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.3rem;
  margin-top: 0.6rem;
}

.narrative-detail__tag {
  padding: 0.15rem 0.45rem;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.04);
  color: rgba(255, 255, 255, 0.6);
  font-size: 0.68rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.narrative-detail__tag:hover {
  border-color: rgba(240, 138, 75, 0.35);
  color: rgba(255, 220, 200, 0.9);
  background: rgba(240, 138, 75, 0.1);
}

.narrative-detail__meta {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 0.6rem;
  font-size: 0.7rem;
  color: rgba(255, 255, 255, 0.35);
}

.narrative-detail__meta-item {
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
}

/* ── Transitions ── */
.detail-slide-enter-active,
.detail-slide-leave-active {
  transition: all 0.22s cubic-bezier(0.22, 1, 0.36, 1);
}

.detail-slide-enter-from,
.detail-slide-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
