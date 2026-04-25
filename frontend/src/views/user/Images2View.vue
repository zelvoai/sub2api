<template>
  <AppLayout>
    <div class="images2-shell">
      <header class="images2-header">
        <div>
          <h1 class="images2-title">{{ pageTitle }}</h1>
        </div>
        <div class="images2-balance-pill" :class="canGenerate ? '' : 'is-alert'">
          <span>{{ t('images2.balance') }} ${{ balanceText }}</span>
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
          <div class="images2-size-options">
            <button
              v-for="option in sizeOptions"
              :key="option.value"
              type="button"
              class="images2-size-option"
              :class="{ 'is-active': selectedSize === option.value }"
              :disabled="isGenerating"
              @click="selectedSize = option.value"
            >
              {{ option.label }}
            </button>
          </div>
        </div>

        <div v-if="imageUrl" class="images2-edit-hint">
          {{ t('images2.editHint') }}
        </div>

        <div class="images2-toolbar">
          <button class="images2-primary" :class="!canGenerate ? 'is-alert' : ''" :disabled="isGenerating || !prompt.trim() || !canGenerate" @click="generateImage">
            {{ isGenerating ? t('images2.generating') : (imageUrl ? t('images2.editCurrent') : t('images2.generate')) }}
          </button>
          <button class="images2-secondary" :disabled="isGenerating" @click="resetCanvas">
            {{ t('images2.newImage') }}
          </button>
          <button class="images2-link" @click="goRecharge">
            {{ rechargeCtaText }}
          </button>
        </div>
      </section>

      <section class="images2-stage" :class="isGenerating ? 'is-generating' : ''">
        <div v-if="isGenerating" class="images2-loader">
          <div class="images2-loader-ring"></div>
          <p>{{ t('images2.loadingHint') }}</p>
        </div>

        <template v-else-if="imageUrl">
          <img :src="imageUrl" :alt="revisedPrompt || prompt" class="images2-image" />
          <div class="images2-result-hint">
            {{ t('images2.resultEditHint') }}
          </div>
          <div class="images2-stage-footer">
            <p class="images2-notice">{{ noticeText }}</p>
            <button class="images2-secondary" @click="downloadImage">{{ t('images2.saveImage') }}</button>
          </div>
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
import images2API, { type Images2Size } from '@/api/images2'
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
const selectedSize = ref<Images2Size>('1024x1024')

const sizeOptions: Array<{ value: Images2Size; label: string }> = [
  { value: '1024x1024', label: t('images2.sizeSquare') },
  { value: '1536x1024', label: t('images2.sizeLandscape') },
  { value: '1024x1536', label: t('images2.sizePortrait') },
]

const settings = computed(() => appStore.cachedPublicSettings)
const user = computed(() => authStore.user)
const pageTitle = computed(() => settings.value?.images2_page_title || 'ChatGPT Images 2 生图')
const noticeText = computed(() => settings.value?.images2_notice_text || t('images2.defaultNotice'))
const rechargePath = computed(() => settings.value?.images2_recharge_path || '/purchase')
const unitPrice = computed(() => settings.value?.images2_price_per_image ?? 0.5)
const canGenerate = computed(() => (user.value?.balance ?? 0) >= unitPrice.value)
const balanceText = computed(() => (user.value?.balance ?? 0).toFixed(2))
const priceText = computed(() => unitPrice.value.toFixed(2))
const rechargeCtaText = computed(() => canGenerate.value
  ? t('images2.goRecharge')
  : t('images2.insufficientRechargeCta'))

onMounted(async () => {
  await Promise.allSettled([appStore.fetchPublicSettings(), authStore.refreshUser()])
})

async function generateImage() {
  if (!prompt.value.trim() || isGenerating.value || !canGenerate.value) return
  isGenerating.value = true
  revisedPrompt.value = ''
  errorMessage.value = ''
  try {
    const result = imageUrl.value
      ? await images2API.edit(prompt.value.trim(), imageUrl.value, selectedSize.value)
      : await images2API.generate(prompt.value.trim(), selectedSize.value)
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

function resetCanvas() {
  prompt.value = ''
  imageUrl.value = ''
  revisedPrompt.value = ''
  errorMessage.value = ''
}

function goRecharge() {
  router.push(rechargePath.value)
}

async function downloadImage() {
  if (!imageUrl.value) return
  const filename = `chatgpt-images-2-${new Date().toISOString().replace(/[:.]/g, '-')}.png`
  const link = document.createElement('a')
  link.download = filename

  if (imageUrl.value.startsWith('data:')) {
    link.href = imageUrl.value
    link.click()
    return
  }

  try {
    const response = await fetch(imageUrl.value, { mode: 'cors' })
    if (!response.ok) throw new Error(`download failed: ${response.status}`)
    const blob = await response.blob()
    const objectUrl = URL.createObjectURL(blob)
    link.href = objectUrl
    link.click()
    URL.revokeObjectURL(objectUrl)
  } catch {
    link.href = imageUrl.value
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
  justify-content: flex-end;
  gap: 0.5rem;
}

.images2-size-option {
  appearance: none;
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

.images2-edit-hint,
.images2-result-hint {
  border-radius: 16px;
  border: 1px solid rgba(14, 165, 233, 0.2);
  background: rgba(240, 249, 255, 0.86);
  color: #0369a1;
  font-size: 0.9rem;
  line-height: 1.55;
}

.images2-edit-hint {
  margin-top: 0.9rem;
  padding: 0.75rem 0.9rem;
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

.images2-link {
  background: transparent;
  color: #fca5a5;
  padding: 0.4rem 0.2rem;
}

.images2-stage {
  margin-top: 1.25rem;
  min-height: 560px;
  padding: 1rem;
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
  margin-top: 0.9rem;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.images2-notice {
  margin: 0;
  color: #64748b;
  font-size: 0.92rem;
}

.images2-result-hint {
  margin-top: 0.9rem;
  padding: 0.8rem 0.95rem;
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

.dark .images2-edit-hint,
.dark .images2-result-hint {
  border-color: rgba(56, 189, 248, 0.22);
  background: rgba(8, 47, 73, 0.46);
  color: #bae6fd;
}

.dark .images2-secondary {
  background: rgba(255, 255, 255, 0.08);
  color: #e2e8f0;
}

.dark .images2-link {
  color: #fda4af;
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

  .images2-options-row {
    align-items: stretch;
    flex-direction: column;
  }

  .images2-size-options {
    display: grid;
    grid-template-columns: 1fr;
  }

  .images2-toolbar {
    display: grid;
    grid-template-columns: 1fr 1fr;
    align-items: center;
  }

  .images2-primary,
  .images2-secondary {
    width: 100%;
  }

  .images2-link {
    grid-column: 1 / -1;
    justify-self: center;
    margin-top: 0.15rem;
    white-space: nowrap;
  }

  .images2-stage,
  .images2-loader,
  .images2-empty {
    min-height: 360px;
  }
}
</style>
