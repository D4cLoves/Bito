import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { loginUser, refreshAccessToken, revokeSession, sendVerificationCode, verifyCodeAndRegister } from '../api/auth'
import { ApiError } from '../api/errors'
import { expiresSoon } from '../api/token'
import type { LoginRequest, RegisterRequest, VerifyCodeRequest, UserResponse } from '../types/auth'

const SESSION_KEYS = ['token', 'refreshToken', 'user']

function storedUser(): UserResponse | null {
  const saved = localStorage.getItem('user')
  if (!saved) return null
  try {
    return JSON.parse(saved) as UserResponse
  } catch {
    localStorage.removeItem('user')
    return null
  }
}

function withRefreshLock<T>(task: () => Promise<T>): Promise<T> {
  if (typeof navigator !== 'undefined' && navigator.locks) {
    return navigator.locks.request('bito-session-refresh', task)
  }
  return task()
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const refreshToken = ref<string | null>(localStorage.getItem('refreshToken'))
  const user = ref<UserResponse | null>(storedUser())
  const registrationTicket = ref<string | null>(null)

  function adoptStoredSession() {
    token.value = localStorage.getItem('token')
    refreshToken.value = localStorage.getItem('refreshToken')
    user.value = storedUser()
  }

  window.addEventListener('storage', (e) => {
    if (e.key === null || SESSION_KEYS.includes(e.key)) adoptStoredSession()
  })

  const isLoading = ref(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!token.value)

  function clearError() {
    error.value = null
  }

  function persistSession(data: { accessToken: string; refreshToken: string; user: UserResponse }) {
    token.value = data.accessToken
    refreshToken.value = data.refreshToken
    user.value = data.user
    localStorage.setItem('token', data.accessToken)
    localStorage.setItem('refreshToken', data.refreshToken)
    localStorage.setItem('user', JSON.stringify(data.user))
  }

  async function login(payload: LoginRequest): Promise<boolean> {
    isLoading.value = true
    error.value = null
    try {
      const data = await loginUser(payload)
      persistSession(data)
      return true
    } catch (err: unknown) {
      if (err instanceof Error) {
        error.value = err.message
      } else {
        error.value = 'Неизвестная ошибка'
      }
      return false
    } finally {
      isLoading.value = false
    }
  }

  async function requestCode(payload: RegisterRequest): Promise<boolean> {
    isLoading.value = true
    error.value = null
    try {
      const { ticket } = await sendVerificationCode(payload)
      registrationTicket.value = ticket
      return true
    } catch (err: unknown) {
      if (err instanceof Error) {
        error.value = err.message
      } else {
        error.value = 'Неизвестная ошибка'
      }
      return false
    } finally {
      isLoading.value = false
    }
  }

  async function verifyAndRegister(payload: Omit<VerifyCodeRequest, 'ticket'>): Promise<boolean> {
    isLoading.value = true
    error.value = null
    try {
      const data = await verifyCodeAndRegister({ ...payload, ticket: registrationTicket.value ?? '' })
      registrationTicket.value = null
      persistSession(data)
      return true
    } catch (err: unknown) {
      if (err instanceof Error) {
        error.value = err.message
      } else {
        error.value = 'Неизвестная ошибка'
      }
      return false
    } finally {
      isLoading.value = false
    }
  }

  let refreshing: Promise<boolean> | null = null

  function rotateSession(): Promise<boolean> {
    const tokenBefore = token.value
    return withRefreshLock(async () => {
      adoptStoredSession()
      if (token.value && token.value !== tokenBefore && !expiresSoon(token.value)) return true

      const used = refreshToken.value
      if (!used) return false
      try {
        persistSession(await refreshAccessToken(used))
        return true
      } catch (err) {
        if (!(err instanceof ApiError) || (err.status !== 400 && err.status !== 401)) return false
        adoptStoredSession()
        if (refreshToken.value && refreshToken.value !== used) return true
        forget()
        return false
      }
    })
  }

  function refreshSession(): Promise<boolean> {
    refreshing ??= rotateSession().finally(() => {
      refreshing = null
    })
    return refreshing
  }

  function setUser(next: UserResponse) {
    user.value = next
    localStorage.setItem('user', JSON.stringify(next))
  }

  function logout() {
    revokeSession(refreshToken.value)
    forget()
  }

  function forget() {
    token.value = null
    refreshToken.value = null
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('refreshToken')
    localStorage.removeItem('user')
  }

  return {
    user,
    token,
    refreshToken,
    isLoading,
    error,
    isAuthenticated,
    clearError,
    login,
    requestCode,
    verifyAndRegister,
    refreshSession,
    setUser,
    logout,
  }
})
