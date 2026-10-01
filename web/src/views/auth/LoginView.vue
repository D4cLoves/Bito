<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'
import Message from 'primevue/message'

const router = useRouter()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const clientError = ref('')

async function handleSubmit() {
  clientError.value = ''
  authStore.clearError()

  if (!email.value.trim()) {
    clientError.value = 'Введите ваш email'
    return
  }
  if (!password.value) {
    clientError.value = 'Введите пароль'
    return
  }

  const success = await authStore.login({
    email: email.value.trim(),
    password: password.value,
  })

  if (success) {
    router.push('/lobby')
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-4 relative overflow-hidden bg-zinc-950">
    <!-- Ambient Aura Glow -->
    <div class="fixed inset-0 pointer-events-none flex items-center justify-center overflow-hidden">
      <div class="w-[600px] h-[600px] bg-emerald-500/10 rounded-full blur-[140px] -translate-y-28"></div>
      <div class="w-[450px] h-[450px] bg-amber-500/10 rounded-full blur-[120px] translate-x-40 translate-y-36"></div>
    </div>

    <div class="relative w-full max-w-md z-10">
      <!-- Outer Card -->
      <div class="bg-zinc-900/80 backdrop-blur-2xl border border-zinc-800/90 rounded-3xl p-8 sm:p-10 shadow-2xl shadow-black/90">
        <!-- Logo and Header -->
        <div class="text-center mb-8">
          <div class="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-zinc-950 border border-zinc-800 mb-4 shadow-inner ring-1 ring-amber-500/20">
            <i class="pi pi-sparkles text-2xl text-amber-400"></i>
          </div>
          <h1 class="text-3xl font-extrabold tracking-tight text-white m-0">
            BITO
          </h1>
          <p class="text-xs uppercase tracking-widest font-semibold text-amber-400/90 mt-1">
            Дурак Онлайн
          </p>
          <p class="text-zinc-400 text-sm mt-3">
            Войдите в аккаунт, чтобы продолжить игру
          </p>
        </div>

        <!-- Server or Client Error Alert -->
        <div v-if="clientError || authStore.error" class="mb-6">
          <Message severity="error" :closable="false" class="w-full">
            {{ clientError || authStore.error }}
          </Message>
        </div>

        <!-- Form -->
        <form @submit.prevent="handleSubmit" class="space-y-5">
          <!-- Email Input -->
          <div>
            <label class="block text-xs font-semibold text-zinc-300 mb-2 uppercase tracking-wider">
              Email
            </label>
            <div class="relative">
              <i class="pi pi-envelope absolute left-3.5 top-1/2 -translate-y-1/2 text-zinc-500 pointer-events-none z-10"></i>
              <InputText
                v-model="email"
                type="email"
                placeholder="player@bito.local"
                class="w-full pl-10! bg-zinc-950/70! border-zinc-800! text-zinc-100! placeholder:text-zinc-600! focus:border-amber-500! rounded-xl!"
              />
            </div>
          </div>

          <!-- Password Input -->
          <div>
            <label class="block text-xs font-semibold text-zinc-300 mb-2 uppercase tracking-wider">
              Пароль
            </label>
            <div class="relative">
              <i class="pi pi-lock absolute left-3.5 top-1/2 -translate-y-1/2 text-zinc-500 pointer-events-none z-10"></i>
              <Password
                v-model="password"
                :feedback="false"
                toggleMask
                placeholder="••••••••"
                class="w-full"
                inputClass="w-full pl-10! bg-zinc-950/70! border-zinc-800! text-zinc-100! placeholder:text-zinc-600! focus:border-amber-500! rounded-xl!"
              />
            </div>
          </div>

          <!-- Submit Button -->
          <div class="pt-2">
            <Button
              type="submit"
              label="Войти в игру"
              icon="pi pi-sign-in"
              :loading="authStore.isLoading"
              class="w-full py-3.5! rounded-xl! font-bold! bg-amber-500! hover:bg-amber-400! border-amber-500! text-zinc-950! shadow-lg shadow-amber-500/20! transition duration-200"
            />
          </div>
        </form>

        <!-- Footer Navigation -->
        <div class="mt-8 pt-6 border-t border-zinc-800/80 text-center text-sm text-zinc-400">
          Еще нет аккаунта?
          <router-link
            to="/register"
            class="ml-1 text-amber-400 hover:text-amber-300 font-semibold transition hover:underline"
          >
            Создать профиль
          </router-link>
        </div>
      </div>
    </div>
  </div>
</template>
