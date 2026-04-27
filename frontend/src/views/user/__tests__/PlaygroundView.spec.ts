import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PlaygroundView from '@/views/user/PlaygroundView.vue'

const {
  getGroupsMock,
  getModelsMock,
  sendChatCompletionMock,
  sendImageGenerationMock,
  streamChatCompletionMock,
  copyToClipboardMock,
} = vi.hoisted(() => ({
  getGroupsMock: vi.fn(),
  getModelsMock: vi.fn(),
  sendChatCompletionMock: vi.fn(),
  sendImageGenerationMock: vi.fn(),
  streamChatCompletionMock: vi.fn(),
  copyToClipboardMock: vi.fn(),
}))

vi.mock('@/api', () => ({
  playgroundAPI: {
    getGroups: getGroupsMock,
    getModels: getModelsMock,
    sendChatCompletion: sendChatCompletionMock,
    sendImageGeneration: sendImageGenerationMock,
    streamChatCompletion: streamChatCompletionMock,
  }
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: copyToClipboardMock,
  })
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    })
  }
})

describe('PlaygroundView', () => {
  beforeEach(() => {
    localStorage.clear()
    getGroupsMock.mockReset()
    getModelsMock.mockReset()
    sendChatCompletionMock.mockReset()
    sendImageGenerationMock.mockReset()
    streamChatCompletionMock.mockReset()
    copyToClipboardMock.mockReset()

    getGroupsMock.mockResolvedValue([
      { id: 1, name: 'default', platform: 'openai' }
    ])
    getModelsMock.mockResolvedValue([
      { model_name: 'gpt-4o' }
    ])
    sendChatCompletionMock.mockResolvedValue({
      choices: [
        {
          message: {
            content: 'hello from assistant',
            reasoning_content: 'thinking...'
          }
        }
      ]
    })
    sendImageGenerationMock.mockResolvedValue({
      data: [
        { b64_json: 'ZmFrZWltYWdl', revised_prompt: 'A flying pig over clouds.' }
      ]
    })
    streamChatCompletionMock.mockImplementation(async (_payload, handlers) => {
      handlers.onChunk?.({ choices: [{ delta: { reasoning_content: 'step1 ' } }] })
      handlers.onChunk?.({ choices: [{ delta: { content: 'streamed answer' } }] })
      handlers.onDone?.()
    })
  })

  it('loads groups and models on mount', async () => {
    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()

    expect(getGroupsMock).toHaveBeenCalledTimes(1)
    expect(getModelsMock).toHaveBeenCalledWith({ group_id: 1 })
    expect(wrapper.html()).toContain('playground.title')
  })

  it('sends a message and renders assistant response', async () => {
    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()
    const textarea = wrapper.find('textarea')
    await textarea.setValue('hello playground')
    const buttons = wrapper.findAll('button')
    const sendButton = buttons[buttons.length - 1]
    await sendButton.trigger('click')
    await flushPromises()

    expect(streamChatCompletionMock).toHaveBeenCalledTimes(1)
    expect(wrapper.html()).toContain('hello playground')
    expect(wrapper.html()).toContain('streamed answer')
    expect(wrapper.html()).toContain('step1')
  })

  it('restores persisted config and messages', async () => {
    localStorage.setItem('playground_config', JSON.stringify({ group_id: 1, model: 'gpt-4o', stream: true, temperature: 0.3, top_p: 0.8, max_tokens: 1024, frequency_penalty: 0, presence_penalty: 0, seed: 7 }))
    localStorage.setItem('playground_messages', JSON.stringify([
      { id: '1', role: 'user', content: 'persisted', created_at: Date.now(), status: 'complete' }
    ]))

    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()

    expect(wrapper.html()).toContain('persisted')
    expect(wrapper.html()).toContain('gpt-4o')
  })

  it('tolerates malformed persisted messages payload', async () => {
    localStorage.setItem('playground_messages', JSON.stringify({ bad: true }))

    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()

    expect(wrapper.html()).toContain('playground.emptyTitle')
  })

  it('streams response when stream mode is enabled', async () => {
    localStorage.setItem('playground_config', JSON.stringify({ group_id: 1, model: 'gpt-4o', stream: true, temperature: 0.7, top_p: 1, max_tokens: 4096, frequency_penalty: 0, presence_penalty: 0, seed: null }))

    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()
    await wrapper.find('textarea').setValue('stream please')
    const buttons = wrapper.findAll('button')
    const sendButton = buttons[buttons.length - 1]
    await sendButton.trigger('click')
    await flushPromises()

    expect(streamChatCompletionMock).toHaveBeenCalledTimes(1)
    expect(wrapper.html()).toContain('streamed answer')
    expect(wrapper.html()).toContain('step1')
  })

  it('syncs custom request body messages into the conversation', async () => {
    localStorage.setItem('playground_custom_mode', 'true')
    localStorage.setItem('playground_custom_body', JSON.stringify({
      model: 'gpt-4o',
      messages: [
        { role: 'system', content: 'system preset' },
        { role: 'user', content: 'custom body prompt' }
      ]
    }))

    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()

    expect(wrapper.html()).toContain('system preset')
    expect(wrapper.html()).toContain('custom body prompt')
  })

  it('extracts think tags into reasoning content', async () => {
    streamChatCompletionMock.mockImplementationOnce(async (_payload, handlers) => {
      handlers.onChunk?.({ choices: [{ delta: { content: '<think>internal thought</think>' } }] })
      handlers.onChunk?.({ choices: [{ delta: { content: 'final answer' } }] })
      handlers.onDone?.()
    })

    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()
    await wrapper.find('textarea').setValue('trigger think')
    const buttons = wrapper.findAll('button')
    const sendButton = buttons[buttons.length - 1]
    await sendButton.trigger('click')
    await flushPromises()

    expect(wrapper.html()).toContain('internal thought')
    expect(wrapper.html()).toContain('final answer')
    const articles = wrapper.findAll('article')
    expect(articles[1].html()).not.toContain('&lt;think&gt;')
  })

  it('restores persisted image mode and image urls', async () => {
    localStorage.setItem('playground_image_enabled', 'true')
    localStorage.setItem('playground_image_urls', JSON.stringify(['https://example.com/a.png', 'https://example.com/b.png']))

    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()

    expect(wrapper.html()).toContain('https://example.com/a.png')
    expect(wrapper.html()).toContain('https://example.com/b.png')
  })

  it('loads user message into prompt when editing', async () => {
    localStorage.setItem('playground_messages', JSON.stringify([
      { id: 'user-1', role: 'user', content: 'editable message', created_at: Date.now(), status: 'complete' }
    ]))

    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()
    const editButton = wrapper.findAll('button').find(button => button.text() === 'common.edit')
    expect(editButton).toBeDefined()
    await editButton!.trigger('click')
    await flushPromises()

    expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toContain('editable message')
  })

  it('restores persisted playground mode', async () => {
    localStorage.setItem('playground_mode', 'image')

    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()

    expect(wrapper.html()).toContain('playground.imageModeDescription')
  })

  it('uses image generation endpoint in image mode', async () => {
    localStorage.setItem('playground_mode', 'image')

    const wrapper = mount(PlaygroundView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
        }
      }
    })

    await flushPromises()
    await wrapper.find('textarea').setValue('draw a flying pig')
    const buttons = wrapper.findAll('button')
    const sendButton = buttons[buttons.length - 1]
    await sendButton.trigger('click')
    await flushPromises()

    expect(sendImageGenerationMock).toHaveBeenCalledTimes(1)
    expect(wrapper.html()).toContain('A flying pig over clouds.')
    expect(wrapper.html()).toContain('data:image/png;base64,ZmFrZWltYWdl')
    const articles = wrapper.findAll('article')
    const userArticles = articles.filter(article => article.html().includes('playground.user'))
    expect(userArticles).toHaveLength(1)
  })
})
