<template>
  <div class="rounded-lg border border-dashed border-gray-200 bg-gray-50 p-3 dark:border-dark-600 dark:bg-dark-800/60">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
      <div>
        <div class="text-sm font-medium text-gray-900 dark:text-white">new-api 上游模型</div>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          从兼容 new-api 的 /v1/models 接口检测模型，并导入到当前账号的模型映射。
        </p>
      </div>
      <button
        type="button"
        class="btn btn-secondary btn-sm"
        :disabled="loading || !canDetect"
        @click="detectModels"
      >
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
        <span class="ml-1">{{ accountId ? '检测模型' : '预览模型' }}</span>
      </button>
    </div>

    <p v-if="!canDetect" class="mt-2 text-xs text-amber-600 dark:text-amber-300">
      请先填写 Base URL 和 API Key。Base URL 填站点根地址即可，不要带 /v1/models。
    </p>
    <p v-else-if="!hasGroup" class="mt-2 text-xs text-amber-600 dark:text-amber-300">
      可以先检测模型；应用或保存账号前，请至少选择一个 Zelvo 分组，用来决定这些模型归属到哪个套餐/用户范围。
    </p>

    <div v-if="diff" class="mt-3 space-y-3">
      <div class="grid gap-2 sm:grid-cols-4">
        <div class="rounded bg-white p-2 text-xs dark:bg-dark-700">
          <div class="text-gray-500">检测到</div>
          <div class="text-lg font-semibold text-gray-900 dark:text-white">{{ diff.models.length }}</div>
        </div>
        <div class="rounded bg-white p-2 text-xs dark:bg-dark-700">
          <div class="text-gray-500">新增</div>
          <div class="text-lg font-semibold text-emerald-600">{{ diff.add_models.length }}</div>
        </div>
        <div class="rounded bg-white p-2 text-xs dark:bg-dark-700">
          <div class="text-gray-500">已有</div>
          <div class="text-lg font-semibold text-blue-600">{{ diff.existing_models.length }}</div>
        </div>
        <div class="rounded bg-white p-2 text-xs dark:bg-dark-700">
          <div class="text-gray-500">失效</div>
          <div class="text-lg font-semibold text-red-600">{{ diff.remove_models.length }}</div>
        </div>
      </div>

      <div v-if="diff.add_models.length" class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-gray-600 dark:text-gray-300">新增模型</span>
          <button type="button" class="text-xs text-primary-600 hover:text-primary-700" @click="toggleAllAddModels">
            {{ allAddModelsSelected ? '清空选择' : '全选' }}
          </button>
        </div>
        <div class="max-h-40 overflow-y-auto rounded border border-gray-200 bg-white p-2 dark:border-dark-600 dark:bg-dark-700">
          <label
            v-for="model in diff.add_models"
            :key="model.id"
            class="flex cursor-pointer items-center justify-between gap-2 rounded px-2 py-1 text-xs hover:bg-gray-50 dark:hover:bg-dark-600"
          >
            <span class="min-w-0">
              <span class="block truncate font-medium text-gray-800 dark:text-gray-100">{{ model.id }}</span>
              <span class="text-gray-400">{{ model.provider || model.owned_by || '未知供应商' }}</span>
            </span>
            <input v-model="selectedAddModels" type="checkbox" class="rounded border-gray-300 text-primary-600" :value="model.id" />
          </label>
        </div>
      </div>

      <div v-if="diff.remove_models.length" class="space-y-2">
        <div class="flex items-center justify-between">
          <span class="text-xs font-medium text-gray-600 dark:text-gray-300">上游已不存在的模型</span>
          <button type="button" class="text-xs text-primary-600 hover:text-primary-700" @click="toggleAllRemoveModels">
            {{ allRemoveModelsSelected ? '清空选择' : '全选' }}
          </button>
        </div>
        <div class="max-h-32 overflow-y-auto rounded border border-red-100 bg-red-50 p-2 dark:border-red-900/50 dark:bg-red-900/20">
          <label
            v-for="model in diff.remove_models"
            :key="model"
            class="flex cursor-pointer items-center justify-between gap-2 rounded px-2 py-1 text-xs"
          >
            <span class="truncate text-red-700 dark:text-red-300">{{ model }}</span>
            <input v-model="selectedRemoveModels" type="checkbox" class="rounded border-gray-300 text-red-600" :value="model" />
          </label>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <label class="flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
          <input v-model="syncToCatalog" type="checkbox" class="rounded border-gray-300 text-primary-600" />
          同步新增模型到模型管理
        </label>
      </div>

      <div class="flex justify-end gap-2">
        <button type="button" class="btn btn-secondary btn-sm" :disabled="loading || selectedAddModels.length === 0" @click="ignoreSelected">
          暂时忽略所选
        </button>
        <button type="button" class="btn btn-primary btn-sm" :disabled="loading || !hasSelection" @click="applySelected">
          {{ accountId ? '应用到账号' : '使用所选映射' }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { AccountUpstreamModelDiff } from '@/api/admin/accountUpstreamModels'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  accountId?: number | null
  platform: string
  baseUrl: string
  apiKey?: string
  groupIds: number[]
  existingMapping: Record<string, string>
}>()

const emit = defineEmits<{
  mappingsUpdated: [mapping: Record<string, string>]
  applied: []
  error: [message: string]
  success: [message: string]
}>()

const loading = ref(false)
const diff = ref<AccountUpstreamModelDiff | null>(null)
const selectedAddModels = ref<string[]>([])
const selectedRemoveModels = ref<string[]>([])
const selectedIgnoreModels = ref<string[]>([])
const syncToCatalog = ref(true)

const hasGroup = computed(() => props.groupIds.length > 0)
const canDetect = computed(() => {
  if (props.accountId) return true
  return props.baseUrl.trim() !== '' && (props.apiKey || '').trim() !== ''
})

const allAddModelsSelected = computed(() =>
  !!diff.value?.add_models.length && selectedAddModels.value.length === diff.value.add_models.length
)
const allRemoveModelsSelected = computed(() =>
  !!diff.value?.remove_models.length && selectedRemoveModels.value.length === diff.value.remove_models.length
)
const hasSelection = computed(() => selectedAddModels.value.length > 0 || selectedRemoveModels.value.length > 0)
const detectedModelsById = computed(() => {
  const models = new Map<string, NonNullable<typeof diff.value>['models'][number]>()
  for (const model of diff.value?.models || []) {
    models.set(model.id, model)
  }
  return models
})

async function detectModels() {
  if (!canDetect.value) return
  loading.value = true
  try {
    const result = props.accountId
      ? await adminAPI.accountUpstreamModels.detect(props.accountId)
      : await adminAPI.accountUpstreamModels.preview({
        base_url: props.baseUrl,
        api_key: props.apiKey,
        platform: props.platform,
        compat_type: 'newapi',
        group_ids: props.groupIds,
        existing_mapping: props.existingMapping
    })
    diff.value = result
    selectedAddModels.value = (result.add_models.length ? result.add_models : result.models).map(model => model.id)
    selectedRemoveModels.value = []
    selectedIgnoreModels.value = []
  } catch (error: any) {
    emit('error', error?.response?.data?.detail || error?.message || '检测模型失败')
  } finally {
    loading.value = false
  }
}

function toggleAllAddModels() {
  if (!diff.value) return
  selectedAddModels.value = allAddModelsSelected.value ? [] : diff.value.add_models.map(model => model.id)
}

function toggleAllRemoveModels() {
  if (!diff.value) return
  selectedRemoveModels.value = allRemoveModelsSelected.value ? [] : [...diff.value.remove_models]
}

function ignoreSelected() {
  selectedIgnoreModels.value = Array.from(new Set([...selectedIgnoreModels.value, ...selectedAddModels.value]))
  selectedAddModels.value = []
}

async function applySelected() {
  if (!diff.value || !hasSelection.value) return
  loading.value = true
  try {
    const selected = new Set(selectedAddModels.value)
    if (props.accountId) {
      await adminAPI.accountUpstreamModels.apply(props.accountId, {
        add_models: selectedAddModels.value,
        remove_models: selectedRemoveModels.value,
        ignore_models: selectedIgnoreModels.value,
        sync_to_model_catalog: syncToCatalog.value
      })
      emit('success', '上游模型已应用到账号')
      emit('applied')
    } else if (syncToCatalog.value && selectedAddModels.value.length > 0) {
      await adminAPI.accountUpstreamModels.importCatalog({
        models: Array.from(selected)
          .map(model => detectedModelsById.value.get(model))
          .filter((model): model is NonNullable<typeof model> => !!model)
      })
      emit('success', '所选模型已同步到模型管理')
    }

    const nextMapping = { ...props.existingMapping }
    for (const model of selectedAddModels.value) {
      nextMapping[model] = model
    }
    for (const model of selectedRemoveModels.value) {
      delete nextMapping[model]
    }
    emit('mappingsUpdated', nextMapping)
  } catch (error: any) {
    emit('error', error?.response?.data?.detail || error?.message || '应用上游模型失败')
  } finally {
    loading.value = false
  }
}
</script>
