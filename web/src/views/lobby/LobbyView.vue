<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import AppHeader from '../../components/app/AppHeader.vue'
import ToastStack from '../../components/app/ToastStack.vue'
import LobbyStage from '../../components/lobby/LobbyStage.vue'
import WaitingTables from '../../components/lobby/WaitingTables.vue'
import Leaderboard from '../../components/lobby/Leaderboard.vue'
import { createTable, listTables, quickMatch } from '../../api/tables'
import { isNotImplemented } from '../../api/http'
import { useAuthStore } from '../../stores/auth'
import { useProfileStore } from '../../stores/profile'
import { useGameSound } from '../../composables/useGameSound'
import { useToast } from '../../composables/useToast'
import type { DeckSize, GameMode, TableInfo } from '../../types/game'

const POLL_MS = 3000

const router = useRouter()
const auth = useAuthStore()
const profileStore = useProfileStore()
const sound = useGameSound()
const toast = useToast()

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

const username = computed(() => profileStore.profile?.username ?? auth.user?.username ?? '?')

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
    if (isNotImplemented(err)) toast.show('Быстрый поиск скоро заработает — пока соберите свой стол')
    else toast.show(err instanceof Error ? err.message : 'Не удалось найти стол', 'error')
  } finally {
    busy.value = null
  }
}

function onVisibility() {
  if (document.visibilityState === 'visible') refreshTables()
}

onMounted(() => {
  profileStore.load()
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
      <LobbyStage
        v-model:mode="mode"
        v-model:deck="deck"
        v-model:players="players"
        :busy="busy"
        :username="username"
        @find="find"
        @create="create"
      />

      <div class="below">
        <div class="tables-col rise" style="--d: 6">
          <WaitingTables
            :tables="tables"
            :loaded="tablesLoaded"
            :refreshing="refreshing"
            :error="tablesError"
            :my-id="auth.user?.id"
            :now="now"
            @join="(t) => goToTable(t.id)"
            @refresh="refreshTables(true)"
            @create="create"
          />
        </div>

        <aside class="leaders-col rise" style="--d: 7">
          <Leaderboard :my-id="auth.user?.id" />
        </aside>
      </div>
    </main>

    <ToastStack />
  </div>
</template>

<style scoped>
.lobby {
  min-height: 100vh;
  background: var(--color-paper);
  color: var(--color-ink);
  overflow-x: hidden;
}

.wrap {
  max-width: 1180px;
  margin: 0 auto;
  padding: 8px 24px 80px;
}

.below {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 300px;
  gap: 28px;
  align-items: start;
  margin-top: 28px;
}

.tables-col {
  min-width: 0;
}

.leaders-col {
  margin-top: 62px;
}

@media (max-width: 960px) {
  .below {
    grid-template-columns: minmax(0, 1fr);
  }

  .leaders-col {
    margin-top: 0;
  }
}

@media (max-width: 520px) {
  .wrap {
    padding: 0 14px 64px;
  }
}
</style>
