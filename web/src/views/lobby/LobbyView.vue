<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { RefreshCw } from 'lucide-vue-next'
import AppHeader from '../../components/app/AppHeader.vue'
import PaperCard from '../../components/landing/PaperCard.vue'
import { createTable, listTables } from '../../api/tables'
import { useAuthStore } from '../../stores/auth'
import { useGameSound } from '../../composables/useGameSound'
import { plural } from '../../composables/useLanding'
import type { DeckSize, GameMode, TableInfo } from '../../types/game'

const POLL_MS = 3000
const HAND = 6

const MODES: { id: GameMode; name: string; text: string }[] = [
  { id: 'podkidnoy', name: 'Подкидной', text: 'Подкидывают все, кроме защитника.' },
  { id: 'perevodnoy', name: 'Переводной', text: 'Атаку можно перевести картой того же достоинства.' },
]
const DECKS: DeckSize[] = [24, 36, 52]
const PLAYERS = [2, 3, 4, 5, 6]

const router = useRouter()
const auth = useAuthStore()
const sound = useGameSound()

const mode = ref<GameMode>('podkidnoy')
const deck = ref<DeckSize>(36)
const players = ref(2)
const creating = ref(false)
const createError = ref('')

const tables = ref<TableInfo[]>([])
const loaded = ref(false)
const listError = ref('')
const refreshing = ref(false)
const now = ref(Date.now())

let pollTimer: number | null = null

const fits = (n: number, size: number = deck.value) => n * HAND <= size
const leftInDeck = computed(() => deck.value - players.value * HAND)

const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return 'Доброй ночи'
  if (h < 12) return 'Доброе утро'
  if (h < 18) return 'Добрый день'
  return 'Добрый вечер'
})

function pickDeck(size: DeckSize) {
  sound.playClick()
  deck.value = size
  if (!fits(players.value, size)) {
    players.value = Math.floor(size / HAND)
  }
}

function pickPlayers(n: number) {
  if (!fits(n)) return
  sound.playClick()
  players.value = n
}

function pickMode(m: GameMode) {
  sound.playClick()
  mode.value = m
}

async function refresh(manual = false) {
  if (manual) refreshing.value = true
  try {
    tables.value = await listTables()
    listError.value = ''
  } catch (err) {
    listError.value = err instanceof Error ? err.message : 'Не удалось загрузить столы'
    if (!auth.isAuthenticated) router.push({ name: 'landing', query: { auth: 'login' } })
  } finally {
    loaded.value = true
    refreshing.value = false
    now.value = Date.now()
  }
}

async function create() {
  if (creating.value) return
  sound.playClick()
  creating.value = true
  createError.value = ''
  try {
    const table = await createTable({ mode: mode.value, deck: deck.value, maxPlayers: players.value })
    sound.playCardDeal()
    router.push({ name: 'table', params: { id: table.id } })
  } catch (err) {
    sound.playError()
    createError.value = err instanceof Error ? err.message : 'Не удалось создать стол'
  } finally {
    creating.value = false
  }
}

function join(t: TableInfo) {
  sound.playCardDeal()
  router.push({ name: 'table', params: { id: t.id } })
}

function modeName(m: GameMode) {
  return MODES.find((x) => x.id === m)?.name ?? m
}

function ago(iso: string) {
  const s = Math.max(0, Math.round((now.value - new Date(iso).getTime()) / 1000))
  if (s < 10) return 'только что'
  if (s < 60) return `${s} сек назад`
  const m = Math.round(s / 60)
  return `${m} ${plural(m, ['минуту', 'минуты', 'минут'])} назад`
}

function onVisibility() {
  if (document.visibilityState === 'visible') refresh()
}

onMounted(() => {
  refresh()
  pollTimer = window.setInterval(() => {
    if (document.visibilityState === 'visible') refresh()
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
      <section class="intro">
        <p class="eyebrow">Лобби</p>
        <h1>{{ greeting }}, {{ auth.user?.username ?? 'игрок' }}.</h1>
        <p class="lead">Соберите свой стол или подсядьте к тем, кто уже ждёт соперника.</p>
      </section>

      <div class="grid">
        <section class="panel create" aria-labelledby="create-title">
          <h2 id="create-title">Новый стол</h2>

          <div class="field">
            <span class="label">Режим</span>
            <div class="segmented" role="radiogroup" aria-label="Режим">
              <button
                v-for="m in MODES"
                :key="m.id"
                type="button"
                role="radio"
                :aria-checked="mode === m.id"
                :class="{ on: mode === m.id }"
                @click="pickMode(m.id)"
              >
                {{ m.name }}
              </button>
            </div>
            <p class="hint">{{ MODES.find((m) => m.id === mode)!.text }}</p>
          </div>

          <div class="field">
            <span class="label">Колода</span>
            <div class="chips" role="radiogroup" aria-label="Колода">
              <button
                v-for="d in DECKS"
                :key="d"
                type="button"
                role="radio"
                :aria-checked="deck === d"
                :class="{ on: deck === d }"
                @click="pickDeck(d)"
              >
                {{ d }}
              </button>
            </div>
          </div>

          <div class="field">
            <span class="label">Игроков</span>
            <div class="chips" role="radiogroup" aria-label="Игроков">
              <button
                v-for="n in PLAYERS"
                :key="n"
                type="button"
                role="radio"
                :aria-checked="players === n"
                :class="{ on: players === n }"
                :disabled="!fits(n)"
                :title="fits(n) ? '' : `В колоде ${deck} карт не хватит на ${n} игроков`"
                @click="pickPlayers(n)"
              >
                {{ n }}
              </button>
            </div>
          </div>

          <div class="summary">
            <div class="mini-deck" aria-hidden="true">
              <span v-for="i in 3" :key="i" class="mini" :style="{ transform: `translate(${i * 3}px, ${-i * 3}px)` }"></span>
            </div>
            <p>
              Раздаём по {{ HAND }} карт, в колоде останется
              <b>{{ leftInDeck }} {{ plural(leftInDeck, ['карта', 'карты', 'карт']) }}</b>.
            </p>
          </div>

          <p v-if="createError" class="error" role="alert">{{ createError }}</p>

          <button type="button" class="primary" :disabled="creating" @click="create">
            {{ creating ? 'Создаём…' : 'Создать стол и сесть' }}
          </button>
        </section>

        <section class="panel tables" aria-labelledby="tables-title">
          <div class="tables-head">
            <h2 id="tables-title">
              Открытые столы
              <span v-if="tables.length" class="count">{{ tables.length }}</span>
            </h2>
            <span class="live"><i></i>обновляется сам</span>
            <button
              type="button"
              class="icon-btn"
              :class="{ spin: refreshing }"
              aria-label="Обновить"
              title="Обновить"
              @click="refresh(true)"
            >
              <RefreshCw :size="17" />
            </button>
          </div>

          <p v-if="listError" class="error" role="alert">{{ listError }}</p>

          <ul v-if="!loaded" class="list" aria-busy="true">
            <li v-for="i in 3" :key="i" class="row skeleton"></li>
          </ul>

          <TransitionGroup v-else-if="tables.length" tag="ul" name="row" class="list">
            <li v-for="t in tables" :key="t.id" class="row">
              <div class="row-cards" aria-hidden="true">
                <span class="rc rc1"><PaperCard rank="6" suit="♠" /></span>
                <span class="rc rc2"><PaperCard :rank="t.mode === 'perevodnoy' ? '6' : 'A'" :suit="t.mode === 'perevodnoy' ? '♥' : '♦'" /></span>
              </div>
              <div class="row-main">
                <div class="row-title">
                  {{ modeName(t.mode) }}
                  <span class="tag">{{ t.deck }} карт</span>
                  <span v-if="t.createdBy === auth.user?.id" class="tag mine">ваш стол</span>
                </div>
                <div class="row-meta">
                  <span class="seats" :aria-label="`Занято ${t.players} из ${t.maxPlayers}`">
                    <i v-for="s in t.maxPlayers" :key="s" :class="{ taken: s <= t.players }"></i>
                  </span>
                  <span>{{ t.players }}/{{ t.maxPlayers }}</span>
                  <span class="dot">·</span>
                  <span>{{ ago(t.createdAt) }}</span>
                </div>
              </div>
              <button type="button" class="join" @click="join(t)">
                {{ t.createdBy === auth.user?.id ? 'Вернуться' : 'Сесть' }}
              </button>
            </li>
          </TransitionGroup>

          <div v-else class="empty">
            <div class="empty-cards" aria-hidden="true">
              <span class="ec ec1"><PaperCard rank="K" suit="♣" /></span>
              <span class="ec ec2"><PaperCard rank="Q" suit="♥" /></span>
            </div>
            <p class="empty-title">Пока никто не ждёт</p>
            <p class="empty-text">Создайте стол слева — он появится здесь, и соперники смогут подсесть.</p>
          </div>
        </section>
      </div>
    </main>
  </div>
</template>

<style scoped>
.lobby {
  min-height: 100vh;
  background: #f5f4ef;
  color: #1b1916;
  font-family: 'Inter', -apple-system, 'Helvetica Neue', Arial, sans-serif;
}

.wrap {
  max-width: 1120px;
  margin: 0 auto;
  padding: 48px 24px 80px;
}

.eyebrow {
  margin: 0 0 10px;
  font-size: 13px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #9a9384;
}

h1 {
  margin: 0;
  font-size: clamp(34px, 5vw, 56px);
  font-weight: 500;
  letter-spacing: -0.035em;
  line-height: 1.02;
}

.lead {
  margin: 14px 0 0;
  max-width: 520px;
  font-size: 17px;
  line-height: 1.5;
  color: #57534b;
}

.grid {
  display: grid;
  grid-template-columns: minmax(0, 380px) minmax(0, 1fr);
  gap: 20px;
  margin-top: 40px;
  align-items: start;
}

.panel {
  padding: 28px;
  border-radius: 28px;
  background: rgba(250, 249, 245, 0.85);
  box-shadow:
    0 0 0 1px rgba(214, 208, 196, 0.9),
    0 24px 48px -32px rgba(70, 62, 48, 0.35);
}

h2 {
  display: flex;
  align-items: center;
  gap: 10px;
  margin: 0;
  font-size: 22px;
  font-weight: 500;
  letter-spacing: -0.02em;
}

.field {
  margin-top: 22px;
}

.label {
  display: block;
  margin-bottom: 10px;
  font-size: 13px;
  color: #8a8375;
}

.segmented {
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding: 4px;
  border-radius: 999px;
  background: #e9e5db;
}

.segmented button,
.chips button {
  height: 40px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: #57534b;
  font: inherit;
  font-size: 15px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    color 0.2s,
    box-shadow 0.2s;
}

.segmented button.on {
  background: #fdfcf8;
  color: #1b1916;
  box-shadow: 0 2px 8px -4px rgba(70, 62, 48, 0.4);
}

.hint {
  margin: 10px 4px 0;
  font-size: 14px;
  line-height: 1.45;
  color: #6f695e;
}

.chips {
  display: flex;
  gap: 8px;
}

.chips button {
  flex: 1;
  min-width: 44px;
  height: 44px;
  background: #ede9e0;
  font-variant-numeric: tabular-nums;
}

.chips button:hover:not(:disabled):not(.on) {
  background: #e3ded2;
  color: #1b1916;
}

.chips button.on {
  background: #36322c;
  color: #f5f4ef;
}

.chips button:disabled {
  opacity: 0.35;
  cursor: not-allowed;
  text-decoration: line-through;
}

.summary {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-top: 24px;
  padding: 14px 16px;
  border-radius: 18px;
  background: #efebe2;
}

.summary p {
  margin: 0;
  font-size: 14px;
  line-height: 1.45;
  color: #57534b;
}

.summary b {
  color: #1b1916;
  font-weight: 600;
}

.mini-deck {
  position: relative;
  flex: none;
  width: 34px;
  height: 46px;
}

.mini {
  position: absolute;
  left: 0;
  bottom: 0;
  width: 28px;
  height: 38px;
  border-radius: 5px;
  background:
    repeating-linear-gradient(45deg, transparent 0 4px, rgba(160, 148, 124, 0.25) 4px 5px),
    #e7e1d4;
  box-shadow: 0 0 0 1px rgba(160, 150, 132, 0.5);
}

.primary {
  width: 100%;
  height: 54px;
  margin-top: 20px;
  border: 0;
  border-radius: 999px;
  background: #36322c;
  color: #f5f4ef;
  font: inherit;
  font-size: 16px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.2s;
}

.primary:hover:not(:disabled) {
  background: #1b1916;
}

.primary:active:not(:disabled) {
  transform: scale(0.98);
}

.primary:disabled {
  opacity: 0.6;
  cursor: wait;
}

.error {
  margin: 16px 0 0;
  padding: 10px 14px;
  border-radius: 14px;
  background: #f5e4df;
  color: #8f3a32;
  font-size: 14px;
}

.tables-head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.count {
  display: grid;
  place-items: center;
  min-width: 26px;
  height: 26px;
  padding: 0 8px;
  box-sizing: border-box;
  border-radius: 999px;
  background: #36322c;
  color: #f5f4ef;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.live {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-left: auto;
  font-size: 13px;
  color: #8a8375;
}

.live i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #6f9a6a;
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  50% {
    opacity: 0.35;
  }
}

.icon-btn {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border: 0;
  border-radius: 999px;
  background: #ede9e0;
  color: #1b1916;
  cursor: pointer;
}

.icon-btn:hover {
  background: #e3ded2;
}

.icon-btn.spin svg {
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 20px 0 0;
  padding: 0;
  list-style: none;
}

.row {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 76px;
  padding: 12px 12px 12px 16px;
  box-sizing: border-box;
  border-radius: 20px;
  background: #fdfcf8;
  box-shadow: 0 0 0 1px rgba(214, 208, 196, 0.8);
  transition: box-shadow 0.2s;
}

.row:hover {
  box-shadow:
    0 0 0 1px rgba(184, 176, 160, 0.9),
    0 14px 28px -20px rgba(70, 62, 48, 0.45);
}

.skeleton {
  background: linear-gradient(90deg, #efebe2 0%, #f7f5ef 50%, #efebe2 100%);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

.row-cards {
  position: relative;
  flex: none;
  width: 52px;
  height: 52px;
}

.rc {
  position: absolute;
  top: 2px;
  width: 32px;
  height: 46px;
}

.rc :deep(.pip) {
  font-size: 18px;
}
.rc :deep(.index) {
  display: none;
}

.rc1 {
  left: 0;
  transform: rotate(-10deg);
}

.rc2 {
  left: 16px;
  transform: rotate(8deg);
}

.row-main {
  flex: 1;
  min-width: 0;
}

.row-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 17px;
  letter-spacing: -0.01em;
}

.tag {
  padding: 3px 9px;
  border-radius: 999px;
  background: #ede9e0;
  font-size: 12px;
  color: #57534b;
}

.tag.mine {
  background: #f1e6cf;
  color: #7a5a22;
}

.row-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  font-size: 13px;
  color: #8a8375;
  font-variant-numeric: tabular-nums;
}

.seats {
  display: flex;
  gap: 4px;
}

.seats i {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  box-shadow: inset 0 0 0 1.5px #b9b1a1;
}

.seats i.taken {
  background: #36322c;
  box-shadow: none;
}

.join {
  flex: none;
  height: 44px;
  padding: 0 22px;
  border: 0;
  border-radius: 999px;
  background: #36322c;
  color: #f5f4ef;
  font: inherit;
  font-size: 15px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.2s;
}

.join:hover {
  background: #1b1916;
}

.join:active {
  transform: scale(0.97);
}

.empty {
  padding: 48px 16px 32px;
  text-align: center;
}

.empty-cards {
  position: relative;
  width: 120px;
  height: 100px;
  margin: 0 auto 18px;
}

.ec {
  position: absolute;
  top: 0;
  width: 62px;
  height: 88px;
}

.ec1 {
  left: 10px;
  transform: rotate(-12deg);
}

.ec2 {
  left: 46px;
  transform: rotate(9deg);
}

.empty-title {
  margin: 0;
  font-size: 18px;
}

.empty-text {
  max-width: 320px;
  margin: 8px auto 0;
  font-size: 14px;
  line-height: 1.5;
  color: #6f695e;
}

.row-enter-active,
.row-leave-active {
  transition:
    opacity 0.35s,
    transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.row-enter-from,
.row-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

@media (max-width: 860px) {
  .grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 520px) {
  .wrap {
    padding: 32px 16px 64px;
  }

  .panel {
    padding: 22px 18px;
  }

  .live {
    display: none;
  }

  .tables-head .icon-btn {
    margin-left: auto;
  }

  .row-cards {
    display: none;
  }
}
</style>
