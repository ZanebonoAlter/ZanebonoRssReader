<script setup lang="ts">
import { Icon } from '@iconify/vue'
import type { TopicCategory } from '~/api/topicGraph'

interface TopicInfo {
  slug: string
  label: string
  category: TopicCategory
}

interface Props {
  topic: TopicInfo | null
  totalCount: number
}

defineProps<Props>()

const categoryLabels: Record<TopicCategory, string> = {
  event: '事件',
  person: '人物',
  keyword: '关键词',
}
</script>

<template>
  <header class="timeline-header">
    <div class="timeline-header__topic">
      <template v-if="topic">
        <h2 class="timeline-header__title">{{ topic.label }}</h2>
        <span class="timeline-header__category" :class="`timeline-header__category--${topic.category}`">
          {{ categoryLabels[topic.category] }}
        </span>
        <span class="timeline-header__count">
          <Icon icon="mdi:file-document-outline" width="14" />
          {{ totalCount }} 篇日报
        </span>
      </template>
      <template v-else>
        <h2 class="timeline-header__title timeline-header__title--placeholder">选择题材查看日报</h2>
      </template>
    </div>
  </header>
</template>

<style scoped>
.timeline-header {
  padding: 1rem 1.25rem;
  border-radius: 1.25rem;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: linear-gradient(180deg, rgba(20, 30, 42, 0.9), rgba(12, 18, 26, 0.95));
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
}

.timeline-header__topic {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.timeline-header__title {
  font-size: 1.35rem;
  font-weight: 700;
  color: rgba(255, 255, 255, 0.95);
  line-height: 1.3;
}

.timeline-header__title--placeholder {
  color: rgba(255, 255, 255, 0.4);
  font-weight: 500;
}

.timeline-header__category {
  font-size: 0.7rem;
  padding: 0.22rem 0.55rem;
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
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.8rem;
  color: rgba(255, 255, 255, 0.55);
  padding: 0.3rem 0.65rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.08);
}

@media (max-width: 640px) {
  .timeline-header {
    padding: 0.85rem 1rem;
  }

  .timeline-header__title {
    font-size: 1.15rem;
  }
}
</style>