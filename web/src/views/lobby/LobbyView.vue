<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { AnimatePresence } from 'motion-v'
import ToastStack from '../../components/app/ToastStack.vue'
import NavRail from '../../components/client/NavRail.vue'
import TopBar from '../../components/client/TopBar.vue'
import ReturnBanner from '../../components/client/ReturnBanner.vue'
import PlayPanel from '../../components/client/PlayPanel.vue'
import TablesBoard from '../../components/client/TablesBoard.vue'
import SocialPanel from '../../components/client/SocialPanel.vue'
import OverlayPanel from '../../components/client/OverlayPanel.vue'
import ProfilePanel from '../../components/panels/ProfilePanel.vue'
import SettingsPanel from '../../components/panels/SettingsPanel.vue'
import ShopPanel from '../../components/panels/ShopPanel.vue'
import Leaderboard from '../../components/lobby/Leaderboard.vue'
import { createTable, listTables, quickMatch } from '../../api/tables'
import { isNotImplemented } from '../../api/http'
import { useAuthStore } from '../../stores/auth'
import { useProfileStore } from '../../stores/profile'
import { useSettingsStore } from '../../stores/settings'
import { usePanel } from '../../composables/usePanel'
import { useGameSound } from '../../composables/useGameSound'
import { useToast } from '../../composables/useToast'
import { MIN_STAKE, TOP_UP_THRESHOLD, TOP_UP_TO } from '../../constants/economy'
import { MOCK_ACTIVE_TABLE, MOCK_FRIENDS, MOCK_HISTORY, MOCK_MESSAGES, MOCK_ONLINE, MOCK_REQUESTS } from '../../mocks/lobby'
import type { DeckSize, GameMode, TableInfo } from '../../types/game'
import type { Friend, FriendRequest, ChatEntry } from '../../types/social'

const POLL_MS = 3000

const router = useRouter()
const auth = useAuthStore()
const profileStore = useProfileStore()
const settings = useSettingsStore()
const panels = usePanel()
const sound = useGameSound()
const toast = useToast()

const mode = ref<GameMode>('podkidnoy')
const deck = ref<DeckSize>(36)
const players = ref(2)
const stake = ref(MIN_STAKE)
const busy = ref<'find' | 'create' | null>(null)

const tables = ref<TableInfo[]>([])
const tablesLoaded = ref(false)
const tablesError = ref('')
const refreshing = ref(false)
const now = ref(Date.now())

const friends = ref<Friend[]>([...MOCK_FRIENDS])
const requests = ref<FriendRequest[]>([...MOCK_REQUESTS])
const messages = ref<ChatEntry[]>([...MOCK_MESSAGES])
const activeTable = ref(MOCK_ACTIVE_TABLE)

let pollTimer: number | null = null

const profile = computed(() => profileStore.profile)
const username = computed(() => profile.value?.username ?? auth.user?.username ?? 'Игрок')
const balance = computed(() => profile.value?.wallet.balance ?? null)

watch(balance, (b) => {
  if (b !== null && stake.value > b) stake.value = MIN_STAKE
})

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

function soon(text: string) {
  sound.playClick()
  toast.show(text)
}

function topUp() {
  if (balance.value !== null && balance.value < TOP_UP_THRESHOLD) soon(`Скоро здесь можно будет пополнить баланс до ${TOP_UP_TO} фишек`)
  else soon(`Пополнение доступно, когда фишек меньше ${TOP_UP_THRESHOLD}`)
}

function sendMessage(text: string) {
  messages.value = [
    ...messages.value,
    { id: `local-${Date.now()}`, userId: auth.user?.id ?? 'me', username: username.value, text, at: new Date().toISOString() },
  ]
}

function acceptRequest(req: FriendRequest) {
  requests.value = requests.value.filter((r) => r.id !== req.id)
  friends.value = [...friends.value, { id: req.id, username: req.username, status: 'online' }]
  sound.playSuccess()
}

function declineRequest(req: FriendRequest) {
  requests.value = requests.value.filter((r) => r.id !== req.id)
}

function invite(friend: Friend) {
  soon(`Приглашения скоро заработают — ${friend.username} пока не узнает`)
}

function toggleSound() {
  settings.soundOn = !settings.soundOn
  if (settings.soundOn) sound.playClick()
}

function logout() {
  auth.logout()
  profileStore.reset()
  router.push('/')
}

const PANEL_TITLES = {
  profile: 'Профиль',
  settings: 'Настройки',
  leaderboard: 'Рейтинг',
  shop: 'Магазин',
  social: 'Друзья и чат',
} as const

function onVisibility() {
  if (document.visibilityState === 'visible') refreshTables()
}

onMounted(() => {
  profileStore.load(true)
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
  <div class="client" :data-theme="settings.theme" :data-felt="settings.felt">
    <div class="glow" aria-hidden="true"></div>

    <NavRail
      class="rail"
      :active="panels.panel.value"
      :sound-on="settings.soundOn"
      @home="panels.close()"
      @open="panels.open"
      @toggle-sound="toggleSound"
      @logout="logout"
    />

    <main class="main">
      <div class="main-inner">
        <TopBar
          :username="username"
          :balance="balance"
          :online="MOCK_ONLINE"
          :social-badge="requests.length"
          @profile="panels.open('profile')"
          @social="panels.open('social')"
          @top-up="topUp"
        />

        <ReturnBanner
          v-if="activeTable"
          class="rise"
          style="--d: 1"
          :table="activeTable"
          @return="soon('Возврат за стол скоро заработает')"
          @dismiss="activeTable = null"
        />

        <PlayPanel
          v-model:mode="mode"
          v-model:deck="deck"
          v-model:players="players"
          v-model:stake="stake"
          class="rise"
          style="--d: 2"
          :busy="busy"
          :balance="balance"
          :card-back="settings.cardBack"
          @find="find"
          @create="create"
          @private="soon('Приватные столы по ссылке скоро появятся')"
        />

        <TablesBoard
          class="rise"
          style="--d: 3"
          :tables="tables"
          :loaded="tablesLoaded"
          :refreshing="refreshing"
          :error="tablesError"
          :my-id="auth.user?.id"
          :now="now"
          :stake="stake"
          :balance="balance"
          @join="(t) => goToTable(t.id)"
          @refresh="refreshTables(true)"
          @create="create"
        />
      </div>
    </main>

    <SocialPanel
      class="social-col"
      :friends="friends"
      :requests="requests"
      :messages="messages"
      :my-id="auth.user?.id"
      :show-chat="settings.showChat"
      @add-friend="soon('Поиск друзей по имени скоро появится')"
      @invite="invite"
      @accept="acceptRequest"
      @decline="declineRequest"
      @send="sendMessage"
    />

    <AnimatePresence>
      <OverlayPanel
        v-if="panels.panel.value && panels.panel.value !== 'social'"
        :key="panels.panel.value"
        :title="PANEL_TITLES[panels.panel.value]"
        :width="panels.panel.value === 'leaderboard' ? 520 : 880"
        @close="panels.close()"
      >
        <ProfilePanel
          v-if="panels.panel.value === 'profile'"
          :profile="profile"
          :error="profileStore.error"
          :history="MOCK_HISTORY"
          @settings="(s) => panels.open('settings', s)"
          @retry="profileStore.load(true)"
        />
        <SettingsPanel
          v-else-if="panels.panel.value === 'settings'"
          :section="panels.section.value"
          :profile="profile"
          @section="panels.setSection"
        />
        <Leaderboard v-else-if="panels.panel.value === 'leaderboard'" :my-id="auth.user?.id" />
        <ShopPanel v-else-if="panels.panel.value === 'shop'" @buy="soon('Магазин откроется вместе с экономикой')" />
      </OverlayPanel>

      <OverlayPanel
        v-if="panels.panel.value === 'social'"
        key="social"
        side="right"
        :width="380"
        :title="PANEL_TITLES.social"
        @close="panels.close()"
      >
        <SocialPanel
          class="social-sheet"
          :friends="friends"
          :requests="requests"
          :messages="messages"
          :my-id="auth.user?.id"
          :show-chat="settings.showChat"
          @add-friend="soon('Поиск друзей по имени скоро появится')"
          @invite="invite"
          @accept="acceptRequest"
          @decline="declineRequest"
          @send="sendMessage"
        />
      </OverlayPanel>
    </AnimatePresence>

    <ToastStack />
  </div>
</template>

<style scoped>
.client {
  --rail-w: 84px;
  --rail-h: 0px;

  position: relative;
  display: grid;
  grid-template-columns: var(--rail-w) minmax(0, 1fr) 360px;
  height: 100dvh;
  overflow: hidden;
  background: var(--color-paper);
  color: var(--color-ink);
}

.glow {
  position: absolute;
  top: -30vh;
  left: 20%;
  width: 60vw;
  height: 80vh;
  background: radial-gradient(ellipse at 50% 40%, var(--color-glow), transparent 65%);
  pointer-events: none;
}

.rail,
.main,
.social-col {
  position: relative;
  min-height: 0;
}

.main {
  overflow-y: auto;
  overscroll-behavior: contain;
}

.main-inner {
  display: flex;
  flex-direction: column;
  gap: 22px;
  max-width: 1280px;
  margin: 0 auto;
  padding: 8px 32px 48px;
}

.social-col {
  border-left: 1px solid var(--color-line);
  background: color-mix(in srgb, var(--color-paper-sunken) 60%, transparent);
}

.social-sheet {
  height: 100%;
  padding: 0;
}

@media (min-width: 1700px) {
  .client {
    grid-template-columns: var(--rail-w) minmax(0, 1fr) 400px;
  }
}

@media (max-width: 1279px) {
  .client {
    grid-template-columns: var(--rail-w) minmax(0, 1fr);
  }

  .social-col {
    display: none;
  }
}

@media (max-width: 767px) {
  .client {
    --rail-w: 0px;
    --rail-h: 68px;

    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: minmax(0, 1fr) auto;
  }

  .rail {
    grid-row: 2;
    z-index: 110;
  }

  .main {
    grid-row: 1;
  }

  .main-inner {
    gap: 16px;
    padding: 4px 16px 32px;
  }
}
</style>
