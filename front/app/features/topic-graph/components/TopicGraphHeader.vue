<script setup lang="ts">
import type { TopicGraphType } from '~/api/topicGraph'

interface Props {
  selectedType: TopicGraphType
  selectedDate: string
  loading?: boolean
  heroLabel: string
  heroSubline: string
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
})

const emit = defineEmits<{
  'update:type': [value: TopicGraphType]
  'update:date': [value: string]
  refresh: []
}>()

const typeOptions: TopicGraphType[] = ['daily', 'weekly']

function updateDate(event: Event) {
  emit('update:date', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <header class="topic-hero rounded-[34px] px-5 py-5 md:px-7 md:py-6">
    <div class="flex flex-col gap-6 lg:flex-row lg:items-end lg:justify-between">
      <div class="max-w-3xl space-y-3">
        <p class="text-xs uppercase tracking-[0.34em] text-white/70">Topic Field</p>
        <div class="space-y-2">
          <h1 class="font-serif text-3xl text-white md:text-[3.4rem]">{{ heroLabel }}</h1>
          <p class="max-w-2xl text-sm leading-6 text-white/72 md:text-base">{{ heroSubline }}</p>
        </div>
      </div>

      <div class="topic-toolbar rounded-[28px] p-4 md:p-5">
        <div class="topic-toolbar__switcher" role="tablist" aria-label="主题图谱窗口切换">
          <button
            v-for="type in typeOptions"
            :key="type"
            type="button"
            class="topic-toolbar__switch"
            :class="{ 'topic-toolbar__switch--active': props.selectedType === type }"
            @click="emit('update:type', type)"
          >
            {{ type === 'daily' ? '日报图谱' : '周报图谱' }}
          </button>
        </div>

        <label class="topic-toolbar__date">
          <span class="topic-toolbar__eyebrow">时间锚点</span>
          <input class="topic-toolbar__input" :value="props.selectedDate" type="date" @input="updateDate">
        </label>

        <button class="topic-toolbar__button" type="button" :disabled="props.loading" @click="emit('refresh')">
          {{ props.loading ? '图谱载入中...' : '刷新图谱' }}
        </button>
      </div>
    </div>
  </header>
</template>

<style scoped>
.topic-hero {
  background:
    radial-gradient(circle at top left, rgba(240, 138, 75, 0.34), transparent 38%),
    radial-gradient(circle at 82% 20%, rgba(104, 164, 255, 0.26), transparent 30%),
    linear-gradient(135deg, #111a22 0%, #1b3041 54%, #3b2d1f 100%);
  box-shadow: 0 24px 70px rgba(11, 19, 28, 0.28);
}

.topic-toolbar {
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(250, 247, 241, 0.08);
  backdrop-filter: blur(16px);
}

.topic-toolbar__eyebrow {
  font-size: 0.7rem;
  letter-spacing: 0.24em;
  text-transform: uppercase;
  color: rgba(255, 255, 255, 0.56);
}

.topic-toolbar {
  display: grid;
  gap: 0.9rem;
  min-width: min(100%, 22rem);
}

.topic-toolbar__switcher {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0.5rem;
}

.topic-toolbar__switch,
.topic-toolbar__button,
.topic-toolbar__input {
  min-height: 3rem;
  border-radius: 999px;
}

.topic-toolbar__switch,
.topic-toolbar__button {
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.06);
  color: white;
}

.topic-toolbar__switch--active {
  background: rgba(240, 138, 75, 0.92);
  color: #1d120a;
}

.topic-toolbar__date {
  display: grid;
  gap: 0.55rem;
}

.topic-toolbar__input {
  width: 100%;
  border: 1px solid rgba(255, 255, 255, 0.14);
  background: rgba(255, 255, 255, 0.08);
  color: white;
  padding: 0 1rem;
}

.topic-toolbar__button {
  font-weight: 600;
}
</style>
