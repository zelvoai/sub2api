<template>
  <AppLayout>
    <div class="space-y-6">
      <div v-if="loading" class="card p-6 text-sm text-gray-500 dark:text-gray-400">
        {{ t('playground.loading') }}
      </div>

      <div v-else-if="loadError" class="card p-6 text-sm text-red-600 dark:text-red-300">
        {{ loadError }}
      </div>

      <div v-else class="grid grid-cols-1 gap-6 xl:grid-cols-[320px_minmax(0,1fr)_340px]">
        <section class="card space-y-4 p-5">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-gray-100">{{ t('playground.settings') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('playground.description') }}</p>
          </div>

          <Select v-model="playgroundMode" :options="modeOptions" :label="t('playground.mode', 'Mode')" />
          <p class="text-xs text-gray-500 dark:text-gray-400">
            {{ isImageMode ? t('playground.imageModeDescription', 'Generate images with models that support image generation.') : t('playground.chatModeDescription', 'Use chat-capable models for multi-turn conversation and debugging.') }}
          </p>
          <Select v-model="selectedGroupId" :options="groupOptions" :label="t('playground.group')" />
          <Select v-model="selectedModel" :options="modelOptions" :label="loadingModels ? `${t('playground.model')}...` : t('playground.model')" />
          <p v-if="loadingModels" class="text-xs text-gray-500 dark:text-gray-400">Loading models...</p>
          <Toggle v-if="!isImageMode" v-model="inputs.stream" :label="t('playground.stream')" />
          <Toggle v-if="!isImageMode" v-model="customRequestMode" :label="t('playground.customRequestMode', 'Custom Request Body')" />
          <Toggle v-if="isImageMode" v-model="imageEnabled" :label="t('playground.imageMode', 'Image Mode')" />
          <Input v-if="!isImageMode" v-model="temperatureInput" type="number" :label="t('playground.temperature')" />
          <Input v-if="!isImageMode" v-model="topPInput" type="number" :label="t('playground.topP')" />
          <Input v-if="!isImageMode" v-model="maxTokensInput" type="number" :label="t('playground.maxTokens')" />
          <Input v-if="!isImageMode" v-model="frequencyPenaltyInput" type="number" :label="t('playground.frequencyPenalty')" />
          <Input v-if="!isImageMode" v-model="presencePenaltyInput" type="number" :label="t('playground.presencePenalty')" />
          <Input v-if="!isImageMode" v-model="seedInput" type="number" :label="t('playground.seed')" />
          <Input v-if="isImageMode" v-model="imageSize" :label="t('playground.imageSize', 'Image Size')" />
          <Input v-if="isImageMode" v-model="imageQuality" :label="t('playground.imageQuality', 'Image Quality')" />
          <Input v-if="isImageMode" v-model="imageBackground" :label="t('playground.imageBackground', 'Background')" />
          <Input v-if="isImageMode" v-model="imageOutputFormat" :label="t('playground.imageOutputFormat', 'Output Format')" />
          <TextArea
            v-if="customRequestMode"
            v-model="customRequestBody"
            :label="t('playground.customRequestBody', 'Request Body JSON')"
            :rows="10"
          />
          <div v-if="imageEnabled" class="space-y-3">
            <Input
              v-for="(_url, index) in imageUrls"
              :key="`image-${index}`"
              v-model="imageUrls[index]"
              :label="index === 0 ? t('playground.imageUrls', 'Image URLs') : undefined"
              :placeholder="t('playground.imageUrlPlaceholder', 'https://example.com/image.png')"
            />
            <div class="flex items-center gap-2">
              <button class="btn btn-secondary btn-sm" @click="addImageUrl">{{ t('common.add', 'Add') }}</button>
              <button class="btn btn-ghost btn-sm" :disabled="imageUrls.length <= 1" @click="removeImageUrl">{{ t('common.remove', 'Remove') }}</button>
            </div>
          </div>
        </section>

        <section class="card flex min-h-[640px] flex-col overflow-hidden">
          <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
            <div class="flex items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-gray-100">{{ t('playground.title') }}</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ currentModelSubtitle }}</p>
              </div>
              <button class="btn btn-secondary btn-sm" :disabled="sending" @click="clearMessages">{{ t('playground.clear') }}</button>
            </div>
          </div>

          <div class="flex-1 overflow-y-auto px-5 py-5">
            <div v-if="messages.length === 0" class="flex min-h-[300px] items-center justify-center">
              <div class="max-w-md text-center">
                <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-2xl bg-primary-50 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
                  <Icon name="chatBubble" size="lg" />
                </div>
                <h3 class="mt-4 text-lg font-semibold text-gray-900 dark:text-gray-100">{{ t('playground.emptyTitle') }}</h3>
                <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('playground.emptyDescription') }}</p>
              </div>
            </div>

            <div v-else class="space-y-4">
              <article
                v-for="message in messages"
                :key="message.id"
                class="rounded-2xl border p-4"
                :class="message.role === 'user'
                  ? 'border-primary-200 bg-primary-50/70 dark:border-primary-500/20 dark:bg-primary-500/10'
                  : message.role === 'system'
                    ? 'border-amber-200 bg-amber-50 dark:border-amber-500/20 dark:bg-amber-500/10'
                    : 'border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800'"
              >
                <div class="mb-2 flex items-center justify-between gap-3">
                  <div class="flex items-center gap-2 text-sm font-medium text-gray-700 dark:text-gray-200">
                    <Icon :name="message.role === 'assistant' ? 'sparkles' : message.role === 'system' ? 'brain' : 'user'" size="sm" />
                    <span>{{ roleLabel(message.role) }}</span>
                  </div>
                  <div class="flex items-center gap-2">
                    <button class="btn btn-ghost btn-sm" @click="copyMessage(message.content)">{{ t('playground.copy') }}</button>
                    <button v-if="message.role === 'user'" class="btn btn-ghost btn-sm" @click="editMessage(message.id)">{{ t('common.edit', 'Edit') }}</button>
                    <button v-if="message.role === 'user'" class="btn btn-ghost btn-sm" @click="reusePrompt(message.content)">{{ t('playground.retry') }}</button>
                    <button class="btn btn-ghost btn-sm" @click="removeMessage(message.id)">{{ t('common.delete', 'Delete') }}</button>
                  </div>
                </div>

                <div v-if="message.reasoning_content" class="mb-3 rounded-xl border border-dashed border-violet-200 bg-violet-50/70 p-3 dark:border-violet-500/30 dark:bg-violet-500/10">
                  <div class="mb-1 text-xs font-semibold uppercase tracking-wide text-violet-700 dark:text-violet-300">{{ t('playground.assistantThinking') }}</div>
                  <pre class="whitespace-pre-wrap text-xs text-violet-700 dark:text-violet-200">{{ message.reasoning_content }}</pre>
                </div>

                <div v-if="message.image_urls?.length" class="mb-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <a
                    v-for="(image, index) in message.image_urls"
                    :key="`${image}-${index}`"
                    :href="image"
                    target="_blank"
                    rel="noreferrer"
                    class="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-600 dark:bg-dark-800"
                  >
                    <div class="flex min-h-[12rem] items-center justify-center bg-gray-50 p-2 dark:bg-dark-900/60">
                      <img :src="image" :alt="`generated-${index}`" class="h-auto max-h-[28rem] w-full object-contain" />
                    </div>
                  </a>
                </div>

                <div class="markdown-body prose prose-sm max-w-none dark:prose-invert" v-html="renderMarkdown(message.content)"></div>
              </article>
            </div>
          </div>

          <div class="border-t border-gray-100 p-5 dark:border-dark-700">
            <TextArea v-model="prompt" :label="t('playground.title')" :placeholder="t('playground.inputPlaceholder')" :disabled="sending" :rows="4" />
            <div class="mt-4 flex items-center justify-end gap-3">
              <button class="btn btn-secondary" :disabled="sending || (!prompt.trim() && !customRequestMode)" @click="clearMessages">{{ t('playground.clear') }}</button>
              <button v-if="sending" class="btn btn-secondary" @click="stopStreaming">{{ t('playground.stop') }}</button>
              <button v-else class="btn btn-primary" :disabled="(!prompt.trim() && !customRequestMode) || !selectedGroupId || !selectedModel" @click="sendMessage">{{ t('playground.send') }}</button>
            </div>
          </div>
        </section>

        <section class="card space-y-4 p-5">
          <div>
            <h2 class="text-base font-semibold text-gray-900 dark:text-gray-100">{{ t('playground.debug') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('playground.preview') }} / {{ t('playground.request') }} / {{ t('playground.response') }}</p>
          </div>

          <div>
            <div class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('playground.preview') }}</div>
            <pre class="max-h-[180px] overflow-auto rounded-xl bg-gray-950 p-3 text-xs text-green-200">{{ stringifyDebug(previewPayload) }}</pre>
          </div>

          <div>
            <div class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('playground.request') }}</div>
            <pre class="max-h-[180px] overflow-auto rounded-xl bg-gray-950 p-3 text-xs text-sky-200">{{ stringifyDebug(debugData.request) }}</pre>
          </div>

          <div>
            <div class="mb-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('playground.response') }}</div>
            <pre class="max-h-[220px] overflow-auto rounded-xl bg-gray-950 p-3 text-xs text-amber-200">{{ stringifyDebug(debugData.response) }}</pre>
          </div>
        </section>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import AppLayout from '@/components/layout/AppLayout.vue'
import Select from '@/components/common/Select.vue'
import Input from '@/components/common/Input.vue'
import TextArea from '@/components/common/TextArea.vue'
import Toggle from '@/components/common/Toggle.vue'
import { Icon } from '@/components/icons'
import { playgroundAPI } from '@/api'
import { useClipboard } from '@/composables/useClipboard'
import type { PlaygroundDebugData, PlaygroundInputs, PlaygroundMessage, PlaygroundMessageRole, PlaygroundMessageStatus } from '@/types'

marked.setOptions({ breaks: true, gfm: true })

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const loading = ref(true)
const loadingModels = ref(false)
const loadError = ref('')
const sending = ref(false)
const customRequestMode = ref(false)
const customRequestBody = ref('')
const imageEnabled = ref(false)
const imageUrls = ref<string[]>([''])
const playgroundMode = ref<'chat' | 'image'>('chat')
const imageSize = ref('1024x1024')
const imageQuality = ref('high')
const imageBackground = ref('auto')
const imageOutputFormat = ref('png')
const generatedImages = ref<string[]>([])
const groups = ref<any[]>([])
const models = ref<any[]>([])
const prompt = ref('')
const messages = ref<PlaygroundMessage[]>([])
const editingMessageId = ref<string | null>(null)
const inputs = ref<PlaygroundInputs>({
  group_id: null,
  model: '',
  stream: true,
  temperature: 0.7,
  top_p: 1,
  max_tokens: 4096,
  frequency_penalty: 0,
  presence_penalty: 0,
  seed: null,
})
const debugData = ref<PlaygroundDebugData>({
  preview_request: null,
  request: null,
  response: null,
  timestamp: null,
})
const configStorageKey = 'playground_config'
const messagesStorageKey = 'playground_messages'
const customModeStorageKey = 'playground_custom_mode'
const customBodyStorageKey = 'playground_custom_body'
const imageEnabledStorageKey = 'playground_image_enabled'
const imageUrlsStorageKey = 'playground_image_urls'
const playgroundModeStorageKey = 'playground_mode'
let abortController: AbortController | null = null
let syncingMessagesFromCustomBody = false
let syncingCustomBodyFromMessages = false

const selectedGroupId = computed({
  get: () => inputs.value.group_id,
  set: (value: string | number | boolean | null) => {
    inputs.value.group_id = typeof value === 'number' ? value : Number(value)
  }
})

const selectedModel = computed({
  get: () => inputs.value.model,
  set: (value: string | number | boolean | null) => {
    inputs.value.model = String(value || '')
  }
})

const temperatureInput = computed({ get: () => String(inputs.value.temperature), set: (value: string) => { inputs.value.temperature = Number(value || 0) } })
const topPInput = computed({ get: () => String(inputs.value.top_p), set: (value: string) => { inputs.value.top_p = Number(value || 0) } })
const maxTokensInput = computed({ get: () => String(inputs.value.max_tokens), set: (value: string) => { inputs.value.max_tokens = Number(value || 0) } })
const frequencyPenaltyInput = computed({ get: () => String(inputs.value.frequency_penalty), set: (value: string) => { inputs.value.frequency_penalty = Number(value || 0) } })
const presencePenaltyInput = computed({ get: () => String(inputs.value.presence_penalty), set: (value: string) => { inputs.value.presence_penalty = Number(value || 0) } })
const seedInput = computed({ get: () => inputs.value.seed === null ? '' : String(inputs.value.seed), set: (value: string) => { inputs.value.seed = value.trim() ? Number(value) : null } })

const groupOptions = computed(() => Array.isArray(groups.value) ? groups.value.map(group => ({ value: group.id, label: `${group.name} (${group.platform})` })) : [])
const modeOptions = computed(() => [
  { value: 'chat', label: t('playground.chatMode', 'Chat') },
  { value: 'image', label: t('playground.imageGenerationMode', 'Image Generation') }
])
const modelOptions = computed(() => Array.isArray(models.value) ? models.value.map(model => ({ value: model.model_name, label: model.model_name })) : [])
const currentModelSubtitle = computed(() => selectedModel.value || t('playground.model'))
const isImageMode = computed(() => playgroundMode.value === 'image')

const parsedCustomRequestBody = computed<Record<string, any> | null>(() => {
  if (!customRequestMode.value || !customRequestBody.value.trim()) return null
  try {
    return JSON.parse(customRequestBody.value)
  } catch {
    return null
  }
})

const previewPayload = computed(() => ({
  group_id: inputs.value.group_id,
  model: inputs.value.model,
  messages: parsedCustomRequestBody.value?.messages || buildPreviewMessages(),
  stream: inputs.value.stream,
  temperature: inputs.value.temperature,
  top_p: inputs.value.top_p,
  max_tokens: inputs.value.max_tokens,
  frequency_penalty: inputs.value.frequency_penalty,
  presence_penalty: inputs.value.presence_penalty,
  seed: inputs.value.seed,
}))

watch(previewPayload, value => {
  debugData.value.preview_request = value
}, { immediate: true, deep: true })

watch(inputs, value => {
  localStorage.setItem(configStorageKey, JSON.stringify(value))
}, { deep: true })

watch(messages, value => {
  if (!Array.isArray(value)) {
    messages.value = normalizeMessages(value)
    return
  }
  localStorage.setItem(messagesStorageKey, JSON.stringify(value))
}, { deep: true })

watch(customRequestMode, value => {
  localStorage.setItem(customModeStorageKey, value ? 'true' : 'false')
})

watch(customRequestBody, value => {
  localStorage.setItem(customBodyStorageKey, value)
})

watch(imageEnabled, value => {
  localStorage.setItem(imageEnabledStorageKey, value ? 'true' : 'false')
})

watch(imageUrls, value => {
  localStorage.setItem(imageUrlsStorageKey, JSON.stringify(value))
}, { deep: true })

watch(playgroundMode, value => {
  localStorage.setItem(playgroundModeStorageKey, value)
  generatedImages.value = []
  if (value === 'image') {
    customRequestMode.value = false
    inputs.value.stream = false
  }
})

async function loadModelsForGroup(groupId: number | null) {
  if (!groupId) return
  loadingModels.value = true
  try {
    models.value = await playgroundAPI.getModels({ group_id: groupId })
    if (!models.value.some(model => model.model_name === inputs.value.model)) {
      inputs.value.model = models.value[0]?.model_name || ''
    }
  } catch (error: any) {
    models.value = []
    inputs.value.model = ''
    debugData.value.response = error?.message || String(error)
  } finally {
    loadingModels.value = false
  }
}

watch(parsedCustomRequestBody, value => {
  if (!customRequestMode.value || !value || !Array.isArray(value.messages) || syncingCustomBodyFromMessages) return
  syncingMessagesFromCustomBody = true
  messages.value = normalizeMessages(value.messages)
  syncingMessagesFromCustomBody = false
}, { deep: true })

watch(() => inputs.value.group_id, async (groupId) => {
  await loadModelsForGroup(groupId)
})

onMounted(async () => {
  try {
    const savedConfig = localStorage.getItem(configStorageKey)
    if (savedConfig) {
      Object.assign(inputs.value, JSON.parse(savedConfig))
    }
    const savedMessages = localStorage.getItem(messagesStorageKey)
    if (savedMessages) {
      messages.value = normalizeMessages(JSON.parse(savedMessages))
    }
    customRequestMode.value = localStorage.getItem(customModeStorageKey) === 'true'
    customRequestBody.value = localStorage.getItem(customBodyStorageKey) || ''
    imageEnabled.value = localStorage.getItem(imageEnabledStorageKey) === 'true'
    imageUrls.value = JSON.parse(localStorage.getItem(imageUrlsStorageKey) || '[""]')
    playgroundMode.value = localStorage.getItem(playgroundModeStorageKey) === 'image' ? 'image' : 'chat'
  } catch {
    // Ignore malformed persisted state and fall back to defaults.
  }

  loading.value = true
  loadError.value = ''
  try {
    groups.value = await playgroundAPI.getGroups()
    if (!inputs.value.group_id) {
      inputs.value.group_id = groups.value[0]?.id ?? null
    }
    if (!inputs.value.group_id) {
      loadError.value = 'No available playground groups found.'
      return
    }
    // Explicitly trigger initial model load so restored persisted group_id also
    // fetches models even when the watcher does not observe a fresh change.
    void loadModelsForGroup(inputs.value.group_id)
  } finally {
    loading.value = false
  }
})

watch(previewPayload, value => {
  if (!customRequestMode.value) return
  if (!customRequestBody.value.trim()) {
    customRequestBody.value = JSON.stringify(value, null, 2)
  }
}, { deep: true })

watch(messages, value => {
  if (!customRequestMode.value || !parsedCustomRequestBody.value || syncingMessagesFromCustomBody) return
  syncingCustomBodyFromMessages = true
  const nextPayload = {
    ...parsedCustomRequestBody.value,
    messages: value.map(message => ({ role: message.role, content: message.content }))
  }
  customRequestBody.value = JSON.stringify(nextPayload, null, 2)
  syncingCustomBodyFromMessages = false
}, { deep: true })

async function sendMessage() {
  if ((!prompt.value.trim() && !customRequestMode.value) || !inputs.value.group_id || !inputs.value.model) return

  let payload: Record<string, any>
  let displayPrompt = prompt.value
  let userMessageAdded = false

  if (customRequestMode.value) {
    payload = JSON.parse(customRequestBody.value || '{}')
    payload.group_id = inputs.value.group_id
    payload.model = payload.model || inputs.value.model
    if (typeof payload.stream !== 'boolean') {
      payload.stream = inputs.value.stream
    }
    const lastMessage = Array.isArray(payload.messages) ? payload.messages[payload.messages.length - 1] : null
    displayPrompt = typeof lastMessage?.content === 'string' ? lastMessage.content : (prompt.value || '[custom request]')
  } else {
    const content = buildUserMessageContent(prompt.value)
    const userMessage: PlaygroundMessage = {
      id: crypto.randomUUID(),
      role: 'user',
      content,
      created_at: Date.now(),
      status: 'complete'
    }
    messages.value.push(userMessage)
    userMessageAdded = true
    payload = {
      ...previewPayload.value,
      messages: [...previewPayload.value.messages, { role: 'user', content }]
    }
  }

  debugData.value.request = payload
  debugData.value.response = null
  debugData.value.timestamp = new Date().toISOString()
  sending.value = true
  abortController = new AbortController()

  try {
    if (playgroundMode.value === 'image') {
      if (!userMessageAdded) {
        const userMessage: PlaygroundMessage = {
          id: crypto.randomUUID(),
          role: 'user',
          content: prompt.value,
          created_at: Date.now(),
          status: 'complete'
        }
        messages.value.push(userMessage)
      }

      const assistantMessage: PlaygroundMessage = {
        id: crypto.randomUUID(),
        role: 'assistant',
        content: t('playground.generatingImage'),
        created_at: Date.now(),
        status: 'loading'
      }
      messages.value.push(assistantMessage)

      const response = await playgroundAPI.sendImageGeneration({
        group_id: Number(inputs.value.group_id),
        model: inputs.value.model,
        prompt: prompt.value,
        size: imageSize.value,
        quality: imageQuality.value,
        background: imageBackground.value,
        output_format: imageOutputFormat.value,
      })
      debugData.value.response = response
      generatedImages.value = Array.isArray(response?.data)
        ? response.data.map((item: any) => resolveGeneratedImageURL(item, imageOutputFormat.value)).filter(Boolean)
        : []
      assistantMessage.image_urls = [...generatedImages.value]
      assistantMessage.content = response?.data?.[0]?.revised_prompt || t('playground.imageGenerated', 'Image generated successfully.')
      assistantMessage.status = 'complete'
      messages.value = [...messages.value.slice(0, -1), { ...assistantMessage }]
      prompt.value = ''
      return
    }

    if (payload.stream) {
      if (customRequestMode.value) {
        messages.value.push({
          id: crypto.randomUUID(),
          role: 'user',
          content: displayPrompt,
          created_at: Date.now(),
          status: 'complete'
        })
      }

      const assistantMessage: PlaygroundMessage = {
        id: crypto.randomUUID(),
        role: 'assistant',
        content: '',
        reasoning_content: '',
        created_at: Date.now(),
        status: 'loading'
      }
      messages.value.push(assistantMessage)

      const streamEvents: string[] = []
      await playgroundAPI.streamChatCompletion(payload as any, {
        onChunk: (chunk) => {
          streamEvents.push(typeof chunk === 'string' ? chunk : JSON.stringify(chunk))
          debugData.value.response = streamEvents.join('\n')

          const delta = chunk?.choices?.[0]?.delta
          if (!delta) return
          if (delta.reasoning_content || delta.reasoning) {
            assistantMessage.reasoning_content = `${assistantMessage.reasoning_content || ''}${delta.reasoning_content || delta.reasoning || ''}`
          }
          if (delta.content) {
            assistantMessage.content = `${assistantMessage.content}${delta.content}`
          }
          normalizeAssistantMessageContent(assistantMessage)
          assistantMessage.status = 'incomplete'
          messages.value = [...messages.value.slice(0, -1), { ...assistantMessage }]
        },
        onDone: () => {
          normalizeAssistantMessageContent(assistantMessage)
          assistantMessage.status = 'complete'
          messages.value = [...messages.value.slice(0, -1), { ...assistantMessage }]
        }
      }, abortController.signal)
    } else {
      const response = await playgroundAPI.sendChatCompletion(payload as any)
      debugData.value.response = response
      const choice = response?.choices?.[0]
      const assistantMessage: PlaygroundMessage = {
        id: crypto.randomUUID(),
        role: 'assistant',
        content: choice?.message?.content || '',
        reasoning_content: choice?.message?.reasoning_content || choice?.message?.reasoning || '',
        created_at: Date.now(),
        status: 'complete'
      }
      normalizeAssistantMessageContent(assistantMessage)
      messages.value.push(assistantMessage)
    }
    prompt.value = ''
  } catch (error: any) {
    if (error instanceof DOMException && error.name === 'AbortError') {
      return
    }
    debugData.value.response = error?.message || String(error)
    messages.value.push({
      id: crypto.randomUUID(),
      role: 'assistant',
      content: error?.message || 'Request failed',
      created_at: Date.now(),
      status: 'error'
    })
  } finally {
    sending.value = false
    abortController = null
  }
}

function stopStreaming() {
  abortController?.abort()
}

function clearMessages() {
  messages.value = []
  prompt.value = ''
  localStorage.removeItem(messagesStorageKey)
}

function reusePrompt(content: string) {
  prompt.value = content
}

function editMessage(messageID: string) {
  const message = normalizeMessages(messages.value).find(item => item.id === messageID)
  if (!message) return
  prompt.value = message.content
  editingMessageId.value = messageID
}

function removeMessage(messageID: string) {
  messages.value = normalizeMessages(messages.value).filter(message => message.id !== messageID)
  if (editingMessageId.value === messageID) {
    editingMessageId.value = null
  }
}

function addImageUrl() {
  imageUrls.value.push('')
}

function removeImageUrl() {
  if (imageUrls.value.length <= 1) return
  imageUrls.value.pop()
}

function roleLabel(role: PlaygroundMessageRole) {
  if (role === 'assistant') return t('playground.assistant')
  if (role === 'system') return t('playground.system')
  return t('playground.user')
}

function renderMarkdown(content: string) {
  const html = marked.parse(content || '') as string
  return DOMPurify.sanitize(html)
}

function normalizeAssistantMessageContent(message: PlaygroundMessage) {
  const content = message.content || ''
  const thinkMatches = Array.from(content.matchAll(/<think>([\s\S]*?)<\/think>/g))
  if (thinkMatches.length === 0) return

  const extracted = thinkMatches.map(match => match[1]).join('\n')
  message.reasoning_content = `${message.reasoning_content || ''}${message.reasoning_content ? '\n' : ''}${extracted}`.trim()
  message.content = content.replace(/<think>[\s\S]*?<\/think>/g, '').trim()
}

function buildPreviewMessages() {
  const normalizedMessages = normalizeMessages(messages.value).map(message => ({ role: message.role, content: message.content }))
  if (!imageEnabled.value) return normalizedMessages
  const lastUserIndex = normalizedMessages.map(item => item.role).lastIndexOf('user')
  return normalizedMessages.map((message, index) => {
    if (index !== lastUserIndex || message.role !== 'user') return message
    return {
      role: message.role,
      content: buildUserMessageContent(String(message.content))
    }
  })
}

function normalizeMessages(value: unknown): PlaygroundMessage[] {
  if (!Array.isArray(value)) return []
  return value
    .filter((message): message is Record<string, any> => !!message && typeof message === 'object' && typeof (message as Record<string, any>).role === 'string')
    .map((message) => ({
      id: typeof message.id === 'string' ? message.id : crypto.randomUUID(),
      role: message.role,
      content: typeof message.content === 'string' ? message.content : JSON.stringify(message.content ?? ''),
      reasoning_content: typeof message.reasoning_content === 'string' ? message.reasoning_content : '',
      created_at: typeof message.created_at === 'number' ? message.created_at : Date.now(),
      status: normalizeMessageStatus(message.status)
    }))
}

function normalizeMessageStatus(status: unknown): PlaygroundMessageStatus {
  switch (status) {
    case 'idle':
    case 'loading':
    case 'complete':
    case 'error':
    case 'incomplete':
      return status
    default:
      return 'complete'
  }
}

function buildUserMessageContent(text: string) {
  if (!imageEnabled.value) return text
  const validImageUrls = imageUrls.value.map(item => item.trim()).filter(Boolean)
  if (validImageUrls.length === 0) return text
  return JSON.stringify([
    { type: 'text', text },
    ...validImageUrls.map(url => ({ type: 'image_url', image_url: { url } }))
  ])
}

function resolveGeneratedImageURL(item: any, outputFormat: string) {
  if (!item || typeof item !== 'object') return ''
  if (typeof item.url === 'string' && item.url.trim()) return item.url
  if (typeof item.b64_json === 'string' && item.b64_json.trim()) {
    const normalizedFormat = (outputFormat || 'png').trim().toLowerCase()
    return `data:image/${normalizedFormat};base64,${item.b64_json}`
  }
  return ''
}

function stringifyDebug(value: unknown) {
  if (value == null) return ''
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

async function copyMessage(content: string) {
  await copyToClipboard(content)
}
</script>
