import { apiClient } from './client'

export interface Images2GenerateResponse {
  images?: Array<{
    b64_json?: string
    url?: string
    revised_prompt?: string
  }>
  revised_prompt?: string
}

export async function generate(prompt: string): Promise<Images2GenerateResponse> {
  const { data } = await apiClient.post<Images2GenerateResponse>('/images2/generate', { prompt }, {
    timeout: 180000,
  })
  return data
}

export async function edit(prompt: string, imageUrl: string): Promise<Images2GenerateResponse> {
  const { data } = await apiClient.post<Images2GenerateResponse>('/images2/generate', {
    prompt,
    image_url: imageUrl,
  }, {
    timeout: 180000,
  })
  return data
}

export const images2API = {
  generate,
  edit,
}

export default images2API
