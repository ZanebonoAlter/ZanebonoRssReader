<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { TopicCategory, TopicGraphDetailPayload } from '~/api/topicGraph'
import type { TimelineDigestSelection } from '~/types/timeline'
import { normalizeTopicCategory } from '~/features/topic-graph/utils/normalizeTopicCategory'
import KeywordCloud, { type Keyword } from './KeywordCloud.vue'

interface Props {
  detail: TopicGraphDetailPayload | null
  selectedDigest?: TimelineDigestSelection | null
  loading?: boolean
  error?: string | null
  dataState?: string
  selectedKeyword?: string | null
  selectedTagSlug?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  selectedDigest: null,
  error: null,
  dataState: 'empty',
  selectedKeyword: null,
  selectedTagSlug: null,
})

const emit = defineEmits<{
  openArticle: [articleId: number]
  highlightKeyword: [keywordSlug: string | null]
}>()

// Internal selected keyword state (for toggle behavior)
const internalSelectedKeyword = ref<string | null>(null)

// Computed selected keyword (prefer prop, fallback to internal)
const activeKeywordSlug = computed(() => props.selectedKeyword !== undefined ? props.selectedKeyword : internalSelectedKeyword.value)

const deduplicatedArticles = computed(() => {
  if (!props.detail || !props.selectedDigest) return []

  const topicArticleIds = new Set(props.detail.articles.map(article => article.id))
  const matchedIds = new Set(props.selectedDigest.matchedArticleIds)

  // 如果有选中的标签slug，只显示与该标签相关的文章
  const hasSelectedTag = props.selectedTagSlug && props.selectedTagSlug.trim() !== ''

  return props.selectedDigest.articles
    .filter(article => {
      // 如果没有选中的标签，显示所有文章
      if (!hasSelectedTag) return true

      // 如果选中了标签，只显示与当前话题标签匹配的文章
      // matchedIds 包含与当前标签相关的文章ID
      return matchedIds.has(article.id) || topicArticleIds.has(article.id)
    })
    .map(article => ({
      ...article,
      feedName: props.selectedDigest?.feedName || props.detail?.topic.label || '来源文章',
      categoryName: props.selectedDigest?.categoryName || '日报',
      summaryTitle: props.selectedDigest?.title || props.detail?.topic.label || '当前日报',
      matchedTopic: matchedIds.has(article.id) || topicArticleIds.has(article.id),
      matchedBySummaryOnly: !topicArticleIds.has(article.id),
    }))
    .sort((left, right) => {
      if (left.matchedTopic === right.matchedTopic) return 0
      return left.matchedTopic ? -1 : 1
    })
})

const keywords = computed((): Keyword[] => {
  if (!props.detail?.related_tags?.length) return []

  const maxCooccurrence = Math.max(...props.detail.related_tags.map(tag => tag.cooccurrence), 1)

  return props.detail.related_tags.slice(0, 18).map(tag => ({
    slug: tag.slug,
    label: tag.label,
    count: tag.cooccurrence,
    relevance: Math.max(tag.cooccurrence / maxCooccurrence, 0.28),
  }))
})

const shouldScrollFeaturedArticles = computed(() => deduplicatedArticles.value.length > 8)

const topicCategoryLabels: Record<TopicCategory, string> = {
  event: '事件',
  person: '人物',
  keyword: '关键词',
}

const displayTopicCategory = computed<TopicCategory>(() => {
  if (!props.detail) return 'keyword'
  return normalizeTopicCategory(props.detail.topic.category, props.detail.topic.kind)
})

function handleKeywordSelect(keyword: Keyword) {
  if (activeKeywordSlug.value === keyword.slug) {
    internalSelectedKeyword.value = null
    emit('highlightKeyword', null)
  } else {
    internalSelectedKeyword.value = keyword.slug
    emit('highlightKeyword', keyword.slug)
  }
}

watch(() => props.selectedKeyword, (value) => {
  if (value === null) {
    internalSelectedKeyword.value = null
  }
})
</script>

<template>
  <aside
    class="topic-sidebar rounded-[34px] px-5 py-5 md:px-6 md:py-6"
    data-testid="topic-graph-sidebar"
    :data-state="props.dataState"
  >
    <div v-if="props.loading" class="topic-sidebar__empty">正在展开话题脉络...</div>
    <div v-else-if="props.error" class="topic-sidebar__empty">{{ props.error }}</div>
    <div v-else-if="!props.detail" class="topic-sidebar__empty">点一个节点，右侧就会展开这类题材的近期总结、历史轨迹和外部入口。</div>
    <div v-else class="topic-sidebar__content">
      <!-- Current topic header -->
      <section class="space-y-3">
        <p class="topic-sidebar__eyebrow">当前焦点</p>
        <div class="flex flex-wrap items-center gap-3">
          <h2 class="font-serif text-3xl text-[var(--topic-ink-strong)]">{{ props.detail.topic.label }}</h2>
          <span class="topic-pill" :class="`topic-pill--${displayTopicCategory}`">
            {{ topicCategoryLabels[displayTopicCategory] }}
          </span>
        </div>
        <p class="text-sm text-[var(--topic-ink-medium)]">
          {{ props.selectedDigest ? '当前日报来源文章' : '先从下方选择一条日报' }}
        </p>
      </section>

      <!-- Related Articles (deduplicated) -->
      <section class="topic-panel topic-panel--featured rounded-[28px] p-4 md:p-5">
        <div class="flex items-center justify-between gap-3">
          <div>
            <p class="topic-sidebar__eyebrow">日报文章</p>
            <p v-if="props.selectedDigest" class="topic-related-card__context mt-2">{{ props.selectedDigest.title }}</p>
          </div>
          <span class="topic-summary__count">{{ deduplicatedArticles.length }} 条</span>
        </div>
        <div
          v-if="deduplicatedArticles.length"
          class="topic-sidebar__news-scroll mt-4"
          :class="{ 'topic-sidebar__news-scroll--bounded': shouldScrollFeaturedArticles }"
          data-testid="topic-graph-related-articles"
        >
          <div class="grid gap-3">
            <button
              v-for="article in deduplicatedArticles"
              :key="article.link"
              class="topic-related-card"
              type="button"
              data-testid="sidebar-article"
              :data-article-id="String(article.id)"
              @click="emit('openArticle', article.id)"
            >
              <p class="topic-related-card__meta">{{ article.feedName }} · {{ article.categoryName }}</p>
              <h3 class="topic-related-card__title">{{ article.title }}</h3>
              <p class="topic-related-card__context">来自：{{ article.summaryTitle }}</p>
              <p class="topic-related-card__note" :class="{ 'topic-related-card__note--soft': article.matchedBySummaryOnly }">
                {{ article.matchedBySummaryOnly ? '命中日报关键词，article 本身暂未打上当前 topic 标签' : '命中当前 topic/article 标签' }}
              </p>
            </button>
          </div>
        </div>
        <div v-else class="topic-sidebar__empty topic-sidebar__empty--soft">点击下方日报后，这里只展示该日报里命中当前主题的文章。</div>
      </section>

      <!-- Keyword Cloud (Related Topics) -->
      <section v-if="keywords.length > 0" class="topic-panel rounded-[26px] p-4">
        <p class="topic-sidebar__eyebrow">相关主题</p>
        <div class="mt-4">
          <KeywordCloud
            :keywords="keywords"
            :selected-keyword="activeKeywordSlug"
            @select="handleKeywordSelect"
          />
          <p class="keywords-hint">
            点击标签，只高亮当前标签节点和它的一跳邻居
          </p>
        </div>
      </section>
    </div>
  </aside>
</template>

<style scoped>
.topic-sidebar {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  position: relative;
  overflow: hidden;
  --topic-ink-strong: rgba(248, 251, 255, 0.96);
  --topic-ink-medium: rgba(210, 221, 232, 0.82);
  --topic-ink-soft: rgba(148, 168, 188, 0.7);
  --topic-border: rgba(123, 154, 192, 0.18);
  --topic-border-strong: rgba(123, 154, 192, 0.28);
  --topic-card: linear-gradient(180deg, rgba(20, 30, 42, 0.86), rgba(11, 17, 25, 0.92));
  --topic-card-raised: linear-gradient(180deg, rgba(25, 37, 50, 0.94), rgba(13, 21, 30, 0.96));
  --topic-chip: rgba(12, 19, 28, 0.82);
  background:
    radial-gradient(circle at 16% 14%, rgba(240, 138, 75, 0.18), transparent 26%),
    radial-gradient(circle at 82% 10%, rgba(74, 129, 219, 0.16), transparent 24%),
    linear-gradient(180deg, rgba(17, 28, 39, 0.96), rgba(8, 14, 22, 0.98));
  border: 1px solid rgba(153, 187, 227, 0.18);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.05),
    0 28px 90px rgba(2, 6, 12, 0.4);
}

.topic-sidebar::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.05), transparent 18%),
    linear-gradient(90deg, rgba(255, 255, 255, 0.03), transparent 28%);
  pointer-events: none;
}

.topic-sidebar::after {
  content: '';
  position: absolute;
  inset: 1rem auto 1rem 1rem;
  width: 1px;
  background: linear-gradient(180deg, rgba(240, 138, 75, 0.4), rgba(115, 150, 198, 0.08) 42%, rgba(115, 150, 198, 0));
  pointer-events: none;
}

.topic-sidebar__content {
  position: relative;
  z-index: 1;
  display: grid;
  min-height: 0;
  flex: 1;
  gap: 1.5rem;
  grid-template-rows: auto minmax(0, 1fr) auto;
}

.topic-sidebar__content > section:first-child {
  padding-left: 0.75rem;
}

.topic-sidebar__empty--soft {
  border-style: solid;
  background: rgba(10, 16, 23, 0.56);
}

.topic-sidebar__eyebrow {
  font-size: 0.72rem;
  letter-spacing: 0.24em;
  text-transform: uppercase;
  color: var(--topic-ink-soft);
}

.topic-sidebar__empty {
  position: relative;
  z-index: 1;
  border-radius: 1.6rem;
  border: 1px dashed rgba(153, 187, 227, 0.2);
  background: rgba(9, 15, 23, 0.5);
  padding: 1.2rem;
  color: var(--topic-ink-medium);
}

.topic-panel {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--topic-border);
  background: var(--topic-card);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.04),
    0 22px 60px rgba(2, 6, 12, 0.24);
  backdrop-filter: blur(16px);
}

.topic-panel::before {
  content: '';
  position: absolute;
  inset: 0 auto auto 0;
  width: 100%;
  height: 1px;
  background: linear-gradient(90deg, rgba(240, 138, 75, 0.44), rgba(120, 167, 230, 0.14), rgba(255, 255, 255, 0));
  pointer-events: none;
}

.topic-panel--featured {
  display: flex;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  background: var(--topic-card-raised);
}

.topic-sidebar__news-scroll {
  min-height: 0;
  flex: 1;
  padding-right: 0.25rem;
}

.topic-sidebar__news-scroll--bounded {
  overflow-y: auto;
  max-height: 75vh;
}

.topic-sidebar__news-scroll--bounded::-webkit-scrollbar {
  width: 0.45rem;
}

.topic-sidebar__news-scroll--bounded::-webkit-scrollbar-track {
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.05);
}

.topic-sidebar__news-scroll--bounded::-webkit-scrollbar-thumb {
  border-radius: 999px;
  background: rgba(240, 138, 75, 0.28);
}

.topic-pill {
  border-radius: 999px;
  background: linear-gradient(180deg, rgba(22, 29, 39, 0.88), rgba(11, 17, 24, 0.96));
  padding: 0.45rem 0.85rem;
  font-size: 0.78rem;
  font-weight: 600;
  color: rgba(248, 251, 255, 0.9);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.04);
}

.topic-pill--event {
  border: 1px solid rgba(245, 158, 11, 0.32);
}

.topic-pill--person {
  border: 1px solid rgba(16, 185, 129, 0.32);
}

.topic-pill--keyword {
  border: 1px solid rgba(99, 102, 241, 0.32);
}

.topic-related-card {
  position: relative;
  width: 100%;
  text-align: left;
  display: block;
  border-radius: 1.25rem;
  border: 1px solid var(--topic-border);
  background: linear-gradient(180deg, rgba(18, 27, 38, 0.96), rgba(10, 16, 24, 0.98));
  padding: 1rem 1rem 1.05rem;
  text-decoration: none;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.04),
    0 16px 40px rgba(3, 8, 14, 0.28);
  transition:
    transform 0.22s ease,
    border-color 0.22s ease,
    background 0.22s ease,
    box-shadow 0.22s ease;
}

.topic-related-card::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  border-radius: 999px;
  background: linear-gradient(180deg, rgba(240, 138, 75, 0.9), rgba(92, 143, 226, 0.52));
  opacity: 0.7;
}

.topic-related-card__meta,
.topic-related-card__context {
  font-size: 0.74rem;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--topic-ink-soft);
}

.topic-related-card__title {
  margin-top: 0.45rem;
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.45;
  color: var(--topic-ink-strong);
}

.topic-related-card__context {
  margin-top: 0.65rem;
}

.topic-related-card__note {
  margin-top: 0.55rem;
  font-size: 0.78rem;
  line-height: 1.5;
  color: rgba(255, 227, 203, 0.86);
}

.topic-related-card__note--soft {
  color: rgba(173, 193, 214, 0.72);
}

.topic-related-card {
  cursor: pointer;
}

.topic-related-card:hover,
.topic-related-card:focus-visible {
  transform: translateY(-2px);
  border-color: rgba(240, 138, 75, 0.36);
  background: linear-gradient(180deg, rgba(24, 35, 48, 0.98), rgba(12, 19, 28, 1));
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.05),
    0 24px 48px rgba(3, 8, 14, 0.36);
}

.topic-related-card:focus-visible {
  outline: 2px solid rgba(240, 138, 75, 0.45);
  outline-offset: 2px;
}

@media (max-width: 1279px) {
  .topic-sidebar__content {
    display: flex;
    flex-direction: column;
  }

  .topic-panel--featured,
  .topic-sidebar__news-scroll {
    min-height: auto;
    flex: none;
    overflow: visible;
  }
}

.topic-summary__count {
  border-radius: 999px;
  border: 1px solid rgba(240, 138, 75, 0.2);
  background: var(--topic-chip);
  padding: 0.32rem 0.7rem;
  font-size: 0.75rem;
  color: rgba(255, 228, 209, 0.88);
}

.keywords-hint {
  font-size: 0.72rem;
  color: rgba(255, 255, 255, 0.4);
  text-align: center;
  margin-top: 0.75rem;
}
</style>
