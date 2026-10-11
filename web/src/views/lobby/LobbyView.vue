<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppHeader from '../../components/app/AppHeader.vue'
import ToastStack from '../../components/app/ToastStack.vue'
import ProfileCard from '../../components/lobby/ProfileCard.vue'
import QuickPlay from '../../components/lobby/QuickPlay.vue'
import TablesList from '../../components/lobby/TablesList.vue'
import Leaderboard from '../../components/lobby/Leaderboard.vue'
import ProfileDrawer from '../../components/lobby/ProfileDrawer.vue'
import { createTable, listTables, quickMatch } from '../../api/tables'
import { getProfile } from '../../api/users'
import { isNotImplemented } from '../../api/http'
import { useAuthStore } from '../../stores/auth'
import { useGameSound } from '../../composables/useGameSound'
import { useToast } from '../../composables/useToast'
import type { DeckSize, GameMode, TableInfo } from '../../types/game'
import type { Profile } from '../../types/user'

const POLL_MS = 3000

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const sound = useGameSound()
const toast = useToast()

const profile = ref<Profile | null>(null)
const profileError = ref('')

const mode = ref<GameMode>('podkidnoy')
const deck = ref<DeckSize>(36)
const players = ref(2)
const busy = ref<'find' | 'create' | null>(null)

const tables = ref<TableInfo[]>([])
const tablesLoaded = ref(false)
const tablesError = ref('')
const refreshing = ref(false)
const now = ref(Date.now())

let pollTimer: number | null = null

const drawerOpen = computed(() => route.query.panel === 'profile')
const firstName = computed(() => profile.value?.username ?? auth.user?.username ?? 'игрок')

const greeting = computed(() => {
  const h = new Date(now.value).getHours()
  if (h < 6) return 'Доброй ночи'
  if (h < 12) return 'Доброе утро'
  if (h < 18) return 'Добрый день'
  return 'Добрый вечер'
})

function openProfile() {
  router.replace({ query: { ...route.query, panel: 'profile' } })
}

function closeProfile() {
  const { panel: _panel, ...rest } = route.query
  router.replace({ query: rest })
}

async function loadProfile() {
  profileError.value = ''
  try {
    profile.value = await getProfile()
  } catch (err) {
    profileError.value = err instanceof Error ? err.message : 'Не удалось загрузить профиль'
    if (!auth.isAuthenticated) router.push({ name: 'landing', query: { auth: 'login' } })
  }
}

async function refreshTables(manual = false) {
  if (manual) refreshing.value = true
  try {
    tables.value = await listTables()
    tablesError.value = ''
  } catch (err) {
    tablesError.value = err instanceof Error ? err.message : 'Не удалось загрузить столы'
    if (!auth.isAuthenticated) router.push({ name: 'landing', query: { auth: 'login' } })
  } finally {
    tablesLoaded.value = true
    refreshing.value = false
    now.value = Date.now()
  }
}

function goToTable(id: string) {
  sound.playCardDeal()
  router.push({ name: 'table', params: { id } })
}

async function create() {
  if (busy.value) return
  sound.playClick()
  busy.value = 'create'
  try {
    const table = await createTable({ mode: mode.value, deck: deck.value, maxPlayers: players.value })
    goToTable(table.id)
  } catch (err) {
    sound.playError()
    toast.show(err instanceof Error ? err.message : 'Не удалось создать стол', 'error')
  } finally {
    busy.value = null
  }
}

async function find() {
  if (busy.value) return
  sound.playClick()
  busy.value = 'find'
  try {
    const table = await quickMatch({ mode: mode.value, deck: deck.value, maxPlayers: players.value })
    goToTable(table.id)
  } catch (err) {
    sound.playError()
    if (isNotImplemented(err)) toast.show('Быстрый поиск скоро заработает. Пока можно создать свой стол.')
    else toast.show(err instanceof Error ? err.message : 'Не удалось найти стол', 'error')
  } finally {
    busy.value = null
  }
}

function onRenamed(username: string) {
  if (profile.value) profile.value = { ...profile.value, username }
}

function onVisibility() {
  if (document.visibilityState === 'visible') refreshTables()
}

onMounted(() => {
  loadProfile()
  refreshTables()
  pollTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') refreshTables()
  }, POLL_MS)
  document.addEventListener('visibilitychange', onVisibility)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  document.removeEventListener('visibilitychange', onVisibility)
})
</script>

<template>
  <div class="lobby">
    <AppHeader />

    <main class="wrap">
      <section class="intro" style="--d: 0">
        <p class="eyebrow">Лобби</p>
        <h1>{{ greeting }}, <span class="name">{{ firstName }}</span>.</h1>
        <p class="lead">Сыграйте партию, загляните в профиль или подсядьте к тем, кто уже ждёт.</p>
      </section>

      <div class="grid">
        <aside class="side">
          <ProfileCard
            class="enter"
            style="--d: 1"
            :profile="profile"
            :error="profileError"
            @open-settings="openProfile"
            @retry="loadProfile"
          />
          <Leaderboard class="enter" style="--d: 3" :my-id="auth.user?.id" />
        </aside>

        <div class="main">
          <QuickPlay
            v-model:mode="mode"
            v-model:deck="deck"
            v-model:players="players"
            class="enter"
            style="--d: 2"
            :busy="busy"
            :open-count="tables.length"
            @find="find"
            @create="create"
          />
          <TablesList
            class="enter"
            style="--d: 3"
            :tables="tables"
            :loaded="tablesLoaded"
            :refreshing="refreshing"
            :error="tablesError"
            :my-id="auth.user?.id"
            :now="now"
            @join="(t) => goToTable(t.id)"
            @refresh="refreshTables(true)"
          />
        </div>
      </div>
    </main>

    <ProfileDrawer :open="drawerOpen" :profile="profile" @close="closeProfile" @renamed="onRenamed" />
    <ToastStack />
  </div>
</template>

<style scoped>
.lobby {
  min-height: 100vh;
  background:
    radial-gradient(ellipse 80% 50% at 50% -10%, rgba(246, 235, 211, 0.75), rgba(246, 235, 211, 0) 70%),
    var(--color-paper);
  color: var(--color-ink);
}

.wrap {
  max-width: 1120px;
  margin: 0 auto;
  padding: 44px 24px 80px;
}

.intro,
.enter {
  animation: enter 0.7s cubic-bezier(0.2, 0.8, 0.2, 1) both;
  animation-delay: calc(var(--d, 0) * 90ms);
}

.eyebrow {
  margin: 0 0 10px;
  font-size: 13px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-ink-faint);
}

h1 {
  margin: 0;
  font-size: clamp(34px, 5vw, 56px);
  font-weight: 500;
  letter-spacing: -0.035em;
  line-height: 1.04;
}

.name {
  font-family: var(--font-card);
  font-style: italic;
  font-weight: 400;
}

.lead {
  margin: 14px 0 0;
  max-width: 540px;
  font-size: 17px;
  line-height: 1.5;
  color: var(--color-ink-muted);
}

.grid {
  display: grid;
  grid-template-columns: minmax(0, 340px) minmax(0, 1fr);
  gap: 20px;
  margin-top: 36px;
  align-items: start;
}

.side,
.main {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-width: 0;
}

.side {
  position: sticky;
  top: 92px;
}

@keyframes enter {
  from {
    opacity: 0;
    transform: translateY(18px);
  }
}

@media (max-width: 920px) {
  .grid {
    display: flex;
    flex-direction: column;
  }

  .side,
  .main {
    display: contents;
  }

  .profile-card {
    order: 1;
  }

  .quick {
    order: 2;
  }

  .tables {
    order: 3;
  }

  .board {
    order: 4;
  }
}

@media (max-width: 520px) {
  .wrap {
    padding: 32px 16px 64px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .intro,
  .enter {
    animation: none;
  }
}
</style>
