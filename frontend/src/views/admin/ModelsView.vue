<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="space-y-4">
          <div class="flex flex-col justify-between gap-3 lg:flex-row lg:items-start">
            <div class="flex flex-1 flex-wrap items-center gap-3">
              <div class="relative w-full sm:w-72">
                <Icon name="search" size="md" class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
                <input
                  v-model="searchQuery"
                  type="text"
                  class="input pl-10"
                  placeholder="搜索模型..."
                  @input="handleSearch"
                />
              </div>
              <Select v-model="filters.status" class="w-36" :options="statusOptions" @change="reload" />
              <Select v-model="filters.name_rule" class="w-40" :options="ruleOptions" @change="reload" />
            </div>

            <div class="flex flex-wrap items-center justify-end gap-2">
              <button class="btn btn-secondary" :disabled="loading" @click="reload">
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>
              <button class="btn btn-secondary" @click="openMissingDialog">
                <Icon name="search" size="md" class="mr-2" />
                缺失模型
              </button>
              <button class="btn btn-secondary" @click="openSyncDialog">
                <Icon name="download" size="md" class="mr-2" />
                同步上游
              </button>
              <button class="btn btn-secondary" @click="openVendorDialog()">
                <Icon name="cog" size="md" class="mr-2" />
                供应商
              </button>
              <button class="btn btn-primary" @click="openModelDialog()">
                <Icon name="plus" size="md" class="mr-2" />
                创建模型
              </button>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <button
              class="rounded-md px-3 py-1.5 text-sm transition-colors"
              :class="filters.vendor_id === null ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-200'"
              @click="setVendor(null)"
            >
              全部供应商
            </button>
            <button
              v-for="vendor in vendors"
              :key="vendor.id"
              class="rounded-md px-3 py-1.5 text-sm transition-colors"
              :class="filters.vendor_id === vendor.id ? 'bg-primary-600 text-white' : 'bg-gray-100 text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-200'"
              @click="setVendor(vendor.id)"
            >
              {{ vendor.icon ? `${vendor.icon} ` : '' }}{{ vendor.name }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <div v-if="selectedIds.length" class="mb-3 flex items-center justify-between rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-sm dark:border-amber-900/60 dark:bg-amber-900/20">
          <span>已选择 {{ selectedIds.length }} 个模型</span>
          <button class="btn btn-danger btn-sm" @click="handleBatchDelete">删除所选</button>
        </div>

        <DataTable :columns="columns" :data="models" :loading="loading">
          <template #cell-select="{ row }">
            <input
              type="checkbox"
              class="rounded border-gray-300 text-primary-600"
              :checked="selectedIds.includes(row.id)"
              @change="toggleSelected(row.id)"
            />
          </template>

          <template #cell-model_name="{ row }">
            <div class="min-w-0">
              <div class="font-medium text-gray-900 dark:text-white">{{ row.model_name }}</div>
              <div v-if="row.description" class="max-w-md truncate text-xs text-gray-500">{{ row.description }}</div>
            </div>
          </template>

          <template #cell-vendor="{ row }">
            <span class="text-sm text-gray-700 dark:text-gray-300">
              {{ row.vendor_name || '-' }}
            </span>
          </template>

          <template #cell-tags="{ row }">
            <div class="flex max-w-xs flex-wrap gap-1">
              <span v-for="tag in tagList(row.tags)" :key="tag" class="rounded bg-gray-100 px-1.5 py-0.5 text-xs text-gray-700 dark:bg-dark-700 dark:text-gray-300">{{ tag }}</span>
              <span v-if="!tagList(row.tags).length" class="text-xs text-gray-400">-</span>
            </div>
          </template>

          <template #cell-endpoints="{ row }">
            <span class="text-xs text-gray-600 dark:text-gray-400">{{ endpointKeys(row).join(', ') || '-' }}</span>
          </template>

          <template #cell-status="{ row }">
            <Toggle :modelValue="row.status === 'active'" @update:modelValue="toggleStatus(row)" />
          </template>

          <template #cell-bound_channels="{ row }">
            <span class="rounded bg-gray-100 px-2 py-0.5 text-xs dark:bg-dark-700">{{ row.bound_channels?.length || 0 }}</span>
          </template>

          <template #cell-enable_groups="{ row }">
            <span class="text-xs text-gray-600 dark:text-gray-400">{{ (row.enable_groups || []).slice(0, 3).join(', ') || '-' }}</span>
          </template>

          <template #cell-matched_count="{ row }">
            <span class="rounded bg-primary-100 px-2 py-0.5 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">{{ row.matched_count || 0 }}</span>
          </template>

          <template #cell-updated_at="{ value }">
            <span class="text-xs text-gray-500">{{ formatDate(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button class="rounded p-1.5 text-gray-500 hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700" @click="openModelDialog(row)">
                <Icon name="edit" size="sm" />
              </button>
              <button class="rounded p-1.5 text-gray-500 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20" @click="handleDelete(row)">
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState title="暂无模型" description="可以手动创建模型，或从上游元数据同步。" action-text="创建模型" @action="openModelDialog()" />
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog :show="showModelDialog" :title="editingModel ? '编辑模型' : '创建模型'" width="wide" @close="closeModelDialog">
      <form class="space-y-4" @submit.prevent="saveModel">
        <div class="grid gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">模型名称 <span class="text-red-500">*</span></label>
            <input v-model="modelForm.model_name" class="input" required />
          </div>
          <div>
            <label class="input-label">供应商</label>
            <select v-model.number="modelForm.vendor_id" class="input">
              <option :value="null">无</option>
              <option v-for="vendor in vendors" :key="vendor.id" :value="vendor.id">{{ vendor.name }}</option>
            </select>
          </div>
          <div>
            <label class="input-label">图标</label>
            <input v-model="modelForm.icon" class="input" placeholder="可选 emoji 或 URL" />
          </div>
          <div>
            <label class="input-label">名称规则</label>
            <Select v-model="modelForm.name_rule" :options="ruleOptionsWithoutAll" />
          </div>
          <div>
            <label class="input-label">状态</label>
            <Select v-model="modelForm.status" :options="statusOptionsWithoutAll" />
          </div>
          <label class="mt-6 flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="modelForm.sync_official" type="checkbox" class="rounded border-gray-300 text-primary-600" />
            跟随官方元数据同步
          </label>
        </div>
        <div>
          <label class="input-label">标签</label>
          <input v-model="tagsInput" class="input" placeholder="chat, vision, reasoning" />
        </div>
        <div>
          <label class="input-label">描述</label>
          <textarea v-model="modelForm.description" class="input min-h-20" />
        </div>
        <div>
          <label class="input-label">端点 JSON</label>
          <textarea v-model="endpointsInput" class="input min-h-28 font-mono text-xs" placeholder="{ &quot;chat&quot;: true }" />
        </div>
      </form>
      <template #footer>
        <button class="btn btn-secondary" @click="closeModelDialog">取消</button>
        <button class="btn btn-primary" :disabled="saving" @click="saveModel">保存</button>
      </template>
    </BaseDialog>

    <BaseDialog :show="showVendorDialog" title="模型供应商" width="wide" @close="closeVendorDialog">
      <div class="grid gap-4 lg:grid-cols-[1fr_280px]">
        <div class="space-y-2">
          <div v-for="vendor in vendors" :key="vendor.id" class="flex items-center justify-between rounded-md border border-gray-200 p-3 dark:border-dark-700">
            <div>
              <div class="font-medium text-gray-900 dark:text-white">{{ vendor.icon ? `${vendor.icon} ` : '' }}{{ vendor.name }}</div>
              <div class="text-xs text-gray-500">{{ vendor.description || '-' }}</div>
            </div>
            <div class="flex items-center gap-1">
              <button class="rounded p-1.5 text-gray-500 hover:bg-gray-100 dark:hover:bg-dark-700" @click="editVendor(vendor)"><Icon name="edit" size="sm" /></button>
              <button class="rounded p-1.5 text-gray-500 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20" @click="deleteVendor(vendor)"><Icon name="trash" size="sm" /></button>
            </div>
          </div>
        </div>
        <form class="space-y-3" @submit.prevent="saveVendor">
          <input v-model="vendorForm.name" class="input" placeholder="供应商名称" required />
          <input v-model="vendorForm.icon" class="input" placeholder="图标" />
          <textarea v-model="vendorForm.description" class="input min-h-20" placeholder="描述" />
          <Select v-model="vendorForm.status" :options="statusOptionsWithoutAll" />
          <button class="btn btn-primary w-full" :disabled="saving">{{ editingVendor ? '更新供应商' : '创建供应商' }}</button>
          <button v-if="editingVendor" type="button" class="btn btn-secondary w-full" @click="resetVendorForm">取消编辑</button>
        </form>
      </div>
    </BaseDialog>

    <BaseDialog :show="showMissingDialog" title="缺失模型" width="extra-wide" @close="showMissingDialog = false">
      <div class="max-h-[60vh] space-y-2 overflow-y-auto">
        <div v-if="missingLoading" class="py-8 text-center text-gray-500">加载中...</div>
        <div v-for="item in missingModels" :key="item.model_name" class="flex items-center justify-between rounded-md border border-gray-200 p-3 dark:border-dark-700">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">{{ item.model_name }}</div>
            <div class="text-xs text-gray-500">
              {{ ruleLabel(item.name_rule) }} · {{ item.platforms.join(', ') || '-' }} · {{ item.channels.map(channel => channel.name).join(', ') || '-' }}
            </div>
          </div>
          <span class="rounded bg-gray-100 px-2 py-0.5 text-xs dark:bg-dark-700">{{ item.matched_count }}</span>
        </div>
        <div v-if="!missingLoading && !missingModels.length" class="py-8 text-center text-gray-500">没有发现缺失模型。</div>
      </div>
    </BaseDialog>

    <BaseDialog :show="showSyncDialog" title="同步上游元数据" width="extra-wide" @close="showSyncDialog = false">
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <div class="text-sm text-gray-600 dark:text-gray-400">
            {{ syncPreviewData ? `${syncPreviewData.missing.length} 个新增模型，${syncPreviewData.conflicts.length} 个冲突` : '应用前先预览上游元数据。' }}
          </div>
          <button class="btn btn-secondary" :disabled="syncLoading" @click="loadSyncPreview">预览</button>
        </div>
        <div v-if="syncLoading" class="py-8 text-center text-gray-500">加载中...</div>
        <div v-else-if="syncPreviewData" class="grid max-h-[58vh] gap-4 overflow-y-auto lg:grid-cols-2">
          <div>
            <h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-white">新增模型</h4>
            <div class="space-y-1">
              <div v-for="model in syncPreviewData.missing" :key="model.model_name" class="rounded border border-gray-200 p-2 text-sm dark:border-dark-700">
                <div class="truncate font-medium">{{ model.model_name }}</div>
                <div class="text-xs text-gray-500">{{ model.vendor_name || '-' }} · {{ model.tags || '-' }}</div>
              </div>
            </div>
          </div>
          <div>
            <h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-white">冲突项</h4>
            <div class="space-y-1">
              <div v-for="conflict in syncPreviewData.conflicts" :key="conflict.model_name" class="rounded border border-amber-200 bg-amber-50 p-2 text-xs dark:border-amber-900/60 dark:bg-amber-900/20">
                <div class="font-medium">{{ conflict.model_name }}</div>
                <label v-for="field in conflict.fields" :key="field.field" class="mt-1 flex items-center gap-2">
                  <input type="checkbox" class="rounded border-gray-300 text-primary-600" :checked="isConflictSelected(conflict.model_name, field.field)" @change="toggleConflict(conflict.model_name, field.field)" />
                  覆盖 {{ field.field }}
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>
      <template #footer>
        <button class="btn btn-secondary" @click="showSyncDialog = false">取消</button>
        <button class="btn btn-primary" :disabled="syncLoading || !syncPreviewData" @click="applySync">应用</button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { MissingModel, ModelCatalog, ModelCatalogPayload, ModelSyncPreview } from '@/api/admin/models'
import type { ModelVendor } from '@/api/admin/modelVendors'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'

const appStore = useAppStore()

const models = ref<ModelCatalog[]>([])
const vendors = ref<ModelVendor[]>([])
const loading = ref(false)
const saving = ref(false)
const searchQuery = ref('')
const selectedIds = ref<number[]>([])
const pagination = reactive({ page: 1, page_size: getPersistedPageSize(), total: 0 })
const filters = reactive<{ status: string; name_rule: number | null; vendor_id: number | null }>({
  status: '',
  name_rule: null,
  vendor_id: null
})

const showModelDialog = ref(false)
const editingModel = ref<ModelCatalog | null>(null)
const modelForm = reactive<ModelCatalogPayload>(emptyModelForm())
const tagsInput = ref('')
const endpointsInput = ref('{}')

const showVendorDialog = ref(false)
const editingVendor = ref<ModelVendor | null>(null)
const vendorForm = reactive({ name: '', description: '', icon: '', status: 'active' })

const showMissingDialog = ref(false)
const missingLoading = ref(false)
const missingModels = ref<MissingModel[]>([])

const showSyncDialog = ref(false)
const syncLoading = ref(false)
const syncPreviewData = ref<ModelSyncPreview | null>(null)
const syncSelection = reactive<{ overwrite: Record<string, string[]> }>({
  overwrite: {}
})

let searchTimeout: ReturnType<typeof setTimeout> | null = null

const statusOptions = [
  { value: '', label: '全部状态' },
  { value: 'active', label: '启用' },
  { value: 'disabled', label: '禁用' }
]
const statusOptionsWithoutAll = statusOptions.slice(1)
const ruleOptions = [
  { value: null, label: '全部规则' },
  { value: 0, label: '精确匹配' },
  { value: 1, label: '前缀匹配' },
  { value: 2, label: '包含匹配' },
  { value: 3, label: '后缀匹配' }
]
const ruleOptionsWithoutAll = ruleOptions.slice(1)

const columns = computed<Column[]>(() => [
  { key: 'select', label: '', sortable: false },
  { key: 'model_name', label: '模型', sortable: true },
  { key: 'vendor', label: '供应商', sortable: false },
  { key: 'tags', label: '标签', sortable: false },
  { key: 'endpoints', label: '端点', sortable: false },
  { key: 'status', label: '启用', sortable: false },
  { key: 'bound_channels', label: '渠道', sortable: false },
  { key: 'enable_groups', label: '分组', sortable: false },
  { key: 'matched_count', label: '匹配数', sortable: false },
  { key: 'updated_at', label: '更新时间', sortable: true },
  { key: 'actions', label: '操作', sortable: false }
])

function emptyModelForm(): ModelCatalogPayload {
  return {
    model_name: '',
    description: '',
    icon: '',
    tags: '',
    vendor_id: null,
    endpoints: [],
    status: 'active',
    sync_official: true,
    name_rule: 0
  }
}

async function reload() {
  loading.value = true
  try {
    const res = await adminAPI.models.list(pagination.page, pagination.page_size, {
      search: searchQuery.value.trim() || undefined,
      status: filters.status || undefined,
      name_rule: filters.name_rule,
      vendor_id: filters.vendor_id,
      sort_by: 'updated_at',
      sort_order: 'desc'
    })
    models.value = res.items || []
    pagination.total = res.total
    selectedIds.value = selectedIds.value.filter(id => models.value.some(model => model.id === id))
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载模型失败'))
  } finally {
    loading.value = false
  }
}

async function loadVendors() {
  try {
    const res = await adminAPI.modelVendors.list(1, 200)
    vendors.value = res.items || []
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载供应商失败'))
  }
}

function handleSearch() {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    reload()
  }, 300)
}

function handlePageChange(page: number) {
  pagination.page = page
  reload()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  reload()
}

function setVendor(id: number | null) {
  filters.vendor_id = id
  pagination.page = 1
  reload()
}

function endpointKeys(row: ModelCatalog): string[] {
  return row.endpoints || []
}

function tagList(tags: string): string[] {
  return (tags || '').split(',').map(tag => tag.trim()).filter(Boolean)
}

function ruleLabel(rule: number): string {
  return ruleOptionsWithoutAll.find(option => option.value === rule)?.label || '精确匹配'
}

function formatDate(value: string): string {
  return value ? new Date(value).toLocaleString() : '-'
}

function toggleSelected(id: number) {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter(existing => existing !== id)
    : [...selectedIds.value, id]
}

function openModelDialog(model?: ModelCatalog) {
  editingModel.value = model || null
  Object.assign(modelForm, emptyModelForm(), model ? {
    model_name: model.model_name,
    description: model.description,
    icon: model.icon,
    tags: model.tags || '',
    vendor_id: model.vendor_id,
    endpoints: model.endpoints || {},
    status: model.status,
    sync_official: model.sync_official,
    name_rule: model.name_rule
  } : {})
  tagsInput.value = modelForm.tags || ''
  endpointsInput.value = JSON.stringify(modelForm.endpoints || {}, null, 2)
  showModelDialog.value = true
}

function closeModelDialog() {
  showModelDialog.value = false
  editingModel.value = null
}

async function saveModel() {
  if (!modelForm.model_name.trim()) {
    appStore.showError('模型名称不能为空')
    return
  }
  let endpoints: string[]
  try {
    const parsed = endpointsInput.value.trim() ? JSON.parse(endpointsInput.value) : []
    endpoints = Array.isArray(parsed) ? parsed.map(String) : Object.keys(parsed || {})
  } catch {
    appStore.showError('端点必须是合法 JSON')
    return
  }
  saving.value = true
  try {
    const payload: ModelCatalogPayload = {
      ...modelForm,
      model_name: modelForm.model_name.trim(),
      tags: tagsInput.value.split(',').map(tag => tag.trim()).filter(Boolean).join(','),
      endpoints
    }
    if (editingModel.value) {
      await adminAPI.models.update(editingModel.value.id, payload)
      appStore.showSuccess('模型已更新')
    } else {
      await adminAPI.models.create(payload)
      appStore.showSuccess('模型已创建')
    }
    closeModelDialog()
    reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存模型失败'))
  } finally {
    saving.value = false
  }
}

async function toggleStatus(row: ModelCatalog) {
  const next = row.status === 'active' ? 'disabled' : 'active'
  try {
    await adminAPI.models.updateStatus(row.id, next)
    row.status = next
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '更新状态失败'))
  }
}

async function handleDelete(row: ModelCatalog) {
  if (!window.confirm(`确定删除模型 ${row.model_name} 吗？`)) return
  try {
    await adminAPI.models.remove(row.id)
    appStore.showSuccess('模型已删除')
    reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除模型失败'))
  }
}

async function handleBatchDelete() {
  if (!selectedIds.value.length || !window.confirm(`确定删除已选择的 ${selectedIds.value.length} 个模型吗？`)) return
  try {
    await adminAPI.models.batchDelete(selectedIds.value)
    selectedIds.value = []
    appStore.showSuccess('模型已删除')
    reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '批量删除模型失败'))
  }
}

function openVendorDialog() {
  resetVendorForm()
  showVendorDialog.value = true
}

function closeVendorDialog() {
  showVendorDialog.value = false
  resetVendorForm()
}

function resetVendorForm() {
  editingVendor.value = null
  Object.assign(vendorForm, { name: '', description: '', icon: '', status: 'active' })
}

function editVendor(vendor: ModelVendor) {
  editingVendor.value = vendor
  Object.assign(vendorForm, {
    name: vendor.name,
    description: vendor.description || '',
    icon: vendor.icon || '',
    status: vendor.status || 'active'
  })
}

async function saveVendor() {
  saving.value = true
  try {
    if (editingVendor.value) {
      await adminAPI.modelVendors.update(editingVendor.value.id, vendorForm)
      appStore.showSuccess('供应商已更新')
    } else {
      await adminAPI.modelVendors.create(vendorForm)
      appStore.showSuccess('供应商已创建')
    }
    resetVendorForm()
    await loadVendors()
    reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '保存供应商失败'))
  } finally {
    saving.value = false
  }
}

async function deleteVendor(vendor: ModelVendor) {
  if (!window.confirm(`确定删除供应商 ${vendor.name} 吗？模型会继续可用，但不再关联该供应商。`)) return
  try {
    await adminAPI.modelVendors.remove(vendor.id)
    appStore.showSuccess('供应商已删除')
    await loadVendors()
    reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '删除供应商失败'))
  }
}

async function openMissingDialog() {
  showMissingDialog.value = true
  missingLoading.value = true
  try {
    missingModels.value = await adminAPI.models.missing()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '加载缺失模型失败'))
  } finally {
    missingLoading.value = false
  }
}

function openSyncDialog() {
  showSyncDialog.value = true
  if (!syncPreviewData.value) loadSyncPreview()
}

async function loadSyncPreview() {
  syncLoading.value = true
  try {
    const preview = await adminAPI.models.syncPreview('zh-CN')
    syncPreviewData.value = preview
    syncSelection.overwrite = {}
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '预览上游同步失败'))
  } finally {
    syncLoading.value = false
  }
}

function isConflictSelected(modelName: string, field: string): boolean {
  return (syncSelection.overwrite[modelName] || []).includes(field)
}

function toggleConflict(modelName: string, field: string) {
  const fields = syncSelection.overwrite[modelName] || []
  syncSelection.overwrite[modelName] = fields.includes(field)
    ? fields.filter(existing => existing !== field)
    : [...fields, field]
}

async function applySync() {
  syncLoading.value = true
  try {
    const result = await adminAPI.models.syncUpstream({
      locale: 'zh-CN',
      overwrite: Object.entries(syncSelection.overwrite).map(([model_name, fields]) => ({ model_name, fields }))
    })
    appStore.showSuccess(`已创建 ${result.created_models} 个模型，已更新 ${result.updated_models} 个模型`)
    showSyncDialog.value = false
    syncPreviewData.value = null
    await loadVendors()
    reload()
  } catch (err) {
    appStore.showError(extractApiErrorMessage(err, '同步上游失败'))
  } finally {
    syncLoading.value = false
  }
}

onMounted(async () => {
  await loadVendors()
  await reload()
})
</script>
