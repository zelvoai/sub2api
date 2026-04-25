import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface AIRequestLog {
  id: number
  created_at: string
  request_id: string
  client_request_id: string
  user_id?: number | null
  api_key_id?: number | null
  account_id?: number | null
  group_id?: number | null
  platform: string
  model: string
  request_path: string
  inbound_endpoint: string
  upstream_endpoint: string
  method: string
  status_code: number
  stream: boolean
  request_body: string
  response_body: string
  error_message: string
  duration_ms?: number | null
  content_type: string
  response_content_type: string
}

export interface AIRequestLogQueryParams {
  page?: number
  page_size?: number
  request_id?: string
  client_request_id?: string
  platform?: string
  model?: string
  status_code?: number
  user_id?: number
  api_key_id?: number
  account_id?: number
  group_id?: number
  q?: string
  start_time?: string
  end_time?: string
}

export interface AIRequestLogRetentionSettings {
  enabled: boolean
  retention_hours: number
  cleanup_interval_minutes: number
  delete_batch_size: number
}

export async function listAIRequestLogs(params: AIRequestLogQueryParams): Promise<PaginatedResponse<AIRequestLog>> {
  const { data } = await apiClient.get<PaginatedResponse<AIRequestLog>>('/admin/ops/ai-request-logs', { params })
  return data
}

export async function getAIRequestLogByID(id: number): Promise<AIRequestLog> {
  const { data } = await apiClient.get<AIRequestLog>(`/admin/ops/ai-request-logs/${id}`)
  return data
}

export async function getAIRequestLogRetentionSettings(): Promise<AIRequestLogRetentionSettings> {
  const { data } = await apiClient.get<AIRequestLogRetentionSettings>('/admin/ops/ai-request-logs/settings/retention')
  return data
}

export async function updateAIRequestLogRetentionSettings(payload: AIRequestLogRetentionSettings): Promise<AIRequestLogRetentionSettings> {
  const { data } = await apiClient.put<AIRequestLogRetentionSettings>('/admin/ops/ai-request-logs/settings/retention', payload)
  return data
}

const aiRequestLogsAPI = {
  list: listAIRequestLogs,
  getByID: getAIRequestLogByID,
  getRetentionSettings: getAIRequestLogRetentionSettings,
  updateRetentionSettings: updateAIRequestLogRetentionSettings,
}

export default aiRequestLogsAPI
