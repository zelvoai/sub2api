import { describe, expect, it, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

const { previewAvailableModelsMock, showWarning, showSuccess, showInfo } = vi.hoisted(() => ({
  previewAvailableModelsMock: vi.fn(),
  showWarning: vi.fn(),
  showSuccess: vi.fn(),
  showInfo: vi.fn()
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showWarning,
    showSuccess,
    showInfo,
    showError: vi.fn()
  })
}))

vi.mock('@/api/admin/accounts', () => ({
  previewAvailableModels: previewAvailableModelsMock
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) => {
        if (params?.count !== undefined) {
          return `${key}:${params.count}`
        }
        return key
      }
    })
  }
})

import ModelWhitelistSelector from '../ModelWhitelistSelector.vue'

function mountSelector(props: Record<string, unknown> = {}) {
  return mount(ModelWhitelistSelector, {
    props: {
      modelValue: ['gpt-5.2'],
      platform: 'openai',
      fetchRequest: {
        platform: 'openai',
        type: 'apikey',
        credentials: {
          base_url: 'https://api.openai.com',
          api_key: 'sk-test'
        }
      },
      ...props
    },
    global: {
      stubs: {
        ModelIcon: { template: '<span />' },
        Icon: { template: '<span />' },
        BaseDialog: {
          props: ['show', 'title'],
          template: '<div v-if="show"><slot /><slot name="footer" /></div>'
        }
      }
    }
  })
}

describe('ModelWhitelistSelector', () => {
  beforeEach(() => {
    previewAvailableModelsMock.mockReset()
    showWarning.mockReset()
    showSuccess.mockReset()
    showInfo.mockReset()
  })

  it('fetches remote models and appends the chosen model to the whitelist', async () => {
    previewAvailableModelsMock.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4', provider: 'OpenAI', source: 'remote' },
      { id: 'gpt-5.2', display_name: 'GPT-5.2', provider: 'OpenAI', source: 'remote' },
      { id: 'claude-sonnet-4', display_name: 'Claude Sonnet 4', provider: 'Anthropic', source: 'remote' }
    ])

    const wrapper = mountSelector()

    const fetchButton = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.fetchModels'))
    expect(fetchButton).toBeTruthy()
    await fetchButton!.trigger('click')
    await nextTick()
    await nextTick()

    expect(previewAvailableModelsMock).toHaveBeenCalledWith({
      platform: 'openai',
      type: 'apikey',
      credentials: {
        base_url: 'https://api.openai.com',
        api_key: 'sk-test'
      }
    })
    expect(wrapper.text()).toContain('admin.accounts.fetchModelsNewGroup (2)')
    expect(wrapper.text()).toContain('admin.accounts.fetchModelsExistingGroup (1)')
    expect(wrapper.text()).not.toContain('gpt-5.4')

    const openProviderButton = wrapper.findAll('button').find(button => button.text().includes('OpenAI'))
    expect(openProviderButton).toBeTruthy()
    await openProviderButton!.trigger('click')
    await nextTick()

    expect(wrapper.text()).toContain('gpt-5.4')

    const newModelCheckbox = wrapper.findAll('input[type="checkbox"]').find(input => {
      return input.element.parentElement?.textContent?.includes('gpt-5.4')
    })
    expect(newModelCheckbox).toBeTruthy()
    await newModelCheckbox!.setValue(true)
    await wrapper.get('.btn.btn-primary').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toBeTruthy()
    expect(emitted?.at(-1)?.[0]).toEqual(['gpt-5.2', 'gpt-5.4'])
    expect(showSuccess).toHaveBeenCalled()
  })

  it('falls back to local platform models when remote fetch fails', async () => {
    previewAvailableModelsMock.mockRejectedValue(new Error('boom'))

    const wrapper = mountSelector()

    const fetchButton = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.fetchModels'))
    expect(fetchButton).toBeTruthy()
    await fetchButton!.trigger('click')
    await nextTick()
    await nextTick()

    expect(showWarning).toHaveBeenCalled()
    expect(wrapper.text()).toContain('admin.accounts.fetchModelsHintFallback')
  })

  it('toggles provider checkbox to select all models in the collapsed group', async () => {
    previewAvailableModelsMock.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4', provider: 'OpenAI', source: 'remote' },
      { id: 'gpt-5.3', display_name: 'GPT-5.3', provider: 'OpenAI', source: 'remote' }
    ])

    const wrapper = mountSelector()

    const fetchButton = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.fetchModels'))
    expect(fetchButton).toBeTruthy()
    await fetchButton!.trigger('click')
    await nextTick()
    await nextTick()

    const providerCheckbox = wrapper.findAll('input[type="checkbox"]').find(input => {
      return input.element.parentElement?.textContent?.includes('OpenAI')
    })
    expect(providerCheckbox).toBeTruthy()
    await providerCheckbox!.setValue(true)

    await wrapper.get('.btn.btn-primary').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.at(-1)?.[0]).toEqual(['gpt-5.2', 'gpt-5.3', 'gpt-5.4'])
  })

  it('toggles the outer select-all checkbox to select all provider groups', async () => {
    previewAvailableModelsMock.mockResolvedValue([
      { id: 'gpt-5.4', display_name: 'GPT-5.4', provider: 'OpenAI', source: 'remote' },
      { id: 'claude-sonnet-4', display_name: 'Claude Sonnet 4', provider: 'Anthropic', source: 'remote' }
    ])

    const wrapper = mountSelector()

    const fetchButton = wrapper.findAll('button').find(button => button.text().includes('admin.accounts.fetchModels'))
    expect(fetchButton).toBeTruthy()
    await fetchButton!.trigger('click')
    await nextTick()
    await nextTick()

    const selectAllCheckbox = wrapper.findAll('input[type="checkbox"]').find(input => {
      return input.element.parentElement?.textContent?.includes('admin.accounts.fetchModelsSelectAllProviders')
    })
    expect(selectAllCheckbox).toBeTruthy()
    await selectAllCheckbox!.setValue(true)

    await wrapper.get('.btn.btn-primary').trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted?.at(-1)?.[0]).toEqual(['gpt-5.2', 'claude-sonnet-4', 'gpt-5.4'])
  })
})
