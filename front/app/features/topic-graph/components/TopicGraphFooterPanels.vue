<script setup lang="ts">
import { computed } from 'vue'
import type { TopicGraphDetailPayload } from '~/api/topicGraph'

interface Props {
  detail: TopicGraphDetailPayload | null
}

const props = defineProps<Props>()

const historyPeak = computed(() => Math.max(...(props.detail?.history.map(item => item.count) || [0]), 1))

function barWidth(count: number) {
  return `${Math.max(10, (count / historyPeak.value) * 100)}%`
}
</script>

<template>
  <section class="topic-footer-grid grid gap-4 xl:grid-cols-[1.1fr_0.9fr_0.9fr]">
    <article class="topic-footer-card rounded-[28px] p-4 md:p-5">
      <p class="topic-footer-card__eyebrow">历史温度</p>
      <div v-if="detail?.history?.length" class="mt-4 space-y-3">
        <div v-for="point in detail.history" :key="point.anchor_date" class="space-y-1">
          <div class="flex items-center justify-between text-xs text-white/62">
            <span>{{ point.label }}</span>
            <span>{{ point.count }}</span>
          </div>
          <div class="topic-history__rail">
            <div class="topic-history__bar" :style="{ width: barWidth(point.count) }" />
          </div>
        </div>
      </div>
      <div v-else class="topic-footer-card__empty">选中一个主题后，这里会显示最近几个时间窗里的热度变化。</div>
    </article>

    <article class="topic-footer-card rounded-[28px] p-4 md:p-5">
      <p class="topic-footer-card__eyebrow">站内动作</p>
      <div v-if="detail?.app_links" class="mt-4 grid gap-2">
        <NuxtLink
          v-for="(href, label) in detail.app_links"
          :key="label"
          class="topic-footer-link"
          :to="href"
        >
          {{ label === 'digest_view' ? '打开 Digest 视图' : '回到图谱主页' }}
        </NuxtLink>
      </div>
      <div v-else class="topic-footer-card__empty">这里会放和当前主题有关的站内跳转。</div>
    </article>

    <article class="topic-footer-card rounded-[28px] p-4 md:p-5">
      <p class="topic-footer-card__eyebrow">外部入口</p>
      <div v-if="detail?.search_links" class="mt-4 grid gap-2">
        <a v-for="(href, label) in detail.search_links" :key="label" class="topic-footer-link" :href="href" target="_blank" rel="noreferrer">
          {{ label === 'youtube_live' ? 'YouTube Live' : 'YouTube Videos' }}
        </a>
      </div>
      <div v-else class="topic-footer-card__empty">这里会放当前主题的外部搜索入口。</div>
    </article>
  </section>
</template>

<style scoped>
.topic-footer-card {
  border: 1px solid rgba(255, 255, 255, 0.08);
  background: rgba(11, 18, 24, 0.72);
  box-shadow: 0 24px 80px rgba(6, 10, 16, 0.18);
  backdrop-filter: blur(12px);
}

.topic-footer-card__eyebrow {
  font-size: 0.72rem;
  letter-spacing: 0.24em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.5);
}

.topic-footer-card__empty {
  margin-top: 1rem;
  border-radius: 1rem;
  border: 1px dashed rgba(255, 255, 255, 0.14);
  padding: 0.9rem 1rem;
  color: rgba(255, 255, 255, 0.6);
  font-size: 0.92rem;
}

.topic-footer-link {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 2.85rem;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.04);
  color: white;
  text-decoration: none;
}

.topic-history__rail {
  height: 0.5rem;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
}

.topic-history__bar {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, #f08a4b, #3f7cff);
}
</style>
