import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { loginUser, refreshAccessToken, sendVerificationCode, verifyCodeAndRegister } from '../api/auth'
import type { LoginRequest, RegisterRequest, VerifyCodeRequest, UserResponse } from '../types/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const refreshToken = ref<string | null>(localStorage.getItem('refreshToken'))
  const user = ref<UserResponse | null>(null)
  
  const savedUser = localStorage.getItem('user')
  if (savedUser) {
    try {
      user.value = JSON.parse(savedUser) as UserResponse
    } catch {
      localStorage.removeItem('user')
    }
  }

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
      await sendVerificationCode(payload)
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

  async function verifyAndRegister(payload: VerifyCodeRequest): Promise<boolean> {
    isLoading.value = true
    error.value = null
    try {
      const data = await verifyCodeAndRegister(payload)
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

  async function refreshSession(): Promise<boolean> {
    if (!refreshToken.value) return false
    try {
      const data = await refreshAccessToken(refreshToken.value)
      persistSession(data)
      return true
    } catch {
      logout()
      return false
    }
  }

  function logout() {
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
    logout,
  }
})
