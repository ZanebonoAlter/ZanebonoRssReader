<script setup lang="ts">
import { Icon } from '@iconify/vue'
import type { TopicCategory } from '~/api/topicGraph'
import type { TimelineDigest, TimelineFilters } from '~/types/timeline'
import AIAnalysisPanel from './AIAnalysisPanel.vue'
import TimelineHeader from './TimelineHeader.vue'
import TimelineItem from './TimelineItem.vue'

interface TopicInfo {
  slug: string
  label: string
  category: TopicCategory
}

interface AnalysisResult {
  timeline?: Array<{
    date: string
    title: string
    summary: string
    sources: Array<{ articleId: number; title: string }>
  }>
  keyMoments?: string[]
  relatedEntities?: Array<{ name: string; type: string }>
  summary?: string
  profile?: {
    name: string
    role: string
    background: string
  }
  appearances?: Array<{
    date: string
    context: string
    quote: string
    articleId: number
  }>
  trend?: Array<{ date: string; value: number }>
  relatedTopics?: string[]
  coOccurrence?: Array<{ term: string; count: number }>
  contextExamples?: string[]
}

interface Props {
  selectedTopic: TopicInfo | null
  items: TimelineDigest[]
  filters: TimelineFilters
  activeDigestId?: string | null
  aiAnalysisStatus?: 'idle' | 'loading' | 'completed' | 'error'
  aiAnalysisProgress?: number
  aiAnalysisResult?: AnalysisResult | null
  aiAnalysisError?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  activeDigestId: null,
  aiAnalysisStatus: 'idle',
  aiAnalysisProgress: 0,
  aiAnalysisResult: null,
  aiAnalysisError: null,
})

const emit = defineEmits<{
  'filter-change': [filters: TimelineFilters]
  'ai-analysis': []
  'ai-analysis-start': []
  'ai-analysis-retry': []
  'open-article': [articleId: number]
  'select-digest': [digestId: string]
  'preview-digest': [digestId: string]
}>()

function handleFilterChange(filters: TimelineFilters) {
  emit('filter-change', filters)
}

function handleAIAnalysis() {
  emit('ai-analysis')
}

function handleAIAnalysisStart() {
  emit('ai-analysis-start')
}

function handleAIAnalysisRetry() {
  emit('ai-analysis-retry')
}

function handleOpenArticle(articleId: number) {
  emit('open-article', articleId)
}

function handleSelectDigest(digestId: string) {
  emit('select-digest', digestId)
}

function handlePreviewDigest(digestId: string) {
  emit('preview-digest', digestId)
}
</script>

<template>
  <div class="topic-timeline">
    <TimelineHeader
      :topic="selectedTopic"
      :total-count="items.length"
      :filters="filters"
      @filter-change="handleFilterChange"
      @ai-analysis="handleAIAnalysis"
    />

    <AIAnalysisPanel
      v-if="selectedTopic"
      :selected-topic="selectedTopic"
      :analysis-type="selectedTopic.category"
      :status="aiAnalysisStatus"
      :progress="aiAnalysisProgress"
      :result="aiAnalysisResult"
      :error="aiAnalysisError"
      @start-analysis="handleAIAnalysisStart"
      @retry="handleAIAnalysisRetry"
      @open-article="handleOpenArticle"
    />

    <div class="timeline-content">
      <div v-if="!selectedTopic" class="timeline-empty">
        <Icon icon="mdi:cursor-default-click" width="32" />
        <span>请先选择一个题材查看相关日报</span>
      </div>

      <div v-else-if="items.length === 0" class="timeline-empty">
        <Icon icon="mdi:file-search" width="32" />
        <span>这个题材在当前窗口里还没有日报</span>
      </div>

      <div v-else class="timeline-list">
        <TimelineItem
          v-for="(item, index) in items"
          :key="item.id"
          :item="item"
          :is-first="index === 0"
          :is-last="index === items.length - 1"
          :is-active="props.activeDigestId === item.id"
          :highlighted-tag-slugs="selectedTopic ? [selectedTopic.slug] : []"
          @open-article="handleOpenArticle"
          @select="handleSelectDigest"
          @preview-digest="handlePreviewDigest"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.topic-timeline {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  height: 100%;
}

.timeline-content {
  flex: 1;
  min-height: 0;
  padding-right: 0.25rem;
}

.timeline-empty {
  display: flex;
  min-height: 16rem;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 0.75rem;
  border-radius: 1.2rem;
  border: 1px dashed rgba(255, 255, 255, 0.1);
  background: rgba(255, 255, 255, 0.02);
  padding: 2rem 1rem;
  color: rgba(255, 255, 255, 0.52);
  text-align: center;
}

.timeline-list {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}
</style>
