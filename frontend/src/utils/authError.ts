interface APIErrorLike {
  message?: string
  response?: {
    data?: {
      detail?: string
      message?: string
      reason?: string
    }
  }
}

function extractErrorMessage(error: unknown): string {
  const err = (error || {}) as APIErrorLike
  return err.response?.data?.detail || err.response?.data?.message || err.message || ''
}

function extractErrorReason(error: unknown): string {
  const err = (error || {}) as APIErrorLike
  return err.response?.data?.reason || ''
}

export function buildAuthErrorMessage(
  error: unknown,
  options: {
    fallback: string
    t?: (key: string) => string
  }
): string {
  const { fallback, t } = options
  const reason = extractErrorReason(error)
  if (t && reason) {
    switch (reason) {
      case 'INVALID_CREDENTIALS':
        return t('auth.invalidCredentials')
      case 'EMAIL_EXISTS':
        return t('auth.emailExists')
      case 'USERNAME_EXISTS':
        return t('auth.usernameExists')
      case 'USERNAME_INVALID':
        return t('auth.invalidUsername')
    }
  }
  const message = extractErrorMessage(error)
  return message || fallback
}
