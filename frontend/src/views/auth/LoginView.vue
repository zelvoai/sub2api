<template>
  <AuthLayout>
    <div class="space-y-6">
      <!-- Title -->
      <div class="text-center">
        <h2 class="text-2xl font-bold text-gray-900 dark:text-white">
          {{ t('auth.welcomeBack') }}
        </h2>
        <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ t('auth.signInToAccount') }}
        </p>
      </div>

  <div v-if="!backendModeEnabled && (linuxdoOAuthEnabled || wechatOAuthEnabled || oidcOAuthEnabled)" class="space-y-4">
        <LinuxDoOAuthSection
          v-if="linuxdoOAuthEnabled"
          :disabled="isLoading"
          :show-divider="false"
        />
        <WechatOAuthSection
          v-if="wechatOAuthEnabled"
          :disabled="isLoading"
          :show-divider="false"
        />
        <OidcOAuthSection
          v-if="oidcOAuthEnabled"
          :disabled="isLoading"
          :provider-name="oidcOAuthProviderName"
          :show-divider="false"
        />
        <div class="flex items-center gap-3">
          <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
          <span class="text-xs text-gray-500 dark:text-dark-400">
            {{ t('auth.oauthOrContinue') }}
          </span>
          <div class="h-px flex-1 bg-gray-200 dark:bg-dark-700"></div>
        </div>
      </div>

      <!-- Account Type Tabs -->
      <div class="flex rounded-xl bg-gray-100 p-1 dark:bg-dark-800">
        <button
          type="button"
          class="flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all"
          :class="accountType === 'username'
            ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
            : 'text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-300'"
          @click="accountType = 'username'"
        >
          {{ t('auth.usernameLogin') }}
        </button>
        <button
          type="button"
          class="flex-1 rounded-lg px-4 py-2 text-sm font-medium transition-all"
          :class="accountType === 'email'
            ? 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
            : 'text-gray-500 hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-300'"
          @click="accountType = 'email'"
        >
          {{ t('auth.emailLogin') }}
        </button>
      </div>

      <!-- Login Form -->
      <form @submit.prevent="handleLogin" class="space-y-5">
        <!-- Username Input -->
        <div v-if="accountType === 'username'">
          <label for="username" class="input-label">
            {{ t('auth.usernameLabel') }}
          </label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="user" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="username"
              v-model="formData.username"
              type="text"
              required
              autofocus
              autocomplete="username"
              :disabled="isLoading"
              class="input pl-11"
              :class="{ 'input-error': errors.username }"
              :placeholder="t('auth.usernamePlaceholder')"
            />
          </div>
        </div>

        <!-- Email Input -->
        <div v-if="accountType === 'email'">
          <label for="email" class="input-label">
            {{ t('auth.emailLabel') }}
          </label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="mail" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="email"
              v-model="formData.email"
              type="email"
              required
              autofocus
              autocomplete="email"
              :disabled="isLoading"
              class="input pl-11"
              :class="{ 'input-error': errors.email }"
              :placeholder="t('auth.emailPlaceholder')"
            />
          </div>
        </div>

        <!-- Password Input -->
        <div>
          <label for="password" class="input-label">
            {{ t('auth.passwordLabel') }}
          </label>
          <div class="relative">
            <div class="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3.5">
              <Icon name="lock" size="md" class="text-gray-400 dark:text-dark-500" />
            </div>
            <input
              id="password"
              v-model="formData.password"
              :type="showPassword ? 'text' : 'password'"
              required
              autocomplete="current-password"
              :disabled="isLoading"
              class="input pl-11 pr-11"
              :class="{ 'input-error': errors.password }"
              :placeholder="t('auth.passwordPlaceholder')"
            />
            <button
              type="button"
              @click="showPassword = !showPassword"
              class="absolute inset-y-0 right-0 flex items-center pr-3.5 text-gray-400 transition-colors hover:text-gray-600 dark:hover:text-dark-300"
            >
              <Icon v-if="showPassword" name="eyeOff" size="md" />
              <Icon v-else name="eye" size="md" />
            </button>
          </div>
          <div class="mt-1 flex items-center justify-between">
            <span></span>
            <router-link
              v-if="passwordResetEnabled && !backendModeEnabled && accountType === 'email'"
              to="/forgot-password"
              class="text-sm font-medium text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
            >
              {{ t('auth.forgotPassword') }}
            </router-link>
          </div>
        </div>

        <!-- Turnstile Widget -->
        <div v-if="turnstileEnabled && turnstileSiteKey">
          <TurnstileWidget
            ref="turnstileRef"
            :site-key="turnstileSiteKey"
            @verify="onTurnstileVerify"
            @expire="onTurnstileExpire"
            @error="onTurnstileError"
          />
        </div>

        <!-- Submit Button -->
        <button
          type="submit"
          :disabled="isLoading || (turnstileEnabled && !turnstileToken)"
          class="btn btn-primary w-full"
        >
          <svg
            v-if="isLoading"
            class="-ml-1 mr-2 h-4 w-4 animate-spin text-white"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              class="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              stroke-width="4"
            ></circle>
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            ></path>
          </svg>
          <Icon v-else name="login" size="md" class="mr-2" />
          {{ isLoading ? t('auth.signingIn') : t('auth.signIn') }}
        </button>
      </form>
    </div>

    <!-- Footer -->
    <template v-if="!backendModeEnabled" #footer>
      <p class="text-gray-500 dark:text-dark-400">
        {{ t('auth.dontHaveAccount') }}
        <router-link
          to="/register"
          class="font-medium text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
        >
          {{ t('auth.signUp') }}
        </router-link>
      </p>
    </template>
  </AuthLayout>

  <!-- 2FA Modal -->
  <TotpLoginModal
    v-if="show2FAModal"
    ref="totpModalRef"
    :temp-token="totpTempToken"
    :user-email-masked="totpUserEmailMasked"
    @verify="handle2FAVerify"
    @cancel="handle2FACancel"
  />
</template>

<script setup lang="ts">
import { computed, ref, reactive, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AuthLayout } from '@/components/layout'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'
import OidcOAuthSection from '@/components/auth/OidcOAuthSection.vue'
import WechatOAuthSection from '@/components/auth/WechatOAuthSection.vue'
import TotpLoginModal from '@/components/auth/TotpLoginModal.vue'
import Icon from '@/components/icons/Icon.vue'
import TurnstileWidget from '@/components/TurnstileWidget.vue'
import { useAuthStore, useAppStore } from '@/stores'
import { getPublicSettings, isTotp2FARequired, isWeChatWebOAuthEnabled } from '@/api/auth'
import { buildAuthErrorMessage } from '@/utils/authError'
import type { AuthAccountType, TotpLoginResponse } from '@/types'

const { t } = useI18n()

const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()

const isLoading = ref<boolean>(false)
const errorMessage = ref<string>('')
const showPassword = ref<boolean>(false)
const accountType = ref<AuthAccountType>('username')

const turnstileEnabled = ref<boolean>(false)
const turnstileSiteKey = ref<string>('')
const linuxdoOAuthEnabled = ref<boolean>(false)
const wechatOAuthEnabled = ref<boolean>(false)
const backendModeEnabled = ref<boolean>(false)
const oidcOAuthEnabled = ref<boolean>(false)
const oidcOAuthProviderName = ref<string>('OIDC')
const passwordResetEnabled = ref<boolean>(false)

const turnstileRef = ref<InstanceType<typeof TurnstileWidget> | null>(null)
const turnstileToken = ref<string>('')

const show2FAModal = ref<boolean>(false)
const totpTempToken = ref<string>('')
const totpUserEmailMasked = ref<string>('')
const totpModalRef = ref<InstanceType<typeof TotpLoginModal> | null>(null)

const formData = reactive({
  email: '',
  username: '',
  password: ''
})

const errors = reactive({
  email: '',
  username: '',
  password: '',
  turnstile: ''
})

const validationToastMessage = computed(
  () => errors.email || errors.username || errors.password || errors.turnstile || ''
)

watch(validationToastMessage, (value, previousValue) => {
  if (value && value !== previousValue) {
    appStore.showError(value)
  }
})

onMounted(async () => {
  const expiredFlag = sessionStorage.getItem('auth_expired')
  if (expiredFlag) {
    sessionStorage.removeItem('auth_expired')
    const message = t('auth.reloginRequired')
    errorMessage.value = message
    appStore.showWarning(message)
  }

  try {
    const settings = await getPublicSettings()
    turnstileEnabled.value = settings.turnstile_enabled
    turnstileSiteKey.value = settings.turnstile_site_key || ''
    linuxdoOAuthEnabled.value = settings.linuxdo_oauth_enabled
    wechatOAuthEnabled.value = isWeChatWebOAuthEnabled(settings)
    backendModeEnabled.value = settings.backend_mode_enabled
    oidcOAuthEnabled.value = settings.oidc_oauth_enabled
    oidcOAuthProviderName.value = settings.oidc_oauth_provider_name || 'OIDC'
    backendModeEnabled.value = settings.backend_mode_enabled
    passwordResetEnabled.value = settings.password_reset_enabled
  } catch (error) {
    console.error('Failed to load public settings:', error)
  }
})

function onTurnstileVerify(token: string): void {
  turnstileToken.value = token
  errors.turnstile = ''
}

function onTurnstileExpire(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileExpired')
}

function onTurnstileError(): void {
  turnstileToken.value = ''
  errors.turnstile = t('auth.turnstileFailed')
}

function validateForm(): boolean {
  errors.email = ''
  errors.username = ''
  errors.password = ''
  errors.turnstile = ''

  let isValid = true

  if (accountType.value === 'email') {
    if (!formData.email.trim()) {
      errors.email = t('auth.emailRequired')
      isValid = false
    } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) {
      errors.email = t('auth.invalidEmail')
      isValid = false
    }
  } else {
    if (!formData.username.trim()) {
      errors.username = t('auth.usernameRequired')
      isValid = false
    } else if (!/^[\u4e00-\u9fffA-Za-z0-9_]{2,32}$/.test(formData.username.trim())) {
      errors.username = t('auth.invalidUsername')
      isValid = false
    }
  }

  if (!formData.password) {
    errors.password = t('auth.passwordRequired')
    isValid = false
  } else if (formData.password.length < 6) {
    errors.password = t('auth.passwordMinLength')
    isValid = false
  }

  if (turnstileEnabled.value && !turnstileToken.value) {
    errors.turnstile = t('auth.completeVerification')
    isValid = false
  }

  return isValid
}

async function handleLogin(): Promise<void> {
  errorMessage.value = ''

  if (!validateForm()) {
    return
  }

  isLoading.value = true

  try {
    const response = await authStore.login({
      account_type: accountType.value,
      email: accountType.value === 'email' ? formData.email : undefined,
      username: accountType.value === 'username' ? formData.username : undefined,
      password: formData.password,
      turnstile_token: turnstileEnabled.value ? turnstileToken.value : undefined
    })

    if (isTotp2FARequired(response)) {
      const totpResponse = response as TotpLoginResponse
      totpTempToken.value = totpResponse.temp_token || ''
      totpUserEmailMasked.value = totpResponse.user_email_masked || ''
      show2FAModal.value = true
      isLoading.value = false
      return
    }

    appStore.showSuccess(t('auth.loginSuccess'))

    const redirectTo = (router.currentRoute.value.query.redirect as string) || '/dashboard'
    await router.push(redirectTo)
  } catch (error: unknown) {
    if (turnstileRef.value) {
      turnstileRef.value.reset()
      turnstileToken.value = ''
    }

    errorMessage.value = buildAuthErrorMessage(error, {
      fallback: t('auth.loginFailed'),
      t,
    })

    appStore.showError(errorMessage.value)
  } finally {
    isLoading.value = false
  }
}

async function handle2FAVerify(code: string): Promise<void> {
  if (totpModalRef.value) {
    totpModalRef.value.setVerifying(true)
  }

  try {
    await authStore.login2FA(totpTempToken.value, code)

    show2FAModal.value = false
    appStore.showSuccess(t('auth.loginSuccess'))

    const redirectTo = (router.currentRoute.value.query.redirect as string) || '/dashboard'
    await router.push(redirectTo)
  } catch (error: unknown) {
    const err = error as { message?: string; response?: { data?: { message?: string } } }
    const message = err.response?.data?.message || err.message || t('profile.totp.loginFailed')

    if (totpModalRef.value) {
      totpModalRef.value.setError(message)
      totpModalRef.value.setVerifying(false)
    }
  }
}

function handle2FACancel(): void {
  show2FAModal.value = false
  totpTempToken.value = ''
  totpUserEmailMasked.value = ''
}
</script>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: all 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
