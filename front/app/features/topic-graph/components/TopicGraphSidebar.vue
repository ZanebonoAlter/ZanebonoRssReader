<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
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

  return Array.from(new Map(items.map(item => [item.link, item])).values()).slice(0, 8)
})

function renderMarkdown(content?: string | null) {
  if (!content) return ''
  return marked.parse(content) as string
}

import '~/components/article/ArticleContent.css'
</script>

<template>
  <aside class="topic-sidebar rounded-[34px] px-5 py-5 md:px-6 md:py-6">
    <div v-if="props.loading" class="topic-sidebar__empty">正在展开话题脉络...</div>
    <div v-else-if="props.error" class="topic-sidebar__empty">{{ props.error }}</div>
    <div v-else-if="!props.detail" class="topic-sidebar__empty">点一个节点，右侧就会展开这类题材的近期总结、历史轨迹和外部入口。</div>
    <div v-else class="space-y-6">
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
        <div v-if="featuredArticles.length" class="mt-4 grid gap-3">
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
        <div v-else class="topic-sidebar__empty topic-sidebar__empty--soft">这一话题当前还没有挂上文章链接。</div>
      </section>

      <section class="topic-panel rounded-[26px] p-4">
        <p class="topic-sidebar__eyebrow">相关主题</p>
        <div class="mt-4 flex flex-wrap gap-2">
          <span v-for="item in props.detail.related_topics" :key="item.slug" class="topic-pill">{{ item.label }}</span>
        </div>
      </section>

      <section class="space-y-3">
        <p class="topic-sidebar__eyebrow">摘要纵览</p>
        <article v-for="summary in props.detail.summaries" :key="summary.id" class="topic-summary rounded-[26px] p-4">
          <div class="flex items-center justify-between gap-3">
            <div>
              <h3 class="text-base font-semibold text-[var(--topic-ink-strong)]">{{ summary.title }}</h3>
              <p class="mt-1 text-xs uppercase tracking-[0.18em] text-[var(--topic-ink-soft)]">{{ summary.feed_name }} · {{ summary.category_name }}</p>
            </div>
            <span class="topic-summary__count">{{ summary.article_count }} 篇</span>
          </div>
          <div class="topic-summary__markdown markdown-body markdown-summary mt-4" v-html="renderMarkdown(summary.summary)" />
          <div class="mt-3 flex flex-wrap gap-2">
            <span v-for="topic in summary.topics" :key="topic.slug" class="topic-pill topic-pill--muted">{{ topic.label }}</span>
          </div>
          <div v-if="summary.articles?.length" class="mt-4 grid gap-2">
            <button
              v-for="article in summary.articles"
              :key="article.id"
              class="topic-article-link"
              type="button"
              @click="emit('openArticle', article.id)"
            >
              {{ article.title }}
            </button>
          </div>
        </article>
      </section>
    </div>
  </aside>
</template>

<style scoped>
.topic-sidebar {
  --topic-ink-strong: #14212c;
  --topic-ink-medium: #455563;
  --topic-ink-soft: #778792;
  background: linear-gradient(180deg, rgba(247, 243, 235, 0.96), rgba(239, 235, 226, 0.98));
  border: 1px solid rgba(133, 105, 78, 0.16);
  box-shadow: 0 22px 68px rgba(23, 30, 36, 0.12);
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

.topic-panel,
.topic-summary {
  background: rgba(255, 255, 255, 0.64);
  border: 1px solid rgba(133, 105, 78, 0.1);
}

.topic-panel--featured {
  background: linear-gradient(180deg, rgba(255,255,255,0.82), rgba(247, 240, 230, 0.92));
}

.topic-pill {
  border-radius: 999px;
  background: rgba(240, 138, 75, 0.14);
  padding: 0.45rem 0.85rem;
  font-size: 0.78rem;
  font-weight: 600;
  color: #7a4724;
}

.topic-pill--muted {
  background: rgba(20, 33, 44, 0.08);
  color: var(--topic-ink-medium);
}

.topic-article-link {
  display: block;
  border-radius: 1rem;
  background: rgba(20, 33, 44, 0.05);
  padding: 0.7rem 0.9rem;
  color: var(--topic-ink-strong);
  font-size: 0.86rem;
  text-decoration: none;
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

.topic-summary__markdown {
  padding: 0;
  color: var(--topic-ink-medium);
  font-size: 0.98rem;
  line-height: 1.85;
}

.topic-related-card,
.topic-article-link {
  cursor: pointer;
}

:deep(.topic-summary__markdown h1),
:deep(.topic-summary__markdown h2),
:deep(.topic-summary__markdown h3) {
  font-size: 1.06rem;
}

.topic-summary__count {
  border-radius: 999px;
  background: rgba(20, 33, 44, 0.08);
  padding: 0.32rem 0.7rem;
  font-size: 0.75rem;
  color: var(--topic-ink-medium);
}
</style>
