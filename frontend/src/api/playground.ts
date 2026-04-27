import { apiClient } from './client'

export interface PlaygroundGroup {
  id: number
  name: string
  description: string
  platform: string
  rate_multiplier: number
  effective_multiplier: number
  subscription_type: string
  supported_model_scopes?: string[]
}

export interface PlaygroundModel {
  model_name: string
  provider: string
  vendor_name: string
  vendor_icon: string
  account_count: number
  group_ids: number[]
  groups: string[]
  priced: boolean
}

export interface PlaygroundChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string | Array<Record<string, any>>
}

export interface PlaygroundChatRequest {
  group_id: number
  model: string
  messages: PlaygroundChatMessage[]
  stream?: boolean
  temperature?: number
  top_p?: number
  max_tokens?: number
  frequency_penalty?: number
  presence_penalty?: number
  seed?: number | null
}

export interface PlaygroundImageRequest {
  group_id: number
  model: string
  prompt: string
  size?: string
  quality?: string
  background?: string
  output_format?: string
}

export interface PlaygroundStreamHandlers {
  onChunk?: (payload: any) => void
  onDone?: () => void
}

function getAuthHeaders() {
  return {
    Authorization: `Bearer ${localStorage.getItem('auth_token') || ''}`,
    'Content-Type': 'application/json',
  }
}

async function parsePlaygroundError(response: Response): Promise<Error> {
  const rawText = await response.text()
  try {
    const parsed = rawText ? JSON.parse(rawText) : null
    if (parsed && typeof parsed === 'object' && 'message' in parsed) {
      return new Error(String((parsed as Record<string, unknown>).message || `HTTP error ${response.status}`))
    }
  } catch {
    // Fall back to raw text below.
  }
  return new Error(rawText || `HTTP error ${response.status}`)
}

async function parsePlaygroundResponse(response: Response): Promise<any> {
  const rawText = await response.text()
  const parsed = rawText ? JSON.parse(rawText) : null

  if (!response.ok) {
    if (parsed && typeof parsed === 'object') {
      const parsedRecord = parsed as Record<string, unknown>
      const nestedError = parsedRecord.error
      if (nestedError && typeof nestedError === 'object' && 'message' in (nestedError as Record<string, unknown>)) {
        throw new Error(String((nestedError as Record<string, unknown>).message || `HTTP error ${response.status}`))
      }
      if ('message' in parsedRecord) {
        throw new Error(String(parsedRecord.message || `HTTP error ${response.status}`))
      }
    }
    throw new Error(rawText || `HTTP error ${response.status}`)
  }

  if (parsed && typeof parsed === 'object' && 'code' in parsed) {
    if ((parsed as Record<string, unknown>).code === 0) {
      return (parsed as Record<string, unknown>).data
    }
    throw new Error(String((parsed as Record<string, unknown>).message || 'Request failed'))
  }

  return parsed
}

export async function getPlaygroundGroups(): Promise<PlaygroundGroup[]> {
  const { data } = await apiClient.get<PlaygroundGroup[]>('/playground/groups')
  return data
}

export async function getPlaygroundModels(params?: {
  group_id?: number
  search?: string
  limit?: number
}): Promise<PlaygroundModel[]> {
  const { data } = await apiClient.get<PlaygroundModel[] | null>('/playground/models', { params })
  return Array.isArray(data) ? data : []
}

export async function sendPlaygroundChatCompletion(payload: PlaygroundChatRequest): Promise<any> {
  const response = await fetch('/api/v1/playground/chat/completions', {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(payload),
  })

  return parsePlaygroundResponse(response)
}

export async function sendPlaygroundImageGeneration(payload: PlaygroundImageRequest): Promise<any> {
  const response = await fetch('/api/v1/playground/images/generations', {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(payload),
  })

  return parsePlaygroundResponse(response)
}

export async function streamPlaygroundChatCompletion(
  payload: PlaygroundChatRequest,
  handlers: PlaygroundStreamHandlers,
  signal?: AbortSignal,
): Promise<void> {
  const response = await fetch('/api/v1/playground/chat/completions', {
    method: 'POST',
    headers: getAuthHeaders(),
    body: JSON.stringify(payload),
    signal,
  })

  if (!response.ok) {
    throw await parsePlaygroundError(response)
  }

  const reader = response.body?.getReader()
  if (!reader) {
    throw new Error('No response body')
  }

  const decoder = new TextDecoder()
  let buffer = ''

  while (true) {
    const { done, value } = await reader.read()
    if (done) break

    buffer += decoder.decode(value, { stream: true })
    const parts = buffer.split('\n\n')
    buffer = parts.pop() || ''

    for (const part of parts) {
      const line = part
        .split('\n')
        .find(item => item.startsWith('data: '))

      if (!line) continue
      const payloadText = line.slice(6).trim()
      if (!payloadText) continue
      if (payloadText === '[DONE]') {
        handlers.onDone?.()
        continue
      }

      try {
        handlers.onChunk?.(JSON.parse(payloadText))
      } catch {
        handlers.onChunk?.(payloadText)
      }
    }
  }

  handlers.onDone?.()
}

export const playgroundAPI = {
  getGroups: getPlaygroundGroups,
  getModels: getPlaygroundModels,
  sendChatCompletion: sendPlaygroundChatCompletion,
  sendImageGeneration: sendPlaygroundImageGeneration,
  streamChatCompletion: streamPlaygroundChatCompletion,
}

export default playgroundAPI
