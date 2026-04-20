<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { ref, computed, watch } from 'vue'
import { useNarrativeApi, type NarrativeItem, type NarrativeTimelineDay, type NarrativeScopeCategory } from '~/api/topicGraph'
import NarrativeCanvas from './NarrativeCanvas.client.vue'
import NarrativeDetailCard from './NarrativeDetailCard.vue'

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
const timelineDays = ref<NarrativeTimelineDay[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const expandedIds = ref<Set<number>>(new Set())
const selectedId = ref<number | null>(null)
const hoveredId = ref<number | null>(null)
const triggering = ref(false)
const triggerMessage = ref<string | null>(null)

type ScopeMode = 'all' | 'category'
const scopeMode = ref<ScopeMode>('all')
const selectedCategoryId = ref<number | null>(null)
const scopeCategories = ref<NarrativeScopeCategory[]>([])
const scopesLoading = ref(false)
const categoryTimelineDays = ref<NarrativeTimelineDay[]>([])

const statusStyle: Record<string, { label: string; dot: string; ring: string; bg: string; border: string }> = {
  emerging:   { label: '新兴', dot: '#34d399', ring: 'rgba(52,211,153,0.25)',  bg: 'rgba(52,211,153,0.08)',  border: 'rgba(52,211,153,0.35)' },
  continuing: { label: '持续', dot: '#60a5fa', ring: 'rgba(96,165,250,0.25)',  bg: 'rgba(96,165,250,0.08)',  border: 'rgba(96,165,250,0.35)' },
  splitting:  { label: '分化', dot: '#fb923c', ring: 'rgba(251,146,60,0.25)', bg: 'rgba(251,146,60,0.08)', border: 'rgba(251,146,60,0.35)' },
  merging:    { label: '融合', dot: '#c084fc', ring: 'rgba(192,132,252,0.25)', bg: 'rgba(192,132,252,0.08)', border: 'rgba(192,132,252,0.35)' },
  ending:     { label: '终结', dot: '#6b7280', ring: 'rgba(107,114,128,0.25)', bg: 'rgba(107,114,128,0.08)', border: 'rgba(107,114,128,0.35)' },
}

const isInCategoryDetail = computed(() => scopeMode.value === 'category' && selectedCategoryId.value !== null)

const activeTimelineDays = computed(() => {
  if (scopeMode.value === 'category' && selectedCategoryId.value !== null) {
    return categoryTimelineDays.value.filter(d => d.narratives.length > 0)
  }
  return timelineDays.value.filter(d => d.narratives.length > 0)
})

const allNarratives = computed(() => {
  const all: NarrativeItem[] = []
  const days = isInCategoryDetail.value ? categoryTimelineDays.value : timelineDays.value
  for (const day of days) all.push(...day.narratives)
  return all
})

const selectedNarrative = computed(() => {
  if (selectedId.value === null) return null
  return allNarratives.value.find(n => n.id === selectedId.value) ?? null
})

const totalCount = computed(() => allNarratives.value.length)

const showTrigger = computed(() => {
  if (scopeMode.value === 'all') return true
  return selectedCategoryId.value !== null
})

const activeCategoryName = computed(() => {
  if (selectedCategoryId.value === null) return ''
  const cat = scopeCategories.value.find(c => c.category_id === selectedCategoryId.value)
  return cat?.category_name ?? ''
})

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

function switchScope(mode: ScopeMode) {
  scopeMode.value = mode
  selectedCategoryId.value = null
  selectedId.value = null
  hoveredId.value = null
  expandedIds.value = new Set()
  categoryTimelineDays.value = []

  if (mode === 'category') {
    void loadScopes()
  }
}

function selectCategory(catId: number) {
  selectedCategoryId.value = catId
  selectedId.value = null
  expandedIds.value = new Set()
  void loadCategoryTimeline(catId)
}

function backToCategoryList() {
  selectedCategoryId.value = null
  selectedId.value = null
  expandedIds.value = new Set()
  categoryTimelineDays.value = []
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

async function loadScopes() {
  scopesLoading.value = true
  try {
    const response = await narrativeApi.getNarrativeScopes(props.date)
    if (response.success && response.data) {
      scopeCategories.value = response.data.categories ?? []
    } else {
      scopeCategories.value = []
    }
  } catch (err) {
    console.error('Failed to load narrative scopes:', err)
    scopeCategories.value = []
  } finally {
    scopesLoading.value = false
  }
}

async function loadCategoryTimeline(catId: number) {
  loading.value = true
  error.value = null
  try {
    const response = await narrativeApi.getNarrativeTimeline(props.date, 7, 'feed_category', catId)
    if (response.success && response.data) {
      categoryTimelineDays.value = response.data
    } else {
      error.value = response.error || '分类叙事数据加载失败'
      categoryTimelineDays.value = []
    }
  } catch (err) {
    console.error('Failed to load category narrative timeline:', err)
    error.value = '分类叙事数据加载失败'
    categoryTimelineDays.value = []
  } finally {
    loading.value = false
  }
}

async function triggerGeneration() {
  triggering.value = true
  triggerMessage.value = null
  try {
    const scopeType = isInCategoryDetail.value ? 'feed_category' : undefined
    const categoryId = isInCategoryDetail.value ? selectedCategoryId.value ?? undefined : undefined

    const response = await narrativeApi.regenerateNarratives(props.date, scopeType, categoryId)
    if (response.success) {
      triggerMessage.value = '叙事重新整理完成'
      setTimeout(() => { triggerMessage.value = null }, 3000)
      if (isInCategoryDetail.value && selectedCategoryId.value !== null) {
        void loadCategoryTimeline(selectedCategoryId.value)
      } else {
        void loadTimeline()
      }
    } else {
      triggerMessage.value = response.error || '重新整理失败'
      setTimeout(() => { triggerMessage.value = null }, 5000)
    }
  } catch (err) {
    console.error('Failed to trigger narrative generation:', err)
    triggerMessage.value = '重新整理失败'
    setTimeout(() => { triggerMessage.value = null }, 5000)
  } finally {
    triggering.value = false
  }
}

watch(() => props.date, () => {
  timelineDays.value = []
  categoryTimelineDays.value = []
  expandedIds.value = new Set()
  selectedId.value = null
  hoveredId.value = null
  scopeCategories.value = []
  selectedCategoryId.value = null
  void loadTimeline()
  if (scopeMode.value === 'category') {
    void loadScopes()
  }
}, { immediate: true })
</script>

<template>
  <section class="narrative-panel">
    <!-- Header -->
    <div class="narrative-panel__header">
      <div>
        <p class="narrative-panel__eyebrow">叙事脉络</p>
        <h3 class="narrative-panel__title">
          <template v-if="isInCategoryDetail">{{ activeCategoryName }} · {{ date }} 叙事脉络</template>
          <template v-else>话题演化时间线</template>
        </h3>
      </div>
      <div class="narrative-panel__actions">
        <span v-if="totalCount" class="narrative-panel__count">{{ totalCount }} 条叙事</span>
        <button
          v-if="showTrigger"
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

    <!-- Scope switcher -->
    <div v-if="!isInCategoryDetail" class="narrative-panel__scope-switcher">
      <button
        type="button"
        class="narrative-panel__scope-btn"
        :class="{ 'narrative-panel__scope-btn--active': scopeMode === 'all' }"
        @click="switchScope('all')"
      >
        全部
      </button>
      <button
        type="button"
        class="narrative-panel__scope-btn"
        :class="{ 'narrative-panel__scope-btn--active': scopeMode === 'category' }"
        @click="switchScope('category')"
      >
        按分类
      </button>
    </div>

    <!-- Back button for category detail -->
    <button
      v-if="isInCategoryDetail"
      type="button"
      class="narrative-panel__back"
      @click="backToCategoryList"
    >
      <Icon icon="mdi:chevron-left" width="16" />
      <span>全部叙事</span>
    </button>

    <div v-if="triggerMessage" class="narrative-panel__msg">
      {{ triggerMessage }}
    </div>

    <!-- ═══ Scope: ALL (default) ═══ -->
    <template v-if="scopeMode === 'all'">
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
      <div v-else-if="!activeTimelineDays.length" class="narrative-panel__centered">
        <Icon icon="mdi:text-box-search-outline" width="28" class="text-white/20" />
        <p>近 7 天暂无叙事记录</p>
        <p class="narrative-panel__empty-hint">叙事会在话题分析过程中自动生成</p>
      </div>

      <!-- Canvas + Detail -->
      <div v-else class="narrative-panel__body">
        <ClientOnly>
          <NarrativeCanvas
            :days="activeTimelineDays"
            :selected-id="selectedId"
            @select="handleCanvasSelect"
            @hover="handleCanvasHover"
          />
        </ClientOnly>

        <!-- Floating detail card -->
        <Transition name="detail-slide">
          <NarrativeDetailCard
            v-if="selectedNarrative"
            :narrative="selectedNarrative"
            :expanded="expandedIds.has(selectedNarrative.id)"
            :status-style="statusStyle"
            @select-tag="(tag: NarrativeTag) => emit('select-tag', tag)"
            @toggle-expand="toggleExpand(selectedNarrative.id)"
            @close="selectedId = null"
          />
        </Transition>
      </div>
    </template>

    <!-- ═══ Scope: CATEGORY list ═══ -->
    <template v-else-if="scopeMode === 'category' && !isInCategoryDetail">
      <!-- Loading scopes -->
      <div v-if="scopesLoading" class="narrative-panel__centered">
        <Icon icon="mdi:loading" width="20" class="animate-spin text-white/40" />
        <span>正在加载分类叙事...</span>
      </div>

      <!-- Empty scopes -->
      <div v-else-if="scopeCategories.length === 0" class="narrative-panel__centered">
        <Icon icon="mdi:text-box-search-outline" width="28" class="text-white/20" />
        <p>当天无分类叙事</p>
        <p class="narrative-panel__empty-hint">请先完成一次叙事整理</p>
      </div>

      <!-- Category list -->
      <div v-else class="narrative-panel__cat-list">
        <button
          v-for="cat in scopeCategories"
          :key="cat.category_id"
          type="button"
          class="narrative-panel__cat-card"
          @click="selectCategory(cat.category_id)"
        >
          <div class="narrative-panel__cat-icon" :style="{ background: cat.category_color + '22', color: cat.category_color }">
            <Icon :icon="cat.category_icon || 'mdi:folder'" width="18" />
          </div>
          <div class="narrative-panel__cat-info">
            <span class="narrative-panel__cat-name">{{ cat.category_name }}</span>
          </div>
          <span class="narrative-panel__cat-badge">{{ cat.narrative_count }}</span>
          <Icon icon="mdi:chevron-right" width="16" class="narrative-panel__cat-arrow" />
        </button>
      </div>
    </template>

    <!-- ═══ Scope: CATEGORY detail (reuse canvas) ═══ -->
    <template v-else-if="isInCategoryDetail">
      <!-- Loading -->
      <div v-if="loading" class="narrative-panel__centered">
        <Icon icon="mdi:loading" width="20" class="animate-spin text-white/40" />
        <span>正在加载分类叙事...</span>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="narrative-panel__centered narrative-panel__centered--error">
        <Icon icon="mdi:alert-circle-outline" width="18" />
        <span>{{ error }}</span>
      </div>

      <!-- Empty -->
      <div v-else-if="!activeTimelineDays.length" class="narrative-panel__centered">
        <Icon icon="mdi:text-box-search-outline" width="28" class="text-white/20" />
        <p>该分类当天未生成叙事</p>
        <p class="narrative-panel__empty-hint">文章数或标签数可能不足</p>
      </div>

      <!-- Canvas + Detail -->
      <div v-else class="narrative-panel__body">
        <ClientOnly>
          <NarrativeCanvas
            :days="activeTimelineDays"
            :selected-id="selectedId"
            @select="handleCanvasSelect"
            @hover="handleCanvasHover"
          />
        </ClientOnly>

        <Transition name="detail-slide">
          <NarrativeDetailCard
            v-if="selectedNarrative"
            :narrative="selectedNarrative"
            :expanded="expandedIds.has(selectedNarrative.id)"
            :status-style="statusStyle"
            @select-tag="(tag: NarrativeTag) => emit('select-tag', tag)"
            @toggle-expand="toggleExpand(selectedNarrative.id)"
            @close="selectedId = null"
          />
        </Transition>
      </div>
    </template>
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

/* ── Scope switcher ── */
.narrative-panel__scope-switcher {
  display: flex;
  gap: 0.25rem;
  padding: 0.2rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

.narrative-panel__scope-btn {
  flex: 1;
  padding: 0.3rem 0.9rem;
  border-radius: 999px;
  border: none;
  background: none;
  color: rgba(186, 206, 226, 0.55);
  font-size: 0.75rem;
  cursor: pointer;
  transition: all 0.15s ease;
}

.narrative-panel__scope-btn--active {
  background: rgba(255, 255, 255, 0.08);
  color: rgba(241, 247, 252, 0.9);
}

.narrative-panel__scope-btn:hover:not(.narrative-panel__scope-btn--active) {
  color: rgba(186, 206, 226, 0.8);
}

/* ── Back button ── */
.narrative-panel__back {
  display: inline-flex;
  align-items: center;
  gap: 0.2rem;
  padding: 0.25rem 0.5rem;
  border: none;
  background: none;
  color: rgba(186, 206, 226, 0.55);
  font-size: 0.75rem;
  cursor: pointer;
  transition: color 0.15s ease;
}

.narrative-panel__back:hover {
  color: rgba(241, 247, 252, 0.9);
}

/* ── Category list ── */
.narrative-panel__cat-list {
  display: grid;
  gap: 0.5rem;
}

.narrative-panel__cat-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.65rem 0.85rem;
  border-radius: 12px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.02);
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
}

.narrative-panel__cat-card:hover {
  border-color: rgba(255, 255, 255, 0.15);
  background: rgba(255, 255, 255, 0.04);
}

.narrative-panel__cat-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.narrative-panel__cat-info {
  flex: 1;
  min-width: 0;
}

.narrative-panel__cat-name {
  font-size: 0.82rem;
  color: rgba(241, 247, 252, 0.85);
}

.narrative-panel__cat-badge {
  padding: 0.15rem 0.5rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.06);
  color: rgba(186, 206, 226, 0.6);
  font-size: 0.7rem;
  font-variant-numeric: tabular-nums;
}

.narrative-panel__cat-arrow {
  color: rgba(186, 206, 226, 0.3);
  flex-shrink: 0;
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
