<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import { useDebounceFn } from '@vueuse/core'
import type { TopicCategory } from '~/api/topicGraph'
import type { TimelineFilters } from '~/types/timeline'

interface TopicInfo {
  slug: string
  label: string
  category: TopicCategory
}

interface Props {
  topic: TopicInfo | null
  totalCount: number
  filters: TimelineFilters
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'filter-change': [filters: TimelineFilters]
  'ai-analysis': []
}>()

const categoryLabels: Record<TopicCategory, string> = {
  event: '事件',
  person: '人物',
  keyword: '关键词',
}

const dateRangeOptions = [
  { value: null, label: '全部' },
  { value: 'today', label: '今天' },
  { value: 'week', label: '本周' },
  { value: 'month', label: '本月' },
  { value: 'custom', label: '自定义' },
] as const

const showCustomDate = computed(() => props.filters.dateRange === 'custom')

const localFilters = ref<TimelineFilters>({ ...props.filters })

// Sync local filters with props
watch(() => props.filters, (newFilters) => {
  localFilters.value = { ...newFilters }
}, { deep: true })

// Debounced filter change handler for custom date inputs
const debouncedFilterChange = useDebounceFn((filters: TimelineFilters) => {
  emit('filter-change', filters)
}, 300)

function handleDateRangeChange(dateRange: typeof dateRangeOptions[number]['value']) {
  const newFilters: TimelineFilters = {
    ...localFilters.value,
    dateRange,
    startDate: dateRange === 'custom' ? localFilters.value.startDate : undefined,
    endDate: dateRange === 'custom' ? localFilters.value.endDate : undefined,
  }
  localFilters.value = newFilters
  emit('filter-change', newFilters)
}

function handleCustomDateChange() {
  if (localFilters.value.startDate && localFilters.value.endDate) {
    debouncedFilterChange({ ...localFilters.value, dateRange: 'custom' })
  }
}

function handleAIAnalysis() {
  emit('ai-analysis')
}
</script>

<template>
  <header class="timeline-header">
    <div class="timeline-header__main">
      <div class="timeline-header__topic">
        <template v-if="topic">
          <h2 class="timeline-header__title">{{ topic.label }}</h2>
          <span class="timeline-header__category" :class="`timeline-header__category--${topic.category}`">
            {{ categoryLabels[topic.category] }}
          </span>
        </template>
        <template v-else>
          <h2 class="timeline-header__title timeline-header__title--placeholder">选择题材查看日报</h2>
        </template>
      </div>

      <div class="timeline-header__count">
        <Icon icon="mdi:file-document-outline" width="16" />
        <span>{{ totalCount }} 篇日报</span>
      </div>
    </div>

    <div class="timeline-header__filters">
      <div class="timeline-header__filter-group">
        <span class="timeline-header__filter-label">时间范围</span>
        <div class="timeline-header__filter-buttons">
          <button
            v-for="option in dateRangeOptions"
            :key="String(option.value)"
            type="button"
            class="timeline-header__filter-btn"
            :class="{ 'timeline-header__filter-btn--active': filters.dateRange === option.value }"
            @click="handleDateRangeChange(option.value)"
          >
            {{ option.label }}
          </button>
        </div>
      </div>

      <div v-if="showCustomDate" class="timeline-header__custom-date">
        <input
          v-model="localFilters.startDate"
          type="date"
          class="timeline-header__date-input"
          @change="handleCustomDateChange"
        />
        <span class="timeline-header__date-separator">至</span>
        <input
          v-model="localFilters.endDate"
          type="date"
          class="timeline-header__date-input"
          @change="handleCustomDateChange"
        />
      </div>

      <button
        v-if="topic"
        type="button"
        class="timeline-header__ai-btn"
        @click="handleAIAnalysis"
      >
        <Icon icon="mdi:robot-outline" width="18" />
        <span>AI 分析</span>
      </button>
    </div>
  </header>
</template>

<style scoped>
.timeline-header {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  padding: 1.25rem;
  border-radius: 1.25rem;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: linear-gradient(180deg, rgba(20, 30, 42, 0.9), rgba(12, 18, 26, 0.95));
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
}

.timeline-header__main {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.timeline-header__topic {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.timeline-header__title {
  font-size: 1.5rem;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.95);
  line-height: 1.3;
}

.timeline-header__title--placeholder {
  color: rgba(255, 255, 255, 0.4);
  font-weight: 500;
}

.timeline-header__category {
  font-size: 0.75rem;
  padding: 0.25rem 0.6rem;
  border-radius: 999px;
  font-weight: 600;
  letter-spacing: 0.05em;
}

.timeline-header__category--event {
  background: rgba(245, 158, 11, 0.2);
  border: 1px solid rgba(245, 158, 11, 0.4);
  color: rgba(252, 211, 77, 0.9);
}

.timeline-header__category--person {
  background: rgba(16, 185, 129, 0.2);
  border: 1px solid rgba(16, 185, 129, 0.4);
  color: rgba(110, 231, 183, 0.9);
}

.timeline-header__category--keyword {
  background: rgba(99, 102, 241, 0.2);
  border: 1px solid rgba(99, 102, 241, 0.4);
  color: rgba(165, 180, 252, 0.9);
}

.timeline-header__count {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.85rem;
  color: rgba(255, 255, 255, 0.6);
  padding: 0.4rem 0.75rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
}

.timeline-header__filters {
  display: flex;
  align-items: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.timeline-header__filter-group {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.timeline-header__filter-label {
  font-size: 0.75rem;
  color: rgba(255, 255, 255, 0.5);
  letter-spacing: 0.05em;
}

.timeline-header__filter-buttons {
  display: flex;
  gap: 0.25rem;
}

.timeline-header__filter-btn {
  font-size: 0.75rem;
  padding: 0.35rem 0.65rem;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: transparent;
  color: rgba(255, 255, 255, 0.6);
  cursor: pointer;
  transition: all 0.15s ease;
}

.timeline-header__filter-btn:hover {
  border-color: rgba(255, 255, 255, 0.2);
  color: rgba(255, 255, 255, 0.8);
}

.timeline-header__filter-btn--active {
  border-color: rgba(240, 138, 75, 0.5);
  background: rgba(240, 138, 75, 0.15);
  color: rgba(255, 200, 150, 0.95);
}

.timeline-header__custom-date {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.timeline-header__date-input {
  font-size: 0.75rem;
  padding: 0.35rem 0.5rem;
  border-radius: 0.375rem;
  border: 1px solid rgba(255, 255, 255, 0.15);
  background: rgba(255, 255, 255, 0.05);
  color: rgba(255, 255, 255, 0.9);
  outline: none;
  transition: all 0.15s ease;
}

.timeline-header__date-input:focus {
  border-color: rgba(240, 138, 75, 0.5);
  background: rgba(255, 255, 255, 0.08);
}

.timeline-header__date-separator {
  font-size: 0.75rem;
  color: rgba(255, 255, 255, 0.5);
}

.timeline-header__ai-btn {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.8rem;
  padding: 0.5rem 1rem;
  border-radius: 999px;
  border: 1px solid rgba(240, 138, 75, 0.4);
  background: linear-gradient(135deg, rgba(240, 138, 75, 0.2), rgba(255, 160, 100, 0.15));
  color: rgba(255, 200, 150, 0.95);
  cursor: pointer;
  transition: all 0.2s ease;
  margin-left: auto;
}

.timeline-header__ai-btn:hover {
  border-color: rgba(240, 138, 75, 0.6);
  background: linear-gradient(135deg, rgba(240, 138, 75, 0.3), rgba(255, 160, 100, 0.25));
  transform: translateY(-1px);
}

.timeline-header__ai-btn:focus-visible {
  outline: 2px solid rgba(240, 138, 75, 0.5);
  outline-offset: 2px;
}

@media (max-width: 640px) {
  .timeline-header {
    padding: 1rem;
  }

  .timeline-header__main {
    flex-direction: column;
    gap: 0.75rem;
  }

  .timeline-header__title {
    font-size: 1.25rem;
  }

  .timeline-header__filters {
    flex-direction: column;
    align-items: flex-start;
    gap: 0.75rem;
  }

  .timeline-header__ai-btn {
    margin-left: 0;
    width: 100%;
    justify-content: center;
  }
}
</style>