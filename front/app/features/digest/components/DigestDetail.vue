<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { onBeforeUnmount, ref, watch } from 'vue'
import { marked } from 'marked'
import ArticleTagList from '../../articles/components/ArticleTagList.vue'
import ArticleContentView from '~/features/articles/components/ArticleContentView.vue'
import { useArticlesApi } from '~/api/articles'
import type { DigestPreviewSummary } from '~/api/digest'
import { normalizeArticle } from '../../articles/utils/normalizeArticle'
import { useApiStore } from '~/stores/api'
import type { Article } from '~/types'

const props = withDefaults(defineProps<{
  summary: DigestPreviewSummary | null
  activeTypeLabel?: string
  running?: boolean
}>(), {
  activeTypeLabel: '日报',
  running: false,
})

const emit = defineEmits<{
  run: []
  'open-settings': []
}>()

const articlesApi = useArticlesApi()
const apiStore = useApiStore()
const relatedArticles = ref<Article[]>([])
const loadingArticles = ref(false)
const selectedArticle = ref<Article | null>(null)

const renderedSummary = computed(() => {
  if (!props.summary?.summary_text) return ''
  return String(marked.parse(props.summary.summary_text))
})

async function loadRelatedArticles(summary: DigestPreviewSummary | null) {
  if (!summary?.article_ids?.length) {
    relatedArticles.value = []
    selectedArticle.value = null
    return
  }

  loadingArticles.value = true
  try {
    const responses = await Promise.all(summary.article_ids.map(id => articlesApi.getArticle(id)))
    relatedArticles.value = responses
      .filter(response => response.success && response.data)
      .map(response => normalizeArticle(response.data))
    selectedArticle.value = null
  } catch (error) {
    console.error('Failed to load related articles:', error)
    relatedArticles.value = []
    selectedArticle.value = null
  } finally {
    loadingArticles.value = false
  }
}

function openArticle(article: Article) {
  selectedArticle.value = article
}

function closeArticleModal() {
  selectedArticle.value = null
}

async function handleFavorite(articleId: string) {
  const response = await apiStore.toggleFavorite(articleId)
  if (!response.success) return

  const target = relatedArticles.value.find(article => article.id === articleId)
  if (target) {
    target.favorite = !target.favorite
  }

  if (selectedArticle.value?.id === articleId) {
    selectedArticle.value = {
      ...selectedArticle.value,
      favorite: !selectedArticle.value.favorite,
    }
  }
}

function handleNavigate(article: Article) {
  selectedArticle.value = article
}

function normalizeArticleLink(raw: string) {
  try {
    const url = new URL(raw)
    url.hash = ''
    return url.toString().replace(/\/$/, '')
  } catch {
    return raw.replace(/#.*$/, '').replace(/\/$/, '')
  }
}

function findArticleByLink(rawHref: string) {
  const normalizedHref = normalizeArticleLink(rawHref)
  const match = [...relatedArticles.value, ...apiStore.articles].find((article) => {
    if (!article.link) return false
    return normalizeArticleLink(article.link) === normalizedHref
  })
  return match ?? null
}

function handleArticleLinkClick(event: MouseEvent) {
  const target = event.target as HTMLElement | null
  const anchor = target?.closest('a') as HTMLAnchorElement | null
  if (!anchor) return

  const href = anchor.getAttribute('href')?.trim()
  if (!href || href.startsWith('#') || href.startsWith('javascript:') || href.startsWith('mailto:') || href.startsWith('tel:')) {
    return
  }

  const matchedArticle = findArticleByLink(anchor.href || href)
  if (!matchedArticle) {
    return
  }

  event.preventDefault()
  selectedArticle.value = matchedArticle
}

watch(() => props.summary, (summary) => {
  void loadRelatedArticles(summary)
}, { immediate: true })

watch(selectedArticle, (article) => {
  if (!import.meta.client) return
  document.body.style.overflow = article ? 'hidden' : ''
}, { immediate: true })

onBeforeUnmount(() => {
  if (!import.meta.client) return
  document.body.style.overflow = ''
})
</script>

<template>
  <section class="digest-detail-shell min-h-0 h-full overflow-hidden rounded-[36px] border border-[var(--color-border-medium)] bg-[rgba(255,255,255,0.82)] shadow-[0_24px_60px_rgba(18,24,30,0.08)] xl:min-h-[760px]">
    <div v-if="summary" class="flex h-full min-h-0 flex-col">
      <header class="shrink-0 border-b border-[var(--color-border-subtle)] px-6 py-3.5 md:px-7">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="min-w-0 flex-1 xl:pr-4">
            <p class="text-xs uppercase tracking-[0.32em] text-ink-light">{{ activeTypeLabel }} Summary</p>
            
            <h2 class="mt-2.5 max-w-none text-[1.75rem] font-black leading-tight text-ink-dark md:text-[2rem]">{{ summary.feed_name }}</h2>
          </div>
        </div>
      </header>

      <div class="digest-detail-main grid flex-1 min-h-0 gap-0 xl:grid-cols-[minmax(0,1fr)_380px] 2xl:grid-cols-[minmax(0,1fr)_440px]">
        <div class="digest-detail-scroll digest-detail-scroll--summary min-h-0 border-b border-[var(--color-border-subtle)] px-6 py-6 md:px-7 xl:overflow-y-auto xl:border-b-0 xl:border-r">
          <div class="digest-summary-surface" @click.capture="handleArticleLinkClick">
            <div class="digest-summary-content max-w-none" v-html="renderedSummary" />
          </div>
        </div>

        <aside class="digest-detail-scroll digest-detail-scroll--articles min-h-0 bg-[rgba(247,241,230,0.72)] px-5 py-6 md:px-6 xl:overflow-y-auto">
          <div class="space-y-4">
            <div>
              <p class="text-xs uppercase tracking-[0.28em] text-ink-light">关联文章</p>
              <h3 class="mt-2 text-xl font-black text-ink-dark">点开后弹窗读</h3>
              <p class="mt-2 text-sm leading-7 text-ink-medium">总结不动，文章盖上来。少来回切。</p>
            </div>

            <div v-if="summary.aggregated_tags?.length" class="rounded-[20px] border border-[var(--color-border-subtle)] bg-white/70 px-4 py-4">
              <p class="text-[11px] uppercase tracking-[0.24em] text-ink-light">索引标签</p>
              <p class="mt-2 text-sm leading-6 text-ink-medium">这组标签来自当前总结覆盖文章的聚合索引。</p>
              <ArticleTagList class="mt-3" :tags="summary.aggregated_tags" compact />
            </div>

            <div v-if="loadingArticles" class="flex items-center gap-2 rounded-[20px] border border-[var(--color-border-subtle)] bg-white/70 px-4 py-4 text-sm text-ink-medium">
              <Icon icon="mdi:loading" width="18" class="animate-spin" />
              正在拉文章...
            </div>

            <div v-else-if="relatedArticles.length" class="space-y-3">
              <button v-for="article in relatedArticles" :key="article.id" class="digest-article-card" type="button" @click="openArticle(article)">
                <div class="space-y-2 text-left">
                  <p class="line-clamp-2 text-sm font-semibold leading-6 text-ink-dark">{{ article.title }}</p>
                  <p class="text-xs text-ink-medium">{{ article.pubDate || '没有时间' }}</p>
                  <ArticleTagList v-if="article.tags?.length" :tags="article.tags" compact :max-visible="4" :show-article-count="false" />
                </div>
                <div class="mt-3 flex items-center justify-between text-xs text-ink-medium">
                  <span>{{ article.favorite ? '已收藏' : '未收藏' }}</span>
                  <span>弹窗阅读</span>
                </div>
              </button>
            </div>

            <div v-else class="rounded-[20px] border border-[var(--color-border-subtle)] bg-white/70 px-4 py-5 text-sm leading-7 text-ink-medium">
              这条总结没挂文章。
            </div>
          </div>
        </aside>
      </div>
    </div>

    <div v-else class="flex min-h-[760px] items-center justify-center p-6 md:p-10">
      <div class="max-w-xl text-center">
        <p class="text-xs uppercase tracking-[0.36em] text-ink-light">Empty Digest</p>
        <h2 class="mt-4 text-4xl font-black leading-none text-ink-dark md:text-5xl">先选一条总结</h2>
        <p class="mx-auto mt-5 max-w-[24rem] text-sm leading-7 text-ink-medium md:text-base">
          {{ running ? '正在生成。再等一会。' : '当前没有可读的 AI 总结。你可以先执行，或者去检查设置。' }}
        </p>

        <div class="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
          <button class="btn-primary min-h-11 px-5" type="button" :disabled="running" @click="emit('run')">
            {{ running ? '执行中...' : `生成${activeTypeLabel}` }}
          </button>
          <button class="btn-secondary min-h-11 px-5" type="button" @click="emit('open-settings')">
            打开设置
          </button>
        </div>
      </div>
    </div>
  </section>

  <Teleport to="body">
    <div v-if="selectedArticle" class="digest-article-modal" @click.self="closeArticleModal">
      <div class="digest-article-modal__panel">
        <header class="digest-article-modal__header">
          <div class="flex min-w-0 items-center gap-3">
            <div v-if="summary" class="digest-feed-badge max-w-full">
              <Icon :icon="summary.feed_icon || 'mdi:rss'" width="16" :style="{ color: summary.feed_color || '#3b6b87' }" />
              <span class="truncate" :style="{ color: summary.feed_color || '#3b6b87' }">{{ summary.feed_name }}</span>
            </div>
            <p class="truncate text-sm text-ink-medium">文章弹窗里保留收藏、抓取、总结这些动作。</p>
          </div>

          <button class="btn-ghost min-h-11 min-w-11 px-0" type="button" aria-label="关闭文章弹窗" @click="closeArticleModal">
            <Icon icon="mdi:close" width="18" />
          </button>
        </header>

        <div class="digest-article-modal__body" @click.capture="handleArticleLinkClick">
          <ArticleContentView
            :article="selectedArticle"
            :articles="relatedArticles"
            :highlighted-tag-slugs="summary?.aggregated_tags?.map(tag => tag.slug) || []"
            @favorite="handleFavorite"
            @navigate="handleNavigate"
          />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.digest-feed-badge,
.digest-meta-chip,
.digest-fact-chip,
.digest-article-card {
  border: 1px solid var(--color-border-subtle);
}

.digest-feed-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.45rem;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.72);
  padding: 0.45rem 0.85rem;
  font-size: 0.85rem;
  font-weight: 700;
}

.digest-meta-chip {
  display: inline-flex;
  align-items: center;
  min-height: 2rem;
  border-radius: 999px;
  background: rgba(193, 47, 47, 0.08);
  padding: 0 0.85rem;
  color: var(--color-print-red-700);
  font-size: 0.8rem;
  font-weight: 700;
}

.digest-fact-chip {
  min-width: 108px;
  border-radius: 20px;
  background: rgba(255,255,255,0.72);
  padding: 0.7rem 0.9rem;
}

.digest-summary-surface {
  max-width: 88ch;
}

@media (min-width: 1800px) {
  .digest-summary-surface {
    max-width: 96ch;
  }
}

.digest-summary-content {
  color: var(--color-ink-dark);
  line-height: 1.9;
}

.digest-summary-content :deep(h1),
.digest-summary-content :deep(h2),
.digest-summary-content :deep(h3),
.digest-summary-content :deep(h4),
.digest-summary-content :deep(h5),
.digest-summary-content :deep(h6) {
  margin-top: 1.9rem;
  margin-bottom: 0.85rem;
  color: var(--color-ink-dark);
  font-weight: 800;
  letter-spacing: -0.02em;
}

.digest-summary-content :deep(h1) {
  margin-top: 0;
  font-size: 1.9rem;
}

.digest-summary-content :deep(h2) {
  border-top: 1px solid var(--color-border-subtle);
  padding-top: 1.4rem;
  font-size: 1.35rem;
}

.digest-summary-content :deep(p),
.digest-summary-content :deep(li) {
  color: var(--color-ink-medium);
  font-size: 1rem;
}

.digest-summary-content :deep(ul),
.digest-summary-content :deep(ol) {
  margin-bottom: 1.2rem;
  padding-left: 1.4rem;
}

.digest-summary-content :deep(blockquote) {
  margin: 0 0 1.5rem 0;
  border-left: 3px solid var(--color-print-red-500);
  border-radius: 0 16px 16px 0;
  background: rgba(193, 47, 47, 0.05);
  padding: 1rem 1.1rem;
}

.digest-summary-content :deep(code) {
  border-radius: 6px;
  background: rgba(45, 86, 112, 0.08);
  padding: 0.15em 0.4em;
  color: var(--color-ink-700);
}

.digest-summary-content :deep(pre) {
  margin-bottom: 1.5rem;
  overflow-x: auto;
  border-radius: 16px;
  border: 1px solid var(--color-border-subtle);
  background: rgba(247, 241, 230, 0.95);
  padding: 1rem 1.1rem;
}

.digest-summary-content :deep(pre code) {
  background: transparent;
  padding: 0;
}

.digest-summary-content :deep(a) {
  color: var(--color-ink-700);
  border-bottom: 1px solid rgba(45, 86, 112, 0.24);
  text-decoration: none;
}

.digest-article-card {
  width: 100%;
  border-radius: 22px;
  background: rgba(255,255,255,0.8);
  padding: 1rem;
  transition: border-color 180ms ease, background 180ms ease, transform 180ms ease;
}

.digest-topic-tag {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border: 1px solid;
  border-radius: 999px;
  padding: 0.35rem 0.75rem;
  font-size: 0.85rem;
  font-weight: 700;
  cursor: default;
  transition: transform 120ms ease, box-shadow 120ms ease;
}

.digest-topic-tag:hover {
  transform: translateY(-1px);
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.08);
}

.digest-article-card:hover {
  transform: translateY(-1px);
  border-color: rgba(45, 86, 112, 0.28);
  background: rgba(45, 86, 112, 0.06);
}

.digest-article-modal {
  position: fixed;
  inset: 0;
  z-index: 70;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
  overflow: hidden;
  overscroll-behavior: contain;
  background: rgba(16, 20, 25, 0.42);
  backdrop-filter: blur(8px);
}

.digest-article-modal__panel {
  display: flex;
  min-height: 0;
  height: calc(100vh - 2rem);
  max-height: calc(100vh - 2rem);
  width: min(1480px, 100%);
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--color-border-medium);
  border-radius: 32px;
  background: rgba(250, 247, 242, 0.98);
  box-shadow: 0 30px 100px rgba(18, 24, 30, 0.24);
}

.digest-article-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 1rem 1rem 1rem 1.25rem;
  border-bottom: 1px solid var(--color-border-subtle);
  background: linear-gradient(145deg, rgba(255, 255, 255, 0.74), rgba(247, 241, 230, 0.94));
}

.digest-article-modal__body {
  display: flex;
  min-height: 0;
  flex: 1;
  overflow: auto;
  overscroll-behavior: contain;
  background: white;
}

.digest-article-modal__body :deep(.article-content) {
  display: flex;
  min-height: 0;
  height: 100%;
  flex: 1;
}

.digest-article-modal__body :deep(.preview-mode),
.digest-article-modal__body :deep(.iframe-mode) {
  min-height: 0;
  height: 100%;
  overscroll-behavior: contain;
}

@media (prefers-reduced-motion: reduce) {
  .digest-article-card {
    transition: none !important;
  }
}
</style>

