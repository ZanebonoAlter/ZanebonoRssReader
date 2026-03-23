<script setup lang="ts">
import { computed, ref } from 'vue'
import { Icon } from '@iconify/vue'
import ArticleTagList from '../../articles/components/ArticleTagList.vue'
import FeedIcon from '~/components/feed/FeedIcon.vue'
import type { TimelineDigest } from '~/types/timeline'

interface Props {
  item: TimelineDigest
  isFirst: boolean
  isLast: boolean
  isActive?: boolean
  highlightedTagSlugs?: string[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  openArticle: [articleId: number]
  select: [itemId: string]
  previewDigest: [itemId: string]
}>()

const isExpanded = ref(false)

const dateValue = computed(() => parseDateValue(props.item.createdAt))
const formattedDate = computed(() => {
  if (!dateValue.value) return '日期待补'
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'short',
    day: 'numeric',
    weekday: 'short',
  }).format(dateValue.value)
})
const formattedTime = computed(() => {
  if (!dateValue.value) return '时间待补'
  return new Intl.DateTimeFormat('zh-CN', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(dateValue.value)
})

function parseDateValue(value: string | null | undefined) {
  if (!value) return null
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) {
    return null
  }
  return parsed
}

function toggleExpand() {
  isExpanded.value = !isExpanded.value
}

function handleSelect() {
  emit('select', props.item.id)
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' || event.key === ' ') {
    event.preventDefault()
    handleSelect()
  }
}
</script>

<template>
  <article class="timeline-item" :class="{ 'timeline-item--first': isFirst, 'timeline-item--last': isLast }">
    <div class="timeline-item__marker">
      <div class="timeline-item__dot" />
      <div v-if="!isLast" class="timeline-item__line" />
    </div>

    <div class="timeline-item__content">
      <div class="timeline-item__header">
        <time class="timeline-item__time">{{ formattedTime }}</time>
        <span class="timeline-item__date">{{ formattedDate }}</span>
        <span class="timeline-item__source">
          <FeedIcon :icon="item.feedIcon" :size="14" />
          {{ item.feedName }}
        </span>
      </div>

      <div
        class="timeline-item__body"
        :class="{ 'timeline-item__body--active': props.isActive }"
        role="button"
        tabindex="0"
        @click="handleSelect"
        @keydown="handleKeydown"
      >
        <div class="timeline-item__eyebrow-row">
          <span class="timeline-item__eyebrow">{{ item.categoryName || '未分类日报' }}</span>
          <span class="timeline-item__count">{{ item.articleCount }} 篇来源文章</span>
        </div>

        <h3 class="timeline-item__title">{{ item.title }}</h3>
        <p class="timeline-item__summary" :class="{ 'timeline-item__summary--expanded': isExpanded }">
          {{ item.summary }}
        </p>

        <ArticleTagList
          v-if="item.tags.length"
          class="timeline-item__tags"
          :tags="item.tags"
          :highlighted-slugs="highlightedTagSlugs || []"
          compact
          :max-visible="5"
        />

        <div v-if="item.articles.length" class="timeline-item__sources">
          <button
            v-for="article in item.articles"
            :key="article.id"
            type="button"
            class="timeline-item__source-link"
            @click.stop="emit('openArticle', article.id)"
          >
            {{ article.title }}
          </button>
        </div>
      </div>

      <div class="timeline-item__footer">
        <button
          type="button"
          class="timeline-item__expand"
          @click="toggleExpand"
        >
          <Icon :icon="isExpanded ? 'mdi:chevron-up' : 'mdi:chevron-down'" width="16" />
          <span>{{ isExpanded ? '收起日报' : '展开日报' }}</span>
        </button>
        <button
          type="button"
          class="timeline-item__expand"
          @click="emit('previewDigest', item.id)"
        >
          <Icon icon="mdi:text-box-search-outline" width="16" />
          <span>查看日报</span>
        </button>
      </div>
    </div>
  </article>
</template>

<style scoped>
.timeline-item {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr);
  gap: 0.9rem;
  position: relative;
}

.timeline-item--first .timeline-item__dot {
  background: linear-gradient(135deg, rgba(240, 138, 75, 0.94), rgba(255, 182, 123, 0.98));
  box-shadow: 0 0 12px rgba(240, 138, 75, 0.34);
}

.timeline-item__marker {
  display: flex;
  flex-direction: column;
  align-items: center;
  position: relative;
}

.timeline-item__dot {
  width: 12px;
  height: 12px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.28);
  border: 2px solid rgba(255, 255, 255, 0.16);
  flex-shrink: 0;
}

.timeline-item__line {
  width: 2px;
  flex: 1;
  min-height: 24px;
  margin-top: 4px;
  background: linear-gradient(180deg, rgba(240, 138, 75, 0.28), rgba(255, 255, 255, 0.03));
}

.timeline-item__content {
  display: grid;
  gap: 0.55rem;
  padding-bottom: 1.4rem;
}

.timeline-item--last .timeline-item__content {
  padding-bottom: 0;
}

.timeline-item__header {
  display: flex;
  align-items: center;
  gap: 0.55rem;
  flex-wrap: wrap;
}

.timeline-item__time {
  font-size: 0.75rem;
  font-weight: 600;
  color: rgba(240, 138, 75, 0.92);
  letter-spacing: 0.04em;
}

.timeline-item__date {
  font-size: 0.72rem;
  color: rgba(214, 225, 235, 0.56);
}

.timeline-item__source {
  margin-left: auto;
  padding: 0.22rem 0.58rem;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(255, 255, 255, 0.04);
  font-size: 0.72rem;
  color: rgba(220, 230, 239, 0.62);
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
}

.timeline-item__body {
  display: grid;
  gap: 0.72rem;
  text-align: left;
  border-radius: 1.15rem;
  border: 1px solid rgba(141, 173, 214, 0.14);
  background: linear-gradient(180deg, rgba(20, 30, 42, 0.88), rgba(10, 16, 24, 0.94));
  padding: 1rem;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.18);
}

.timeline-item__body--active {
  border-color: rgba(240, 138, 75, 0.34);
  background: linear-gradient(180deg, rgba(28, 39, 53, 0.94), rgba(13, 20, 29, 0.98));
  box-shadow: 0 20px 44px rgba(0, 0, 0, 0.24);
}

.timeline-item__eyebrow-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.timeline-item__eyebrow,
.timeline-item__count {
  font-size: 0.72rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.timeline-item__eyebrow {
  color: rgba(171, 194, 217, 0.74);
}

.timeline-item__count {
  color: rgba(241, 223, 208, 0.76);
}

.timeline-item__title {
  font-size: 1.02rem;
  font-weight: 700;
  line-height: 1.45;
  color: rgba(248, 252, 255, 0.96);
}

.timeline-item__summary {
  font-size: 0.9rem;
  line-height: 1.68;
  color: rgba(212, 225, 236, 0.8);
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.timeline-item__summary--expanded {
  display: block;
}

.timeline-item__tags,
.timeline-item__sources {
  display: flex;
  flex-wrap: wrap;
  gap: 0.45rem;
}

.timeline-item__source-link {
  border-radius: 999px;
  font-size: 0.73rem;
}

.timeline-item__source-link {
  border: 1px solid rgba(240, 138, 75, 0.2);
  background: rgba(255, 255, 255, 0.04);
  padding: 0.34rem 0.68rem;
  color: rgba(255, 231, 213, 0.88);
  transition: all 0.18s ease;
}

.timeline-item__source-link:hover,
.timeline-item__source-link:focus-visible {
  border-color: rgba(240, 138, 75, 0.4);
  background: rgba(240, 138, 75, 0.1);
}

.timeline-item__footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.45rem;
  flex-wrap: wrap;
}

.timeline-item__expand {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  border-radius: 999px;
  padding: 0.28rem 0.62rem;
  color: rgba(214, 225, 236, 0.68);
  background: rgba(255, 255, 255, 0.04);
  border: 1px solid rgba(255, 255, 255, 0.08);
  font-size: 0.74rem;
}

.timeline-item__expand:hover,
.timeline-item__expand:focus-visible {
  color: rgba(248, 252, 255, 0.92);
  border-color: rgba(240, 138, 75, 0.28);
}

@media (max-width: 640px) {
  .timeline-item {
    grid-template-columns: 34px minmax(0, 1fr);
    gap: 0.65rem;
  }

  .timeline-item__body {
    padding: 0.85rem;
  }

  .timeline-item__source {
    margin-left: 0;
  }
}
</style>
