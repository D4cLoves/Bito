import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getProfile } from '../api/users'
import type { Profile } from '../types/user'

export const useProfileStore = defineStore('profile', () => {
  const profile = ref<Profile | null>(null)
  const loading = ref(false)
  const error = ref('')

  async function load(force = false) {
    if (loading.value || (profile.value && !force)) return
    loading.value = true
    error.value = ''
    try {
      profile.value = await getProfile()
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Не удалось загрузить профиль'
    } finally {
      loading.value = false
    }
  }

  function setUsername(username: string) {
    if (profile.value) profile.value = { ...profile.value, username }
  }

  function reset() {
    profile.value = null
    error.value = ''
  }

  return { profile, loading, error, load, setUsername, reset }
})
