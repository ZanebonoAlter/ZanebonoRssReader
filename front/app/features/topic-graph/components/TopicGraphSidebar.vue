<script setup lang="ts">
import { computed } from 'vue'
import type { TopicGraphDetailPayload } from '~/api/topicGraph'

interface Props {
  detail: TopicGraphDetailPayload | null
  loading?: boolean
  error?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  error: null,
})

const emit = defineEmits<{
  openArticle: [articleId: number]
}>()

const featuredArticles = computed(() => {
  if (!props.detail) return []

  const items = props.detail.summaries.flatMap(summary =>
    summary.articles.map(article => ({
      ...article,
      feedName: summary.feed_name,
      categoryName: summary.category_name,
      summaryTitle: summary.title,
    })),
  )

  return Array.from(new Map(items.map(item => [item.link, item])).values())
})

const shouldScrollFeaturedArticles = computed(() => featuredArticles.value.length > 8)

</script>

<template>
  <aside class="topic-sidebar rounded-[34px] px-5 py-5 md:px-6 md:py-6">
    <div v-if="props.loading" class="topic-sidebar__empty">正在展开话题脉络...</div>
    <div v-else-if="props.error" class="topic-sidebar__empty">{{ props.error }}</div>
    <div v-else-if="!props.detail" class="topic-sidebar__empty">点一个节点，右侧就会展开这类题材的近期总结、历史轨迹和外部入口。</div>
    <div v-else class="topic-sidebar__content">
      <section class="space-y-3">
        <p class="topic-sidebar__eyebrow">当前焦点</p>
        <div class="flex flex-wrap items-center gap-3">
          <h2 class="font-serif text-3xl text-[var(--topic-ink-strong)]">{{ props.detail.topic.label }}</h2>
          <span class="topic-pill">{{ props.detail.topic.kind }}</span>
        </div>
      </section>

      <section class="topic-panel topic-panel--featured rounded-[28px] p-4 md:p-5">
        <div class="flex items-center justify-between gap-3">
          <p class="topic-sidebar__eyebrow">相关新闻</p>
          <span class="topic-summary__count">{{ featuredArticles.length }} 条</span>
        </div>
        <div
          v-if="featuredArticles.length"
          class="topic-sidebar__news-scroll mt-4"
          :class="{ 'topic-sidebar__news-scroll--bounded': shouldScrollFeaturedArticles }"
        >
          <div class="grid gap-3">
            <button
              v-for="article in featuredArticles"
              :key="article.link"
              class="topic-related-card"
              type="button"
              @click="emit('openArticle', article.id)"
            >
              <p class="topic-related-card__meta">{{ article.feedName }} · {{ article.categoryName }}</p>
              <h3 class="topic-related-card__title">{{ article.title }}</h3>
              <p class="topic-related-card__context">来自：{{ article.summaryTitle }}</p>
            </button>
          </div>
        </div>
        <div v-else class="topic-sidebar__empty topic-sidebar__empty--soft">这一话题当前还没有挂上文章链接。</div>
      </section>

      <section class="topic-panel rounded-[26px] p-4">
        <p class="topic-sidebar__eyebrow">相关主题</p>
        <div class="mt-4 flex flex-wrap gap-2">
          <span v-for="item in props.detail.related_topics" :key="item.slug" class="topic-pill">{{ item.label }}</span>
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
  --topic-ink-strong: #14212c;
  --topic-ink-medium: #455563;
  --topic-ink-soft: #778792;
  background: linear-gradient(180deg, rgba(247, 243, 235, 0.96), rgba(239, 235, 226, 0.98));
  border: 1px solid rgba(133, 105, 78, 0.16);
  box-shadow: 0 22px 68px rgba(23, 30, 36, 0.12);
}

.topic-sidebar__content {
  display: grid;
  min-height: 0;
  flex: 1;
  gap: 1.5rem;
  grid-template-rows: auto minmax(0, 1fr) auto;
}

.topic-sidebar__empty--soft {
  border-style: solid;
  background: rgba(20, 33, 44, 0.04);
}

.topic-sidebar__eyebrow {
  font-size: 0.72rem;
  letter-spacing: 0.24em;
  text-transform: uppercase;
  color: var(--topic-ink-soft);
}

.topic-sidebar__empty {
  border-radius: 1.6rem;
  border: 1px dashed rgba(100, 109, 117, 0.32);
  padding: 1.2rem;
  color: var(--topic-ink-medium);
}

.topic-panel {
  background: rgba(255, 255, 255, 0.64);
  border: 1px solid rgba(133, 105, 78, 0.1);
}

.topic-panel--featured {
  display: flex;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  background: linear-gradient(180deg, rgba(255,255,255,0.82), rgba(247, 240, 230, 0.92));
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

.topic-pill {
  border-radius: 999px;
  background: rgba(240, 138, 75, 0.14);
  padding: 0.45rem 0.85rem;
  font-size: 0.78rem;
  font-weight: 600;
  color: #7a4724;
}

.topic-related-card {
  width: 100%;
  text-align: left;
  display: block;
  border-radius: 1.25rem;
  border: 1px solid rgba(20, 33, 44, 0.08);
  background: rgba(255, 255, 255, 0.88);
  padding: 1rem 1rem 1.05rem;
  text-decoration: none;
  box-shadow: 0 14px 32px rgba(20, 33, 44, 0.06);
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

.topic-related-card {
  cursor: pointer;
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
  background: rgba(20, 33, 44, 0.08);
  padding: 0.32rem 0.7rem;
  font-size: 0.75rem;
  color: var(--topic-ink-medium);
}
</style>
