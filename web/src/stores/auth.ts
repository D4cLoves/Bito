import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { loginUser, registerUser } from '../api/auth'
import type { LoginRequest, RegisterRequest, UserResponse } from '../types/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
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

  async function login(payload: LoginRequest) {
    isLoading.value = true
    error.value = null
    try {
      const data = await loginUser(payload)
      token.value = data.accessToken
      user.value = data.user
      localStorage.setItem('token', data.accessToken)
      localStorage.setItem('user', JSON.stringify(data.user))
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

  async function register(payload: RegisterRequest) {
    isLoading.value = true
    error.value = null
    try {
      const data = await registerUser(payload)
      token.value = data.accessToken
      user.value = data.user
      localStorage.setItem('token', data.accessToken)
      localStorage.setItem('user', JSON.stringify(data.user))
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

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }

  return {
    user,
    token,
    isLoading,
    error,
    isAuthenticated,
    clearError,
    login,
    register,
    logout,
  }
})
