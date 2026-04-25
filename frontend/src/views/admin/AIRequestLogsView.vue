<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="card p-4">
        <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
          <div>
            <h1 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ t('admin.aiRequestLogs.title') }}</h1>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.aiRequestLogs.description') }}</p>
          </div>
          <button class="btn btn-primary" :disabled="loading" @click="loadLogs">{{ t('common.refresh') }}</button>
        </div>

        <div class="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.query') }}</div>
            <input v-model="filters.q" class="input" :placeholder="t('admin.aiRequestLogs.filters.query')" @keyup.enter="applyFilters" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.requestId') }}</div>
            <input v-model="filters.request_id" class="input" :placeholder="t('admin.aiRequestLogs.filters.requestId')" @keyup.enter="applyFilters" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.clientRequestId') }}</div>
            <input v-model="filters.client_request_id" class="input" :placeholder="t('admin.aiRequestLogs.filters.clientRequestId')" @keyup.enter="applyFilters" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.model') }}</div>
            <input v-model="filters.model" class="input" :placeholder="t('admin.aiRequestLogs.filters.model')" @keyup.enter="applyFilters" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.userId') }}</div>
            <input v-model.number="filters.user_id" type="number" min="1" class="input" :placeholder="t('admin.aiRequestLogs.filters.userId')" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.apiKeyId') }}</div>
            <input v-model.number="filters.api_key_id" type="number" min="1" class="input" :placeholder="t('admin.aiRequestLogs.filters.apiKeyId')" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.accountId') }}</div>
            <input v-model.number="filters.account_id" type="number" min="1" class="input" :placeholder="t('admin.aiRequestLogs.filters.accountId')" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.groupId') }}</div>
            <input v-model.number="filters.group_id" type="number" min="1" class="input" :placeholder="t('admin.aiRequestLogs.filters.groupId')" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.platform') }}</div>
            <select v-model="filters.platform" class="input">
              <option value="">{{ t('admin.aiRequestLogs.filters.allPlatforms') }}</option>
              <option value="anthropic">Anthropic</option>
              <option value="openai">OpenAI</option>
              <option value="gemini">Gemini</option>
              <option value="antigravity">Antigravity</option>
            </select>
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.statusCode') }}</div>
            <input v-model.number="filters.status_code" type="number" min="0" class="input" :placeholder="t('admin.aiRequestLogs.filters.statusCode')" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.startTime') }}</div>
            <input v-model="filters.start_time" type="datetime-local" class="input" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.filters.endTime') }}</div>
            <input v-model="filters.end_time" type="datetime-local" class="input" />
          </label>
        </div>

        <div class="mt-4 flex flex-wrap gap-3">
          <button class="btn btn-primary" @click="applyFilters">{{ t('common.search') }}</button>
          <button class="btn btn-secondary" @click="resetFilters">{{ t('common.reset') }}</button>
        </div>
      </section>

      <section class="card p-4">
        <div class="mb-4 flex items-center justify-between gap-3">
          <h2 class="text-base font-semibold text-gray-900 dark:text-gray-100">{{ t('admin.aiRequestLogs.retention.title') }}</h2>
          <button class="btn btn-primary" :disabled="savingRetention" @click="saveRetention">{{ t('common.save') }}</button>
        </div>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-4">
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.retention.enabled') }}</div>
            <div class="flex items-center gap-2 rounded-lg border border-gray-200 px-3 py-2 dark:border-dark-600">
              <input v-model="retention.enabled" type="checkbox" />
              <span class="text-sm text-gray-700 dark:text-gray-300">{{ t('common.enabled') }}</span>
            </div>
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.retention.retentionHours') }}</div>
            <input v-model.number="retention.retention_hours" type="number" min="6" max="168" class="input" :placeholder="t('admin.aiRequestLogs.retention.retentionHours')" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.retention.cleanupInterval') }}</div>
            <input v-model.number="retention.cleanup_interval_minutes" type="number" min="5" max="180" class="input" :placeholder="t('admin.aiRequestLogs.retention.cleanupInterval')" />
          </label>
          <label class="space-y-1">
            <div class="input-label">{{ t('admin.aiRequestLogs.retention.batchSize') }}</div>
            <input v-model.number="retention.delete_batch_size" type="number" min="100" max="10000" class="input" :placeholder="t('admin.aiRequestLogs.retention.batchSize')" />
          </label>
        </div>
      </section>

      <section class="card overflow-hidden">
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.table.time') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.table.platform') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.table.model') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.table.status') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.table.requestId') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.table.duration') }}</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('common.actions') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="item in logs" :key="item.id" class="hover:bg-gray-50 dark:hover:bg-dark-800/70">
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ formatTime(item.created_at) }}</td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ item.platform || '-' }}</td>
                <td class="max-w-[280px] truncate px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ item.model || '-' }}</td>
                <td class="px-4 py-3 text-sm" :class="item.status_code >= 400 ? 'text-red-500' : 'text-emerald-500'">{{ item.status_code }}</td>
                <td class="max-w-[320px] px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
                  <div class="truncate font-mono" :title="item.request_id || '-'">{{ item.request_id || '-' }}</div>
                  <div v-if="item.client_request_id" class="truncate text-xs text-gray-500" :title="item.client_request_id">{{ item.client_request_id }}</div>
                </td>
                <td class="px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ item.duration_ms ?? '-' }}</td>
                <td class="px-4 py-3 text-right text-sm">
                  <button class="btn btn-secondary btn-sm" @click="openDetail(item.id)">{{ t('common.view') }}</button>
                </td>
              </tr>
              <tr v-if="!loading && logs.length === 0">
                <td colspan="7" class="px-4 py-8 text-center text-sm text-gray-500">{{ t('admin.aiRequestLogs.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="p-4">
          <Pagination
            v-if="pagination.total > 0"
            :page="pagination.page"
            :total="pagination.total"
            :page-size="pagination.page_size"
            @update:page="handlePageChange"
            @update:pageSize="handlePageSizeChange"
          />
        </div>
      </section>
    </div>

    <div v-if="detailVisible" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4" @click.self="closeDetail">
      <div class="max-h-[90vh] w-full max-w-6xl overflow-hidden rounded-2xl bg-white shadow-2xl dark:bg-dark-900">
        <div class="flex items-center justify-between border-b border-gray-200 px-5 py-4 dark:border-dark-700">
          <div>
            <div class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ t('admin.aiRequestLogs.detail.title') }}</div>
            <div class="mt-1 text-sm text-gray-500">{{ selectedLog?.request_id || selectedLog?.client_request_id || '#' + selectedLog?.id }}</div>
          </div>
          <button class="btn btn-secondary" @click="closeDetail">{{ t('common.close') }}</button>
        </div>

        <div v-if="selectedLog" class="grid max-h-[calc(90vh-80px)] grid-cols-1 gap-4 overflow-auto p-5 xl:grid-cols-2">
          <section class="xl:col-span-2">
            <div class="grid grid-cols-1 gap-3 rounded-xl border border-gray-200 p-4 dark:border-dark-700 md:grid-cols-2 xl:grid-cols-4">
              <div>
                <div class="text-xs uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.meta.platform') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{{ selectedLog.platform || '-' }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.meta.model') }}</div>
                <div class="mt-1 break-all text-sm font-medium text-gray-900 dark:text-gray-100">{{ selectedLog.model || '-' }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.meta.status') }}</div>
                <div class="mt-1 text-sm font-medium" :class="selectedLog.status_code >= 400 ? 'text-red-500' : 'text-emerald-500'">{{ selectedLog.status_code }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.meta.duration') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{{ selectedLog.duration_ms ?? '-' }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.meta.userId') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{{ selectedLog.user_id ?? '-' }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.meta.apiKeyId') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{{ selectedLog.api_key_id ?? '-' }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.meta.accountId') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{{ selectedLog.account_id ?? '-' }}</div>
              </div>
              <div>
                <div class="text-xs uppercase tracking-wider text-gray-500">{{ t('admin.aiRequestLogs.meta.groupId') }}</div>
                <div class="mt-1 text-sm font-medium text-gray-900 dark:text-gray-100">{{ selectedLog.group_id ?? '-' }}</div>
              </div>
            </div>
          </section>

          <section class="space-y-2">
            <div class="flex items-center justify-between gap-3">
              <h3 class="font-medium text-gray-900 dark:text-gray-100">{{ t('admin.aiRequestLogs.detail.requestBody') }}</h3>
              <div class="flex gap-2">
                <button class="btn btn-secondary btn-sm" @click="toggleSection('request')">{{ expanded.request ? t('admin.aiRequestLogs.detail.collapse') : t('admin.aiRequestLogs.detail.expand') }}</button>
                <button class="btn btn-secondary btn-sm" @click="copyText(selectedLog.request_body)">{{ t('common.copy') }}</button>
              </div>
            </div>
            <div class="rounded-xl bg-gray-950 p-4 text-xs text-gray-100">
              <pre class="overflow-auto" :class="expanded.request ? 'max-h-[60vh]' : 'max-h-60'">{{ prettyJson(selectedLog.request_body || '{}') }}</pre>
            </div>
          </section>

          <section class="space-y-2">
            <div class="flex items-center justify-between gap-3">
              <h3 class="font-medium text-gray-900 dark:text-gray-100">{{ t('admin.aiRequestLogs.detail.responseBody') }}</h3>
              <div class="flex gap-2">
                <button class="btn btn-secondary btn-sm" @click="toggleSection('response')">{{ expanded.response ? t('admin.aiRequestLogs.detail.collapse') : t('admin.aiRequestLogs.detail.expand') }}</button>
                <button class="btn btn-secondary btn-sm" @click="copyText(selectedLog.response_body)">{{ t('common.copy') }}</button>
              </div>
            </div>
            <div class="rounded-xl bg-gray-950 p-4 text-xs text-gray-100">
              <pre class="overflow-auto" :class="expanded.response ? 'max-h-[60vh]' : 'max-h-60'">{{ prettyJson(selectedLog.response_body || '{}') }}</pre>
            </div>
          </section>

          <section v-if="selectedLog.error_message" class="space-y-2 xl:col-span-2">
            <div class="flex items-center justify-between gap-3">
              <h3 class="font-medium text-gray-900 dark:text-gray-100">{{ t('admin.aiRequestLogs.detail.errorMessage') }}</h3>
              <div class="flex gap-2">
                <button class="btn btn-secondary btn-sm" @click="toggleSection('error')">{{ expanded.error ? t('admin.aiRequestLogs.detail.collapse') : t('admin.aiRequestLogs.detail.expand') }}</button>
                <button class="btn btn-secondary btn-sm" @click="copyText(selectedLog.error_message)">{{ t('common.copy') }}</button>
              </div>
            </div>
            <div class="rounded-xl border border-red-200 bg-red-50 p-4 text-xs text-red-900 dark:border-red-900/40 dark:bg-red-950/30 dark:text-red-100">
              <pre class="overflow-auto whitespace-pre-wrap break-all" :class="expanded.error ? 'max-h-80' : 'max-h-24'">{{ selectedLog.error_message }}</pre>
            </div>
          </section>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import {
  type AIRequestLog,
  type AIRequestLogRetentionSettings,
  getAIRequestLogByID,
  getAIRequestLogRetentionSettings,
  listAIRequestLogs,
  updateAIRequestLogRetentionSettings,
} from '@/api/admin/aiRequestLogs'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const savingRetention = ref(false)
const logs = ref<AIRequestLog[]>([])
const selectedLog = ref<AIRequestLog | null>(null)
const detailVisible = ref(false)
const pagination = reactive({ page: 1, page_size: 20, total: 0 })
const filters = reactive({
  q: '',
  request_id: '',
  client_request_id: '',
  platform: '',
  model: '',
  user_id: undefined as number | undefined,
  api_key_id: undefined as number | undefined,
  account_id: undefined as number | undefined,
  group_id: undefined as number | undefined,
  status_code: undefined as number | undefined,
  start_time: '',
  end_time: ''
})
const retention = reactive<AIRequestLogRetentionSettings>({
  enabled: true,
  retention_hours: 24,
  cleanup_interval_minutes: 30,
  delete_batch_size: 2000
})
const expanded = reactive({
  request: false,
  response: false,
  error: false,
})

function toRFC3339(value: string): string | undefined {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return undefined
  return date.toISOString()
}

function formatTime(value: string): string {
  if (!value) return '-'
  return new Date(value).toLocaleString()
}

function prettyJson(raw: string): string {
  if (!raw) return '{}'
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

async function loadLogs() {
  loading.value = true
  try {
    const result = await listAIRequestLogs({
      page: pagination.page,
      page_size: pagination.page_size,
      q: filters.q || undefined,
      request_id: filters.request_id || undefined,
      client_request_id: filters.client_request_id || undefined,
      platform: filters.platform || undefined,
      model: filters.model || undefined,
      user_id: typeof filters.user_id === 'number' && !Number.isNaN(filters.user_id) ? filters.user_id : undefined,
      api_key_id: typeof filters.api_key_id === 'number' && !Number.isNaN(filters.api_key_id) ? filters.api_key_id : undefined,
      account_id: typeof filters.account_id === 'number' && !Number.isNaN(filters.account_id) ? filters.account_id : undefined,
      group_id: typeof filters.group_id === 'number' && !Number.isNaN(filters.group_id) ? filters.group_id : undefined,
      status_code: typeof filters.status_code === 'number' && !Number.isNaN(filters.status_code) ? filters.status_code : undefined,
      start_time: toRFC3339(filters.start_time),
      end_time: toRFC3339(filters.end_time)
    })
    logs.value = result.items
    pagination.total = result.total
  } catch {
    appStore.showError(t('admin.aiRequestLogs.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadRetention() {
  try {
    const result = await getAIRequestLogRetentionSettings()
    Object.assign(retention, result)
  } catch {
    appStore.showError(t('admin.aiRequestLogs.retention.loadFailed'))
  }
}

async function saveRetention() {
  savingRetention.value = true
  try {
    const result = await updateAIRequestLogRetentionSettings({ ...retention })
    Object.assign(retention, result)
    appStore.showSuccess(t('admin.aiRequestLogs.retention.saveSuccess'))
  } catch {
    appStore.showError(t('admin.aiRequestLogs.retention.saveFailed'))
  } finally {
    savingRetention.value = false
  }
}

function applyFilters() {
  pagination.page = 1
  void loadLogs()
}

function resetFilters() {
  filters.q = ''
  filters.request_id = ''
  filters.client_request_id = ''
  filters.platform = ''
  filters.model = ''
  filters.user_id = undefined
  filters.api_key_id = undefined
  filters.account_id = undefined
  filters.group_id = undefined
  filters.status_code = undefined
  filters.start_time = ''
  filters.end_time = ''
  applyFilters()
}

async function openDetail(id: number) {
  try {
    selectedLog.value = await getAIRequestLogByID(id)
    expanded.request = false
    expanded.response = false
    expanded.error = false
    detailVisible.value = true
  } catch {
    appStore.showError(t('admin.aiRequestLogs.detail.loadFailed'))
  }
}

function closeDetail() {
  detailVisible.value = false
  selectedLog.value = null
}

function toggleSection(section: 'request' | 'response' | 'error') {
  expanded[section] = !expanded[section]
}

async function copyText(value: string) {
  try {
    await navigator.clipboard.writeText(value || '')
    appStore.showSuccess(t('admin.aiRequestLogs.detail.copySuccess'))
  } catch {
    appStore.showError(t('admin.aiRequestLogs.detail.copyFailed'))
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadLogs()
}

function handlePageSizeChange(size: number) {
  pagination.page_size = size
  pagination.page = 1
  void loadLogs()
}

onMounted(() => {
  void Promise.all([loadLogs(), loadRetention()])
})
</script>
