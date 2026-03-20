<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { marked } from 'marked'
import { computed, onMounted, ref, watch } from 'vue'
import DigestDetail from '~/features/digest/components/DigestDetail.vue'
import DigestSettings from '~/features/digest/components/DigestSettings.vue'
import {
  useDigestApi,
  type DigestPreview,
  type DigestPreviewCategory,
  type DigestPreviewSummary,
  type DigestStatus,
  type DigestType,
  type OpenNotebookConfig,
  type OpenNotebookRunResult,
} from '~/api/digest'

const props = withDefaults(defineProps<{
  lockedType?: DigestType | null
  showDigestBack?: boolean
}>(), {
  lockedType: null,
  showDigestBack: false,
})

const digestApi = useDigestApi()

function formatDateInput(date = new Date()) {
  const year = date.getFullYear()
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  const day = `${date.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

function shiftDate(value: string, days: number) {
  const date = new Date(`${value}T12:00:00`)
  if (Number.isNaN(date.getTime())) return formatDateInput()
  date.setDate(date.getDate() + days)
  return formatDateInput(date)
}

const selectedType = ref<DigestType>(props.lockedType || 'daily')
const selectedDates = ref<Record<DigestType, string>>({
  daily: formatDateInput(),
  weekly: formatDateInput(),
})
const selectedCategoryId = ref<number | null>(null)
const selectedSummaryId = ref<number | null>(null)
const loading = ref(true)
const showSettings = ref(false)
const status = ref<DigestStatus | null>(null)
const previews = ref<Record<DigestType, DigestPreview | null>>({
  daily: null,
  weekly: null,
})
const refreshingType = ref<DigestType | null>(null)
const runningType = ref<DigestType | null>(null)
const openNotebookSending = ref(false)
const notice = ref<{ type: 'success' | 'error' | 'info', text: string } | null>(null)
const openNotebookConfig = ref<OpenNotebookConfig | null>(null)
const openNotebookResult = ref<OpenNotebookRunResult | null>(null)

const typeMeta: Record<DigestType, { label: string, short: string, kicker: string }> = {
  daily: {
    label: '日报',
    short: 'Daily',
    kicker: '这一天收了哪些 AI 总结。',
  },
  weekly: {
    label: '周报',
    short: 'Weekly',
    kicker: '这一周哪些 feed 值得翻。',
  },
}

const activePreview = computed(() => previews.value[selectedType.value])
const activeMeta = computed(() => typeMeta[selectedType.value])
const activeDateValue = computed(() => selectedDates.value[selectedType.value])
const activeDateLabel = computed(() => selectedType.value === 'daily' ? '看哪天' : '锚点日')
const activeCategories = computed(() => activePreview.value?.categories || [])
const activeCategory = computed(() => activeCategories.value.find(category => category.id === selectedCategoryId.value) || null)
const activeSummaries = computed(() => activeCategory.value?.summaries || [])
const activeSummary = computed(() => activeSummaries.value.find(summary => summary.id === selectedSummaryId.value) || null)
const activeOpenNotebookResult = computed(() => {
  const result = openNotebookResult.value
  if (!result) return null
  if (result.digest_type !== selectedType.value) return null
  if (result.anchor_date !== selectedDates.value[selectedType.value]) return null
  return result
})
const renderedOpenNotebookResult = computed(() => {
  if (!activeOpenNotebookResult.value?.summary_markdown) return ''
  return String(marked.parse(activeOpenNotebookResult.value.summary_markdown))
})
const openNotebookReady = computed(() => Boolean(openNotebookConfig.value?.enabled && openNotebookConfig.value?.base_url))

const statusChips = computed(() => {
  if (!status.value) return []

  return [
    {
      label: '日报',
      value: status.value.daily_enabled ? status.value.daily_time : '已关闭',
    },
    {
      label: '周报',
      value: status.value.weekly_enabled ? `${weekdayLabel(status.value.weekly_day)} ${status.value.weekly_time}` : '已关闭',
    },
    {
      label: '任务数',
      value: String(status.value.active_jobs ?? 0),
    },
  ]
})

const bottomFacts = computed(() => {
  const preview = activePreview.value
  if (!preview) {
    return [
      { label: '当前状态', value: '还没取到内容' },
      { label: '建议动作', value: '先执行一版' },
      { label: '备用动作', value: '检查设置' },
    ]
  }

  return [
    { label: '分类数', value: `${preview.category_count}` },
    { label: '总结数', value: `${preview.summary_count}` },
    { label: '当前焦点', value: activeSummary.value?.feed_name || '还没选中' },
  ]
})

function getTopicCategoryMeta(category: string) {
  const meta: Record<string, { label: string, color: string, defaultIcon: string }> = {
    event: { label: '事件', color: '#f59e0b', defaultIcon: 'mdi:calendar-star' },
    person: { label: '人物', color: '#10b981', defaultIcon: 'mdi:account' },
    keyword: { label: '关键词', color: '#6366f1', defaultIcon: 'mdi:tag' },
  }
  return (meta[category] || meta.keyword)!
}

function weekdayLabel(day?: number) {
  const labels: Record<number, string> = {
    0: '周日',
    1: '周一',
    2: '周二',
    3: '周三',
    4: '周四',
    5: '周五',
    6: '周六',
  }
  return day === undefined ? '--' : labels[day] || '--'
}

function setNotice(type: 'success' | 'error' | 'info', text: string) {
  notice.value = { type, text }
}

function clearOpenNotebookResult() {
  openNotebookResult.value = null
}

function openSettings() {
  showSettings.value = true
}

function closeSettings() {
  showSettings.value = false
}

function stripMarkdown(content: string) {
  return content
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, ' ')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/[>#*_~-]/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

function summaryExcerpt(summary: DigestPreviewSummary) {
  const plain = stripMarkdown(summary.summary_text || '')
  if (plain.length <= 92) return plain
  return `${plain.slice(0, 92)}...`
}

function applyDefaultSelection(preview: DigestPreview | null) {
  if (!preview) {
    selectedCategoryId.value = null
    selectedSummaryId.value = null
    return
  }

  const categories = preview.categories || []
  if (!categories.length) {
    selectedCategoryId.value = null
    selectedSummaryId.value = null
    return
  }

  const fallbackCategory = categories[0]
  const preferredCategory = categories.find(category => category.id === preview.default_category_id) || fallbackCategory
  selectedCategoryId.value = preferredCategory?.id ?? null

  const summaries = preferredCategory?.summaries || []
  const fallbackSummary = summaries[0]
  const preferredSummary = summaries.find(summary => summary.id === preview.default_summary_id) || fallbackSummary
  selectedSummaryId.value = preferredSummary?.id ?? null
}

function selectCategory(category: DigestPreviewCategory) {
  selectedCategoryId.value = category.id
  selectedSummaryId.value = category.summaries[0]?.id ?? null
}

function selectSummary(summary: DigestPreviewSummary) {
  selectedSummaryId.value = summary.id
}

function handleDateInput(value: string) {
  if (!value) return
  selectedDates.value[selectedType.value] = value
  void loadPreview(selectedType.value, { resetSelection: true })
}

function jumpDate(days: number) {
  const nextValue = shiftDate(activeDateValue.value, days)
  handleDateInput(nextValue)
}

function jumpToToday() {
  handleDateInput(formatDateInput())
}

async function loadStatus() {
  const response = await digestApi.getStatus()
  if (response.success && response.data) {
    status.value = response.data
  }
}

async function loadOpenNotebookConfig() {
  const response = await digestApi.getOpenNotebookConfig()
  if (response.success && response.data) {
    openNotebookConfig.value = response.data
  }
}

async function loadPreview(type: DigestType, options?: { silent?: boolean; resetSelection?: boolean }) {
  if (!options?.silent) {
    refreshingType.value = type
  }

  try {
    const response = await digestApi.getPreview(type, selectedDates.value[type])
    if (response.success && response.data) {
      previews.value[type] = response.data
      selectedDates.value[type] = response.data.anchor_date || selectedDates.value[type]
      clearOpenNotebookResult()
      if (type === selectedType.value || options?.resetSelection) {
        applyDefaultSelection(response.data)
      }
      return
    }

    setNotice('error', response.error || `${typeMeta[type].label}没拉下来`)
  } catch (error) {
    console.error(`Failed to load ${type} preview:`, error)
    setNotice('error', `${typeMeta[type].label}加载失败`)
  } finally {
    if (!options?.silent) {
      refreshingType.value = null
    }
  }
}

async function runNow(type: DigestType) {
  runningType.value = type
  try {
    const response = await digestApi.runNow(type, selectedDates.value[type])
    if (response.success && response.data) {
      previews.value[type] = response.data.preview
      selectedDates.value[type] = response.data.preview.anchor_date || selectedDates.value[type]
      selectedType.value = type
      clearOpenNotebookResult()
      applyDefaultSelection(response.data.preview)

       const hints = []
       if (response.data.sent_to_feishu) hints.push('飞书已发')
       if (response.data.exported_to_obsidian) hints.push('Obsidian 已写')
       if (response.data.sent_to_open_notebook) hints.push('Open Notebook 已收')
       setNotice('success', hints.length ? `这版已跑完，${hints.join('，')}` : '这版已跑完')
       return
    }

    setNotice('error', response.error || '执行失败')
  } catch (error) {
    console.error(`Failed to run ${type} digest:`, error)
    setNotice('error', '执行失败')
  } finally {
    runningType.value = null
  }
}

async function sendToOpenNotebook() {
  if (!openNotebookReady.value) {
    setNotice('info', '先把 Open Notebook 配好')
    return
  }

  openNotebookSending.value = true
  try {
    const response = await digestApi.sendToOpenNotebook(selectedType.value, selectedDates.value[selectedType.value])
    if (response.success && response.data) {
      openNotebookResult.value = response.data
      setNotice('success', response.data.remote_url ? '二次总结回来了，外链也带上了' : '二次总结回来了')
      return
    }

    setNotice('error', response.error || 'Open Notebook 没接住')
  } catch (error) {
    console.error('Failed to send digest to open notebook:', error)
    setNotice('error', 'Open Notebook 发送失败')
  } finally {
    openNotebookSending.value = false
  }
}

async function refreshCurrent() {
  await Promise.all([
    loadStatus(),
    loadPreview(selectedType.value, { resetSelection: true }),
  ])
}

async function loadDashboard() {
  loading.value = true
  try {
    if (props.lockedType) {
      await Promise.all([
        loadStatus(),
        loadOpenNotebookConfig(),
        loadPreview(props.lockedType, { silent: true, resetSelection: true }),
      ])
      selectedType.value = props.lockedType
      return
    }

    await Promise.all([
      loadStatus(),
      loadOpenNotebookConfig(),
      loadPreview('daily', { silent: true, resetSelection: true }),
      loadPreview('weekly', { silent: true }),
    ])
  } finally {
    loading.value = false
  }
}

watch(() => props.lockedType, (value) => {
  if (value) {
    selectedType.value = value
  }
})

watch(selectedType, async (type) => {
  if (props.lockedType && type !== props.lockedType) {
    selectedType.value = props.lockedType
    return
  }

  clearOpenNotebookResult()

  const preview = previews.value[type]
  if (!preview || preview.anchor_date !== selectedDates.value[type]) {
    await loadPreview(type, { silent: true, resetSelection: true })
    return
  }

  applyDefaultSelection(preview)
})

onMounted(loadDashboard)
</script>

<template>
  <div class="digest-stage min-h-screen px-4 py-5 md:px-6 md:py-7">
    <div class="digest-shell mx-auto max-w-[1840px] space-y-5">
      <header class="digest-topbar paper-card rounded-[34px] px-5 py-5 md:px-7 md:py-6">
        <div class="flex flex-col gap-5 xl:flex-row xl:items-center xl:justify-between">
          <div class="flex flex-wrap items-center gap-3">
            <button v-if="showDigestBack" class="btn-ghost min-h-11 px-4" type="button" aria-label="返回 digest 总览" @click="navigateTo('/digest')">
              <Icon icon="mdi:arrow-left" width="18" />
              回总览
            </button>
            <button class="btn-ghost min-h-11 px-4" type="button" aria-label="返回主界面" @click="navigateTo('/')">
              <Icon icon="mdi:home-outline" width="18" />
              返回主界面
            </button>

            <div v-if="!lockedType" class="digest-switcher" role="tablist" aria-label="日报周报切换">
              <button
                v-for="type in (['daily', 'weekly'] as DigestType[])"
                :key="type"
                class="digest-switcher__item"
                :class="{ 'digest-switcher__item--active': selectedType === type }"
                type="button"
                :aria-selected="selectedType === type"
                @click="selectedType = type"
              >
                <span class="text-[10px] uppercase tracking-[0.26em] opacity-60">{{ typeMeta[type].short }}</span>
                <span class="text-sm font-bold">{{ typeMeta[type].label }}</span>
              </button>
            </div>
            <div v-else class="digest-lock-chip rounded-full px-4 py-2">
              <span class="text-[10px] uppercase tracking-[0.26em] text-ink-light">{{ activeMeta.short }}</span>
              <strong class="ml-2 text-sm text-ink-dark">{{ activeMeta.label }}</strong>
            </div>

            <div class="digest-date-rail">
              <button class="digest-date-button" type="button" aria-label="看前一天" @click="jumpDate(-1)">
                <Icon icon="mdi:chevron-left" width="18" />
              </button>
              <label class="digest-date-box">
                <span class="text-[10px] uppercase tracking-[0.24em] text-ink-light">{{ activeDateLabel }}</span>
                <input class="digest-date-input" :value="activeDateValue" type="date" @input="handleDateInput(($event.target as HTMLInputElement).value)">
              </label>
              <button class="digest-date-button" type="button" aria-label="看后一天" @click="jumpDate(1)">
                <Icon icon="mdi:chevron-right" width="18" />
              </button>
              <button class="digest-date-today" type="button" @click="jumpToToday">
                回今天
              </button>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <button
              class="btn-secondary min-h-11 px-4"
              type="button"
              :disabled="refreshingType === selectedType || loading"
              @click="refreshCurrent"
            >
              {{ refreshingType === selectedType ? '刷新中...' : '刷新这版' }}
            </button>
            <button
              class="btn-secondary min-h-11 px-4"
              type="button"
              :disabled="openNotebookSending || !openNotebookReady"
              @click="sendToOpenNotebook"
            >
              {{ openNotebookSending ? '发送中...' : '丢给 Open Notebook' }}
            </button>
            <button
              class="btn-primary min-h-11 px-4"
              type="button"
              :disabled="runningType === selectedType"
              @click="runNow(selectedType)"
            >
              {{ runningType === selectedType ? '执行中...' : '立即执行' }}
            </button>
            <button class="btn-ghost min-h-11 px-4" type="button" aria-label="打开设置抽屉" @click="openSettings">
              <Icon icon="mdi:tune-vertical" width="18" />
              打开设置
            </button>
          </div>
        </div>
      </header>

      <div v-if="notice" class="rounded-[24px] border px-5 py-4 text-sm"
        :class="{
          'border-[rgba(61,138,74,0.25)] bg-[rgba(61,138,74,0.08)] text-[var(--color-success)]': notice.type === 'success',
          'border-[rgba(196,47,60,0.25)] bg-[rgba(196,47,60,0.08)] text-[var(--color-error)]': notice.type === 'error',
          'border-[rgba(61,122,138,0.25)] bg-[rgba(61,122,138,0.08)] text-[var(--color-info)]': notice.type === 'info',
        }">
        {{ notice.text }}
      </div>

      <div v-if="loading" class="paper-card flex min-h-[760px] items-center justify-center rounded-[34px]">
        <Icon icon="mdi:loading" width="44" class="animate-spin text-ink-medium" />
      </div>

      <main v-else class="digest-main-grid grid min-h-0 gap-5 xl:h-[calc(100vh-18rem)] xl:grid-cols-[260px_360px_minmax(0,1fr)]">
        <aside class="digest-column digest-column--meta digest-panel paper-card rounded-[34px] px-5 py-6 md:px-6 xl:flex xl:min-h-0 xl:flex-col xl:overflow-hidden">
          <div class="digest-column__scroll space-y-5 xl:min-h-0 xl:overflow-y-auto xl:pr-1">
            <section class="space-y-3">
              <p class="text-xs uppercase tracking-[0.3em] text-ink-light">{{ activeMeta.short }}</p>
              <div>
                <h1 class="max-w-[8ch] text-4xl font-black leading-none text-ink-dark">{{ activeMeta.label }}</h1>
                <p class="mt-3 text-sm leading-7 text-ink-medium">{{ activeMeta.kicker }}</p>
              </div>
            </section>

            <section class="digest-panel__sub rounded-[24px] p-4">
              <p class="text-xs uppercase tracking-[0.28em] text-ink-light">时间窗</p>
              <p class="mt-3 text-xl font-bold text-ink-dark">{{ activePreview?.period_label || '还没拿到这版' }}</p>
              <p class="mt-2 text-sm leading-7 text-ink-medium">{{ activePreview?.generated_at ? `最近生成于 ${activePreview.generated_at}` : '先选日期，再看这版。' }}</p>
            </section>

            <section class="space-y-3">
              <div class="flex items-center justify-between">
                <p class="text-xs uppercase tracking-[0.28em] text-ink-light">分类</p>
                <span class="status-badge" :class="status?.running ? 'status-badge-success' : 'status-badge-warning'">
                  {{ status?.running ? 'running' : 'idle' }}
                </span>
              </div>

              <div v-if="activeCategories.length" class="space-y-2">
                <button
                  v-for="category in activeCategories"
                  :key="category.id"
                  class="digest-category-card"
                  :class="{ 'digest-category-card--active': selectedCategoryId === category.id }"
                  type="button"
                  @click="selectCategory(category)"
                >
                  <div class="text-left">
                    <p class="text-sm font-semibold text-ink-dark">{{ category.name }}</p>
                    <p class="mt-1 text-xs text-ink-medium">{{ category.feed_count }} 个源，{{ category.summary_count }} 条总结</p>
                  </div>
                  <span class="digest-category-card__count">{{ category.summary_count }}</span>
                </button>
              </div>

              <div v-else class="digest-empty-note rounded-[22px] px-4 py-4 text-sm text-ink-medium">
                这一版还没有分类。
              </div>
            </section>

            <section class="space-y-3">
              <p class="text-xs uppercase tracking-[0.28em] text-ink-light">执行状态</p>
              <div class="grid gap-2">
                <div v-for="item in statusChips" :key="item.label" class="digest-chip-row">
                  <span class="text-xs text-ink-light">{{ item.label }}</span>
                  <span class="text-sm font-semibold text-ink-dark">{{ item.value }}</span>
                </div>
              </div>
            </section>
          </div>
        </aside>

        <section class="digest-column digest-column--list digest-panel paper-card rounded-[34px] px-5 py-6 md:px-6 xl:flex xl:min-h-0 xl:flex-col xl:overflow-hidden">
          <div class="digest-column__scroll space-y-4 xl:min-h-0 xl:overflow-y-auto xl:pr-1">
            <div>
              <p class="text-xs uppercase tracking-[0.28em] text-ink-light">AI 总结列表</p>
              <h2 class="mt-2 text-2xl font-black text-ink-dark">{{ activeCategory?.name || '还没选分类' }}</h2>
              <p class="mt-2 text-sm leading-7 text-ink-medium">中间只列自动总结出来的内容，不列原始文章。</p>
            </div>

            <div v-if="activeSummaries.length" class="space-y-3">
              <button
                v-for="summary in activeSummaries"
                :key="summary.id"
                class="digest-summary-card"
                :class="{ 'digest-summary-card--active': selectedSummaryId === summary.id }"
                type="button"
                @click="selectSummary(summary)"
              >
                <div class="flex items-start justify-between gap-3">
                  <div class="min-w-0 text-left">
                    <div class="flex items-center gap-2 text-sm font-bold text-ink-dark">
                      <Icon :icon="summary.feed_icon || 'mdi:rss'" width="14" :style="{ color: summary.feed_color || '#3b6b87' }" />
                      <span class="truncate">{{ summary.feed_name }}</span>
                    </div>
                    <p class="mt-2 line-clamp-3 text-sm leading-7 text-ink-medium">{{ summaryExcerpt(summary) }}</p>
                  </div>
                  <span class="digest-summary-card__count">{{ summary.article_count }}</span>
                </div>
                <div class="mt-3 flex flex-wrap items-center gap-2 text-xs text-ink-light">
                  <span>{{ summary.category_name }}</span>
                  <span>·</span>
                  <span>{{ summary.created_at }}</span>
                </div>
                <div v-if="summary.topics?.length" class="mt-2 flex flex-wrap items-center gap-1.5">
                  <button
                    v-for="topic in summary.topics.slice(0, 5)"
                    :key="topic.slug"
                    class="digest-topic-tag"
                    :style="{ borderColor: getTopicCategoryMeta(topic.category).color + '40', backgroundColor: getTopicCategoryMeta(topic.category).color + '12' }"
                    type="button"
                  >
                    <Icon
                      :icon="topic.icon || getTopicCategoryMeta(topic.category).defaultIcon"
                      width="12"
                      :style="{ color: getTopicCategoryMeta(topic.category).color }"
                    />
                    <span :style="{ color: getTopicCategoryMeta(topic.category).color }">{{ topic.label }}</span>
                  </button>
                  <span v-if="summary.topics.length > 5" class="text-xs text-ink-light">+{{ summary.topics.length - 5 }}</span>
                </div>
              </button>
            </div>

            <div v-else class="digest-empty-note rounded-[22px] px-4 py-5 text-sm text-ink-medium">
              这个时间窗还没有 AI 总结。
            </div>
          </div>
        </section>

        <DigestDetail
          :summary="activeSummary"
          :active-type-label="activeMeta.label"
          :running="runningType === selectedType"
          @run="runNow(selectedType)"
          @open-settings="openSettings"
        />
      </main>

      <footer class="digest-footer paper-card rounded-[34px] px-5 py-5 md:px-6">
        <div class="grid gap-4 md:grid-cols-3">
          <article v-for="fact in bottomFacts" :key="fact.label" class="digest-footer__item">
            <p class="text-xs uppercase tracking-[0.24em] text-ink-light">{{ fact.label }}</p>
            <p class="mt-2 text-lg font-bold text-ink-dark">{{ fact.value }}</p>
          </article>
        </div>
      </footer>

      <section v-if="activeOpenNotebookResult" class="paper-card rounded-[34px] px-5 py-5 md:px-6 md:py-6">
        <div class="grid gap-5 xl:grid-cols-[280px_minmax(0,1fr)]">
          <aside class="open-notebook-meta rounded-[28px] px-5 py-5">
            <p class="text-xs uppercase tracking-[0.3em] text-ink-light">Open Notebook</p>
            <h3 class="mt-3 text-3xl font-black leading-none text-ink-dark">二次总结</h3>
            <p class="mt-3 text-sm leading-7 text-ink-medium">这版更短。适合扫一眼。</p>

            <div class="mt-5 space-y-3">
              <div class="digest-chip-row">
                <span class="text-xs text-ink-light">类型</span>
                <span class="text-sm font-semibold text-ink-dark">{{ activeOpenNotebookResult.digest_type }}</span>
              </div>
              <div class="digest-chip-row">
                <span class="text-xs text-ink-light">日期</span>
                <span class="text-sm font-semibold text-ink-dark">{{ activeOpenNotebookResult.anchor_date }}</span>
              </div>
              <a
                v-if="activeOpenNotebookResult.remote_url"
                class="open-notebook-link"
                :href="activeOpenNotebookResult.remote_url"
                target="_blank"
                rel="noreferrer"
              >
                去外部笔记看
                <Icon icon="mdi:arrow-top-right" width="16" />
              </a>
            </div>
          </aside>

          <div class="open-notebook-surface rounded-[28px] px-5 py-5 md:px-6">
            <div class="open-notebook-content max-w-none" v-html="renderedOpenNotebookResult" />
          </div>
        </div>
      </section>
    </div>

    <div
      v-if="showSettings"
      class="fixed inset-0 z-50 bg-[rgba(16,20,25,0.28)] backdrop-blur-[2px]"
      @click.self="closeSettings"
    >
      <div class="absolute inset-y-0 right-0 w-full max-w-[480px] overflow-hidden bg-transparent md:w-[460px]">
        <div class="digest-drawer h-full w-full transform-gpu overflow-y-auto p-3 md:p-4">
          <div class="digest-drawer__panel h-full rounded-[32px] border border-[var(--color-border-medium)] bg-[rgba(250,247,242,0.97)] shadow-[0_24px_80px_rgba(18,24,30,0.18)]">
            <div class="flex items-center justify-between border-b border-[var(--color-border-subtle)] px-5 py-4 md:px-6">
              <div>
                <p class="text-xs uppercase tracking-[0.28em] text-ink-light">Digest Setup</p>
                <h2 class="mt-1 text-xl font-black text-ink-dark">版面设置</h2>
              </div>
              <button class="btn-ghost min-h-11 min-w-11 px-0" type="button" aria-label="关闭设置" @click="closeSettings">
                <Icon icon="mdi:close" width="18" />
              </button>
            </div>

            <div class="h-[calc(100%-88px)] overflow-y-auto px-4 py-4 md:px-5">
              <DigestSettings
                @saved="async () => { closeSettings(); await loadStatus(); await loadPreview(selectedType, { silent: true, resetSelection: true }); if (!lockedType) { await loadPreview(selectedType === 'daily' ? 'weekly' : 'daily', { silent: true }); } }"
                @close="closeSettings"
              />
            </div>
          </div>
        </div>
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

.digest-shell {
  position: relative;
}

.digest-topbar,
.digest-panel,
.digest-footer {
  background: linear-gradient(145deg, rgba(255, 255, 255, 0.72), rgba(250, 247, 242, 0.9));
}

.digest-switcher {
  display: inline-grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.35rem;
  padding: 0.35rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid var(--color-border-subtle);
}

.digest-switcher__item,
.digest-lock-chip {
  min-height: 48px;
  display: flex;
  align-items: center;
  gap: 0.35rem;
  padding: 0.55rem 1rem;
  border-radius: 999px;
}

.digest-switcher__item {
  min-width: 112px;
  flex-direction: column;
  justify-content: center;
  align-items: flex-start;
  color: var(--color-ink-medium);
  transition: transform 180ms ease, background 180ms ease, color 180ms ease;
}

.digest-switcher__item:hover {
  background: rgba(45, 86, 112, 0.08);
  color: var(--color-ink-dark);
}

.digest-switcher__item--active {
  background: var(--color-ink-700);
  color: white;
  box-shadow: 0 10px 24px rgba(31, 59, 77, 0.18);
}

.digest-date-rail {
  display: inline-flex;
  align-items: stretch;
  gap: 0.55rem;
  flex-wrap: wrap;
}

.digest-date-box,
.digest-date-button,
.digest-date-today,
.digest-lock-chip,
.digest-panel__sub,
.digest-empty-note,
.digest-footer__item,
.digest-category-card,
.digest-chip-row,
.digest-summary-card {
  border: 1px solid var(--color-border-subtle);
  background: rgba(255, 255, 255, 0.72);
}

.digest-date-box {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 0.2rem;
  min-height: 48px;
  min-width: 164px;
  border-radius: 18px;
  padding: 0.45rem 0.8rem;
}

.digest-date-input {
  border: 0;
  background: transparent;
  color: var(--color-ink-dark);
  font-size: 0.95rem;
  font-weight: 700;
  outline: none;
}

.digest-date-button,
.digest-date-today {
  min-height: 48px;
  border-radius: 18px;
  color: var(--color-ink-dark);
  transition: transform 180ms ease, background 180ms ease, border-color 180ms ease;
}

.digest-date-button {
  min-width: 48px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.digest-date-today {
  padding: 0 1rem;
  font-size: 0.9rem;
  font-weight: 700;
}

.digest-date-button:hover,
.digest-date-today:hover,
.digest-category-card:hover,
.digest-summary-card:hover {
  transform: translateY(-1px);
  border-color: rgba(45, 86, 112, 0.28);
}

.digest-category-card,
.digest-chip-row,
.digest-summary-card {
  width: 100%;
  text-align: left;
}

.digest-category-card,
.digest-chip-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  border-radius: 20px;
  padding: 0.85rem 1rem;
}

.digest-category-card {
  transition: border-color 180ms ease, background 180ms ease, transform 180ms ease;
}

.digest-category-card--active,
.digest-summary-card--active {
  border-color: rgba(45, 86, 112, 0.32);
  background: rgba(45, 86, 112, 0.08);
}

.digest-category-card__count,
.digest-summary-card__count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 2rem;
  height: 2rem;
  padding: 0 0.6rem;
  border-radius: 999px;
  background: rgba(193, 47, 47, 0.08);
  color: var(--color-print-red-700);
  font-size: 0.85rem;
  font-weight: 700;
}

.digest-summary-card {
  border-radius: 24px;
  padding: 1rem;
  transition: border-color 180ms ease, background 180ms ease, transform 180ms ease;
}

.open-notebook-meta,
.open-notebook-surface,
.open-notebook-link {
  border: 1px solid var(--color-border-subtle);
  background: rgba(255, 255, 255, 0.74);
}

.open-notebook-meta {
  background:
    radial-gradient(circle at top left, rgba(193, 47, 47, 0.08), transparent 42%),
    rgba(255, 250, 244, 0.94);
}

.open-notebook-link {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  border-radius: 18px;
  padding: 0.9rem 1rem;
  color: var(--color-ink-dark);
  font-size: 0.92rem;
  font-weight: 700;
}

.open-notebook-content :deep(h1),
.open-notebook-content :deep(h2),
.open-notebook-content :deep(h3) {
  color: var(--color-ink-dark);
  font-weight: 900;
}

.open-notebook-content :deep(p),
.open-notebook-content :deep(li) {
  color: var(--color-ink-medium);
  line-height: 1.9;
}

.open-notebook-content :deep(ul),
.open-notebook-content :deep(ol) {
  padding-left: 1.25rem;
}

.digest-footer__item {
  border-radius: 24px;
  padding: 1rem 1.1rem;
}

.digest-drawer {
  animation: drawer-slide-in 220ms cubic-bezier(0.22, 1, 0.36, 1);
}

@keyframes drawer-slide-in {
  from {
    opacity: 0;
    transform: translateX(28px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .digest-drawer,
  .digest-switcher__item,
  .digest-date-button,
  .digest-date-today,
  .digest-category-card,
  .digest-summary-card,
  .paper-card {
    animation: none !important;
    transition: none !important;
  }
}

@media (max-width: 1279px) {
  .digest-stage {
    padding-bottom: 2rem;
  }
}

.digest-topic-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  border-radius: 999px;
  padding: 0.15rem 0.5rem;
  font-size: 0.7rem;
  font-weight: 600;
  transition: transform 120ms ease, box-shadow 120ms ease;
}

.digest-topic-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
}
</style>
