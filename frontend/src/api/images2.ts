import { apiClient } from './client'

export interface Images2GenerateResponse {
  images?: Array<{
    b64_json?: string
    url?: string
    revised_prompt?: string
  }>
  revised_prompt?: string
}

export type Images2Size = '1024x1024' | '1536x1024' | '1024x1536'

export async function generate(prompt: string, size: Images2Size): Promise<Images2GenerateResponse> {
  const { data } = await apiClient.post<Images2GenerateResponse>('/images2/generate', { prompt, size }, {
    timeout: 180000,
  })
  return data
}

export async function edit(prompt: string, imageUrl: string, size: Images2Size): Promise<Images2GenerateResponse> {
  const { data } = await apiClient.post<Images2GenerateResponse>('/images2/generate', {
    prompt,
    image_url: imageUrl,
    size,
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
