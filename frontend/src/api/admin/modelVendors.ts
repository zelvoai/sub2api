import { apiClient } from '../client'

export interface ModelVendor {
  id: number
  name: string
  description: string
  icon: string
  status: string
  created_at: string
  updated_at: string
}

export interface ModelVendorPayload {
  name: string
  description?: string
  icon?: string
  status?: string
}

interface PaginatedResponse<T> {
  items: T[]
  total: number
}

export async function list(
  page = 1,
  pageSize = 50,
  filters?: { search?: string; status?: string }
): Promise<PaginatedResponse<ModelVendor>> {
  const { data } = await apiClient.get<PaginatedResponse<ModelVendor>>('/admin/model-vendors', {
    params: { page, page_size: pageSize, ...filters }
  })
  return data
}

export async function create(payload: ModelVendorPayload): Promise<ModelVendor> {
  const { data } = await apiClient.post<ModelVendor>('/admin/model-vendors', payload)
  return data
}

export async function update(id: number, payload: ModelVendorPayload): Promise<ModelVendor> {
  const { data } = await apiClient.put<ModelVendor>(`/admin/model-vendors/${id}`, payload)
  return data
}

export async function remove(id: number): Promise<void> {
  await apiClient.delete(`/admin/model-vendors/${id}`)
}

const modelVendorsAPI = { list, create, update, remove }
export default modelVendorsAPI
