<template>
  <AppLayout>
    <div class="images2-shell">
      <header class="images2-header">
        <div>
          <h1 class="images2-title">{{ pageTitle }}</h1>
        </div>
        <div class="images2-balance-pill" :class="canGenerate ? '' : 'is-alert'">
          <span class="images2-inline-icon">
            <svg
              class="images2-balance-icon"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="1.5"
              aria-hidden="true"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z"
              />
            </svg>
            {{ t('images2.balance') }} ${{ balanceText }}
          </span>
          <span class="images2-balance-separator"></span>
          <span>{{ t('images2.priceText', { price: priceText }) }}</span>
        </div>
      </header>

      <div v-if="errorMessage" class="images2-error-banner">
        <strong>{{ t('images2.generateFailed') }}</strong>
        <span>{{ errorMessage }}</span>
      </div>

      <section class="images2-composer">
        <textarea
          v-model="prompt"
          class="images2-textarea"
          :placeholder="t('images2.promptPlaceholder')"
          rows="5"
        />

        <div class="images2-options-row" :aria-label="t('images2.sizeLabel')">
          <span class="images2-options-label">{{ t('images2.sizeLabel') }}</span>
          <div v-if="!hasImage" class="images2-size-control-group">
            <div class="images2-size-options images2-size-options-fixed">
              <button
                v-for="option in sizeOptions"
                :key="option.value"
                type="button"
                class="images2-size-option"
                :class="{ 'is-active': selectedSize === option.value && !followReferenceAspect }"
                :disabled="isGenerating"
                @click="selectedSize = option.value; followReferenceAspect = false"
              >
                <Icon :name="option.icon" size="sm" :stroke-width="1.8" />
                {{ option.label }}
              </button>
            </div>
            <div v-if="canFollowReferenceAspect" class="images2-size-options images2-size-options-reference">
              <button
                type="button"
                class="images2-size-option is-reference-aspect"
                :class="{ 'is-active': followReferenceAspect }"
                :disabled="isGenerating"
                @click="followReferenceAspect = true"
              >
                <Icon name="link" size="sm" :stroke-width="1.8" />
                {{ t('images2.sizeFollowReference') }}
              </button>
            </div>
          </div>
          <div v-else class="images2-size-lock">
            <p class="images2-size-lock-title">{{ t('images2.editPreserveAspectTitle') }}</p>
            <p class="images2-size-lock-copy">{{ editSizeHintText }}</p>
          </div>
        </div>

        <p v-if="!hasImage && canFollowReferenceAspect && followReferenceAspect" class="images2-size-follow-hint">
          {{ referenceAspectHintText }}
        </p>

        <div class="images2-attachments-panel">
          <div class="images2-attachments-header">
            <div>
              <p class="images2-options-label">{{ t('images2.attachmentsTitle') }}</p>
              <p class="images2-attachments-copy">{{ t('images2.attachmentsHint') }}</p>
            </div>
            <button class="images2-size-option images2-upload-button" type="button" :disabled="isGenerating || attachments.length >= maxAttachments" @click="openAttachmentPicker">
              <Icon name="plus" size="sm" :stroke-width="2" />
              {{ t('images2.uploadAttachments') }}
            </button>
            <input ref="fileInput" type="file" accept="image/*" multiple class="images2-file-input" @change="handleAttachmentChange" />
          </div>

          <p class="images2-attachments-meta">{{ t('images2.attachmentsMeta', { count: maxAttachments }) }}</p>
          <p v-if="attachmentError" class="images2-attachments-error">{{ attachmentError }}</p>

          <div v-if="hasImage || attachments.length" class="images2-attachments-list">
            <div v-if="hasImage" class="images2-attachment-card is-current-result">
              <img :src="displayImageUrl" :alt="t('images2.currentResultAlt')" class="images2-attachment-image" />
              <div class="images2-attachment-badge">{{ t('images2.currentResultBadge') }}</div>
            </div>

            <div v-for="(attachment, index) in attachments" :key="`${index}-${attachment.dataUrl.slice(0, 24)}`" class="images2-attachment-card">
              <img :src="attachment.dataUrl" :alt="t('images2.attachmentPreviewAlt', { index: index + 1 })" class="images2-attachment-image" />
              <button type="button" class="images2-attachment-remove" :aria-label="t('images2.removeAttachment')" @click="removeAttachment(index)">
                <Icon name="x" size="sm" :stroke-width="2.2" />
              </button>
            </div>
          </div>
        </div>

        <div class="images2-toolbar" :class="{ 'is-single': !hasImage }">
          <button class="images2-primary" :class="!canGenerate ? 'is-alert' : ''" :disabled="isGenerating || (canGenerate && !prompt.trim())" @click="handlePrimaryAction">
            <Icon :name="primaryActionIcon" size="sm" :stroke-width="2" />
            {{ primaryActionText }}
          </button>
          <button v-if="hasImage" class="images2-secondary" :disabled="isGenerating || !prompt.trim()" @click="handleEditAction">
            <Icon name="edit" size="sm" :stroke-width="2" />
            {{ t('images2.editCurrent') }}
          </button>
        </div>

        <div v-if="hasImage" class="images2-result-hint">
          <p>
            <strong>{{ t('images2.resultActionHintRegenerateLabel') }}</strong>{{ t('images2.resultActionHintRegenerateBody') }}
          </p>
          <p>
            <strong>{{ t('images2.resultActionHintEditLabel') }}</strong>{{ t('images2.resultActionHintEditBody') }}
          </p>
        </div>
      </section>

      <section class="images2-stage" :class="isGenerating ? 'is-generating' : ''">
        <div v-if="isGenerating" class="images2-loader">
          <div class="images2-loader-ring"></div>
          <p>{{ t('images2.loadingHint') }}</p>
        </div>

        <template v-else-if="hasImage">
          <div class="images2-stage-meta">
            <div class="images2-stage-footer">
              <p class="images2-notice">{{ noticeText }}</p>
              <button class="images2-secondary images2-download-button" @click="downloadImage">
                <Icon name="download" size="sm" :stroke-width="2" />
                {{ t('images2.saveImage') }}
              </button>
            </div>
          </div>
          <img :src="displayImageUrl" :alt="revisedPrompt || prompt" class="images2-image" />
        </template>

        <div v-else class="images2-empty">
          <p>{{ noticeText }}</p>
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { Icon } from '@/components/icons'
import images2API, { type Images2Attachment, type Images2Size } from '@/api/images2'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()

const prompt = ref('')
const isGenerating = ref(false)
const imageUrl = ref('')
const revisedPrompt = ref('')
const errorMessage = ref('')
const attachmentError = ref('')
const selectedSize = ref<Images2Size>('1024x1024')
const followReferenceAspect = ref(false)
const attachments = ref<Images2Attachment[]>([])
const fileInput = ref<HTMLInputElement | null>(null)

const sizeOptions: Array<{ value: Images2Size; label: string; icon: 'square' | 'rectangleHorizontal' | 'rectangleVertical' }> = [
  { value: '1024x1024', label: t('images2.sizeSquare'), icon: 'square' },
  { value: '1536x1024', label: t('images2.sizeLandscape'), icon: 'rectangleHorizontal' },
  { value: '1024x1536', label: t('images2.sizePortrait'), icon: 'rectangleVertical' },
]

const settings = computed(() => appStore.cachedPublicSettings)
const user = computed(() => authStore.user)
const pageTitle = computed(() => settings.value?.images2_page_title || 'ChatGPT Images 2 生图')
const noticeText = computed(() => settings.value?.images2_notice_text || t('images2.defaultNotice'))
const rechargePath = computed(() => settings.value?.images2_recharge_path || '/purchase')
const unitPrice = computed(() => settings.value?.images2_price_per_image ?? 0.5)
const maxAttachments = computed(() => settings.value?.images2_max_attachments ?? 5)
const canGenerate = computed(() => (user.value?.balance ?? 0) >= unitPrice.value)
const balanceText = computed(() => (user.value?.balance ?? 0).toFixed(2))
const priceText = computed(() => unitPrice.value.toFixed(2))
const hasImage = computed(() => Boolean(imageUrl.value))
const displayImageUrl = computed(() => imageUrl.value)
const currentImageDimensions = computed(() => getImageDimensions(displayImageUrl.value))
const firstReferenceDimensions = computed(() => {
  const first = attachments.value[0]
  return first ? getImageDimensions(first.dataUrl) : null
})
const firstReferenceAspectRatio = computed(() => {
  const dimensions = firstReferenceDimensions.value
  if (!dimensions) return ''
  return trimTrailingZeros((dimensions.width / dimensions.height).toFixed(2))
})
const canFollowReferenceAspect = computed(() => attachments.value.length > 0 && Boolean(firstReferenceDimensions.value))
const referenceAspectHintText = computed(() => {
  const dimensions = firstReferenceDimensions.value
  if (!dimensions || !firstReferenceAspectRatio.value) {
    return t('images2.sizeFollowReferenceBodyUnknown')
  }
  return t('images2.sizeFollowReferenceBody', {
    width: dimensions.width,
    height: dimensions.height,
    ratio: firstReferenceAspectRatio.value,
  })
})
const currentImageAspectRatio = computed(() => {
  const dimensions = currentImageDimensions.value
  if (!dimensions) return ''
  return trimTrailingZeros((dimensions.width / dimensions.height).toFixed(2))
})
const editSizeHintText = computed(() => {
  const dimensions = currentImageDimensions.value
  if (!dimensions || !currentImageAspectRatio.value) {
    return t('images2.editPreserveAspectBodyUnknown')
  }
  return t('images2.editPreserveAspectBody', {
    width: dimensions.width,
    height: dimensions.height,
    ratio: currentImageAspectRatio.value,
  })
})
const primaryActionIcon = computed(() => {
  if (!canGenerate.value) return 'creditCard'
  return 'sparkles'
})
const primaryActionText = computed(() => {
  if (isGenerating.value) return t('images2.generating')
  if (!canGenerate.value) return t('images2.rechargePrimary')
  return hasImage.value ? t('images2.regenerate') : t('images2.generate')
})

onMounted(async () => {
  await Promise.allSettled([appStore.fetchPublicSettings(), authStore.refreshUser()])
})

async function generateImage() {
  if (!prompt.value.trim() || isGenerating.value || !canGenerate.value) return
  isGenerating.value = true
  revisedPrompt.value = ''
  errorMessage.value = ''
  try {
    const result = await images2API.generate(prompt.value.trim(), selectedSize.value, attachments.value, followReferenceAspect.value)
    const first = result.images?.[0]
    if (typeof first?.b64_json === 'string' && first.b64_json) {
      imageUrl.value = `data:image/png;base64,${first.b64_json}`
    } else if (typeof first?.url === 'string' && first.url) {
      imageUrl.value = first.url
    } else {
      imageUrl.value = ''
    }
    revisedPrompt.value = typeof first?.revised_prompt === 'string' ? first.revised_prompt : (result.revised_prompt || '')
    await authStore.refreshUser()
  } catch (error: any) {
    const message = error?.response?.data?.error?.message
      || error?.response?.data?.message
      || error?.message
      || t('images2.generateFailed')
    const normalized = message === t('images2.generateFailed')
      ? t('common.error')
      : message
    errorMessage.value = normalized
    appStore.showError(normalized)
  } finally {
    isGenerating.value = false
  }
}

async function editCurrentImage() {
  if (!prompt.value.trim() || isGenerating.value || !canGenerate.value || !imageUrl.value) return
  isGenerating.value = true
  revisedPrompt.value = ''
  errorMessage.value = ''
  const previousImageUrl = imageUrl.value
  try {
    const result = await images2API.edit(prompt.value.trim(), imageUrl.value, selectedSize.value, attachments.value, true)
    const first = result.images?.[0]
    if (typeof first?.b64_json === 'string' && first.b64_json) {
      imageUrl.value = `data:image/png;base64,${first.b64_json}`
    } else if (typeof first?.url === 'string' && first.url) {
      imageUrl.value = first.url
    } else {
      imageUrl.value = ''
    }
    revisedPrompt.value = typeof first?.revised_prompt === 'string' ? first.revised_prompt : (result.revised_prompt || '')
    if (previousImageUrl) {
      attachments.value = retainEditSourceReference(previousImageUrl, attachments.value, maxAttachments.value)
    }
    await authStore.refreshUser()
  } catch (error: any) {
    const message = error?.response?.data?.error?.message
      || error?.response?.data?.message
      || error?.message
      || t('images2.generateFailed')
    const normalized = message === t('images2.generateFailed')
      ? t('common.error')
      : message
    errorMessage.value = normalized
    appStore.showError(normalized)
  } finally {
    isGenerating.value = false
  }
}

function handlePrimaryAction() {
  if (isGenerating.value) return
  if (!canGenerate.value) {
    goRecharge()
    return
  }
  void generateImage()
}

function handleEditAction() {
  void editCurrentImage()
}

function goRecharge() {
  router.push(rechargePath.value)
}

function openAttachmentPicker() {
  fileInput.value?.click()
}

async function handleAttachmentChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files ?? [])
  input.value = ''
  attachmentError.value = ''
  if (files.length === 0) return

  if (attachments.value.length >= maxAttachments.value) {
    attachmentError.value = t('images2.attachmentsLimitError', { count: maxAttachments.value })
    return
  }

  const availableSlots = maxAttachments.value - attachments.value.length
  const nextFiles = files.slice(0, availableSlots)
  if (files.length > availableSlots) {
    attachmentError.value = t('images2.attachmentsLimitError', { count: maxAttachments.value })
  }

  for (const file of nextFiles) {
    if (!file.type.startsWith('image/')) {
      attachmentError.value = t('images2.attachmentsOnlyImages')
      continue
    }
    try {
      const dataUrl = await readFileAsDataUrl(file)
      attachments.value.push({ dataUrl })
    } catch {
      attachmentError.value = t('images2.attachmentsReadFailed')
    }
  }
}

function removeAttachment(index: number) {
  attachments.value.splice(index, 1)
  attachmentError.value = ''
  if (!attachments.value.length) {
    followReferenceAspect.value = false
  }
}

function retainEditSourceReference(previousImageUrl: string, existingAttachments: Images2Attachment[], maxCount: number): Images2Attachment[] {
  const deduped = [
    { dataUrl: previousImageUrl },
    ...existingAttachments,
  ].filter((attachment, index, list) => list.findIndex((item) => item.dataUrl === attachment.dataUrl) === index)

  return deduped.slice(0, Math.max(1, maxCount))
}

function readFileAsDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      if (typeof reader.result === 'string') {
        resolve(reader.result)
        return
      }
      reject(new Error('invalid file result'))
    }
    reader.onerror = () => reject(reader.error || new Error('failed to read file'))
    reader.readAsDataURL(file)
  })
}

function getImageDimensions(dataUrl: string): { width: number; height: number } | null {
  const match = dataUrl.match(/^data:image\/(png|jpeg|jpg|gif|webp);base64,([A-Za-z0-9+/=]+)$/i)
  if (!match) return null
  try {
    const binary = atob(match[2])
    const bytes = Uint8Array.from(binary, (char) => char.charCodeAt(0))
    const png = readPngDimensions(bytes)
    if (png) return png
    const gif = readGifDimensions(bytes)
    if (gif) return gif
    const jpeg = readJpegDimensions(bytes)
    if (jpeg) return jpeg
    const webp = readWebpDimensions(bytes)
    if (webp) return webp
  } catch {
    return null
  }
  return null
}

function readPngDimensions(bytes: Uint8Array): { width: number; height: number } | null {
  if (bytes.length < 24) return null
  const pngSignature = [137, 80, 78, 71, 13, 10, 26, 10]
  for (let i = 0; i < pngSignature.length; i += 1) {
    if (bytes[i] !== pngSignature[i]) return null
  }
  return {
    width: readUint32BE(bytes, 16),
    height: readUint32BE(bytes, 20),
  }
}

function readGifDimensions(bytes: Uint8Array): { width: number; height: number } | null {
  if (bytes.length < 10) return null
  const header = String.fromCharCode(...bytes.slice(0, 6))
  if (header !== 'GIF87a' && header !== 'GIF89a') return null
  return {
    width: bytes[6] | (bytes[7] << 8),
    height: bytes[8] | (bytes[9] << 8),
  }
}

function readJpegDimensions(bytes: Uint8Array): { width: number; height: number } | null {
  if (bytes.length < 4 || bytes[0] !== 0xff || bytes[1] !== 0xd8) return null
  let offset = 2
  while (offset + 9 < bytes.length) {
    if (bytes[offset] !== 0xff) {
      offset += 1
      continue
    }
    const marker = bytes[offset + 1]
    if (marker === 0xd8 || marker === 0xd9) {
      offset += 2
      continue
    }
    if (offset + 4 >= bytes.length) return null
    const segmentLength = (bytes[offset + 2] << 8) | bytes[offset + 3]
    if (segmentLength < 2 || offset + 2 + segmentLength > bytes.length) return null
    const isSOF = (marker >= 0xc0 && marker <= 0xc3) || (marker >= 0xc5 && marker <= 0xc7) || (marker >= 0xc9 && marker <= 0xcb) || (marker >= 0xcd && marker <= 0xcf)
    if (isSOF && offset + 8 < bytes.length) {
      return {
        height: (bytes[offset + 5] << 8) | bytes[offset + 6],
        width: (bytes[offset + 7] << 8) | bytes[offset + 8],
      }
    }
    offset += 2 + segmentLength
  }
  return null
}

function readWebpDimensions(bytes: Uint8Array): { width: number; height: number } | null {
  if (bytes.length < 30) return null
  const riff = String.fromCharCode(...bytes.slice(0, 4))
  const webp = String.fromCharCode(...bytes.slice(8, 12))
  if (riff !== 'RIFF' || webp !== 'WEBP') return null
  const chunk = String.fromCharCode(...bytes.slice(12, 16))
  if (chunk === 'VP8X' && bytes.length >= 30) {
    const width = 1 + bytes[24] + (bytes[25] << 8) + (bytes[26] << 16)
    const height = 1 + bytes[27] + (bytes[28] << 8) + (bytes[29] << 16)
    return { width, height }
  }
  if (chunk === 'VP8 ' && bytes.length >= 30) {
    return {
      width: bytes[26] | (bytes[27] << 8),
      height: bytes[28] | (bytes[29] << 8),
    }
  }
  if (chunk === 'VP8L' && bytes.length >= 25) {
    const b0 = bytes[21]
    const b1 = bytes[22]
    const b2 = bytes[23]
    const b3 = bytes[24]
    const width = 1 + (((b1 & 0x3f) << 8) | b0)
    const height = 1 + (((b3 & 0x0f) << 10) | (b2 << 2) | ((b1 & 0xc0) >> 6))
    return { width, height }
  }
  return null
}

function readUint32BE(bytes: Uint8Array, offset: number): number {
  return ((bytes[offset] << 24) >>> 0) + (bytes[offset + 1] << 16) + (bytes[offset + 2] << 8) + bytes[offset + 3]
}

function trimTrailingZeros(value: string): string {
  return value.replace(/\.0+$/, '').replace(/(\.\d*?)0+$/, '$1')
}

async function downloadImage() {
  if (!displayImageUrl.value) return
  const filename = `chatgpt-images-2-${new Date().toISOString().replace(/[:.]/g, '-')}.png`
  const link = document.createElement('a')
  link.download = filename

  if (displayImageUrl.value.startsWith('data:')) {
    link.href = displayImageUrl.value
    link.click()
    return
  }

  try {
    const response = await fetch(displayImageUrl.value, { mode: 'cors' })
    if (!response.ok) throw new Error(`download failed: ${response.status}`)
    const blob = await response.blob()
    const objectUrl = URL.createObjectURL(blob)
    link.href = objectUrl
    link.click()
    URL.revokeObjectURL(objectUrl)
  } catch {
    link.href = displayImageUrl.value
    link.target = '_blank'
    link.rel = 'noopener noreferrer'
    link.click()
  }
}
</script>

<style>
.images2-shell {
  margin: 0 auto;
  max-width: 980px;
  padding: 1rem 0 1.5rem;
  color: #0f172a;
  min-height: calc(100vh - 120px);
}

.images2-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1.5rem;
}

.images2-title {
  margin: 0;
  font-size: clamp(1.8rem, 3.4vw, 3.1rem);
  line-height: 0.96;
  font-weight: 600;
  color: #0f172a;
}

.images2-balance-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.75rem;
  border-radius: 9999px;
  border: 1px solid rgba(148, 163, 184, 0.2);
  background: rgba(255, 255, 255, 0.9);
  padding: 0.75rem 1rem;
  color: #334155;
  backdrop-filter: blur(16px);
}

.images2-inline-icon {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
}

.images2-balance-icon {
  width: 1rem;
  height: 1rem;
  flex-shrink: 0;
}

.images2-balance-pill.is-alert {
  border-color: rgba(248, 113, 113, 0.24);
  color: #fca5a5;
}

.images2-balance-separator {
  width: 1px;
  height: 1rem;
  background: rgba(148, 163, 184, 0.35);
}

.images2-composer,
.images2-stage {
  border-radius: 28px;
  border: 1px solid rgba(226, 232, 240, 0.95);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.98), rgba(248, 250, 252, 0.98));
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(22px);
}

.images2-composer {
  padding: 1rem;
}

.images2-error-banner {
  display: flex;
  flex-direction: column;
  gap: 0.2rem;
  margin-bottom: 0.9rem;
  border-radius: 18px;
  border: 1px solid rgba(248, 113, 113, 0.28);
  background: rgba(254, 242, 242, 0.92);
  padding: 0.85rem 1rem;
  color: #b91c1c;
  font-size: 0.92rem;
}

.images2-textarea {
  width: 100%;
  min-height: 160px;
  resize: vertical;
  border: 0;
  outline: none;
  background: transparent;
  color: #0f172a;
  font-size: 1rem;
  line-height: 1.75;
}

.images2-textarea::placeholder {
  color: rgba(100, 116, 139, 0.72);
}

.images2-options-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 1rem;
  border-top: 1px solid rgba(148, 163, 184, 0.16);
  padding-top: 1rem;
}

.images2-options-label {
  color: #475569;
  font-size: 0.9rem;
  font-weight: 600;
}

.images2-size-options {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.images2-size-control-group {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 0.5rem;
}

.images2-size-options-fixed {
  flex-wrap: nowrap;
}

.images2-size-options-reference {
  width: 100%;
  justify-content: flex-end;
}

.images2-size-lock {
  max-width: 420px;
  margin-left: auto;
  padding: 0.85rem 1rem;
  border-radius: 18px;
  border: 1px solid rgba(14, 165, 233, 0.18);
  background: rgba(240, 249, 255, 0.82);
  text-align: left;
}

.images2-size-lock-title {
  margin: 0;
  color: #0369a1;
  font-size: 0.9rem;
  font-weight: 700;
}

.images2-size-lock-copy {
  margin: 0.3rem 0 0;
  color: #0369a1;
  font-size: 0.88rem;
  line-height: 1.55;
}

.images2-size-follow-hint {
  margin: 0.75rem 0 0;
  color: #64748b;
  font-size: 0.82rem;
  line-height: 1.55;
}

.images2-size-option {
  appearance: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 1px solid rgba(148, 163, 184, 0.28);
  border-radius: 9999px;
  background: rgba(255, 255, 255, 0.72);
  color: #334155;
  cursor: pointer;
  padding: 0.55rem 0.8rem;
  font-size: 0.88rem;
  transition: 180ms ease;
}

.images2-size-option:hover:not(:disabled),
.images2-size-option.is-active {
  border-color: rgba(15, 23, 42, 0.76);
  background: #0f172a;
  color: #f8fafc;
}

.images2-size-option:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.images2-size-option.is-reference-aspect {
  padding-inline: 1rem;
}

.images2-dev-toggle {
  margin-top: 0.85rem;
  display: inline-flex;
  align-items: center;
  gap: 0.55rem;
  color: #64748b;
  font-size: 0.88rem;
}

.images2-dev-toggle input {
  margin: 0;
}

.images2-result-hint {
  margin: 0.85rem 0 0;
  padding: 0.85rem 1rem;
  border-radius: 16px;
  border: 1px solid rgba(14, 165, 233, 0.2);
  background: rgba(240, 249, 255, 0.86);
  color: #0369a1;
  font-size: 0.9rem;
  line-height: 1.55;
}

.images2-result-hint p {
  margin: 0;
}

.images2-result-hint p + p {
  margin-top: 0.35rem;
}

.images2-result-hint strong {
  font-weight: 700;
}

.images2-attachments-panel {
  margin-top: 1rem;
  border-top: 1px solid rgba(148, 163, 184, 0.16);
  padding-top: 1rem;
}

.images2-attachments-header {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 1rem;
}

.images2-attachments-copy {
  margin: 0.3rem 0 0;
  color: #64748b;
  font-size: 0.9rem;
  line-height: 1.55;
}

.images2-upload-button {
  white-space: nowrap;
}

.images2-file-input {
  display: none;
}

.images2-attachments-meta {
  margin: 0.7rem 0 0;
  color: #64748b;
  font-size: 0.82rem;
}

.images2-attachments-error {
  margin: 0.55rem 0 0;
  color: #dc2626;
  font-size: 0.84rem;
}

.images2-attachments-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(96px, 1fr));
  gap: 0.75rem;
  margin-top: 0.8rem;
}

.images2-attachment-card {
  position: relative;
  overflow: hidden;
  border-radius: 18px;
  border: 1px solid rgba(148, 163, 184, 0.22);
  background: rgba(255, 255, 255, 0.72);
  aspect-ratio: 1;
}

.images2-attachment-card.is-current-result {
  border-color: rgba(14, 165, 233, 0.38);
  box-shadow: inset 0 0 0 1px rgba(14, 165, 233, 0.18);
}

.images2-attachment-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.images2-attachment-badge {
  position: absolute;
  left: 0.5rem;
  bottom: 0.5rem;
  padding: 0.28rem 0.5rem;
  border-radius: 9999px;
  background: rgba(14, 165, 233, 0.9);
  color: #f8fafc;
  font-size: 0.72rem;
  font-weight: 700;
}

.images2-attachment-remove {
  position: absolute;
  top: 0.45rem;
  right: 0.45rem;
  width: 1.9rem;
  height: 1.9rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 9999px;
  background: rgba(15, 23, 42, 0.72);
  color: #f8fafc;
  cursor: pointer;
}

.images2-toolbar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  margin-top: 1rem;
}

.images2-primary,
.images2-secondary,
.images2-link {
  appearance: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.45rem;
  border: 0;
  border-radius: 9999px;
  cursor: pointer;
  transition: 180ms ease;
}

.images2-primary {
  padding: 0.78rem 1.2rem;
  background: #0f172a;
  color: #f8fafc;
  font-weight: 600;
}

.images2-primary:disabled {
  cursor: not-allowed;
  opacity: 0.45;
}

.images2-primary.is-alert {
  background: #dc2626;
  color: #fff7f7;
}

.images2-secondary {
  padding: 0.78rem 1.1rem;
  background: rgba(15, 23, 42, 0.05);
  color: #0f172a;
}

.images2-download-button {
  padding: 0.56rem 0.9rem;
  gap: 0.35rem;
  font-size: 0.88rem;
}

.images2-stage {
  margin-top: 1.25rem;
  min-height: 560px;
  padding: 1rem;
}

.images2-stage-meta {
  display: flex;
  align-items: flex-start;
  justify-content: center;
  gap: 1rem;
  margin-bottom: 0.9rem;
}

.images2-stage.is-generating {
  display: grid;
  place-items: center;
}

.images2-loader,
.images2-empty {
  min-height: 520px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  color: #64748b;
}

.images2-loader {
  gap: 0;
}

.images2-loader-ring {
  width: 68px;
  height: 68px;
  border-radius: 9999px;
  border: 2px solid rgba(15, 23, 42, 0.08);
  border-top-color: rgba(15, 23, 42, 0.72);
  animation: images2-spin 1s linear infinite;
  margin-bottom: 0;
}

.images2-loader p {
  margin: 1.25rem 0 0;
}

.images2-image {
  width: 100%;
  display: block;
  border-radius: 22px;
  object-fit: contain;
  background: rgba(255, 255, 255, 0.8);
}

.images2-stage-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  width: 100%;
}

.images2-notice {
  margin: 0;
  color: #64748b;
  font-size: 0.92rem;
  text-align: center;
}

.dark .images2-title {
  color: #f8fafc;
}

.dark .images2-shell {
  color: #e2e8f0;
  background: transparent;
}

.dark .images2-balance-pill {
  border-color: rgba(148, 163, 184, 0.16);
  background: rgba(15, 23, 42, 0.72);
  color: #cbd5e1;
}

.dark .images2-error-banner {
  border-color: rgba(248, 113, 113, 0.26);
  background: rgba(69, 10, 10, 0.55);
  color: #fecaca;
}

.dark .images2-dev-toggle {
  color: #94a3b8;
}

.dark .images2-composer,
.dark .images2-stage {
  border-color: rgba(71, 85, 105, 0.38);
  background: linear-gradient(180deg, rgba(15, 23, 42, 0.74), rgba(30, 41, 59, 0.68));
  box-shadow: 0 24px 56px rgba(2, 6, 23, 0.22);
}

.dark .images2-textarea {
  color: #f8fafc;
}

.dark .images2-textarea::placeholder {
  color: rgba(148, 163, 184, 0.7);
}

.dark .images2-options-row {
  border-top-color: rgba(148, 163, 184, 0.14);
}

.dark .images2-options-label {
  color: #cbd5e1;
}

.dark .images2-size-option {
  border-color: rgba(148, 163, 184, 0.22);
  background: rgba(15, 23, 42, 0.45);
  color: #cbd5e1;
}

.dark .images2-size-option:hover:not(:disabled),
.dark .images2-size-option.is-active {
  border-color: rgba(248, 250, 252, 0.72);
  background: #f8fafc;
  color: #020617;
}

.dark .images2-size-lock {
  border-color: rgba(56, 189, 248, 0.22);
  background: rgba(8, 47, 73, 0.46);
}

.dark .images2-size-lock-title,
.dark .images2-size-lock-copy {
  color: #bae6fd;
}

.dark .images2-size-follow-hint {
  color: #94a3b8;
}

.dark .images2-result-hint {
  border-color: rgba(56, 189, 248, 0.22);
  background: rgba(8, 47, 73, 0.46);
  color: #bae6fd;
}

.dark .images2-attachments-panel {
  border-top-color: rgba(148, 163, 184, 0.14);
}

.dark .images2-attachments-copy,
.dark .images2-attachments-meta {
  color: #94a3b8;
}

.dark .images2-attachment-card {
  border-color: rgba(148, 163, 184, 0.16);
  background: rgba(15, 23, 42, 0.48);
}

.dark .images2-attachment-card.is-current-result {
  border-color: rgba(56, 189, 248, 0.45);
  box-shadow: inset 0 0 0 1px rgba(56, 189, 248, 0.24);
}

.dark .images2-secondary {
  background: rgba(255, 255, 255, 0.08);
  color: #e2e8f0;
}

.dark .images2-primary {
  background: #f8fafc;
  color: #020617;
}

.dark .images2-primary.is-alert {
  background: #ef4444;
  color: #fff7f7;
}

.dark .images2-image {
  background: rgba(30, 41, 59, 0.45);
}

.dark .images2-loader-ring {
  border-color: rgba(255, 255, 255, 0.08);
  border-top-color: rgba(255, 255, 255, 0.72);
}

.dark .images2-loader,
.dark .images2-empty,
.dark .images2-notice {
  color: #94a3b8;
}

@keyframes images2-spin {
  to { transform: rotate(360deg); }
}

@media (max-width: 768px) {
  .images2-header,
  .images2-stage-meta,
  .images2-stage-footer {
    flex-direction: column;
    align-items: stretch;
  }

  .images2-header {
    align-items: center;
    text-align: center;
  }

  .images2-balance-pill {
    justify-content: center;
    align-self: center;
  }

  .images2-size-options {
    width: 100%;
  }

  .images2-size-control-group {
    align-items: stretch;
  }

  .images2-size-options-fixed {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.5rem;
  }

  .images2-size-options-reference {
    display: block;
  }

  .images2-options-row {
    align-items: stretch;
    flex-direction: column;
  }

  .images2-attachments-header {
    flex-direction: column;
    align-items: stretch;
  }

  .images2-size-option {
    width: 100%;
    min-width: 0;
    border-radius: 16px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .images2-size-option.is-reference-aspect {
    justify-content: center;
  }

  .images2-toolbar {
    display: grid;
    grid-template-columns: 1fr;
    align-items: center;
  }

  .images2-toolbar:not(.is-single) {
    grid-template-columns: 1fr 1fr;
  }

  .images2-primary,
  .images2-secondary {
    width: 100%;
  }

  .images2-stage,
  .images2-loader,
  .images2-empty {
    min-height: 360px;
  }
}
</style>
