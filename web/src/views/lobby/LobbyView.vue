<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { AnimatePresence } from 'motion-v'
import { useMediaQuery } from '@vueuse/core'
import ToastStack from '../../components/app/ToastStack.vue'
import NavRail from '../../components/client/NavRail.vue'
import TopBar from '../../components/client/TopBar.vue'
import ReturnBanner from '../../components/client/ReturnBanner.vue'
import PlayPanel from '../../components/client/PlayPanel.vue'
import TablesBoard from '../../components/client/TablesBoard.vue'
import SocialPanel from '../../components/client/SocialPanel.vue'
import FriendsRail from '../../components/client/FriendsRail.vue'
import StatsCard from '../../components/client/StatsCard.vue'
import LobbyChat from '../../components/client/LobbyChat.vue'
import OverlayPanel from '../../components/client/OverlayPanel.vue'
import ProfilePanel from '../../components/panels/ProfilePanel.vue'
import SettingsPanel from '../../components/panels/SettingsPanel.vue'
import ShopPanel from '../../components/panels/ShopPanel.vue'
import Leaderboard from '../../components/lobby/Leaderboard.vue'
import { createTable, getActiveTable, listTables, quickMatch } from '../../api/tables'
import { ApiError, isNotImplemented } from '../../api/http'
import { topUp as requestTopUp } from '../../api/wallet'
import { getGameHistory } from '../../api/users'
import { useAuthStore } from '../../stores/auth'
import { useProfileStore } from '../../stores/profile'
import { useSettingsStore } from '../../stores/settings'
import { usePanel } from '../../composables/usePanel'
import { useSmoothScroll } from '../../composables/useSmoothScroll'
import { useGameSound } from '../../composables/useGameSound'
import { useToast } from '../../composables/useToast'
import { MIN_STAKE, TOP_UP_THRESHOLD, TOP_UP_TO } from '../../constants/economy'
import { MOCK_FRIENDS, MOCK_MESSAGES, MOCK_ONLINE, MOCK_REQUESTS } from '../../mocks/lobby'
import type { DeckSize, GameMode, TableInfo } from '../../types/game'
import type { ActiveTable, Friend, FriendRequest, ChatEntry, GameHistoryEntry } from '../../types/social'

const POLL_MS = 3000

const router = useRouter()
const auth = useAuthStore()
const profileStore = useProfileStore()
const settings = useSettingsStore()
const panels = usePanel()
const sound = useGameSound()
const toast = useToast()
const mainEl = ref<HTMLElement | null>(null)
const sideColumn = useMediaQuery('(min-width: 1024px)')

useSmoothScroll(mainEl)

const mode = ref<GameMode>('podkidnoy')
const deck = ref<DeckSize>(36)
const players = ref(3)
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
const history = ref<GameHistoryEntry[]>([])
const activeTable = ref<ActiveTable | null>(null)
const dismissedTable = ref<string | null>(null)

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
  sound.playChips()
  router.push({ name: 'table', params: { id } })
}

async function create() {
  if (busy.value) return
  sound.playClick()
  busy.value = 'create'
  try {
    const table = await createTable({ mode: mode.value, deck: deck.value, maxPlayers: Math.max(3, players.value), stake: stake.value })
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
    const table = await quickMatch({ mode: mode.value, deck: deck.value, stake: stake.value })
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

const toppingUp = ref(false)

async function topUp() {
  if (toppingUp.value) return
  if (balance.value !== null && balance.value >= TOP_UP_THRESHOLD) {
    soon(`Пополнение доступно, когда фишек меньше ${TOP_UP_THRESHOLD}`)
    return
  }
  toppingUp.value = true
  try {
    await requestTopUp()
    await profileStore.load(true)
    sound.playChipsWin()
    toast.show(`Баланс пополнен до ${TOP_UP_TO} фишек`)
  } catch (err) {
    sound.playError()
    toast.show(err instanceof ApiError ? err.message : 'Не удалось пополнить баланс', 'error')
  } finally {
    toppingUp.value = false
  }
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
  tables: 'Все столы',
} as const

async function refreshActiveTable() {
  try {
    const t = await getActiveTable()
    activeTable.value =
      t && t.id !== dismissedTable.value ? { id: t.id, mode: t.mode, stake: t.stake ?? MIN_STAKE, players: t.maxPlayers } : null
  } catch {
    return
  }
}

async function loadHistory() {
  try {
    history.value = await getGameHistory(20)
  } catch {
    return
  }
}

function refreshAll() {
  refreshTables()
  refreshActiveTable()
}

function returnToTable() {
  if (!activeTable.value) return
  sound.playClick()
  router.push({ name: 'table', params: { id: activeTable.value.id } })
}

function dismissActiveTable() {
  dismissedTable.value = activeTable.value?.id ?? null
  activeTable.value = null
}

function onVisibility() {
  if (document.visibilityState === 'visible') refreshAll()
}

onMounted(() => {
  profileStore.load(true)
  loadHistory()
  refreshAll()
  pollTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') refreshAll()
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

    <main ref="mainEl" class="main">
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
          @return="returnToTable"
          @dismiss="dismissActiveTable"
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

        <div v-if="!sideColumn" class="side-inline">
          <StatsCard class="rise" style="--d: 4" :profile="profile" :history="history" @profile="panels.open('profile')" />
          <LobbyChat
            v-if="settings.showChat"
            class="rise chat-inline"
            style="--d: 5"
            :messages="messages"
            :my-id="auth.user?.id"
            :online="MOCK_ONLINE"
            @send="sendMessage"
          />
        </div>
      </div>
    </main>

    <aside v-if="sideColumn" class="side" aria-label="Статистика и чат">
      <StatsCard class="rise" style="--d: 2" :profile="profile" :history="history" @profile="panels.open('profile')" />
      <LobbyChat
        v-if="settings.showChat"
        class="rise side-chat"
        style="--d: 3"
        :messages="messages"
        :my-id="auth.user?.id"
        :online="MOCK_ONLINE"
        @send="sendMessage"
      />
    </aside>

    <FriendsRail
      class="friends"
      :username="username"
      :friends="friends"
      :requests="requests"
      @profile="panels.open('profile')"
      @social="panels.open('social')"
      @add="soon('Поиск друзей по имени скоро появится')"
    />

    <AnimatePresence>
      <OverlayPanel
        v-if="panels.panel.value && panels.panel.value !== 'social' && panels.panel.value !== 'tables'"
        :key="panels.panel.value"
        :title="PANEL_TITLES[panels.panel.value]"
        :width="panels.panel.value === 'leaderboard' ? 520 : 880"
        @close="panels.close()"
      >
        <ProfilePanel
          v-if="panels.panel.value === 'profile'"
          :profile="profile"
          :error="profileStore.error"
          :history="history"
          @settings="(s) => panels.open('settings', s)"
          @retry="profileStore.load(true)"
        />
        <SettingsPanel
          v-else-if="panels.panel.value === 'settings'"
          :section="panels.section.value"
          :profile="profile"
          @section="panels.setSection"
          @shop="panels.open('shop')"
        />
        <Leaderboard v-else-if="panels.panel.value === 'leaderboard'" :my-id="auth.user?.id" />
        <ShopPanel
          v-else-if="panels.panel.value === 'shop'"
          :balance="balance"
          @buy="(item) => soon(`«${item.name}» можно будет купить, когда заработают покупки на сервере`)"
        />
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
  --rail-w: 98px;
  --side-w: 340px;
  --friends-w: 100px;
  --rail-h: 0px;

  position: relative;
  display: grid;
  grid-template-columns: var(--rail-w) minmax(0, 1fr) var(--side-w) var(--friends-w);
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
.side,
.friends {
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
  gap: 24px;
  max-width: 1180px;
  margin: 0 auto;
  padding: 8px 26px 48px;
}

.side {
  display: flex;
  flex-direction: column;
  gap: 14px;
  height: 100dvh;
  padding: 14px 0;
  box-sizing: border-box;
}

.side-chat {
  flex: 1;
  min-height: 0;
}

.side-inline {
  display: grid;
  grid-template-columns: minmax(260px, 340px) minmax(0, 1fr);
  gap: 18px;
  align-items: stretch;
}

.chat-inline {
  height: 420px;
}

.social-sheet {
  height: 100%;
  padding: 0;
}

@media (min-width: 1800px) {
  .client {
    --side-w: 380px;
  }
}

@media (max-width: 1279px) {
  .client {
    --side-w: 320px;

    grid-template-columns: var(--rail-w) minmax(0, 1fr) var(--side-w);
  }

  .side {
    padding-right: 14px;
  }

  .friends {
    display: none;
  }
}

@media (max-width: 1023px) {
  .client {
    grid-template-columns: var(--rail-w) minmax(0, 1fr);
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
    gap: 18px;
    padding: 4px 16px 32px;
  }

  .side-inline {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
