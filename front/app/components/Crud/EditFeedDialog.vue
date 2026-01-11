<script setup lang="ts">
import { Icon } from "@iconify/vue"
import type { RssFeed } from '~/types'

const props = defineProps<{
  feed: RssFeed
}>()

const emit = defineEmits<{
  close: []
  updated: []
  deleted: []
}>()

const apiStore = useApiStore()

const url = ref(props.feed.url)
const categoryId = ref<number | undefined>(
  props.feed.category ? Number(props.feed.category) : undefined
)
const title = ref(props.feed.title)
const description = ref(props.feed.description)
const loading = ref(false)
const deleting = ref(false)
const error = ref<string | null>(null)
const showDeleteConfirm = ref(false)

async function handleSubmit() {
  if (!url.value) return

  loading.value = true
  error.value = null

  const response = await apiStore.updateFeed(props.feed.id, {
    url: url.value,
    category_id: categoryId.value,
  })

  loading.value = false

  if (response.success) {
    emit('updated')
    emit('close')
  } else {
    error.value = response.error || 'Failed to update feed'
  }
}

async function handleDelete() {
  deleting.value = true
  error.value = null

  const response = await apiStore.deleteFeed(props.feed.id)

  deleting.value = false

  if (response.success) {
    emit('deleted')
    emit('close')
  } else {
    error.value = response.error || 'Failed to delete feed'
    showDeleteConfirm.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition ease-out duration-200"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition ease-in duration-150"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        class="fixed inset-0 bg-black/40 backdrop-blur-sm flex items-center justify-center z-50 p-4"
        @click.self="emit('close')"
      >
        <Transition
          enter-active-class="transition ease-out duration-200"
          enter-from-class="opacity-0 scale-95 translate-y-2"
          enter-to-class="opacity-100 scale-100 translate-y-0"
          leave-active-class="transition ease-in duration-150"
          leave-from-class="opacity-100 scale-100 translate-y-0"
          leave-to-class="opacity-0 scale-95 translate-y-2"
        >
          <div
            class="glass-strong rounded-3xl shadow-2xl w-full max-w-lg overflow-hidden"
            @click.stop
          >
            <!-- Header -->
            <div class="px-6 py-5 bg-linear-to-r from-primary-50 to-primary-100/50 border-b border-primary-100/50">
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-3">
                  <div class="w-11 h-11 rounded-2xl bg-linear-to-br from-primary-600 to-primary-800 flex items-center justify-center shadow-lg">
                    <Icon icon="mdi:pencil-box" class="text-white" width="24" height="24" />
                  </div>
                  <div>
                    <h2 class="text-xl font-bold text-gray-800">编辑订阅源</h2>
                    <p class="text-sm text-gray-500">修改订阅源设置</p>
                  </div>
                </div>
                <button
                  class="p-2.5 hover:bg-white/70 rounded-xl transition-all hover:shadow-md active:scale-95"
                  @click="emit('close')"
                >
                  <Icon icon="mdi:close" width="20" height="20" class="text-gray-500" />
                </button>
              </div>
            </div>

            <div class="p-6 space-y-5">
              <!-- Current Info Display -->
              <div class="p-4 bg-linear-to-br from-primary-50 to-primary-100/50 rounded-2xl border-2 border-primary-200">
                <div class="flex items-start gap-3">
                  <div class="w-11 h-11 rounded-2xl bg-linear-to-br from-primary-500 to-primary-700 flex items-center justify-center shrink-0 shadow-md">
                    <Icon icon="mdi:rss" class="text-white" width="22" height="22" />
                  </div>
                  <div class="flex-1 min-w-0">
                    <h3 class="font-semibold text-gray-900 mb-1 truncate">
                      {{ title }}
                    </h3>
                    <p class="text-sm text-gray-600 line-clamp-2">
                      {{ description || '暂无描述' }}
                    </p>
                  </div>
                </div>
              </div>

              <!-- URL Input -->
              <div class="space-y-2">
                <label class="flex items-center gap-2 text-sm font-semibold text-gray-700">
                  <Icon icon="mdi:link-variant" width="16" height="16" class="text-primary-500" />
                  RSS 订阅地址
                  <span class="text-red-500">*</span>
                </label>
                <input
                  v-model="url"
                  type="url"
                  placeholder="https://example.com/feed.xml"
                  class="input w-full"
                >
              </div>

              <!-- Category Select -->
              <div class="space-y-2">
                <label class="flex items-center gap-2 text-sm font-semibold text-gray-700">
                  <Icon icon="mdi:folder" width="16" height="16" class="text-primary-500" />
                  分类
                  <span class="text-xs font-normal text-gray-400">(可选)</span>
                </label>
                <select
                  v-model="categoryId"
                  class="input w-full cursor-pointer"
                >
                  <option :value="undefined">未分类</option>
                  <option
                    v-for="category in apiStore.categories"
                    :key="category.id"
                    :value="Number(category.id)"
                  >
                    {{ category.name }}
                  </option>
                </select>
              </div>

              <!-- Error -->
              <Transition
                enter-active-class="transition ease-out duration-200"
                enter-from-class="opacity-0 translate-y-1"
                enter-to-class="opacity-100 translate-y-0"
                leave-active-class="transition ease-in duration-150"
                leave-from-class="opacity-100 translate-y-0"
                leave-to-class="opacity-0 translate-y-1"
              >
                <div
                  v-if="error"
                  class="flex items-start gap-3 p-4 bg-red-50/80 backdrop-blur-sm border-2 border-red-200 rounded-2xl"
                >
                  <Icon icon="mdi:alert-circle" width="20" height="20" class="text-red-500 shrink-0 mt-0.5" />
                  <p class="text-sm font-medium text-red-700">{{ error }}</p>
                </div>
              </Transition>

              <!-- Delete Warning -->
              <Transition
                enter-active-class="transition ease-out duration-200"
                enter-from-class="opacity-0 translate-y-1"
                enter-to-class="opacity-100 translate-y-0"
                leave-active-class="transition ease-in duration-150"
                leave-from-class="opacity-100 translate-y-0"
                leave-to-class="opacity-0 translate-y-1"
              >
                <div
                  v-if="showDeleteConfirm"
                  class="p-4 bg-red-50/80 backdrop-blur-sm border-2 border-red-200 rounded-2xl"
                >
                  <div class="flex items-start gap-3">
                    <Icon icon="mdi:alert-circle" class="text-red-600 shrink-0 mt-0.5" width="22" height="22" />
                    <div class="flex-1">
                      <h4 class="font-semibold text-red-900 mb-1">确认删除订阅源？</h4>
                      <p class="text-sm text-red-700 mb-3">
                        删除订阅源 "{{ title }}" 将同时删除该订阅源下的所有文章，此操作不可撤销。
                      </p>
                      <div class="flex gap-2">
                        <button
                          class="btn-secondary px-4 py-2"
                          @click="showDeleteConfirm = false"
                          :disabled="deleting"
                        >
                          取消
                        </button>
                        <button
                          class="px-4 py-2 rounded-xl font-semibold bg-red-600 hover:bg-red-700 text-white shadow-md hover:shadow-lg transition-all active:scale-95 disabled:opacity-50 disabled:cursor-not-allowed disabled:shadow-none disabled:active:scale-100 flex items-center gap-2"
                          @click="handleDelete"
                          :disabled="deleting"
                        >
                          <Icon
                            :icon="deleting ? 'mdi:loading' : 'mdi:delete'"
                            :class="{ 'animate-spin': deleting }"
                            width="16"
                            height="16"
                          />
                          {{ deleting ? '删除中...' : '确认删除' }}
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              </Transition>
            </div>

            <!-- Footer -->
            <div class="px-6 py-4 bg-white/40 border-t border-white/40 flex justify-between items-center">
              <Transition
                enter-active-class="transition ease-out duration-200"
                enter-from-class="opacity-0 scale-95"
                enter-to-class="opacity-100 scale-100"
                leave-active-class="transition ease-in duration-150"
                leave-from-class="opacity-100 scale-100"
                leave-to-class="opacity-0 scale-95"
              >
                <button
                  v-if="!showDeleteConfirm"
                  class="px-4 py-2.5 rounded-xl font-semibold text-red-600 hover:bg-red-50/80 transition-all active:scale-95 flex items-center gap-2"
                  @click="showDeleteConfirm = true"
                >
                  <Icon icon="mdi:delete" width="18" height="18" />
                  删除订阅源
                </button>
                <div v-else class="w-36"></div>
              </Transition>

              <Transition
                enter-active-class="transition ease-out duration-200"
                enter-from-class="opacity-0 translate-x-2"
                enter-to-class="opacity-100 translate-x-0"
                leave-active-class="transition ease-in duration-150"
                leave-from-class="opacity-100 translate-x-0"
                leave-to-class="opacity-0 translate-x-2"
              >
                <div v-if="!showDeleteConfirm" class="flex gap-3">
                  <button class="btn-secondary" @click="emit('close')">
                    取消
                  </button>
                  <button
                    class="btn-primary flex items-center gap-2"
                    :disabled="!url || loading"
                    @click="handleSubmit"
                  >
                    <Icon
                      :icon="loading ? 'mdi:loading' : 'mdi:check'"
                      :class="{ 'animate-spin': loading }"
                      width="18"
                      height="18"
                    />
                    {{ loading ? '保存中...' : '保存更改' }}
                  </button>
                </div>
              </Transition>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>
