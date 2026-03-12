<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { useAIAdminApi, useSummariesApi } from '~/api'
import type { AIProvider, AIRoute, AIProviderUpsertRequest } from '~/types'

const routeLabels: Record<string, string> = {
  summary: '文章总结',
  article_completion: '正文补全',
  topic_tagging: '主题提取',
  digest_polish: '日报润色',
}

const capabilityOrder = ['summary', 'article_completion', 'topic_tagging', 'digest_polish']

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const error = ref<string | null>(null)
const success = ref<string | null>(null)

const providers = ref<AIProvider[]>([])
const routes = ref<AIRoute[]>([])
const routeSelections = ref<Record<string, number[]>>({})

const primaryProviderId = ref<number | null>(null)
const primaryProviderForm = reactive<AIProviderUpsertRequest & { time_range: number }>({
  name: 'default-primary',
  provider_type: 'openai_compatible',
  base_url: '',
  api_key: '',
  model: '',
  enabled: true,
  timeout_seconds: 120,
  time_range: 180,
})

const newProviderForm = reactive<AIProviderUpsertRequest>({
  name: '',
  provider_type: 'openai_compatible',
  base_url: '',
  api_key: '',
  model: '',
  enabled: true,
  timeout_seconds: 120,
})

const showNewProviderForm = ref(false)
const showPrimaryApiKey = ref(false)
const showNewProviderApiKey = ref(false)
const { loadSettings: reloadAISettings } = useAI()
const editingProviderId = ref<number | null>(null)
const draggingProviderId = ref<number | null>(null)
const draggingCapability = ref<string | null>(null)
const editProviderForm = reactive<AIProviderUpsertRequest>({
  name: '',
  provider_type: 'openai_compatible',
  base_url: '',
  api_key: '',
  model: '',
  enabled: true,
  timeout_seconds: 120,
})
const showEditProviderApiKey = ref(false)

const backupProviders = computed(() => providers.value.filter(provider => provider.id !== primaryProviderId.value))

function routeSummary(capability: string) {
  return routeSelections.value[capability] || []
}

function providerName(providerId: number) {
  return providers.value.find(provider => provider.id === providerId)?.name || `#${providerId}`
}

function isProviderLinked(providerId: number) {
  return routes.value.some(route => route.route_providers.some(link => link.provider_id === providerId))
}

function hydratePrimaryProvider() {
  const preferredProvider = providers.value.find(provider => provider.id === primaryProviderId.value)
    || providers.value.find(provider => provider.name === 'default-primary')
    || providers.value[0]

  primaryProviderId.value = preferredProvider?.id || null
  primaryProviderForm.name = preferredProvider?.name || 'default-primary'
  primaryProviderForm.provider_type = preferredProvider?.provider_type || 'openai_compatible'
  primaryProviderForm.base_url = preferredProvider?.base_url || ''
  primaryProviderForm.api_key = ''
  primaryProviderForm.model = preferredProvider?.model || ''
  primaryProviderForm.enabled = preferredProvider?.enabled ?? true
  primaryProviderForm.timeout_seconds = preferredProvider?.timeout_seconds || 120
}

function applyPrimaryProvider(provider: AIProvider | null | undefined) {
  primaryProviderId.value = provider?.id || null
  primaryProviderForm.name = provider?.name || 'default-primary'
  primaryProviderForm.provider_type = provider?.provider_type || 'openai_compatible'
  primaryProviderForm.base_url = provider?.base_url || ''
  primaryProviderForm.api_key = ''
  primaryProviderForm.model = provider?.model || ''
  primaryProviderForm.enabled = provider?.enabled ?? true
  primaryProviderForm.timeout_seconds = provider?.timeout_seconds || 120
}

function hydrateRouteSelections() {
  const nextSelections: Record<string, number[]> = {}
  for (const capability of capabilityOrder) {
    const route = routes.value.find(item => item.capability === capability)
    if (!route) {
      nextSelections[capability] = []
      continue
    }

    const providerIds = route.route_providers
      .slice()
      .sort((a, b) => a.priority - b.priority)
      .map(link => link.provider_id)

    const filtered = providerIds.filter(providerId => providerId !== primaryProviderId.value)
    nextSelections[capability] = filtered
  }
  routeSelections.value = nextSelections
}

async function loadData() {
  loading.value = true
  error.value = null

  try {
    const aiAdminApi = useAIAdminApi()
    const [settingsResponse, providersResponse, routesResponse] = await Promise.all([
      aiAdminApi.getSettings(),
      aiAdminApi.listProviders(),
      aiAdminApi.listRoutes(),
    ])

    if (!providersResponse.success || !routesResponse.success) {
      throw new Error(providersResponse.error || routesResponse.error || '加载 AI 配置失败')
    }

    providers.value = providersResponse.data || []
    routes.value = routesResponse.data || []
    primaryProviderId.value = settingsResponse.success && settingsResponse.data?.provider_id
      ? Number(settingsResponse.data.provider_id)
      : null
    primaryProviderForm.time_range = settingsResponse.success && typeof settingsResponse.data?.time_range === 'number'
      ? settingsResponse.data.time_range
      : 180

    hydratePrimaryProvider()
    hydrateRouteSelections()
  } catch (err) {
    error.value = err instanceof Error ? err.message : '加载 AI 配置失败'
  } finally {
    loading.value = false
  }
}

function pushMessage(kind: 'success' | 'error', message: string) {
  if (kind === 'success') {
    success.value = message
    error.value = null
    setTimeout(() => {
      success.value = null
    }, 2500)
    return
  }

  error.value = message
  success.value = null
}

function addProviderToRoute(capability: string, providerId: number) {
  const current = routeSummary(capability)
  if (current.includes(providerId)) return
  routeSelections.value[capability] = [...current, providerId]
}

function removeProviderFromRoute(capability: string, providerId: number) {
  routeSelections.value[capability] = routeSummary(capability).filter(id => id !== providerId)
}

function moveProvider(capability: string, providerId: number, direction: -1 | 1) {
  const current = [...routeSummary(capability)]
  const index = current.indexOf(providerId)
  const nextIndex = index + direction
  if (index < 0 || nextIndex < 0 || nextIndex >= current.length) return
  const currentValue = current[index]
  const nextValue = current[nextIndex]
  if (currentValue === undefined || nextValue === undefined) return
  current[index] = nextValue
  current[nextIndex] = currentValue
  routeSelections.value[capability] = current
}

function handleDragStart(capability: string, providerId: number) {
  draggingCapability.value = capability
  draggingProviderId.value = providerId
}

function handleDragEnd() {
  draggingCapability.value = null
  draggingProviderId.value = null
}

function handleDropOnProvider(capability: string, targetProviderId: number) {
  if (draggingCapability.value !== capability || draggingProviderId.value === null) {
    handleDragEnd()
    return
  }

  const current = [...routeSummary(capability)]
  const fromIndex = current.indexOf(draggingProviderId.value)
  const toIndex = current.indexOf(targetProviderId)
  if (fromIndex < 0 || toIndex < 0 || fromIndex === toIndex) {
    handleDragEnd()
    return
  }

  const [moved] = current.splice(fromIndex, 1)
  if (moved === undefined) {
    handleDragEnd()
    return
  }
  current.splice(toIndex, 0, moved)
  routeSelections.value[capability] = current
  handleDragEnd()
}

async function savePrimaryProvider() {
  if (!primaryProviderForm.base_url || !primaryProviderForm.model) {
    pushMessage('error', '主模型至少要有 Base URL 和 Model')
    return
  }

  saving.value = true
  error.value = null

  try {
    const aiAdminApi = useAIAdminApi()
    let providerId = primaryProviderId.value

    if (providerId) {
      const response = await aiAdminApi.updateProvider(providerId, primaryProviderForm)
      if (!response.success) {
        throw new Error(response.error || '更新主模型失败')
      }
    } else {
      if (!primaryProviderForm.api_key) {
        throw new Error('首次创建主模型需要填写 API Key')
      }
      const response = await aiAdminApi.createProvider(primaryProviderForm)
      if (!response.success || !response.data) {
        throw new Error(response.error || '创建主模型失败')
      }
      providerId = response.data.id
      primaryProviderId.value = providerId
    }

    const providerSnapshot: AIProvider = {
      id: providerId || 0,
      name: primaryProviderForm.name,
      provider_type: primaryProviderForm.provider_type || 'openai_compatible',
      base_url: primaryProviderForm.base_url,
      model: primaryProviderForm.model,
      enabled: primaryProviderForm.enabled ?? true,
      timeout_seconds: primaryProviderForm.timeout_seconds || 120,
      max_tokens: primaryProviderForm.max_tokens ?? null,
      temperature: primaryProviderForm.temperature ?? null,
      metadata: primaryProviderForm.metadata,
      api_key_configured: true,
    }

    if (providerId) {
      for (const capability of capabilityOrder) {
        const existingRoute = routes.value.find(route => route.capability === capability)
        const providerIds = [providerId, ...routeSummary(capability)]
        const response = await aiAdminApi.updateRoute(capability, {
          name: existingRoute?.name || 'default',
          enabled: existingRoute?.enabled ?? true,
          description: existingRoute?.description,
          provider_ids: providerIds,
        })
        if (!response.success) {
          throw new Error(response.error || `同步 ${routeLabels[capability]} 主路由失败`)
        }
      }
    }

    if (providerId) {
      const summariesApi = useSummariesApi()
      const autoSummaryResponse = await summariesApi.updateAutoSummaryConfig({
        base_url: primaryProviderForm.base_url,
        api_key: primaryProviderForm.api_key || undefined,
        model: primaryProviderForm.model,
        time_range: primaryProviderForm.time_range,
      })
      if (!autoSummaryResponse.success) {
        throw new Error(autoSummaryResponse.error || '保存自动总结配置失败')
      }
    }

    await loadData()
    applyPrimaryProvider(providerSnapshot)
    await reloadAISettings(true)
    pushMessage('success', '主模型配置已保存')
  } catch (err) {
    pushMessage('error', err instanceof Error ? err.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function saveNewProvider() {
  if (!newProviderForm.name || !newProviderForm.base_url || !newProviderForm.api_key || !newProviderForm.model) {
    pushMessage('error', '备用模型表单还没填完整')
    return
  }

  saving.value = true
  try {
    const aiAdminApi = useAIAdminApi()
    const response = await aiAdminApi.createProvider(newProviderForm)
    if (!response.success) {
      throw new Error(response.error || '创建备用模型失败')
    }

    newProviderForm.name = ''
    newProviderForm.base_url = ''
    newProviderForm.api_key = ''
    newProviderForm.model = ''
    newProviderForm.enabled = true
    newProviderForm.timeout_seconds = 120
    showNewProviderForm.value = false
    await loadData()
    pushMessage('success', '备用模型已添加')
  } catch (err) {
    pushMessage('error', err instanceof Error ? err.message : '创建失败')
  } finally {
    saving.value = false
  }
}

function startEditingProvider(provider: AIProvider) {
  editingProviderId.value = provider.id
  editProviderForm.name = provider.name
  editProviderForm.provider_type = provider.provider_type
  editProviderForm.base_url = provider.base_url
  editProviderForm.api_key = ''
  editProviderForm.model = provider.model
  editProviderForm.enabled = provider.enabled
  editProviderForm.timeout_seconds = provider.timeout_seconds
}

function cancelEditingProvider() {
  editingProviderId.value = null
  editProviderForm.name = ''
  editProviderForm.provider_type = 'openai_compatible'
  editProviderForm.base_url = ''
  editProviderForm.api_key = ''
  editProviderForm.model = ''
  editProviderForm.enabled = true
  editProviderForm.timeout_seconds = 120
}

async function saveEditedProvider() {
  if (!editingProviderId.value) return
  if (!editProviderForm.name || !editProviderForm.base_url || !editProviderForm.model) {
    pushMessage('error', '编辑备用模型时，名称、Base URL 和 Model 不能为空')
    return
  }

  saving.value = true
  try {
    const aiAdminApi = useAIAdminApi()
    const response = await aiAdminApi.updateProvider(editingProviderId.value, editProviderForm)
    if (!response.success) {
      throw new Error(response.error || '更新备用模型失败')
    }
    cancelEditingProvider()
    await loadData()
    pushMessage('success', '备用模型已更新')
  } catch (err) {
    pushMessage('error', err instanceof Error ? err.message : '更新失败')
  } finally {
    saving.value = false
  }
}

async function deleteBackupProvider(provider: AIProvider) {
  if (!confirm(`确定删除备用模型 ${provider.name} 吗？`)) {
    return
  }

  saving.value = true
  try {
    const aiAdminApi = useAIAdminApi()
    const response = await aiAdminApi.deleteProvider(provider.id)
    if (!response.success) {
      throw new Error(response.error || '删除备用模型失败')
    }

    for (const capability of capabilityOrder) {
      removeProviderFromRoute(capability, provider.id)
    }

    if (editingProviderId.value === provider.id) {
      cancelEditingProvider()
    }

    await loadData()
    pushMessage('success', '备用模型已删除')
  } catch (err) {
    pushMessage('error', err instanceof Error ? err.message : '删除失败')
  } finally {
    saving.value = false
  }
}

async function saveRoutes() {
  if (!primaryProviderId.value) {
    pushMessage('error', '请先保存主模型，再配置路由')
    return
  }

  saving.value = true
  try {
    const aiAdminApi = useAIAdminApi()
    for (const capability of capabilityOrder) {
      const providerIds = [primaryProviderId.value, ...routeSummary(capability)]
      const response = await aiAdminApi.updateRoute(capability, {
        name: 'default',
        enabled: true,
        provider_ids: providerIds,
      })
      if (!response.success) {
        throw new Error(response.error || `保存 ${routeLabels[capability]} 路由失败`)
      }
    }

    await loadData()
    await reloadAISettings(true)
    pushMessage('success', '多路由顺序已保存')
  } catch (err) {
    pushMessage('error', err instanceof Error ? err.message : '保存路由失败')
  } finally {
    saving.value = false
  }
}

async function testPrimaryProvider() {
  if (!primaryProviderForm.base_url || !primaryProviderForm.model || !primaryProviderForm.api_key) {
    pushMessage('error', '测试连接前请填入 Base URL、Model 和 API Key')
    return
  }

  testing.value = true
  try {
    const aiAdminApi = useAIAdminApi()
    const response = await aiAdminApi.testConnection({
      base_url: primaryProviderForm.base_url,
      api_key: primaryProviderForm.api_key,
      model: primaryProviderForm.model,
    })
    if (!response.success) {
      throw new Error(response.error || '连接测试失败')
    }
    pushMessage('success', response.message || '连接测试成功')
  } catch (err) {
    pushMessage('error', err instanceof Error ? err.message : '连接测试失败')
  } finally {
    testing.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<template>
  <div class="space-y-6">
    <div class="bg-gradient-to-br from-ink-50 to-paper-cream rounded-xl p-6 border border-ink-100">
      <div class="flex items-start justify-between gap-4 mb-4">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-lg bg-gradient-to-br from-ink-500 to-ink-700 flex items-center justify-center">
            <Icon icon="mdi:brain" width="20" height="20" class="text-white" />
          </div>
          <div>
            <h3 class="font-semibold text-gray-900">AI Router</h3>
            <p class="text-xs text-gray-500">主模型 + 备用模型 + 各能力路由顺序</p>
          </div>
        </div>
        <button
          class="px-4 py-2 text-sm font-medium text-white bg-ink-700 rounded-lg hover:bg-ink-800 transition-colors disabled:opacity-50"
          :disabled="saving"
          @click="savePrimaryProvider"
        >
          保存主模型
        </button>
      </div>

      <div v-if="loading" class="py-8 flex justify-center">
        <Icon icon="mdi:loading" width="28" height="28" class="animate-spin text-ink-600" />
      </div>

      <div v-else class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1.5">Primary Name</label>
            <input v-model="primaryProviderForm.name" type="text" class="input w-full" placeholder="default-primary">
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1.5">Model</label>
            <input v-model="primaryProviderForm.model" type="text" class="input w-full" placeholder="gpt-4o-mini">
          </div>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1.5">Base URL</label>
          <input v-model="primaryProviderForm.base_url" type="text" class="input w-full" placeholder="https://api.openai.com/v1">
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1.5">API Key</label>
          <div class="relative">
            <input
              v-model="primaryProviderForm.api_key"
              :type="showPrimaryApiKey ? 'text' : 'password'"
              class="input w-full pr-12"
              placeholder="留空表示沿用已保存密钥"
            >
            <button class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400" @click="showPrimaryApiKey = !showPrimaryApiKey">
              <Icon :icon="showPrimaryApiKey ? 'mdi:eye-off' : 'mdi:eye'" width="16" height="16" />
            </button>
          </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1.5">Timeout</label>
            <input v-model.number="primaryProviderForm.timeout_seconds" type="number" min="30" class="input w-full">
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1.5">自动总结时间范围（分钟）</label>
            <input v-model.number="primaryProviderForm.time_range" type="number" min="60" step="60" class="input w-full">
          </div>
        </div>

        <div class="flex gap-2 pt-1">
          <button class="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50" :disabled="testing" @click="testPrimaryProvider">
            <Icon v-if="testing" icon="mdi:loading" width="14" height="14" class="animate-spin inline-block mr-1" />
            测试主模型
          </button>
        </div>
      </div>
    </div>

    <div class="bg-white rounded-xl border border-gray-200 p-6 space-y-4">
      <div class="flex items-center justify-between gap-4">
        <div>
          <h3 class="font-semibold text-gray-900">备用模型池</h3>
          <p class="text-xs text-gray-500">这里的模型可被不同能力挂成备用链</p>
        </div>
        <button class="px-3 py-1.5 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors" @click="showNewProviderForm = !showNewProviderForm">
          {{ showNewProviderForm ? '收起' : '新增 provider' }}
        </button>
      </div>

      <div v-if="showNewProviderForm" class="grid grid-cols-1 md:grid-cols-2 gap-4 rounded-xl border border-dashed border-gray-300 p-4 bg-gray-50">
        <input v-model="newProviderForm.name" type="text" class="input w-full" placeholder="provider 名称">
        <input v-model="newProviderForm.model" type="text" class="input w-full" placeholder="模型名">
        <input v-model="newProviderForm.base_url" type="text" class="input w-full md:col-span-2" placeholder="https://api.example.com/v1">
        <div class="relative md:col-span-2">
          <input v-model="newProviderForm.api_key" :type="showNewProviderApiKey ? 'text' : 'password'" class="input w-full pr-12" placeholder="API Key">
          <button class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400" @click="showNewProviderApiKey = !showNewProviderApiKey">
            <Icon :icon="showNewProviderApiKey ? 'mdi:eye-off' : 'mdi:eye'" width="16" height="16" />
          </button>
        </div>
        <div class="md:col-span-2 flex justify-end">
          <button class="px-4 py-2 text-sm font-medium text-white bg-ink-700 rounded-lg hover:bg-ink-800 transition-colors disabled:opacity-50" :disabled="saving" @click="saveNewProvider">
            添加备用模型
          </button>
        </div>
      </div>

      <div v-if="backupProviders.length === 0" class="text-sm text-gray-500 rounded-lg bg-gray-50 p-4">
        还没有备用模型，先加一个，失败切换才有地方去。
      </div>

      <div v-else class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <div v-for="provider in backupProviders" :key="provider.id" class="rounded-xl border border-gray-200 p-4 bg-gray-50/70">
          <div class="flex items-center justify-between gap-3">
            <div>
              <div class="font-medium text-gray-900">{{ provider.name }}</div>
              <div class="text-xs text-gray-500 mt-1">{{ provider.model }} · {{ provider.base_url }}</div>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-xs px-2 py-1 rounded-full" :class="provider.enabled ? 'bg-emerald-50 text-emerald-700' : 'bg-gray-100 text-gray-500'">
                {{ provider.enabled ? '已启用' : '已停用' }}
              </span>
              <button class="text-xs px-2 py-1 rounded-lg bg-white border border-gray-200 text-gray-700 hover:bg-gray-50" @click="startEditingProvider(provider)">
                编辑
              </button>
              <button class="text-xs px-2 py-1 rounded-lg bg-red-50 border border-red-200 text-red-700 hover:bg-red-100 disabled:opacity-50 disabled:cursor-not-allowed" :disabled="isProviderLinked(provider.id)" @click="deleteBackupProvider(provider)">
                删除
              </button>
            </div>
          </div>
          <p v-if="isProviderLinked(provider.id)" class="mt-3 text-xs text-amber-700">
            这个 provider 还挂在某条能力路由上，先从路由里移除再删除。
          </p>

          <div v-if="editingProviderId === provider.id" class="mt-4 grid grid-cols-1 gap-3 rounded-lg border border-dashed border-gray-300 bg-white p-3">
            <input v-model="editProviderForm.name" type="text" class="input w-full" placeholder="provider 名称">
            <input v-model="editProviderForm.model" type="text" class="input w-full" placeholder="模型名">
            <input v-model="editProviderForm.base_url" type="text" class="input w-full" placeholder="https://api.example.com/v1">
            <div class="relative">
              <input v-model="editProviderForm.api_key" :type="showEditProviderApiKey ? 'text' : 'password'" class="input w-full pr-12" placeholder="留空表示沿用已保存密钥">
              <button class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400" @click="showEditProviderApiKey = !showEditProviderApiKey">
                <Icon :icon="showEditProviderApiKey ? 'mdi:eye-off' : 'mdi:eye'" width="16" height="16" />
              </button>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <input v-model.number="editProviderForm.timeout_seconds" type="number" min="30" class="input w-full" placeholder="Timeout">
              <label class="flex items-center gap-2 text-sm text-gray-700">
                <input v-model="editProviderForm.enabled" type="checkbox">
                启用该 provider
              </label>
            </div>
            <div class="flex justify-end gap-2">
              <button class="px-3 py-1.5 text-sm rounded-lg border border-gray-200 text-gray-700 hover:bg-gray-50" @click="cancelEditingProvider">
                取消
              </button>
              <button class="px-3 py-1.5 text-sm rounded-lg bg-ink-700 text-white hover:bg-ink-800 disabled:opacity-50" :disabled="saving" @click="saveEditedProvider">
                保存修改
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="bg-white rounded-xl border border-gray-200 p-6 space-y-5">
      <div class="flex items-center justify-between gap-4">
        <div>
          <h3 class="font-semibold text-gray-900">能力路由</h3>
          <p class="text-xs text-gray-500">每个能力默认都会先走主模型，再按下面顺序尝试备用模型</p>
        </div>
        <button class="px-4 py-2 text-sm font-medium text-white bg-ink-700 rounded-lg hover:bg-ink-800 transition-colors disabled:opacity-50" :disabled="saving" @click="saveRoutes">
          保存路由顺序
        </button>
      </div>

      <div v-for="capability in capabilityOrder" :key="capability" class="rounded-xl border border-gray-200 p-4">
        <div class="flex items-center justify-between gap-3 mb-3">
          <div>
            <div class="font-medium text-gray-900">{{ routeLabels[capability] }}</div>
            <div class="text-xs text-gray-500">主模型固定排第一，下面可以拖成备用链的顺序</div>
          </div>
        </div>

        <div class="flex flex-wrap gap-2 mb-3">
          <span class="px-3 py-1 rounded-full text-xs font-medium bg-ink-100 text-ink-700">
            主: {{ primaryProviderForm.name || '未配置' }}
          </span>
          <span
            v-for="providerId in routeSummary(capability)"
            :key="providerId"
            draggable="true"
            class="inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-medium bg-teal-50 text-teal-700 cursor-move"
            @dragstart="handleDragStart(capability, providerId)"
            @dragend="handleDragEnd"
            @dragover.prevent
            @drop.prevent="handleDropOnProvider(capability, providerId)"
          >
            <Icon icon="mdi:drag" width="12" height="12" />
            备: {{ providerName(providerId) }}
            <button @click="moveProvider(capability, providerId, -1)">
              <Icon icon="mdi:arrow-left" width="12" height="12" />
            </button>
            <button @click="moveProvider(capability, providerId, 1)">
              <Icon icon="mdi:arrow-right" width="12" height="12" />
            </button>
            <button @click="removeProviderFromRoute(capability, providerId)">
              <Icon icon="mdi:close" width="12" height="12" />
            </button>
          </span>
        </div>

        <div class="flex flex-wrap gap-2">
          <button
            v-for="provider in backupProviders"
            :key="provider.id"
            class="px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors"
            :class="routeSummary(capability).includes(provider.id) ? 'border-teal-200 bg-teal-50 text-teal-700' : 'border-gray-200 bg-white text-gray-700 hover:bg-gray-50'"
            @click="routeSummary(capability).includes(provider.id) ? removeProviderFromRoute(capability, provider.id) : addProviderToRoute(capability, provider.id)"
          >
            {{ routeSummary(capability).includes(provider.id) ? '移除' : '加入' }} {{ provider.name }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="success" class="rounded-lg bg-emerald-50 border border-emerald-200 px-4 py-3 text-sm text-emerald-700">
      {{ success }}
    </div>
    <div v-if="error" class="rounded-lg bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-700">
      {{ error }}
    </div>
  </div>
</template>
