import { apiClient } from '../client'

export interface DiscoveredUpstreamModel {
  id: string
  display_name: string
  provider: string
  owned_by?: string
  source: string
}

export interface AccountUpstreamModelDiff {
  account_id?: number
  group_ids: number[]
  models: DiscoveredUpstreamModel[]
  add_models: DiscoveredUpstreamModel[]
  existing_models: DiscoveredUpstreamModel[]
  remove_models: string[]
  ignored_models: string[]
  last_check_time: number
}

export interface AccountUpstreamModelPreviewRequest {
  account_id?: number
  base_url?: string
  api_key?: string
  platform?: string
  compat_type?: 'newapi'
  group_ids?: number[]
  credentials?: Record<string, unknown>
  existing_mapping?: Record<string, string>
  upstream_group_name?: string
}

export interface AccountUpstreamModelApplyRequest {
  add_models: string[]
  remove_models?: string[]
  ignore_models?: string[]
  sync_to_model_catalog?: boolean
  upstream_group_name?: string
}

export interface AccountUpstreamModelApplyResult {
  added_models: string[]
  removed_models: string[]
  ignored_models: string[]
  remaining_models: string[]
  created_models: number
  created_vendors: number
}

export interface AccountUpstreamModelImportCatalogRequest {
  models: DiscoveredUpstreamModel[]
}

export interface AccountUpstreamModelImportCatalogResult {
  created_models: number
  created_vendors: number
}

export async function preview(payload: AccountUpstreamModelPreviewRequest): Promise<AccountUpstreamModelDiff> {
  const { data } = await apiClient.post<AccountUpstreamModelDiff>('/admin/accounts/upstream-models/preview', payload)
  return data
}

export async function detect(accountId: number): Promise<AccountUpstreamModelDiff> {
  const { data } = await apiClient.post<AccountUpstreamModelDiff>(`/admin/accounts/${accountId}/upstream-models/detect`)
  return data
}

export async function apply(accountId: number, payload: AccountUpstreamModelApplyRequest): Promise<AccountUpstreamModelApplyResult> {
  const { data } = await apiClient.post<AccountUpstreamModelApplyResult>(`/admin/accounts/${accountId}/upstream-models/apply`, payload)
  return data
}

export async function importCatalog(payload: AccountUpstreamModelImportCatalogRequest): Promise<AccountUpstreamModelImportCatalogResult> {
  const { data } = await apiClient.post<AccountUpstreamModelImportCatalogResult>('/admin/accounts/upstream-models/import-catalog', payload)
  return data
}

export default { preview, detect, apply, importCatalog }
