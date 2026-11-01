<script setup lang="ts">
import { watch } from 'vue'
import { RouterView, useRoute, useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'

const auth = useAuthStore()
const route = useRoute()
const router = useRouter()

watch(
  () => auth.isAuthenticated,
  (signedIn) => {
    if (!signedIn && route.meta.requiresAuth) router.replace({ name: 'landing', query: { auth: 'login' } })
  },
)
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <RouterView :key="$route.path" />
  </div>
</template>
