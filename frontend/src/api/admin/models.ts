import { apiClient } from '../client'

export interface ModelBoundChannel {
  id: number
  name: string
  platform: string
  group_ids: number[]
  groups: string[]
}

export interface ModelSyncConflictField {
  field: string
  local: unknown
  upstream: unknown
}

export interface ModelSyncConflict {
  model_name: string
  fields: ModelSyncConflictField[]
}

export interface UpstreamModelCatalog {
  model_name: string
  description: string
  icon: string
  tags: string
  vendor_name: string
  endpoints: unknown
  status: number
  name_rule: number
}

export interface ModelCatalog {
  id: number
  model_name: string
  description: string
  icon: string
  tags: string
  vendor_id: number | null
  vendor_name: string
  vendor_icon: string
  endpoints: string[]
  status: string
  sync_official: boolean
  name_rule: number
  bound_channels?: ModelBoundChannel[]
  enable_groups?: string[]
  quota_types?: string[]
  matched_models?: string[]
  matched_count?: number
  account_count?: number
  available_groups?: string[]
  created_at: string
  updated_at: string
  deleted_at?: string | null
}

export interface GroupAvailableModel {
  model_name: string
  provider: string
  vendor_name: string
  vendor_icon: string
  account_count: number
  group_ids: number[]
  groups: string[]
  priced: boolean
}

export interface ModelCatalogPayload {
  model_name: string
  description?: string
  icon?: string
  tags?: string
  vendor_id?: number | null
  endpoints?: string[]
  status?: string
  sync_official?: boolean
  name_rule?: number
}

export interface MissingModel {
  model_name: string
  name_rule: number
  sources: string[]
  platforms: string[]
  channels: ModelBoundChannel[]
  matched_count: number
}

export interface ModelSyncPreview {
  missing: UpstreamModelCatalog[]
  conflicts: ModelSyncConflict[]
}

export interface ModelSyncApplyRequest {
  locale?: string
  overwrite?: Array<{ model_name: string; fields: string[] }>
}

export interface ModelSyncApplyResult {
  created_models: number
  created_vendors: number
  updated_models: number
  skipped_models: string[]
  conflict_models: string[]
  upstream_model_url: string
}

interface PaginatedResponse<T> {
  items: T[]
  total: number
}

export async function list(
  page = 1,
  pageSize = 20,
  filters?: {
    search?: string
    status?: string
    vendor_id?: number | null
    name_rule?: number | null
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  }
): Promise<PaginatedResponse<ModelCatalog>> {
  const { data } = await apiClient.get<PaginatedResponse<ModelCatalog>>('/admin/models', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function search(query: string, limit = 20): Promise<ModelCatalog[]> {
  const { data } = await apiClient.get<PaginatedResponse<ModelCatalog>>('/admin/models/search', {
    params: { search: query, page: 1, page_size: limit, status: 'active' }
  })
  return data.items
}

export async function getById(id: number): Promise<ModelCatalog> {
  const { data } = await apiClient.get<ModelCatalog>(`/admin/models/${id}`)
  return data
}

export async function create(payload: ModelCatalogPayload): Promise<ModelCatalog> {
  const { data } = await apiClient.post<ModelCatalog>('/admin/models', payload)
  return data
}

export async function update(id: number, payload: ModelCatalogPayload): Promise<ModelCatalog> {
  const { data } = await apiClient.put<ModelCatalog>(`/admin/models/${id}`, payload)
  return data
}

export async function updateStatus(id: number, status: string): Promise<ModelCatalog> {
  const { data } = await apiClient.patch<ModelCatalog>(`/admin/models/${id}/status`, { status })
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/models/${id}`)
}

export async function batchDelete(ids: number[]): Promise<void> {
  await apiClient.post('/admin/models/batch-delete', { ids })
}

export async function missing(): Promise<MissingModel[]> {
  const { data } = await apiClient.get<MissingModel[]>('/admin/models/missing')
  return data
}

export async function syncPreview(locale = 'zh-CN'): Promise<ModelSyncPreview> {
  const { data } = await apiClient.get<ModelSyncPreview>('/admin/models/sync-upstream/preview', {
    params: { locale }
  })
  return data
}

export async function syncUpstream(payload: ModelSyncApplyRequest): Promise<ModelSyncApplyResult> {
  const { data } = await apiClient.post<ModelSyncApplyResult>('/admin/models/sync-upstream', payload)
  return data
}

export async function groupAvailable(groupIds: number[], search = '', limit = 100): Promise<GroupAvailableModel[]> {
  const { data } = await apiClient.get<GroupAvailableModel[]>('/admin/models/group-available', {
    params: {
      group_ids: groupIds.join(','),
      search,
      limit
    }
  })
  return data
}

const modelsAPI = {
  list,
  search,
  getById,
  create,
  update,
  updateStatus,
  remove,
  batchDelete,
  missing,
  syncPreview,
  syncUpstream,
  groupAvailable
}

export default modelsAPI
