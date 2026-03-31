<script setup lang="ts">
import { Icon } from '@iconify/vue'
import type { TopicCategory } from '~/api/topicGraph'
import type { TimelineDigest } from '~/types/timeline'
import TimelineHeader from './TimelineHeader.vue'
import TimelineItem from './TimelineItem.vue'
import TimelinePendingItem from './TimelinePendingItem.vue'

interface TopicInfo {
  slug: string
  label: string
  category: TopicCategory
}

interface Props {
  selectedTopic: TopicInfo | null
  items: TimelineDigest[]
  activeDigestId?: string | null
  pendingArticleCount?: number
  selectedPendingNode?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  activeDigestId: null,
  pendingArticleCount: 0,
  selectedPendingNode: false,
})

const emit = defineEmits<{
  'open-article': [articleId: number]
  'select-digest': [digestId: string]
  'preview-digest': [digestId: string]
  'select-pending': []
}>()

function handleOpenArticle(articleId: number) {
  emit('open-article', articleId)
}

function handleSelectDigest(digestId: string) {
  emit('select-digest', digestId)
}

function handlePreviewDigest(digestId: string) {
  emit('preview-digest', digestId)
}

function handleSelectPending() {
  emit('select-pending')
}
</script>

<template>
  <div class="topic-timeline">
    <TimelineHeader
      :topic="selectedTopic"
      :total-count="items.length"
    />

    <div class="timeline-content">
      <div v-if="!selectedTopic" class="timeline-empty">
        <Icon icon="mdi:cursor-default-click" width="32" />
        <span>请先选择一个题材查看相关日报</span>
      </div>

      <template v-else>
        <div v-if="items.length === 0 && props.pendingArticleCount === 0" class="timeline-empty">
          <Icon icon="mdi:file-search" width="32" />
          <span>这个题材在当前窗口里还没有日报</span>
        </div>

        <div v-else class="timeline-list">
          <TimelinePendingItem
            v-if="props.pendingArticleCount > 0"
            :count="props.pendingArticleCount"
            :is-active="props.selectedPendingNode"
            @select="handleSelectPending"
          />
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
      </template>
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
