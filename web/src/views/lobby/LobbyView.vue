<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { AnimatePresence } from 'motion-v'
import { PhPlus } from '@phosphor-icons/vue'
import ToastStack from '../../components/app/ToastStack.vue'
import NavRail from '../../components/client/NavRail.vue'
import TopBar from '../../components/client/TopBar.vue'
import ReturnBanner from '../../components/client/ReturnBanner.vue'
import QuickPlay from '../../components/client/QuickPlay.vue'
import TablesCarousel from '../../components/client/TablesCarousel.vue'
import AllTables from '../../components/client/AllTables.vue'
import CreateTableDialog from '../../components/client/CreateTableDialog.vue'
import FriendsList from '../../components/client/FriendsList.vue'
import FriendsRail from '../../components/client/FriendsRail.vue'
import LobbyPlayerCard from '../../components/client/LobbyPlayerCard.vue'
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
import { bestAffordable, useCreatePrefs, useQuickPrefs } from '../../composables/useLobbyPrefs'
import { MIN_STAKE, TOP_UP_THRESHOLD, TOP_UP_TO } from '../../constants/economy'
import { MOCK_FRIENDS, MOCK_ONLINE, MOCK_REQUESTS } from '../../mocks/lobby'
import type { TableInfo } from '../../types/game'
import type { ActiveTable, Friend, FriendRequest, GameHistoryEntry } from '../../types/social'
import type { TablePreset } from '../../composables/tableList'

const POLL_MS = 3000

const router = useRouter()
const auth = useAuthStore()
const profileStore = useProfileStore()
const settings = useSettingsStore()
const panels = usePanel()
const sound = useGameSound()
const toast = useToast()
const mainEl = ref<HTMLElement | null>(null)

useSmoothScroll(mainEl)

const { stake: quickStake, mode: quickMode, deck: quickDeck } = useQuickPrefs()
const createPrefs = useCreatePrefs()
const busy = ref<'find' | 'create' | null>(null)
const creating = ref(false)

const tables = ref<TableInfo[]>([])
const tablesLoaded = ref(false)
const tablesError = ref('')
const now = ref(Date.now())

const friends = ref<Friend[]>([...MOCK_FRIENDS])
const requests = ref<FriendRequest[]>([...MOCK_REQUESTS])
const history = ref<GameHistoryEntry[]>([])
const historyLoaded = ref(false)
const activeTable = ref<ActiveTable | null>(null)
const dismissedTable = ref<string | null>(null)

let pollTimer: number | null = null

const profile = computed(() => profileStore.profile)
const username = computed(() => profile.value?.username ?? auth.user?.username ?? 'Игрок')
const balance = computed(() => profile.value?.wallet.balance ?? null)

watch(balance, (b) => {
  quickStake.value = bestAffordable(b, quickStake.value)
  createPrefs.stake.value = bestAffordable(b, createPrefs.stake.value)
})

async function refreshTables() {
  try {
    tables.value = await listTables()
    tablesError.value = ''
  } catch (err) {
    tablesError.value = err instanceof Error ? err.message : 'Не удалось загрузить столы'
    if (!auth.isAuthenticated) router.push({ name: 'landing', query: { auth: 'login' } })
  } finally {
    tablesLoaded.value = true
    now.value = Date.now()
  }
}

function joinFromList(table: TableInfo) {
  if (panels.panel.value === 'tables') panels.close()
  goToTable(table.id)
}

function goToTable(id: string, search = false) {
  sound.playChips()
  router.push({ name: 'table', params: { id }, query: search ? { search: '1' } : undefined })
}

function openCreate() {
  sound.playClick()
  creating.value = true
}

async function create() {
  if (busy.value) return
  busy.value = 'create'
  try {
    const table = await createTable({
      mode: createPrefs.mode.value,
      deck: createPrefs.deck.value,
      maxPlayers: createPrefs.players.value,
      stake: createPrefs.stake.value,
    })
    creating.value = false
    goToTable(table.id)
  } catch (err) {
    sound.playError()
    toast.show(err instanceof Error ? err.message : 'Не удалось создать стол', 'error')
  } finally {
    busy.value = null
  }
}

function createFromPreset(p: TablePreset) {
  createPrefs.mode.value = p.mode
  createPrefs.deck.value = p.deck
  createPrefs.players.value = p.players
  createPrefs.stake.value = p.stake
  create()
}

async function find() {
  if (busy.value) return
  sound.playClick()
  busy.value = 'find'
  try {
    const table = await quickMatch({ mode: quickMode.value, deck: quickDeck.value, stake: quickStake.value })
    goToTable(table.id, true)
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
  friends: 'Друзья',
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
    history.value = []
  } finally {
    historyLoaded.value = true
  }
}

function playAgain(g: GameHistoryEntry) {
  if (g.players === 2) {
    quickMode.value = g.mode
    quickDeck.value = g.deck
    quickStake.value = bestAffordable(balance.value, g.stake)
    find()
    return
  }
  createPrefs.mode.value = g.mode
  createPrefs.deck.value = g.deck
  createPrefs.players.value = g.players
  createPrefs.stake.value = bestAffordable(balance.value, g.stake)
  openCreate()
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
          :friends-badge="requests.length"
          @profile="panels.open('profile')"
          @friends="panels.open('friends')"
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

        <div class="duo">
          <QuickPlay
            v-model:stake="quickStake"
            v-model:mode="quickMode"
            v-model:deck="quickDeck"
            class="rise"
            style="--d: 2"
            :busy="busy === 'find'"
            :balance="balance"
            :card-back="settings.cardBack"
            @play="find"
            @top-up="topUp"
          />

          <LobbyPlayerCard
            class="rise"
            style="--d: 3"
            :username="username"
            :profile="profile"
            :history="history"
            :loaded="historyLoaded"
            :now="now"
            @profile="panels.open('profile')"
            @again="playAgain"
          />
        </div>

        <TablesCarousel
          class="rise"
          style="--d: 4"
          :tables="tables"
          :loaded="tablesLoaded"
          :error="tablesError"
          :my-id="auth.user?.id"
          :me="username"
          :now="now"
          :balance="balance"
          :busy="busy === 'create'"
          @join="joinFromList"
          @create="openCreate"
          @preset="createFromPreset"
          @all="panels.open('tables')"
        />
      </div>
    </main>

    <FriendsRail
      class="friends-slot"
      :username="username"
      :friends="friends"
      :requests="requests"
      @profile="panels.open('profile')"
      @friends="panels.open('friends')"
      @add="soon('Поиск друзей по имени скоро появится')"
    />

    <AnimatePresence>
      <OverlayPanel
        v-if="panels.panel.value && panels.panel.value !== 'friends' && panels.panel.value !== 'tables'"
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
        v-if="panels.panel.value === 'tables'"
        key="tables"
        :title="PANEL_TITLES.tables"
        :width="920"
        @close="panels.close()"
      >
        <template #head>
          <button type="button" class="head-create" @click="openCreate">
            <PhPlus :size="16" weight="bold" aria-hidden="true" />
            Создать стол
          </button>
        </template>
        <AllTables
          :tables="tables"
          :loaded="tablesLoaded"
          :my-id="auth.user?.id"
          :now="now"
          :balance="balance"
          @join="joinFromList"
          @create="openCreate"
        />
      </OverlayPanel>

      <OverlayPanel
        v-if="panels.panel.value === 'friends'"
        key="friends"
        side="right"
        :width="380"
        :title="PANEL_TITLES.friends"
        @close="panels.close()"
      >
        <FriendsList
          class="friends-sheet"
          :friends="friends"
          :requests="requests"
          @add="soon('Поиск друзей по имени скоро появится')"
          @invite="invite"
          @accept="acceptRequest"
          @decline="declineRequest"
        />
      </OverlayPanel>
    </AnimatePresence>

    <AnimatePresence>
      <CreateTableDialog
        v-if="creating"
        v-model:stake="createPrefs.stake.value"
        v-model:mode="createPrefs.mode.value"
        v-model:deck="createPrefs.deck.value"
        v-model:players="createPrefs.players.value"
        :balance="balance"
        :busy="busy === 'create'"
        @close="creating = false"
        @create="create"
      />
    </AnimatePresence>

    <ToastStack />
  </div>
</template>

<style scoped>
.client {
  --rail-w: 98px;
  --friends-w: 100px;
  --rail-h: 0px;

  position: relative;
  display: grid;
  grid-template-columns: var(--rail-w) minmax(0, 1fr) var(--friends-w);
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
.friends-slot {
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
  max-width: 1480px;
  min-height: 100%;
  margin: 0 auto;
  padding: 8px 26px 26px;
  box-sizing: border-box;
}

.duo {
  display: grid;
  grid-template-columns: minmax(0, 1.75fr) minmax(360px, 1fr);
  gap: 18px;
  align-items: stretch;
}

.friends-sheet {
  height: 100%;
  padding: 0;
}

.head-create {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--color-gold);
  color: #2b1f08;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.head-create:hover {
  filter: brightness(1.05);
}

@media (max-height: 860px) {
  .main-inner {
    gap: 18px;
    padding-bottom: 16px;
  }
}

@media (max-width: 1279px) {
  .client {
    grid-template-columns: var(--rail-w) minmax(0, 1fr);
  }

  .friends-slot {
    display: none;
  }
}

@media (max-width: 1100px) {
  .duo {
    grid-template-columns: minmax(0, 1fr);
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
}
</style>
