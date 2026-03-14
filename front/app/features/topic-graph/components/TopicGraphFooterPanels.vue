<script setup lang="ts">
import { computed } from 'vue'
import type { TopicGraphDetailPayload } from '~/api/topicGraph'

interface Props {
  detail: TopicGraphDetailPayload | null
}

const props = defineProps<Props>()

const historyPeak = computed(() => Math.max(...(props.detail?.history.map(item => item.count) || [0]), 1))

function formatAnchorDate(anchorDate: string) {
  const [year, month, day] = anchorDate.split('-')
  if (year && month && day) {
    return `${year}.${month}.${day}`
  }

  return anchorDate
}

const historyPoints = computed(() => {
  const points = props.detail?.history ?? []

  return [...points]
    .sort((left, right) => new Date(left.anchor_date).getTime() - new Date(right.anchor_date).getTime())
    .map((point, index, collection) => {
      const intensity = Math.max(0.18, point.count / historyPeak.value)

      return {
        ...point,
        step: `${index + 1}`.padStart(2, '0'),
        isLatest: index === collection.length - 1,
        style: {
          '--history-energy': intensity.toFixed(3),
          '--history-scale': (0.92 + intensity * 0.48).toFixed(3),
        },
      }
    })
})
</script>

<template>
  <section
    class="topic-footer-grid grid gap-4 xl:grid-cols-[1.1fr_0.9fr_0.9fr]"
    data-testid="topic-graph-footer"
  >
    <article
      class="topic-footer-card rounded-[28px] p-4 md:p-5"
      data-testid="topic-graph-history-region"
    >
      <div class="flex items-start justify-between gap-3">
        <div>
          <p class="topic-footer-card__eyebrow">历史温度</p>
          <p class="topic-footer-card__lede">沿着时间窗读它的抬升、停顿和延续。</p>
        </div>
        <span v-if="historyPoints.length" class="topic-footer-card__count">{{ historyPoints.length }} 段</span>
      </div>

      <div v-if="historyPoints.length" class="topic-history mt-5">
        <div
          v-for="point in historyPoints"
          :key="point.anchor_date"
          class="topic-history__item"
          :class="{ 'topic-history__item--latest': point.isLatest }"
        >
          <div class="topic-history__spine">
            <span class="topic-history__step">{{ point.step }}</span>
            <span class="topic-history__dot" :style="point.style" />
          </div>

          <div class="topic-history__card" :style="point.style">
            <div class="topic-history__row">
              <div>
                <p class="topic-history__date">{{ formatAnchorDate(point.anchor_date) }}</p>
                <p class="topic-history__label">{{ point.label }}</p>
              </div>

              <div class="topic-history__volume">
                <span class="topic-history__value">{{ point.count }}</span>
                <span class="topic-history__unit">次提及</span>
              </div>
            </div>

            <p class="topic-history__caption">
              {{ point.isLatest ? '最近一段仍在延续这个话题。' : '这段时间窗留下了一次可追溯的热度沉积。' }}
            </p>
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

.topic-footer-card__lede {
  margin-top: 0.75rem;
  max-width: 22rem;
  color: rgba(220, 230, 239, 0.78);
  font-size: 0.95rem;
  line-height: 1.6;
}

.topic-footer-card__count {
  display: inline-flex;
  align-items: center;
  min-height: 2.15rem;
  border-radius: 999px;
  border: 1px solid rgba(240, 138, 75, 0.18);
  background: rgba(255, 255, 255, 0.04);
  padding: 0 0.85rem;
  font-size: 0.74rem;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: rgba(255, 228, 209, 0.82);
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
  transition:
    border-color 0.22s ease,
    background 0.22s ease,
    transform 0.22s ease;
}

.topic-footer-link:hover,
.topic-footer-link:focus-visible {
  transform: translateY(-1px);
  border-color: rgba(240, 138, 75, 0.28);
  background: rgba(255, 255, 255, 0.08);
}

.topic-history {
  position: relative;
  display: grid;
  gap: 1rem;
}

.topic-history::before {
  content: '';
  position: absolute;
  left: 1.75rem;
  top: 0.45rem;
  bottom: 0.45rem;
  width: 1px;
  background: linear-gradient(180deg, rgba(240, 138, 75, 0.46), rgba(111, 151, 218, 0.12) 46%, rgba(111, 151, 218, 0));
}

.topic-history__item {
  position: relative;
  display: grid;
  grid-template-columns: 3.5rem minmax(0, 1fr);
  gap: 1rem;
  align-items: start;
}

.topic-history__spine {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.7rem;
}

.topic-history__step {
  font-size: 0.68rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.44);
}

.topic-history__dot {
  display: block;
  width: 0.82rem;
  height: 0.82rem;
  border-radius: 999px;
  border: 1px solid rgba(255, 255, 255, 0.5);
  background: linear-gradient(180deg, rgba(240, 138, 75, 0.98), rgba(83, 142, 228, 0.9));
  box-shadow:
    0 0 0 0.35rem rgba(240, 138, 75, 0.08),
    0 0 18px rgba(84, 143, 228, 0.22);
  transform: scale(var(--history-scale));
}

.topic-history__card {
  position: relative;
  overflow: hidden;
  border-radius: 1.45rem;
  border: 1px solid rgba(133, 165, 211, 0.16);
  background: linear-gradient(180deg, rgba(18, 28, 39, 0.94), rgba(10, 16, 24, 0.98));
  padding: 1rem 1rem 1.05rem;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.04),
    0 20px 44px rgba(2, 6, 12, 0.22);
}

.topic-history__card::before {
  content: '';
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 14% 24%, rgba(240, 138, 75, 0.22), transparent 26%),
    linear-gradient(135deg, rgba(240, 138, 75, 0.12), rgba(63, 124, 255, 0.1) 62%, transparent 100%);
  opacity: calc(0.32 + var(--history-energy) * 0.45);
  pointer-events: none;
}

.topic-history__row,
.topic-history__caption {
  position: relative;
  z-index: 1;
}

.topic-history__row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}

.topic-history__date {
  font-size: 0.72rem;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(178, 196, 216, 0.7);
}

.topic-history__label {
  margin-top: 0.45rem;
  color: rgba(248, 251, 255, 0.94);
  font-size: 1rem;
  font-weight: 600;
  line-height: 1.45;
}

.topic-history__volume {
  display: grid;
  justify-items: end;
  gap: 0.1rem;
  flex-shrink: 0;
}

.topic-history__value {
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 1.8rem;
  line-height: 1;
  color: rgba(255, 240, 229, 0.96);
}

.topic-history__unit {
  font-size: 0.72rem;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: rgba(178, 196, 216, 0.68);
}

.topic-history__caption {
  margin-top: 0.8rem;
  color: rgba(214, 225, 236, 0.72);
  font-size: 0.88rem;
  line-height: 1.55;
}

.topic-history__item--latest .topic-history__card {
  border-color: rgba(240, 138, 75, 0.22);
}

.topic-history__item--latest .topic-history__step {
  color: rgba(255, 228, 209, 0.72);
}

@media (max-width: 767px) {
  .topic-history__item {
    grid-template-columns: 2.8rem minmax(0, 1fr);
    gap: 0.85rem;
  }

  .topic-history::before {
    left: 1.4rem;
  }

  .topic-history__row {
    flex-direction: column;
  }

  .topic-history__volume {
    justify-items: start;
  }
}
</style>
