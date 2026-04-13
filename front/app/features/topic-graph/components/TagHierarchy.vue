<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { computed, onMounted, ref } from 'vue'
import { useAbstractTagApi } from '~/api/abstractTags'
import TagHierarchyRow from '~/features/topic-graph/components/TagHierarchyRow.vue'
import type { TagHierarchyNode } from '~/types/topicTag'

const abstractTagApi = useAbstractTagApi()

const nodes = ref<TagHierarchyNode[]>([])
const loading = ref(true)
const error = ref<string | null>(null)
const selectedCategory = ref<string>('')

// Inline editing state
const editingNodeId = ref<number | null>(null)
const editingValue = ref('')
const saving = ref(false)

// Detach confirm state
const confirmingDetach = ref<{ parentId: number; childId: number; childLabel: string } | null>(null)

const filteredNodes = computed(() => {
  if (!selectedCategory.value) return nodes.value
  return nodes.value.filter(n => n.category === selectedCategory.value)
})

const categories = computed(() => {
  const cats = new Set<string>()
  nodes.value.forEach(n => cats.add(n.category))
  return Array.from(cats)
})

const totalCount = computed(() => {
  let count = 0
  function walk(list: TagHierarchyNode[]) {
    for (const node of list) {
      count++
      if (node.children.length > 0) walk(node.children)
    }
  }
  walk(filteredNodes.value)
  return count
})

async function loadHierarchy() {
  loading.value = true
  error.value = null
  try {
    const response = await abstractTagApi.fetchHierarchy(selectedCategory.value || undefined)
    if (response.success && response.data) {
      nodes.value = response.data.nodes
    } else {
      error.value = response.error || '加载层级数据失败'
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
  }
}

function startEdit(node: TagHierarchyNode) {
  editingNodeId.value = node.id
  editingValue.value = node.label
}

function cancelEdit() {
  editingNodeId.value = null
  editingValue.value = ''
}

async function confirmEdit() {
  if (!editingNodeId.value) return
  const name = editingValue.value.trim()
  if (!name || name.length > 160) return

  saving.value = true
  try {
    const response = await abstractTagApi.updateAbstractName(editingNodeId.value, name)
    if (response.success) {
      await loadHierarchy()
    } else {
      error.value = response.error || '更新失败'
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : '更新失败'
  } finally {
    saving.value = false
    editingNodeId.value = null
    editingValue.value = ''
  }
}

function requestDetach(node: TagHierarchyNode) {
  // Find parent ID from the tree
  const parentId = findParentId(nodes.value, node.id)
  if (parentId) {
    confirmingDetach.value = { parentId, childId: node.id, childLabel: node.label }
  }
}

function findParentId(tree: TagHierarchyNode[], targetId: number, parentId: number | null = null): number | null {
  for (const node of tree) {
    if (node.id === targetId) return parentId
    if (node.children.length > 0) {
      const found = findParentId(node.children, targetId, node.id)
      if (found !== null) return found
    }
  }
  return null
}

function cancelDetach() {
  confirmingDetach.value = null
}

async function confirmDetach() {
  if (!confirmingDetach.value) return
  const { parentId, childId } = confirmingDetach.value
  try {
    const response = await abstractTagApi.detachChild(parentId, childId)
    if (response.success) {
      await loadHierarchy()
    } else {
      error.value = response.error || '分离失败'
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : '分离失败'
  } finally {
    confirmingDetach.value = null
  }
}

function handleUpdateEditingValue(val: string) {
  editingValue.value = val
}

onMounted(() => {
  void loadHierarchy()
})
</script>

<template>
  <div class="tag-hierarchy">
    <!-- Header with category filter -->
    <div class="flex items-center justify-between gap-3 mb-4">
      <div>
        <h3 class="font-serif text-lg text-white">标签层级</h3>
        <p class="text-xs text-white/50 mt-0.5">
          {{ totalCount }} 个标签 {{ categories.length > 1 ? '· 选择类别筛选' : '' }}
        </p>
      </div>
      <div v-if="categories.length > 1" class="flex gap-1.5">
        <button
          type="button"
          class="th-category-btn"
          :class="{ 'th-category-btn--active': !selectedCategory }"
          @click="selectedCategory = ''; void loadHierarchy()"
        >
          全部
        </button>
        <button
          v-for="cat in categories"
          :key="cat"
          type="button"
          class="th-category-btn"
          :class="{ 'th-category-btn--active': selectedCategory === cat }"
          @click="selectedCategory = cat; void loadHierarchy()"
        >
          {{ cat === 'event' ? '事件' : cat === 'person' ? '人物' : '关键词' }}
        </button>
      </div>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="space-y-3">
      <div v-for="i in 3" :key="i" class="th-skeleton" />
    </div>

    <!-- Error -->
    <div v-else-if="error" class="th-error">
      <Icon icon="mdi:alert-circle-outline" width="16" />
      <span>{{ error }}</span>
    </div>

    <!-- Empty -->
    <div v-else-if="filteredNodes.length === 0" class="th-empty">
      <Icon icon="mdi:file-tree-outline" width="32" class="text-white/20" />
      <p>暂无标签层级关系</p>
      <p class="text-xs text-white/30 mt-1">当新文章入库时，中间相似度标签将自动提取抽象概念</p>
    </div>

    <!-- Tree -->
    <div v-else class="th-tree">
      <TagHierarchyRow
        v-for="node in filteredNodes"
        :key="node.id"
        :node="node"
        :depth="0"
        :editing-id="editingNodeId"
        :saving="saving"
        @start-edit="startEdit"
        @cancel-edit="cancelEdit"
        @confirm-edit="void confirmEdit()"
        @detach="requestDetach"
        @update:editing-value="handleUpdateEditingValue"
      />
    </div>

    <!-- Detach confirm dialog -->
    <Teleport to="body">
      <div v-if="confirmingDetach" class="th-confirm-overlay" @click.self="cancelDetach">
        <div class="th-confirm-dialog">
          <p class="text-sm text-white/80">确认将 <strong class="text-white">{{ confirmingDetach.childLabel }}</strong> 从抽象标签中分离？</p>
          <div class="flex gap-2 mt-4 justify-end">
            <button type="button" class="th-btn th-btn--ghost" @click="cancelDetach">取消</button>
            <button type="button" class="th-btn th-btn--danger" @click="void confirmDetach()">分离</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.tag-hierarchy { min-height: 200px; }

.th-skeleton {
  height: 36px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.04);
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 0.8; }
}

.th-error {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border-radius: 14px;
  border: 1px solid rgba(240, 138, 75, 0.3);
  background: rgba(240, 138, 75, 0.1);
  color: rgba(255, 220, 200, 0.88);
  font-size: 0.82rem;
}

.th-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 3rem 1rem;
  color: rgba(255, 255, 255, 0.4);
  font-size: 0.85rem;
}

.th-tree { display: flex; flex-direction: column; gap: 1px; }

.th-category-btn {
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 999px;
  background: none;
  color: rgba(255, 255, 255, 0.5);
  font-size: 0.7rem;
  padding: 0.2rem 0.65rem;
  cursor: pointer;
  transition: all 0.12s ease;
}
.th-category-btn:hover { border-color: rgba(255, 255, 255, 0.25); color: rgba(255, 255, 255, 0.8); }
.th-category-btn--active { border-color: rgba(240, 138, 75, 0.5); background: rgba(240, 138, 75, 0.12); color: rgba(255, 220, 200, 0.9); }

.th-confirm-overlay {
  position: fixed;
  inset: 0;
  z-index: 90;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 12, 18, 0.7);
  backdrop-filter: blur(8px);
}

.th-confirm-dialog {
  width: min(400px, 90%);
  border-radius: 1.25rem;
  border: 1px solid rgba(255, 255, 255, 0.1);
  background: rgba(17, 27, 38, 0.98);
  padding: 1.5rem;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.4);
}

.th-btn {
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 10px;
  background: none;
  color: rgba(255, 255, 255, 0.7);
  font-size: 0.82rem;
  padding: 0.4rem 1rem;
  cursor: pointer;
  transition: all 0.12s ease;
}
.th-btn--ghost:hover { background: rgba(255, 255, 255, 0.06); }
.th-btn--danger { border-color: rgba(240, 138, 75, 0.4); color: rgba(255, 200, 180, 0.9); }
.th-btn--danger:hover { background: rgba(240, 138, 75, 0.15); }
</style>
