<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Copy, Flag, Volume2, VolumeX } from 'lucide-vue-next'
import AppHeader from '../../components/app/AppHeader.vue'
import GameCard from '../../components/game/GameCard.vue'
import { listTables } from '../../api/tables'
import { useAuthStore } from '../../stores/auth'
import { useGameSocket } from '../../composables/useGameSocket'
import { useGameSound } from '../../composables/useGameSound'
import { plural } from '../../composables/useLanding'
import {
  canBeat,
  cardKey,
  getSuitSymbol,
  sameCard,
  type Card,
  type CardPair,
  type PlayerView,
  type TableInfo,
} from '../../types/game'

const TURN_MS = 20_000

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const sound = useGameSound()

const roomId = String(route.params.id)
const socket = useGameSocket(roomId)
const { state, gameOver, status, offline, lastError } = socket

const me = computed(() => auth.user?.id ?? '')
const selected = ref<Card | null>(null)
const toast = ref<string | null>(null)
const muted = ref(localStorage.getItem('bito_sound_muted') === 'true')
const confirmSurrender = ref(false)
const copied = ref(false)
const waitingInfo = ref<TableInfo | null>(null)
const tick = ref(Date.now())

let toastTimer: number | null = null
let tickTimer: number | null = null
let waitTimer: number | null = null

const pairs = computed<CardPair[]>(() => state.value?.table ?? [])
const hand = computed<Card[]>(() => sortHand(state.value?.myHand ?? []))
const trumpSuit = computed(() => state.value?.trumpSuit ?? -1)
const phase = computed(() => state.value?.phase)
const isPerevodnoy = computed(() => state.value?.mode === 'perevodnoy')

const players = computed<PlayerView[]>(() => state.value?.players ?? [])
const meView = computed(() => players.value.find((p) => p.id === me.value))
const iAmActive = computed(() => meView.value?.isActive ?? false)
const isDefender = computed(() => state.value?.defenderId === me.value)
const isAttacker = computed(() => state.value?.attackerId === me.value)

const opponents = computed(() => {
  const list = players.value
  const idx = list.findIndex((p) => p.id === me.value)
  if (idx < 0) return list
  return [...list.slice(idx + 1), ...list.slice(0, idx)]
})

const nameOf = (id?: string) => players.value.find((p) => p.id === id)?.name ?? 'Соперник'

const unbeaten = computed(() => pairs.value.filter((p) => !p.defend))
const ranksOnTable = computed(() => {
  const set = new Set<number>()
  for (const p of pairs.value) {
    set.add(Number(p.attack.rank))
    if (p.defend) set.add(Number(p.defend.rank))
  }
  return set
})

const canAct = computed(() => !!state.value && iAmActive.value && phase.value !== 'finished' && !gameOver.value)

const canTransferWith = (card: Card) => {
  if (!isPerevodnoy.value || !isDefender.value || phase.value !== 'battle') return false
  if (!pairs.value.length || pairs.value.some((p) => p.defend)) return false
  return pairs.value.every((p) => Number(p.attack.rank) === Number(card.rank))
}

function isPlayable(card: Card): boolean {
  if (!canAct.value) return false
  if (isDefender.value) {
    if (phase.value !== 'battle') return false
    return unbeaten.value.some((p) => canBeat(p.attack, card, trumpSuit.value)) || canTransferWith(card)
  }
  if (!pairs.value.length) return phase.value === 'battle' && isAttacker.value
  return ranksOnTable.value.has(Number(card.rank))
}

const defendTargets = computed(() => {
  if (!selected.value || !isDefender.value) return new Set<string>()
  const card = selected.value
  return new Set(unbeaten.value.filter((p) => canBeat(p.attack, card, trumpSuit.value)).map((p) => cardKey(p.attack)))
})

const showTransfer = computed(() => !!selected.value && canTransferWith(selected.value))
const showTake = computed(() => canAct.value && isDefender.value && phase.value === 'battle' && unbeaten.value.length > 0)
const showBeaten = computed(
  () =>
    canAct.value &&
    !isDefender.value &&
    phase.value === 'battle' &&
    pairs.value.length > 0 &&
    unbeaten.value.length === 0 &&
    !state.value?.youPassed,
)
const showEnough = computed(() => canAct.value && !isDefender.value && phase.value === 'giving_more' && !state.value?.youPassed)

const statusText = computed(() => {
  const s = state.value
  if (!s) return ''
  if (gameOver.value || s.phase === 'finished') return 'Партия окончена'
  if (!iAmActive.value) return 'Вы вышли из игры — досмотрите партию'
  const def = nameOf(s.defenderId)
  const atk = nameOf(s.attackerId)
  if (s.phase === 'giving_more') {
    if (isDefender.value) return 'Вы берёте — соперники могут подкинуть'
    return s.youPassed ? `${def} берёт. Ждём остальных` : `${def} берёт. Подкиньте или нажмите «Хватит»`
  }
  if (!pairs.value.length) return isAttacker.value ? 'Ваш ход — атакуйте любой картой' : `Ходит ${atk}`
  if (unbeaten.value.length) {
    if (isDefender.value) return selected.value ? 'Выберите карту на столе, которую бьёте' : 'Отбивайтесь или берите'
    return `${def} отбивается. Можно подкинуть`
  }
  if (isDefender.value) return 'Всё побито. Ждём «Бито» от соперников'
  return s.youPassed ? 'Ждём остальных' : 'Всё побито. Подкиньте или скажите «Бито»'
})

const myTurn = computed(() => {
  if (!canAct.value) return false
  if (isDefender.value) return phase.value === 'battle' && unbeaten.value.length > 0
  if (!pairs.value.length) return isAttacker.value
  return !state.value?.youPassed && (phase.value === 'giving_more' || unbeaten.value.length === 0)
})

const turnLeft = computed(() => {
  const s = state.value
  if (!s || !s.turnLeftMs) return 0
  return Math.max(0, s.turnLeftMs - (tick.value - socket.stateReceivedAt.value))
})
const turnSeconds = computed(() => Math.ceil(turnLeft.value / 1000))
const turnFraction = computed(() => Math.min(1, turnLeft.value / TURN_MS))

const deckBacks = computed(() => Math.min(5, Math.ceil((state.value?.cardsLeft ?? 0) / 6)))

const result = computed(() => {
  const g = gameOver.value
  if (!g) return null
  if (g.isDraw) return { title: 'Ничья', text: 'Все вышли одновременно — дураков нет.', tone: 'draw' as const }
  if (g.loserId === me.value) return { title: 'Вы — дурак', text: 'Бывает. Отыграетесь в следующей партии.', tone: 'lose' as const }
  if (g.winnerId === me.value) return { title: 'Победа', text: 'Вы первым избавились от карт.', tone: 'win' as const }
  return {
    title: 'Вы не дурак',
    text: g.loserId ? `В дураках — ${nameOf(g.loserId)}.` : 'Партия окончена.',
    tone: 'win' as const,
  }
})

function sortHand(cards: Card[]): Card[] {
  const t = trumpSuit.value
  return [...cards].sort((a, b) => {
    const at = Number(a.suit === t)
    const bt = Number(b.suit === t)
    if (at !== bt) return at - bt
    if (a.suit !== b.suit) return Number(a.suit) - Number(b.suit)
    return Number(a.rank) - Number(b.rank)
  })
}

function fanStyle(i: number, n: number) {
  const mid = (n - 1) / 2
  const spread = Math.min(5, 40 / Math.max(n, 1))
  const angle = (i - mid) * spread
  const lift = Math.abs(i - mid) * Math.abs(i - mid) * 0.9
  return { transform: `rotate(${angle}deg) translateY(${lift}px)` }
}

function onHandClick(card: Card) {
  if (!isPlayable(card)) return
  sound.playClick()

  if (!isDefender.value) {
    socket.attack(card)
    selected.value = null
    return
  }

  if (selected.value && sameCard(selected.value, card)) {
    selected.value = null
    return
  }

  const targets = unbeaten.value.filter((p) => canBeat(p.attack, card, trumpSuit.value))
  if (targets.length === 1 && !canTransferWith(card)) {
    socket.defend(targets[0].attack, card)
    selected.value = null
    return
  }
  selected.value = card
}

function onPairClick(pair: CardPair) {
  if (!selected.value || pair.defend) return
  if (!defendTargets.value.has(cardKey(pair.attack))) return
  socket.defend(pair.attack, selected.value)
  selected.value = null
}

function doTransfer() {
  if (!selected.value) return
  socket.transfer(selected.value)
  selected.value = null
}

function doTake() {
  sound.playClick()
  selected.value = null
  socket.take()
}

function doPass() {
  sound.playClick()
  socket.pass()
}

function doSurrender() {
  confirmSurrender.value = false
  socket.surrender()
}

function toggleMute() {
  muted.value = sound.toggleMuted()
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(location.href)
    copied.value = true
    setTimeout(() => (copied.value = false), 1600)
  } catch {
    showToast('Не удалось скопировать ссылку')
  }
}

function showToast(text: string) {
  toast.value = text
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = window.setTimeout(() => (toast.value = null), 2600)
}

const ERRORS: Record<string, string> = {
  'not your turn': 'Сейчас не ваш ход',
  'card rank is not present on table': 'Подкидывать можно только карты того же достоинства',
  'defending card cannot beat attacking card': 'Эта карта не бьёт',
  'attack limit exceeded': 'Больше подкидывать нельзя',
  'transfer is not allowed': 'Перевести сейчас нельзя',
  'table is not all beaten': 'Сначала дождитесь, пока всё побьют',
  'player has already passed': 'Вы уже сказали «Бито»',
  'defender cannot attack or toss': 'Защитник не подкидывает',
  'only defender can defend': 'Отбивается только защитник',
  'action not allowed in current phase': 'Сейчас так нельзя',
  'card is already beaten': 'Эта карта уже побита',
  'game is over': 'Партия уже окончена',
  'партия уже идет, стол заполнен': 'За этим столом уже играют',
  'стол уже заполнен': 'Стол заполнен',
  'партия уже завершена': 'Партия уже завершена',
}

watch(lastError, (e) => {
  if (!e) return
  sound.playError()
  showToast(ERRORS[e.text] ?? e.text)
})

watch(
  () => [pairs.value.length, pairs.value.filter((p) => p.defend).length, state.value?.cardsLeft],
  (next, prev) => {
    if (!prev || !state.value) return
    if (next[0] !== prev[0] || next[1] !== prev[1]) sound.playCardDrop()
    else if (next[2] !== prev[2]) sound.playCardDeal()
  },
)

watch(state, (s, prev) => {
  if (!s) return
  if (selected.value && !(s.myHand ?? []).some((c) => sameCard(c, selected.value!))) selected.value = null
  if (prev && s.defenderId !== prev.defenderId) selected.value = null
})

watch(gameOver, (g) => {
  if (!g) return
  if (g.loserId === me.value) sound.playError()
  else sound.playChipsWin()
})

async function pollWaiting() {
  if (state.value) return
  try {
    const list = await listTables()
    waitingInfo.value = list.find((t) => t.id === roomId) ?? waitingInfo.value
  } catch {
    // the socket status already tells the user if the server is down
  }
}

onMounted(() => {
  tickTimer = window.setInterval(() => (tick.value = Date.now()), 250)
  pollWaiting()
  waitTimer = window.setInterval(pollWaiting, 2500)
})

onUnmounted(() => {
  if (tickTimer) clearInterval(tickTimer)
  if (waitTimer) clearInterval(waitTimer)
  if (toastTimer) clearTimeout(toastTimer)
})

const modeLabel = computed(() => {
  const m = state.value?.mode ?? waitingInfo.value?.mode
  return m === 'perevodnoy' ? 'Переводной' : m === 'podkidnoy' ? 'Подкидной' : 'Стол'
})
</script>

<template>
  <div class="table-page">
    <AppHeader>
      <div class="room-meta">
        <button type="button" class="back" @click="router.push('/lobby')"><ArrowLeft :size="16" /> Лобби</button>
        <span class="mode">{{ modeLabel }}</span>
        <span v-if="status !== 'open'" class="conn" :class="status">
          {{ status === 'connecting' ? 'Подключаемся…' : status === 'reconnecting' ? 'Переподключаемся…' : 'Нет связи' }}
        </span>
      </div>
    </AppHeader>

    <main class="stage">
      <section v-if="!state" class="waiting">
        <div class="waiting-cards" aria-hidden="true">
          <GameCard v-for="i in 3" :key="i" face-down size="md" :class="`wc wc${i}`" />
        </div>
        <h1>{{ status === 'closed' ? 'Не удалось подключиться' : 'Ждём соперников' }}</h1>
        <p v-if="status === 'closed'" class="waiting-text">
          Стол закрыт или сервер недоступен. Вернитесь в лобби и выберите другой стол.
        </p>
        <template v-else>
          <p class="waiting-text">Партия начнётся сама, как только все сядут за стол.</p>
          <div v-if="waitingInfo" class="waiting-seats">
            <span v-for="s in waitingInfo.maxPlayers" :key="s" class="seat" :class="{ taken: s <= waitingInfo.players }">
              {{ s <= waitingInfo.players ? '●' : '' }}
            </span>
            <span class="seat-count">
              {{ waitingInfo.players }} из {{ waitingInfo.maxPlayers }} · {{ waitingInfo.deck }} карт
            </span>
          </div>
          <button type="button" class="ghost" @click="copyLink">
            <Copy :size="16" /> {{ copied ? 'Ссылка скопирована' : 'Скопировать ссылку на стол' }}
          </button>
        </template>
        <button v-if="status === 'closed'" type="button" class="primary" @click="router.push('/lobby')">В лобби</button>
      </section>

      <template v-else>
        <section class="opponents" aria-label="Соперники">
          <article
            v-for="p in opponents"
            :key="p.id"
            class="opp"
            :class="{
              attacker: p.id === state.attackerId,
              defender: p.id === state.defenderId,
              out: !p.isActive,
              offline: offline.has(p.id),
            }"
          >
            <div class="opp-hand" aria-hidden="true">
              <GameCard
                v-for="i in Math.min(p.handCount, 8)"
                :key="i"
                face-down
                size="xs"
                :style="{ transform: `rotate(${(i - (Math.min(p.handCount, 8) + 1) / 2) * 7}deg)` }"
              />
            </div>
            <div class="opp-info">
              <span class="opp-avatar">{{ p.name[0]?.toUpperCase() }}</span>
              <span class="opp-name">{{ p.name }}</span>
            </div>
            <div class="opp-sub">
              <span v-if="!p.isActive" class="badge">вышел</span>
              <span v-else-if="offline.has(p.id)" class="badge warn">нет связи</span>
              <span v-else-if="p.id === state.defenderId" class="badge def">отбивается</span>
              <span v-else-if="p.id === state.attackerId" class="badge atk">ходит</span>
              <span class="opp-count">{{ p.handCount }} {{ plural(p.handCount, ['карта', 'карты', 'карт']) }}</span>
            </div>
          </article>
        </section>

        <section class="felt" aria-label="Стол">
          <div class="pile deck-pile">
            <div class="deck-stack">
              <GameCard
                v-if="state.cardsLeft > 0"
                :card="state.trumpCard"
                size="sm"
                trump
                class="trump-card"
              />
              <GameCard
                v-for="i in Math.max(0, deckBacks - 1)"
                :key="i"
                face-down
                size="sm"
                class="deck-back"
                :style="{ transform: `translate(${i * 1.5}px, ${-i * 1.5}px)` }"
              />
              <div v-if="state.cardsLeft === 0" class="trump-only" :class="{ red: state.trumpSuit === 2 || state.trumpSuit === 3 }">
                {{ getSuitSymbol(state.trumpSuit) }}
              </div>
            </div>
            <span class="pile-label">
              {{ state.cardsLeft ? `${state.cardsLeft} в колоде` : 'колода пуста' }}
            </span>
          </div>

          <div class="battle">
            <TransitionGroup name="pair" tag="div" class="pairs">
              <div
                v-for="pair in pairs"
                :key="cardKey(pair.attack)"
                class="pair"
                :class="{ target: defendTargets.has(cardKey(pair.attack)) }"
                @click="onPairClick(pair)"
              >
                <GameCard :card="pair.attack" size="md" :trump="pair.attack.suit === state.trumpSuit" />
                <Transition name="defend">
                  <GameCard
                    v-if="pair.defend"
                    :card="pair.defend"
                    size="md"
                    class="defend-card"
                    :trump="pair.defend.suit === state.trumpSuit"
                  />
                </Transition>
              </div>
            </TransitionGroup>
            <p v-if="!pairs.length" class="felt-hint">
              {{ isAttacker && canAct ? 'Положите карту, чтобы начать ход' : `Ход за ${nameOf(state.attackerId)}` }}
            </p>
          </div>

          <div class="pile discard-pile">
            <div class="discard-stack" :class="{ empty: !state.discardSize }">
              <GameCard v-if="state.discardSize" face-down size="sm" style="transform: rotate(14deg)" />
              <GameCard v-if="state.discardSize > 6" face-down size="sm" class="discard-2" />
            </div>
            <span class="pile-label">бито: {{ state.discardSize }}</span>
          </div>
        </section>

        <section class="me-zone" :class="{ 'my-turn': myTurn }">
          <div class="status-row">
            <div class="status">
              <span class="role" v-if="canAct">
                {{ isDefender ? 'Вы отбиваетесь' : isAttacker ? 'Вы атакуете' : 'Вы подкидываете' }}
              </span>
              <span class="status-text">{{ statusText }}</span>
            </div>

            <div class="actions">
              <button v-if="showTransfer" type="button" class="act" @click="doTransfer">Перевести</button>
              <button v-if="showTake" type="button" class="act warn" @click="doTake">Беру</button>
              <button v-if="showBeaten" type="button" class="act" @click="doPass">Бито</button>
              <button v-if="showEnough" type="button" class="act" @click="doPass">Хватит</button>
              <button type="button" class="icon-btn" :title="muted ? 'Включить звук' : 'Выключить звук'" @click="toggleMute">
                <VolumeX v-if="muted" :size="17" />
                <Volume2 v-else :size="17" />
              </button>
              <button
                v-if="canAct"
                type="button"
                class="icon-btn"
                title="Сдаться"
                aria-label="Сдаться"
                @click="confirmSurrender = true"
              >
                <Flag :size="17" />
              </button>
            </div>
          </div>

          <div class="timer" :class="{ urgent: turnSeconds <= 5 }" v-if="turnLeft > 0 && canAct">
            <div class="timer-bar" :style="{ transform: `scaleX(${turnFraction})` }"></div>
            <span class="timer-text">{{ turnSeconds }} с</span>
          </div>

          <div class="hand" :class="{ crowded: hand.length > 10 }">
            <TransitionGroup name="hand">
              <div v-for="(card, i) in hand" :key="cardKey(card)" class="hand-slot" :style="{ zIndex: i }">
                <div class="fan" :style="fanStyle(i, hand.length)">
                <GameCard
                  :card="card"
                  size="lg"
                  :trump="card.suit === state.trumpSuit"
                  :playable="isPlayable(card)"
                  :dimmed="myTurn && !isPlayable(card)"
                  :selected="!!selected && sameCard(selected, card)"
                  @click="onHandClick(card)"
                />
                </div>
              </div>
            </TransitionGroup>
            <p v-if="!hand.length && iAmActive" class="felt-hint">У вас нет карт</p>
          </div>
        </section>
      </template>
    </main>

    <Transition name="toast">
      <div v-if="toast" class="toast" role="status">{{ toast }}</div>
    </Transition>

    <Transition name="fade">
      <div v-if="confirmSurrender" class="overlay" @click.self="confirmSurrender = false">
        <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="surrender-title">
          <h2 id="surrender-title">Сдаться?</h2>
          <p>Вы выйдете из партии, а ваши карты уйдут в отбой.</p>
          <div class="dialog-actions">
            <button type="button" class="ghost" @click="confirmSurrender = false">Играть дальше</button>
            <button type="button" class="primary danger" @click="doSurrender">Сдаться</button>
          </div>
        </div>
      </div>
    </Transition>

    <Transition name="fade">
      <div v-if="result" class="overlay">
        <div class="dialog result" :class="result.tone" role="dialog" aria-modal="true" aria-labelledby="result-title">
          <div class="result-cards" aria-hidden="true">
            <GameCard :card="{ suit: 3, rank: 14 }" size="md" class="rc-a" />
            <GameCard :card="{ suit: 2, rank: 12 }" size="md" class="rc-b" />
          </div>
          <h2 id="result-title">{{ result.title }}</h2>
          <p>{{ result.text }}</p>
          <button type="button" class="primary" @click="router.push('/lobby')">В лобби</button>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.table-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  background:
    radial-gradient(ellipse at 50% 30%, rgba(246, 235, 211, 0.7), rgba(246, 235, 211, 0) 60%),
    #f5f4ef;
  color: #1b1916;
  font-family: 'Inter', -apple-system, 'Helvetica Neue', Arial, sans-serif;
  overflow-x: hidden;
}

.room-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.back {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 14px 0 10px;
  border: 0;
  border-radius: 999px;
  background: #e9e5db;
  color: #1b1916;
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.back:hover {
  background: #e0dace;
}

.mode {
  font-size: 15px;
  color: #57534b;
  white-space: nowrap;
}

.conn {
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  background: #f1e6cf;
  color: #7a5a22;
  white-space: nowrap;
}

.conn.closed {
  background: #f5e4df;
  color: #8f3a32;
}

.stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  width: min(1120px, 100%);
  margin: 0 auto;
  padding: 18px 16px 0;
  box-sizing: border-box;
}

.waiting {
  margin: auto;
  padding: 40px 16px 80px;
  text-align: center;
}

.waiting-cards {
  position: relative;
  width: 200px;
  height: 150px;
  margin: 0 auto 26px;
}

.wc {
  position: absolute;
  left: 54px;
  top: 0;
  animation: float 3.6s ease-in-out infinite;
}

.wc1 {
  transform: rotate(-16deg) translateX(-38px);
  animation-delay: -1.2s;
}

.wc2 {
  animation-delay: -0.6s;
}

.wc3 {
  transform: rotate(16deg) translateX(38px);
}

@keyframes float {
  50% {
    margin-top: -8px;
  }
}

.waiting h1 {
  margin: 0;
  font-size: clamp(30px, 4vw, 44px);
  font-weight: 500;
  letter-spacing: -0.03em;
}

.waiting-text {
  max-width: 420px;
  margin: 12px auto 0;
  font-size: 16px;
  line-height: 1.5;
  color: #57534b;
}

.waiting-seats {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  margin: 26px 0 22px;
}

.seat {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  box-shadow: inset 0 0 0 1.5px #b9b1a1;
  font-size: 0;
}

.seat.taken {
  background: #36322c;
  box-shadow: none;
}

.seat-count {
  margin-left: 6px;
  font-size: 14px;
  color: #6f695e;
}

.opponents {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 14px;
}

.opp {
  min-width: 150px;
  padding: 12px 16px 12px;
  border-radius: 22px;
  background: rgba(250, 249, 245, 0.9);
  box-shadow: 0 0 0 1px rgba(214, 208, 196, 0.9);
  transition:
    box-shadow 0.3s,
    opacity 0.3s;
}

.opp.attacker {
  box-shadow:
    0 0 0 2px #36322c,
    0 16px 30px -22px rgba(54, 50, 44, 0.6);
}

.opp.defender {
  box-shadow:
    0 0 0 2px #a1473f,
    0 16px 30px -22px rgba(161, 71, 63, 0.5);
}

.opp.out {
  opacity: 0.5;
}

.opp.offline .opp-avatar {
  background: #b9b1a1;
}

.opp-hand {
  display: flex;
  justify-content: center;
  height: 52px;
  margin-bottom: 6px;
}

.opp-hand > * {
  margin: 0 -9px;
  transform-origin: 50% 120%;
}

.opp-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.opp-avatar {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: #36322c;
  color: #f5f4ef;
  font-size: 12px;
}

.opp-name {
  font-size: 15px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 120px;
}

.opp-sub {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  font-size: 12px;
  color: #8a8375;
}

.badge {
  padding: 2px 8px;
  border-radius: 999px;
  background: #ede9e0;
  color: #57534b;
}

.badge.atk {
  background: #36322c;
  color: #f5f4ef;
}

.badge.def {
  background: #f5e4df;
  color: #8f3a32;
}

.badge.warn {
  background: #f1e6cf;
  color: #7a5a22;
}

.felt {
  position: relative;
  display: grid;
  grid-template-columns: 110px 1fr 110px;
  align-items: center;
  gap: 12px;
  min-height: 300px;
  margin: 20px 0;
  padding: 24px;
  border-radius: 40px;
  background:
    radial-gradient(ellipse at 50% 40%, rgba(255, 253, 247, 0.9), rgba(239, 235, 226, 0.4) 70%),
    #efebe2;
  box-shadow:
    0 0 0 8px #d3d0c8 inset,
    0 30px 60px -40px rgba(70, 62, 48, 0.5);
}

.pile {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.deck-stack,
.discard-stack {
  position: relative;
  width: 110px;
  height: 90px;
  display: grid;
  place-items: center;
}

.deck-stack > * {
  position: absolute;
}

.trump-card {
  transform: translateX(22px) rotate(90deg);
}

.deck-back {
  left: -8px;
}

.trump-only {
  display: grid;
  place-items: center;
  width: 54px;
  height: 54px;
  border-radius: 50%;
  background: #fdfcf8;
  box-shadow: 0 0 0 2px #c9a66b;
  font-family: Georgia, serif;
  font-size: 28px;
  color: #2b2824;
}

.trump-only.red {
  color: #a1473f;
}

.discard-stack.empty {
  border-radius: 12px;
  box-shadow: inset 0 0 0 1.5px dashed #c8c0ae;
  outline: 1.5px dashed #cdc6b6;
  outline-offset: -14px;
}

.discard-stack > * {
  position: absolute;
}

.discard-2 {
  transform: rotate(-8deg) translate(4px, 4px);
}

.pile-label {
  font-size: 13px;
  color: #8a8375;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.battle {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 220px;
}

.pairs {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 18px 26px;
}

.pair {
  position: relative;
  width: 112px;
  height: 160px;
  border-radius: 14px;
  transition: box-shadow 0.2s;
}

.pair.target {
  cursor: pointer;
  box-shadow: 0 0 0 3px rgba(54, 50, 44, 0.3);
  animation: target 1.2s ease-in-out infinite;
}

@keyframes target {
  50% {
    box-shadow: 0 0 0 6px rgba(54, 50, 44, 0.12);
  }
}

.defend-card {
  position: absolute;
  left: 22px;
  top: 22px;
  transform: rotate(9deg);
}

.felt-hint {
  margin: 0;
  font-size: 15px;
  color: #8a8375;
}

.me-zone {
  position: sticky;
  bottom: 0;
  margin: 0 -16px;
  padding: 14px 16px 0;
  border-radius: 32px 32px 0 0;
  background: rgba(247, 246, 241, 0.86);
  box-shadow: 0 -1px 0 rgba(214, 208, 196, 0.9);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
}

.status-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.status {
  flex: 1;
  min-width: 220px;
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 4px 12px;
}

.role {
  font-size: 13px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #9a9384;
}

.status-text {
  font-size: 18px;
  letter-spacing: -0.01em;
}

.my-turn .status-text {
  font-weight: 500;
}

.actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.act {
  height: 46px;
  padding: 0 24px;
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
  animation: pop 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.act:hover {
  background: #1b1916;
}

.act:active {
  transform: scale(0.97);
}

.act.warn {
  background: #a1473f;
}

.act.warn:hover {
  background: #8a3a33;
}

@keyframes pop {
  from {
    transform: scale(0.85);
    opacity: 0;
  }
}

.icon-btn {
  display: grid;
  place-items: center;
  width: 46px;
  height: 46px;
  border: 0;
  border-radius: 999px;
  background: #e9e5db;
  color: #1b1916;
  cursor: pointer;
}

.icon-btn:hover {
  background: #e0dace;
}

.timer {
  position: relative;
  height: 6px;
  margin-top: 12px;
  border-radius: 999px;
  background: #e5e0d5;
  overflow: hidden;
}

.timer-bar {
  position: absolute;
  inset: 0;
  transform-origin: left;
  background: #36322c;
  transition: transform 0.25s linear;
}

.timer.urgent .timer-bar {
  background: #a1473f;
}

.timer-text {
  position: absolute;
  right: 0;
  top: -22px;
  font-size: 12px;
  color: #8a8375;
  font-variant-numeric: tabular-nums;
}

.timer.urgent .timer-text {
  color: #a1473f;
}

.hand {
  display: flex;
  justify-content: center;
  align-items: flex-end;
  min-height: 190px;
  padding: 34px 8px 18px;
  overflow-x: auto;
  overflow-y: hidden;
}

.hand-slot {
  margin: 0 -14px;
}

.fan {
  transform-origin: 50% 160%;
  transition: transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.hand.crowded .hand-slot {
  margin: 0 -26px;
}

.hand-slot:hover {
  z-index: 50 !important;
}

.ghost {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 46px;
  padding: 0 20px;
  border: 0;
  border-radius: 999px;
  background: #e9e5db;
  color: #1b1916;
  font: inherit;
  font-size: 15px;
  cursor: pointer;
}

.ghost:hover {
  background: #e0dace;
}

.primary {
  height: 50px;
  margin-top: 18px;
  padding: 0 28px;
  border: 0;
  border-radius: 999px;
  background: #36322c;
  color: #f5f4ef;
  font: inherit;
  font-size: 16px;
  cursor: pointer;
}

.primary:hover {
  background: #1b1916;
}

.primary.danger {
  background: #a1473f;
}

.primary.danger:hover {
  background: #8a3a33;
}

.toast {
  position: fixed;
  z-index: 200;
  left: 50%;
  bottom: 250px;
  translate: -50% 0;
  max-width: calc(100% - 32px);
  padding: 12px 20px;
  border-radius: 999px;
  background: #1b1916;
  color: #f5f4ef;
  font-size: 15px;
  box-shadow: 0 18px 40px -16px rgba(27, 25, 22, 0.6);
}

.overlay {
  position: fixed;
  z-index: 300;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 16px;
  background: rgba(40, 36, 30, 0.35);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}

.dialog {
  width: min(420px, 100%);
  padding: 30px 28px 26px;
  box-sizing: border-box;
  border-radius: 30px;
  background: #faf9f5;
  text-align: center;
  box-shadow: 0 40px 80px -30px rgba(40, 36, 30, 0.6);
}

.dialog h2 {
  margin: 0;
  font-size: 30px;
  font-weight: 500;
  letter-spacing: -0.03em;
}

.dialog p {
  margin: 10px 0 0;
  font-size: 16px;
  line-height: 1.5;
  color: #57534b;
}

.dialog-actions {
  display: flex;
  justify-content: center;
  gap: 10px;
  margin-top: 22px;
}

.dialog-actions .primary {
  margin-top: 0;
}

.result-cards {
  position: relative;
  height: 150px;
  margin-bottom: 14px;
}

.rc-a,
.rc-b {
  position: absolute;
  top: 0;
  left: 50%;
}

.rc-a {
  transform: translateX(-80%) rotate(-12deg);
}

.rc-b {
  transform: translateX(-20%) rotate(10deg);
}

.result.lose h2 {
  color: #a1473f;
}

.pair-enter-active {
  transition:
    opacity 0.35s,
    transform 0.45s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.pair-enter-from {
  opacity: 0;
  transform: translateY(60px) scale(0.9);
}

.pair-leave-active {
  pointer-events: none;
  transition:
    opacity 0.3s,
    transform 0.3s;
}

.pair-leave-to {
  opacity: 0;
  transform: scale(0.85);
}

.defend-enter-active {
  transition:
    opacity 0.3s,
    transform 0.4s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.defend-enter-from {
  opacity: 0;
  transform: rotate(30deg) translate(30px, 60px);
}

.hand-enter-active {
  transition:
    opacity 0.35s,
    transform 0.45s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.hand-enter-from {
  opacity: 0;
  transform: translateY(-60px);
}

.hand-leave-active {
  pointer-events: none;
  transition:
    opacity 0.25s,
    transform 0.25s;
  position: absolute;
}

.hand-leave-to {
  opacity: 0;
  transform: translateY(-80px);
}

.toast-enter-active,
.toast-leave-active,
.fade-enter-active,
.fade-leave-active {
  transition:
    opacity 0.25s,
    translate 0.25s;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  translate: -50% 10px;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 760px) {
  .felt {
    grid-template-columns: 1fr;
    padding: 18px 12px;
    border-radius: 30px;
  }

  .pile {
    flex-direction: row;
  }

  .deck-pile {
    order: -1;
    justify-content: center;
  }

  .discard-pile {
    justify-content: center;
  }

  .pair {
    width: 84px;
    height: 120px;
  }

  .pair :deep(.game-card) {
    --w: 70px;
  }

  .defend-card {
    left: 14px;
    top: 14px;
  }

  .hand-slot :deep(.game-card) {
    --w: 72px;
  }

  .hand-slot {
    margin: 0 -18px;
  }

  .opp {
    min-width: 120px;
    padding: 10px 12px;
  }

  .status-text {
    font-size: 16px;
  }

  .toast {
    bottom: 200px;
  }
}
</style>
