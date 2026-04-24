<template>
  <div>
    <div class="relative mb-3">
      <div
        @click="toggleDropdown"
        class="cursor-pointer rounded-lg border border-gray-300 bg-white px-3 py-2 dark:border-dark-500 dark:bg-dark-700"
      >
        <div class="grid grid-cols-2 gap-1.5">
          <span
            v-for="model in modelValue"
            :key="model"
            class="inline-flex items-center justify-between gap-1 rounded bg-gray-100 px-2 py-1 text-xs text-gray-700 dark:bg-dark-600 dark:text-gray-300"
          >
            <span class="flex items-center gap-1 truncate">
              <ModelIcon :model="model" size="14px" />
              <span class="truncate">{{ model }}</span>
            </span>
            <button
              type="button"
              @click.stop="removeModel(model)"
              class="shrink-0 rounded-full hover:bg-gray-200 dark:hover:bg-dark-500"
            >
              <Icon name="x" size="xs" class="h-3.5 w-3.5" :stroke-width="2" />
            </button>
          </span>
        </div>
        <div class="mt-2 flex items-center justify-between border-t border-gray-200 pt-2 dark:border-dark-600">
          <span class="text-xs text-gray-400">{{ t('admin.accounts.modelCount', { count: modelValue.length }) }}</span>
          <svg class="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>
      <div
        v-if="showDropdown"
        class="absolute left-0 right-0 top-full z-50 mt-1 rounded-lg border border-gray-200 bg-white shadow-lg dark:border-dark-600 dark:bg-dark-700"
      >
        <div class="sticky top-0 border-b border-gray-200 bg-white p-2 dark:border-dark-600 dark:bg-dark-700">
          <input
            v-model="searchQuery"
            type="text"
            class="input w-full text-sm"
            :placeholder="t('admin.accounts.searchModels')"
            @click.stop
          />
        </div>
        <div class="max-h-52 overflow-auto">
          <button
            v-for="model in filteredModels"
            :key="model.value"
            type="button"
            @click="toggleModel(model.value)"
            class="flex w-full items-center gap-2 px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-600"
          >
            <span
              :class="[
                'flex h-4 w-4 shrink-0 items-center justify-center rounded border',
                modelValue.includes(model.value)
                  ? 'border-primary-500 bg-primary-500 text-white'
                  : 'border-gray-300 dark:border-dark-500'
              ]"
            >
              <svg v-if="modelValue.includes(model.value)" class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="3" d="M5 13l4 4L19 7" />
              </svg>
            </span>
            <ModelIcon :model="model.value" size="18px" />
            <span class="truncate text-gray-900 dark:text-white">{{ model.value }}</span>
          </button>
          <div v-if="filteredModels.length === 0" class="px-3 py-4 text-center text-sm text-gray-500">
            {{ t('admin.accounts.noMatchingModels') }}
          </div>
        </div>
      </div>
    </div>

    <div class="mb-4 flex flex-wrap gap-2">
      <button
        v-if="canFetchModels"
        type="button"
        @click="handleFetchModels"
        :disabled="isFetchingModels"
        class="rounded-lg border border-emerald-200 px-3 py-1.5 text-sm text-emerald-600 hover:bg-emerald-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-emerald-800 dark:text-emerald-400 dark:hover:bg-emerald-900/30"
      >
        {{ isFetchingModels ? t('admin.accounts.fetchModelsLoading') : t('admin.accounts.fetchModels') }}
      </button>
      <button
        type="button"
        @click="fillRelated"
        class="rounded-lg border border-blue-200 px-3 py-1.5 text-sm text-blue-600 hover:bg-blue-50 dark:border-blue-800 dark:text-blue-400 dark:hover:bg-blue-900/30"
      >
        {{ t('admin.accounts.fillRelatedModels') }}
      </button>
      <button
        type="button"
        @click="clearAll"
        class="rounded-lg border border-red-200 px-3 py-1.5 text-sm text-red-600 hover:bg-red-50 dark:border-red-800 dark:text-red-400 dark:hover:bg-red-900/30"
      >
        {{ t('admin.accounts.clearAllModels') }}
      </button>
    </div>

    <div class="mb-3">
      <label class="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accounts.customModelName') }}</label>
      <div class="flex gap-2">
        <input
          v-model="customModel"
          type="text"
          class="input flex-1"
          :placeholder="t('admin.accounts.enterCustomModelName')"
          @keydown.enter.prevent="handleEnter"
          @compositionstart="isComposing = true"
          @compositionend="isComposing = false"
        />
        <button
          type="button"
          @click="addCustom"
          class="rounded-lg bg-primary-50 px-4 py-2 text-sm font-medium text-primary-600 hover:bg-primary-100 dark:bg-primary-900/30 dark:text-primary-400 dark:hover:bg-primary-900/50"
        >
          {{ t('admin.accounts.addModel') }}
        </button>
      </div>
    </div>

    <BaseDialog
      :show="showFetchDialog"
      :title="t('admin.accounts.fetchModelsDialogTitle')"
      width="wide"
      @close="closeFetchDialog"
    >
      <div class="space-y-4">
        <div v-if="fetchDialogHintKey" :class="fetchDialogHintClasses">
          {{ t(fetchDialogHintKey) }}
        </div>

        <div>
          <input
            v-model="fetchDialogSearch"
            type="text"
            class="input"
            :placeholder="t('admin.accounts.fetchModelsSearch')"
          />
        </div>

        <div class="space-y-4">
          <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800">
            <button
              v-for="tab in dialogTabs"
              :key="tab.key"
              type="button"
              :class="[
                'rounded-md px-3 py-1.5 text-sm transition-colors',
                activeFetchTab === tab.key
                  ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
                  : 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'
              ]"
              @click="activeFetchTab = tab.key"
            >
              {{ t(tab.titleKey) }} ({{ tab.count }})
            </button>
          </div>

          <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
            <div v-if="activeDialogSection.groups.length > 0" class="mb-4 flex items-center justify-between rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-800">
              <label class="flex cursor-pointer items-center gap-3 text-sm text-gray-700 dark:text-gray-200">
                <input
                  :checked="areAllSelected(activeSectionModels)"
                  :indeterminate.prop="isProviderPartiallySelected(activeSectionModels)"
                  type="checkbox"
                  class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
                  @change="toggleProviderSelection(activeSectionModels, ($event.target as HTMLInputElement).checked)"
                />
                <span>{{ t('admin.accounts.fetchModelsSelectAllProviders') }}</span>
              </label>
              <span class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.accounts.fetchModelsProviderCount', { count: activeDialogSection.groups.length }) }}
              </span>
            </div>

            <div v-if="activeDialogSection.groups.length === 0" class="rounded-lg bg-gray-50 px-3 py-4 text-sm text-gray-500 dark:bg-dark-700 dark:text-gray-400">
              {{ t('admin.accounts.fetchModelsEmptySection') }}
            </div>

            <div v-else class="space-y-3">
              <div
                v-for="group in activeDialogSection.groups"
                :key="`${activeDialogSection.key}-${group.provider}`"
                class="overflow-hidden rounded-lg border border-gray-100 dark:border-dark-700"
              >
                <button
                  type="button"
                  class="flex w-full items-center justify-between gap-3 bg-gray-50 px-3 py-3 text-left hover:bg-gray-100 dark:bg-dark-800 dark:hover:bg-dark-700"
                  @click="toggleProviderCollapse(activeDialogSection.key, group.provider)"
                >
                  <div class="flex min-w-0 items-center gap-3">
                    <input
                      :checked="areAllSelected(group.models)"
                      :indeterminate.prop="isProviderPartiallySelected(group.models)"
                      type="checkbox"
                      class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
                      @click.stop
                      @change="toggleProviderSelection(group.models, ($event.target as HTMLInputElement).checked)"
                    />
                    <div class="min-w-0">
                      <div class="truncate text-sm font-medium text-gray-900 dark:text-white">{{ group.provider }}</div>
                      <div class="text-xs text-gray-500 dark:text-gray-400">
                        {{ t('admin.accounts.fetchModelsProviderCount', { count: group.models.length }) }}
                      </div>
                    </div>
                  </div>
                  <Icon
                    :name="isProviderCollapsed(activeDialogSection.key, group.provider) ? 'chevronDown' : 'chevronUp'"
                    size="sm"
                    class="shrink-0 text-gray-400"
                    :stroke-width="2"
                  />
                </button>

                <div v-if="!isProviderCollapsed(activeDialogSection.key, group.provider)" class="space-y-2 p-3">
                  <label
                    v-for="model in group.models"
                    :key="`${activeDialogSection.key}-${model.value}`"
                    class="flex cursor-pointer items-start gap-3 rounded-lg px-2 py-2 hover:bg-gray-50 dark:hover:bg-dark-600"
                  >
                    <input
                      :checked="selectedFetchedModelIds.includes(model.value)"
                      type="checkbox"
                      class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500 dark:border-dark-500"
                      @change="toggleFetchedModel(model.value)"
                    />
                    <div class="min-w-0 flex-1">
                      <div class="flex items-center gap-2 text-sm text-gray-900 dark:text-white">
                        <ModelIcon :model="model.value" size="16px" />
                        <span class="truncate">{{ model.value }}</span>
                      </div>
                      <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        {{ model.label }}
                      </div>
                    </div>
                  </label>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeFetchDialog">
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="btn btn-primary"
            :disabled="selectedFetchedModelIds.length === 0"
            @click="appendFetchedModels"
          >
            {{ t('admin.accounts.fetchModelsAppend', { count: selectedFetchedModelIds.length }) }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import type { PreviewAccountModel, PreviewAccountModelsRequest } from '@/api/admin/accounts'
import { previewAvailableModels } from '@/api/admin/accounts'
import {
  allModels,
  createModelCatalogItem,
  getModelCatalogByPlatform,
  getModelsByPlatform,
  type ModelCatalogItem
} from '@/composables/useModelWhitelist'

const { t } = useI18n()

const props = defineProps<{
  modelValue: string[]
  platform?: string
  platforms?: string[]
  fetchRequest?: PreviewAccountModelsRequest | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
}>()

const appStore = useAppStore()

const showDropdown = ref(false)
const searchQuery = ref('')
const customModel = ref('')
const isComposing = ref(false)
const showFetchDialog = ref(false)
const isFetchingModels = ref(false)
const fetchDialogSearch = ref('')
const fetchedCatalog = ref<ModelCatalogItem[]>([])
const selectedFetchedModelIds = ref<string[]>([])
const fetchDialogHintKey = ref('')
const fetchDialogHintClasses = ref('')
const activeFetchTab = ref<'new' | 'existing'>('new')
const collapsedProviders = ref<Record<string, boolean>>({})

type FetchDialogTabKey = 'new' | 'existing'

type FetchDialogSection = {
  key: FetchDialogTabKey
  titleKey: string
  count: number
  groups: Array<{
    provider: string
    models: ModelCatalogItem[]
  }>
}

const normalizedPlatforms = computed(() => {
  const rawPlatforms =
    props.platforms && props.platforms.length > 0
      ? props.platforms
      : props.platform
        ? [props.platform]
        : []

  return Array.from(
    new Set(
      rawPlatforms
        .map(platform => platform?.trim())
        .filter((platform): platform is string => Boolean(platform))
    )
  )
})

const primaryPlatform = computed(() => normalizedPlatforms.value[0] || '')
const canFetchModels = computed(() => normalizedPlatforms.value.length === 1 && Boolean(props.fetchRequest))

const availableOptions = computed(() => {
  if (normalizedPlatforms.value.length === 0) {
    return allModels
  }

  const allowedModels = new Set<string>()
  for (const platform of normalizedPlatforms.value) {
    for (const model of getModelsByPlatform(platform)) {
      allowedModels.add(model)
    }
  }

  return allModels.filter(model => allowedModels.has(model.value))
})

const filteredModels = computed(() => {
  const query = searchQuery.value.toLowerCase().trim()
  if (!query) return availableOptions.value
  return availableOptions.value.filter(
    m => m.value.toLowerCase().includes(query) || m.label.toLowerCase().includes(query)
  )
})

const searchedFetchedCatalog = computed(() => {
  const query = fetchDialogSearch.value.trim().toLowerCase()
  if (!query) {
    return fetchedCatalog.value
  }
  return fetchedCatalog.value.filter(model =>
    model.value.toLowerCase().includes(query)
    || model.label.toLowerCase().includes(query)
    || model.provider.toLowerCase().includes(query)
  )
})

const existingModelCatalog = computed<ModelCatalogItem[]>(() => {
  const unique = new Map<string, ModelCatalogItem>()
  for (const model of props.modelValue) {
    const fetched = fetchedCatalog.value.find(item => item.value === model)
    unique.set(model, fetched || createModelCatalogItem(model, { platformHint: primaryPlatform.value, source: 'selected' }))
  }
  return Array.from(unique.values()).sort((a, b) => a.value.localeCompare(b.value))
})

function groupByProvider(models: ModelCatalogItem[]) {
  const grouped = new Map<string, ModelCatalogItem[]>()
  for (const model of models) {
    const provider = model.provider || t('admin.accounts.fetchModelsProviderOther')
    if (!grouped.has(provider)) {
      grouped.set(provider, [])
    }
    grouped.get(provider)!.push(model)
  }
  return Array.from(grouped.entries())
    .map(([provider, providerModels]) => ({
      provider,
      models: [...providerModels].sort((a, b) => a.value.localeCompare(b.value))
    }))
    .sort((a, b) => a.provider.localeCompare(b.provider))
}

const dialogSections = computed<FetchDialogSection[]>(() => {
  const selectedValues = new Set(props.modelValue)
  const newModels = searchedFetchedCatalog.value.filter(model => !selectedValues.has(model.value))
  const existingModels = existingModelCatalog.value.filter(model => {
    const query = fetchDialogSearch.value.trim().toLowerCase()
    if (!query) return true
    return model.value.toLowerCase().includes(query)
      || model.label.toLowerCase().includes(query)
      || model.provider.toLowerCase().includes(query)
  })

  return [
    {
      key: 'new',
      titleKey: 'admin.accounts.fetchModelsNewGroup',
      count: newModels.length,
      groups: groupByProvider(newModels)
    },
    {
      key: 'existing',
      titleKey: 'admin.accounts.fetchModelsExistingGroup',
      count: existingModels.length,
      groups: groupByProvider(existingModels)
    }
  ]
})

const dialogTabs = computed<FetchDialogSection[]>(() => dialogSections.value.filter(section => section.count > 0))

const activeDialogSection = computed(() => {
  const matched = dialogSections.value.find(section => section.key === activeFetchTab.value)
  return matched || dialogTabs.value[0] || dialogSections.value[0]
})

const activeSectionModels = computed(() => activeDialogSection.value.groups.flatMap(group => group.models))

const toggleDropdown = () => {
  showDropdown.value = !showDropdown.value
  if (!showDropdown.value) searchQuery.value = ''
}

const removeModel = (model: string) => {
  emit('update:modelValue', props.modelValue.filter(m => m !== model))
}

const toggleModel = (model: string) => {
  if (props.modelValue.includes(model)) {
    removeModel(model)
  } else {
    emit('update:modelValue', [...props.modelValue, model])
  }
}

const addCustom = () => {
  const model = customModel.value.trim()
  if (!model) return
  if (props.modelValue.includes(model)) {
    appStore.showInfo(t('admin.accounts.modelExists'))
    return
  }
  emit('update:modelValue', [...props.modelValue, model])
  customModel.value = ''
}

const handleEnter = () => {
  if (!isComposing.value) addCustom()
}

const fillRelated = () => {
  const newModels = [...props.modelValue]
  for (const platform of normalizedPlatforms.value) {
    for (const model of getModelsByPlatform(platform)) {
      if (!newModels.includes(model)) {
        newModels.push(model)
      }
    }
  }
  emit('update:modelValue', newModels)
}

const clearAll = () => {
  emit('update:modelValue', [])
}

const mapPreviewModel = (model: PreviewAccountModel): ModelCatalogItem => createModelCatalogItem(model.id, {
  label: model.display_name || model.id,
  platformHint: primaryPlatform.value,
  source: model.source === 'static' ? 'static' : 'remote',
  provider: model.provider
})

const openFetchDialog = (models: ModelCatalogItem[]) => {
  fetchedCatalog.value = dedupeCatalog(models)
  selectedFetchedModelIds.value = []
  fetchDialogSearch.value = ''
  collapsedProviders.value = {}
  activeFetchTab.value = dialogSections.value[0]?.count === 0 && dialogSections.value[1]?.count > 0 ? 'existing' : 'new'
  showFetchDialog.value = true
}

const dedupeCatalog = (models: ModelCatalogItem[]) => {
  const unique = new Map<string, ModelCatalogItem>()
  for (const model of models) {
    if (!unique.has(model.value)) {
      unique.set(model.value, model)
    }
  }
  return Array.from(unique.values()).sort((a, b) => a.value.localeCompare(b.value))
}

const handleFetchModels = async () => {
  if (!props.fetchRequest || !primaryPlatform.value) {
    return
  }

  isFetchingModels.value = true
  try {
    const models = await previewAvailableModels(props.fetchRequest)
    openFetchDialog(models.map(mapPreviewModel))
    fetchDialogHintKey.value = primaryPlatform.value === 'antigravity'
      ? 'admin.accounts.fetchModelsHintStatic'
      : 'admin.accounts.fetchModelsHintRemote'
    fetchDialogHintClasses.value = 'rounded-lg bg-emerald-50 px-3 py-2 text-sm text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-300'
    appStore.showSuccess(t('admin.accounts.fetchModelsSuccess', { count: models.length }))
  } catch {
    const fallbackModels = getModelCatalogByPlatform(primaryPlatform.value, 'local')
    openFetchDialog(fallbackModels)
    fetchDialogHintKey.value = 'admin.accounts.fetchModelsHintFallback'
    fetchDialogHintClasses.value = 'rounded-lg bg-amber-50 px-3 py-2 text-sm text-amber-700 dark:bg-amber-900/20 dark:text-amber-300'
    appStore.showWarning(t('admin.accounts.fetchModelsFallbackWarning'))
  } finally {
    isFetchingModels.value = false
  }
}

const closeFetchDialog = () => {
  showFetchDialog.value = false
}

const toggleFetchedModel = (model: string) => {
  if (selectedFetchedModelIds.value.includes(model)) {
    selectedFetchedModelIds.value = selectedFetchedModelIds.value.filter(item => item !== model)
    return
  }
  selectedFetchedModelIds.value = [...selectedFetchedModelIds.value, model]
}

const areAllSelected = (models: ModelCatalogItem[]) => models.length > 0 && models.every(model => selectedFetchedModelIds.value.includes(model.value))

const isProviderPartiallySelected = (models: ModelCatalogItem[]) => {
  const selectedCount = models.filter(model => selectedFetchedModelIds.value.includes(model.value)).length
  return selectedCount > 0 && selectedCount < models.length
}

const getProviderCollapseKey = (sectionKey: string, provider: string) => `${sectionKey}:${provider}`

const isProviderCollapsed = (sectionKey: string, provider: string) => collapsedProviders.value[getProviderCollapseKey(sectionKey, provider)] ?? true

const toggleProviderCollapse = (sectionKey: string, provider: string) => {
  const key = getProviderCollapseKey(sectionKey, provider)
  collapsedProviders.value = {
    ...collapsedProviders.value,
    [key]: !isProviderCollapsed(sectionKey, provider)
  }
}

const toggleProviderSelection = (models: ModelCatalogItem[], checked: boolean) => {
  const values = models.map(model => model.value)
  if (!checked) {
    selectedFetchedModelIds.value = selectedFetchedModelIds.value.filter(model => !values.includes(model))
    return
  }
  selectedFetchedModelIds.value = Array.from(new Set([...selectedFetchedModelIds.value, ...values]))
}

const appendFetchedModels = () => {
  emit('update:modelValue', Array.from(new Set([...props.modelValue, ...selectedFetchedModelIds.value])))
  closeFetchDialog()
}
</script>
