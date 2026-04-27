import { describe, it, expect, vi, beforeEach } from 'vitest'

const getMock = vi.fn()
const postMock = vi.fn()
const fetchMock = vi.fn()

globalThis.fetch = fetchMock as unknown as typeof fetch

vi.mock('@/api/client', () => ({
  apiClient: {
    get: getMock,
    post: postMock,
  }
}))

describe('playgroundAPI', () => {
  beforeEach(() => {
    getMock.mockReset()
    postMock.mockReset()
    fetchMock.mockReset()
  })

  it('loads groups from playground endpoint', async () => {
    getMock.mockResolvedValue({ data: [{ id: 1, name: 'default', platform: 'openai' }] })
    const { playgroundAPI } = await import('@/api/playground')

    const result = await playgroundAPI.getGroups()

    expect(getMock).toHaveBeenCalledWith('/playground/groups')
    expect(result).toEqual([{ id: 1, name: 'default', platform: 'openai' }])
  })

  it('loads models with optional group filter', async () => {
    getMock.mockResolvedValue({ data: [{ model_name: 'gpt-4o' }] })
    const { playgroundAPI } = await import('@/api/playground')

    const result = await playgroundAPI.getModels({ group_id: 9, search: 'gpt' })

    expect(getMock).toHaveBeenCalledWith('/playground/models', {
      params: { group_id: 9, search: 'gpt' }
    })
    expect(result).toEqual([{ model_name: 'gpt-4o' }])
  })

  it('normalizes null model payload to an empty array', async () => {
    getMock.mockResolvedValue({ data: null })
    const { playgroundAPI } = await import('@/api/playground')

    const result = await playgroundAPI.getModels({ group_id: 2 })

    expect(result).toEqual([])
  })

  it('posts chat completion payload', async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      text: async () => JSON.stringify({ code: 0, message: 'success', data: { choices: [] } })
    })
    const { playgroundAPI } = await import('@/api/playground')
    const payload = {
      group_id: 1,
      model: 'gpt-4o',
      messages: [{ role: 'user' as const, content: 'hello' }],
      stream: false,
    }

    const result = await playgroundAPI.sendChatCompletion(payload)

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(result).toEqual({ choices: [] })
  })
})
