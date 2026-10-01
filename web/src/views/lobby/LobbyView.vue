<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '../../stores/auth'
import { LogOut, Coins, Trophy, Users, Play } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-screen p-6 max-w-6xl mx-auto flex flex-col">
    <!-- Header -->
    <header class="flex items-center justify-between py-4 border-b border-slate-800/80 mb-8">
      <div class="flex items-center gap-3">
        <span class="text-3xl">🃏</span>
        <div>
          <h1 class="text-xl font-bold text-white tracking-wide m-0">BITO LOBBY</h1>
          <p class="text-xs text-amber-500 font-semibold uppercase tracking-wider">Дурак Онлайн</p>
        </div>
      </div>

      <div class="flex items-center gap-4">
        <!-- Balance Badge -->
        <div class="flex items-center gap-2 bg-slate-900/90 border border-amber-500/30 px-3.5 py-1.5 rounded-full text-amber-400 font-mono text-sm font-semibold shadow-inner">
          <Coins class="w-4 h-4 text-amber-400" />
          <span>2,500</span>
        </div>

        <!-- User Info -->
        <div class="text-right hidden sm:block">
          <div class="text-sm font-bold text-white">{{ authStore.user?.username || 'Игрок' }}</div>
          <div class="text-xs text-slate-400 font-mono">{{ authStore.user?.email }}</div>
        </div>

        <!-- Logout Button -->
        <button
          @click="handleLogout"
          class="p-2 text-slate-400 hover:text-rose-400 hover:bg-slate-800/60 rounded-xl transition cursor-pointer"
          title="Выйти"
        >
          <LogOut class="w-5 h-5" />
        </button>
      </div>
    </header>

    <!-- Lobby Content -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 flex-1 items-start">
      <!-- Quick Play Card -->
      <div class="md:col-span-2 bg-slate-900/60 border border-slate-800 rounded-3xl p-8 backdrop-blur-md">
        <h2 class="text-2xl font-bold text-white mb-2">Быстрая игра</h2>
        <p class="text-slate-400 text-sm mb-6">Автоматический подбор соперника 1х1 по вашему рейтингу ELO</p>
        
        <div class="grid grid-cols-2 sm:grid-cols-3 gap-4 mb-6">
          <button v-for="bet in [500, 1000, 2500, 5000, 10000]" :key="bet" class="p-4 bg-slate-950/70 border border-slate-800 hover:border-amber-500/50 rounded-2xl text-center transition group cursor-pointer">
            <div class="text-xs text-slate-400 uppercase tracking-wider mb-1">Ставка</div>
            <div class="text-lg font-bold font-mono text-amber-400 group-hover:scale-105 transition-transform flex items-center justify-center gap-1">
              <Coins class="w-4 h-4" />
              {{ bet }}
            </div>
          </button>
        </div>

        <button class="w-full py-4 bg-gradient-to-r from-amber-500 to-amber-600 hover:from-amber-400 hover:to-amber-500 text-slate-950 font-extrabold rounded-2xl shadow-xl shadow-amber-500/20 text-lg flex items-center justify-center gap-2 cursor-pointer transition">
          <Play class="w-6 h-6 fill-slate-950" />
          <span>Найти игру</span>
        </button>
      </div>

      <!-- Stats / Right Column -->
      <div class="space-y-6">
        <div class="bg-slate-900/60 border border-slate-800 rounded-3xl p-6 backdrop-blur-md">
          <div class="flex items-center gap-3 mb-4">
            <Trophy class="w-5 h-5 text-amber-400" />
            <h3 class="text-lg font-bold text-white m-0">Ваш рейтинг</h3>
          </div>
          <div class="text-3xl font-extrabold font-mono text-white mb-1">1000 <span class="text-sm font-sans text-slate-400 font-normal">ELO</span></div>
          <p class="text-xs text-slate-400">Начальный дивизион. Проведите 30 игр для калибровки.</p>
        </div>

        <div class="bg-slate-900/60 border border-slate-800 rounded-3xl p-6 backdrop-blur-md">
          <div class="flex items-center gap-3 mb-4">
            <Users class="w-5 h-5 text-emerald-400" />
            <h3 class="text-lg font-bold text-white m-0">Онлайн</h3>
          </div>
          <div class="flex items-center gap-2">
            <span class="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-pulse"></span>
            <span class="text-slate-300 text-sm font-medium">Сервер активен</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
