<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { AnimatePresence } from 'motion-v'
import { useElementSize, useMediaQuery } from '@vueuse/core'
import { PhX } from '@phosphor-icons/vue'
import AvatarGlyph from '../../components/client/AvatarGlyph.vue'
import ToastStack from '../../components/app/ToastStack.vue'
import OverlayPanel from '../../components/client/OverlayPanel.vue'
import ChatBox from '../../components/client/ChatBox.vue'
import GameCard from '../../components/game/GameCard.vue'
import GameTopBar from '../../components/game/GameTopBar.vue'
import TableSeat from '../../components/game/TableSeat.vue'
import BattleArea from '../../components/game/BattleArea.vue'
import QuickChat from '../../components/game/QuickChat.vue'
import PostGameChat from '../../components/game/PostGameChat.vue'
import FlightLayer from '../../components/game/FlightLayer.vue'
import { getTable, inviteToTable } from '../../api/tables'
import { useAuthStore } from '../../stores/auth'
import { useSettingsStore } from '../../stores/settings'
import { useGameSocket } from '../../composables/useGameSocket'
import { useCardFlights, type FlightKind } from '../../composables/useCardFlights'
import { useGameSound } from '../../composables/useGameSound'
import { useTurnAlert } from '../../composables/useTurnAlert'
import { useLoungeMusic } from '../../composables/useLoungeMusic'
import { useToast } from '../../composables/useToast'
import { useThemeColor } from '../../composables/useThemeColor'
import { useFriends } from '../../composables/useFriends'
import { avatarTone } from '../../composables/avatar'
import { MIN_STAKE } from '../../constants/economy'
import { canBeat, cardKey, sameCard, type Card, type CardPair, type PlayerView, type TableInfo } from '../../types/game'
import type { ChatEntry, Friend } from '../../types/social'

const TURN_MS = 20_000
const BUBBLE_MS = 4500
const SEAT_ANGLES: Record<number, number[]> = {
  1: [270],
  2: [222, 318],
  3: [196, 270, 344],
  4: [190, 238, 302, 350],
  5: [182, 224, 270, 316, 358],
}

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const settings = useSettingsStore()
const sound = useGameSound()
const toast = useToast()
const wide = useMediaQuery('(min-width: 1100px)')
const phone = useMediaQuery('(max-width: 767px)')
const short = useMediaQuery('(max-height: 820px) and (min-width: 768px)')

const roomId = String(route.params.id)
const linkKey = typeof route.query.key === 'string' ? route.query.key : ''
const watching = route.query.watch === '1'
const socket = useGameSocket(roomId, linkKey, watching)
const { state, gameOver, status, offline, lastError, chat, rematch, rematchCancelled, rematchTable, watchers } = socket

const me = computed(() => auth.user?.id ?? '')
const gameEl = ref<HTMLElement | null>(null)
useThemeColor(gameEl, () => settings.theme)
const { flights, reveal: flightReveal, done: flightDone, enabled: flightsEnabled } = useCardFlights(gameEl, state, me)
const launching = ref<string | null>(null)
const landedAt: Partial<Record<FlightKind, number>> = {}
let launchTimer: number | null = null
const selected = ref<Card | null>(null)
const confirmSurrender = ref(false)
const copied = ref(false)
const waitingInfo = ref<TableInfo | null>(null)
const tick = ref(Date.now())
const chatOpen = ref(wide.value)
const resultOpen = ref(false)
const unread = ref(0)
const bubbles = ref<Record<string, ChatEntry>>({})

let tickTimer: number | null = null
let waitTimer: number | null = null
const bubbleTimers = new Map<string, number>()

const pairs = computed<CardPair[]>(() => state.value?.table ?? [])
const hand = computed<Card[]>(() => sortHand(state.value?.myHand ?? []))
const trumpSuit = computed(() => state.value?.trumpSuit ?? -1)
const phase = computed(() => state.value?.phase)
const mode = computed(() => state.value?.mode ?? waitingInfo.value?.mode)
const isPerevodnoy = computed(() => mode.value === 'perevodnoy')

const players = computed<PlayerView[]>(() => state.value?.players ?? [])
const meView = computed(() => players.value.find((p) => p.id === me.value))
const iAmActive = computed(() => meView.value?.isActive ?? false)
const isDefender = computed(() => state.value?.defenderId === me.value)
const isAttacker = computed(() => state.value?.attackerId === me.value)
const myName = computed(() => meView.value?.name ?? auth.user?.username ?? 'Вы')

const anchorId = computed(() => (watching ? (players.value[0]?.id ?? '') : me.value))
const anchor = computed(() => players.value.find((p) => p.id === anchorId.value))

const opponents = computed(() => {
  const list = players.value
  const idx = list.findIndex((p) => p.id === anchorId.value)
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
  if (watching) return watchText.value
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

const watchText = computed(() => {
  const s = state.value
  if (!s) return ''
  const def = nameOf(s.defenderId)
  if (s.phase === 'giving_more') return `${def} берёт, остальные подкидывают`
  if (!pairs.value.length) return `Ходит ${nameOf(s.attackerId)}`
  if (unbeaten.value.length) return `${def} отбивается`
  return 'Всё побито, ждём «Бито»'
})

const roleText = computed(() => {
  if (!canAct.value) return ''
  return isDefender.value ? 'Вы отбиваетесь' : isAttacker.value ? 'Вы атакуете' : 'Вы подкидываете'
})

const myTurn = computed(() => {
  if (!canAct.value) return false
  if (isDefender.value) return phase.value === 'battle' && unbeaten.value.length > 0
  if (!pairs.value.length) return isAttacker.value
  return !state.value?.youPassed && (phase.value === 'giving_more' || unbeaten.value.length === 0)
})

const actorId = computed(() => {
  const s = state.value
  if (!s || s.phase === 'finished' || gameOver.value) return null
  if (s.phase === 'battle' && unbeaten.value.length) return s.defenderId
  return s.attackerId
})

const turnLeft = computed(() => {
  const s = state.value
  if (!s || !s.turnLeftMs) return 0
  return Math.max(0, s.turnLeftMs - (tick.value - socket.stateReceivedAt.value))
})
const turnSeconds = computed(() => Math.ceil(turnLeft.value / 1000))
const turnFraction = computed(() => Math.min(1, turnLeft.value / TURN_MS))

useTurnAlert(myTurn, turnLeft)
useLoungeMusic()

const roleOf = (id: string) => (id === state.value?.defenderId ? 'defender' : id === state.value?.attackerId ? 'attacker' : null)

interface SeatView {
  key: string
  player: PlayerView | null
  waitingName: string | null
  x: number
  y: number
}

const seats = computed<SeatView[]>(() => {
  if (state.value) {
    const angles = SEAT_ANGLES[opponents.value.length] ?? SEAT_ANGLES[5]
    return opponents.value.map((p, i) => ({ key: p.id, player: p, waitingName: p.name, ...polar(angles[i]) }))
  }
  const total = waitingInfo.value?.maxPlayers ?? 2
  const taken = waitingInfo.value?.players ?? 1
  const angles = SEAT_ANGLES[total - 1] ?? SEAT_ANGLES[5]
  return angles.map((a, i) => ({ key: `wait-${i}`, player: null, waitingName: i < taken - 1 ? 'Игрок' : null, ...polar(a) }))
})

function polar(deg: number) {
  const rad = (deg * Math.PI) / 180
  return { x: 50 + 50 * Math.cos(rad), y: 50 + 50 * Math.sin(rad) }
}

type StandingKind = 'place' | 'fool' | 'surrendered' | 'draw'

interface Standing {
  id: string
  name: string
  place: number
  kind: StandingKind
  me: boolean
  payout: number | null
}

const standings = computed<Standing[]>(() => {
  const g = gameOver.value
  if (!g?.places?.length) return []
  const loserIdx = g.loserId ? g.places.indexOf(g.loserId) : -1
  return g.places.map((id, i) => {
    const payout = g.payouts?.[id] ?? null
    let kind: StandingKind = 'place'
    if (g.isDraw) kind = 'draw'
    else if (i === loserIdx) kind = 'fool'
    else if ((loserIdx >= 0 && i > loserIdx) || (payout !== null && payout < 0)) kind = 'surrendered'
    return { id, name: nameOf(id), place: i + 1, kind, me: id === me.value, payout }
  })
})

const STANDING_LABEL: Record<StandingKind, (s: Standing) => string> = {
  place: (s) => `${s.place} место`,
  fool: () => 'дурак',
  surrendered: () => 'сдался',
  draw: () => 'ничья',
}

function chips(n: number): string {
  const abs = Math.abs(n).toLocaleString('ru-RU')
  return n > 0 ? `+${abs}` : n < 0 ? `−${abs}` : '0'
}

const myPayout = computed(() => standings.value.find((s) => s.me)?.payout ?? null)

const payoutLine = computed(() => {
  const p = myPayout.value
  if (p === null) return ''
  if (p > 0) return `Выигрыш ${chips(p)} фишек`
  if (p < 0) return `Ставка ${Math.abs(p).toLocaleString('ru-RU')} ушла в банк`
  return 'Ставка вернулась'
})

const result = computed(() => {
  const g = gameOver.value
  if (!g) return null
  if (watching) {
    if (g.isDraw) return { title: 'Ничья', text: 'Последние вышли одновременно — дураков нет.', tone: 'draw' as const }
    return { title: 'Партия окончена', text: g.loserId ? `В дураках — ${nameOf(g.loserId)}.` : 'Дураков нет.', tone: 'draw' as const }
  }
  if (g.isDraw) return { title: 'Ничья', text: 'Последние вышли одновременно — дураков нет.', tone: 'draw' as const }
  if (g.loserId === me.value) return { title: 'Вы — дурак', text: 'Бывает. Отыграетесь в следующей партии.', tone: 'lose' as const }
  const mine = standings.value.find((s) => s.me)
  if (mine?.kind === 'surrendered') return { title: 'Вы сдались', text: 'Партия доиграна без вас.', tone: 'lose' as const }
  if (mine && mine.place > 1) {
    return { title: `${mine.place} место`, text: g.loserId ? `В дураках — ${nameOf(g.loserId)}.` : 'Партия окончена.', tone: 'win' as const }
  }
  if (g.winnerId === me.value) return { title: 'Победа', text: 'Вы первым избавились от карт.', tone: 'win' as const }
  return {
    title: 'Вы не дурак',
    text: g.loserId ? `В дураках — ${nameOf(g.loserId)}.` : 'Партия окончена.',
    tone: 'win' as const,
  }
})

const modeLabel = computed(() => (mode.value === 'perevodnoy' ? 'Переводной' : mode.value === 'podkidnoy' ? 'Подкидной' : 'Стол'))
const deckSize = computed(() => waitingInfo.value?.deck ?? null)

const searching = computed(
  () => !state.value && status.value !== 'closed' && (waitingInfo.value?.matchmaking ?? route.query.search === '1'),
)

const searchStartedAt = Date.now()

const elapsedText = computed(() => {
  const since = waitingInfo.value ? Math.min(new Date(waitingInfo.value.createdAt).getTime(), searchStartedAt) : searchStartedAt
  const sec = Math.max(0, Math.floor((tick.value - since) / 1000))
  return `${Math.floor(sec / 60)}:${String(sec % 60).padStart(2, '0')}`
})

function cancelSearch() {
  sound.playClick()
  router.push('/lobby')
}
const stake = computed(() => gameOver.value?.stake ?? state.value?.stake ?? waitingInfo.value?.stake ?? MIN_STAKE)

const battleHint = computed(() => {
  if (!state.value) return ''
  return isAttacker.value && canAct.value ? 'Положите карту, чтобы начать ход' : `Ход за ${nameOf(state.value.attackerId)}`
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

const FAN_MAX_LIFT = 4

function fanAngle(i: number, n: number) {
  return (i - (n - 1) / 2) * Math.min(5, 40 / Math.max(n, 1))
}

function fanLift(i: number, n: number) {
  const d = Math.min(Math.abs(i - (n - 1) / 2), FAN_MAX_LIFT)
  return d * d * 0.9
}

function fanStyle(i: number, n: number) {
  return { transform: `rotate(${fanAngle(i, n)}deg) translateY(${fanLift(i, n)}px)` }
}

const handEl = ref<HTMLElement | null>(null)
const { width: handWidth } = useElementSize(handEl)
const handCardW = computed(() => (phone.value ? 66 : short.value ? 84 : wide.value ? 104 : 88))

const handOverlap = computed(() => {
  const n = hand.value.length
  const w = handCardW.value
  const roomy = phone.value ? 16 : 14
  if (n < 2 || !handWidth.value) return roomy
  const step = (handWidth.value - 24 - w * 1.6) / (n - 1)
  return Math.min(Math.max(roomy, (w - step) / 2), (w - 18) / 2)
})

const handStyle = computed(() => {
  const n = hand.value.length
  const w = handCardW.value
  const edge = Math.max(0, n - 1)
  const angle = (Math.abs(fanAngle(edge, n)) * Math.PI) / 180
  const drop = (w / 2) * Math.sin(angle) + fanLift(edge, n)
  return { paddingBottom: `${Math.ceil(10 + drop)}px` }
})

function lift(card: Card) {
  launching.value = cardKey(card)
  if (launchTimer) clearTimeout(launchTimer)
  launchTimer = window.setTimeout(() => (launching.value = null), 1500)
}

function onLand(kind: FlightKind) {
  const now = performance.now()
  if (now - (landedAt[kind] ?? 0) < 40) return
  landedAt[kind] = now
  if (kind === 'throw') sound.playCardDrop()
  else if (kind === 'sweep') sound.playCardSweep()
  else if (kind === 'take') sound.playCardTake()
  else sound.playCardDeal()
}

function onHandClick(card: Card) {
  if (!isPlayable(card)) return
  sound.playCardPick()

  if (!isDefender.value) {
    lift(card)
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
    lift(card)
    socket.defend(targets[0].attack, card)
    selected.value = null
    return
  }
  selected.value = card
}

function onPairClick(pair: CardPair) {
  if (!selected.value || pair.defend) return
  if (!defendTargets.value.has(cardKey(pair.attack))) return
  lift(selected.value)
  socket.defend(pair.attack, selected.value)
  selected.value = null
}

function doTransfer() {
  if (!selected.value) return
  lift(selected.value)
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

function toggleSound() {
  settings.soundOn = !settings.soundOn
  if (settings.soundOn) sound.playClick()
}

function toggleChat() {
  chatOpen.value = !chatOpen.value
}

function say(text: string) {
  socket.say(text)
}

async function copyLink() {
  try {
    await navigator.clipboard.writeText(tableLink())
    copied.value = true
    setTimeout(() => (copied.value = false), 1600)
  } catch {
    toast.show('Не удалось скопировать ссылку', 'error')
  }
}

const chatPlaceholder = watching ? 'Зрители только читают чат' : 'Написать за столом…'

function tableLink() {
  const key = waitingInfo.value?.key || linkKey
  return `${location.origin}/table/${roomId}${key ? `?key=${encodeURIComponent(key)}` : ''}`
}

const { friends, refresh: refreshFriends } = useFriends()
const calledUntil = ref<Record<string, number>>({})
const seatsLeft = computed(() => (waitingInfo.value ? waitingInfo.value.maxPlayers - Math.max(waitingInfo.value.players, 1) : 0))
const canCall = computed(() => !watching && !state.value && status.value !== 'closed' && !!waitingInfo.value && !waitingInfo.value.matchmaking)
const showCall = computed(() => canCall.value && seatsLeft.value > 0)
const callable = computed(() => friends.value.filter((f) => f.status === 'online' || isCalled(f.id)))

function isCalled(id: string) {
  return (calledUntil.value[id] ?? 0) > tick.value
}

async function callFriend(friend: Friend) {
  if (isCalled(friend.id)) return
  try {
    const { expiresAt } = await inviteToTable(roomId, friend.id)
    calledUntil.value = { ...calledUntil.value, [friend.id]: new Date(expiresAt).getTime() }
    toast.show(`Позвали ${friend.username}, место держим 45 секунд`, 'success')
    pollWaiting()
  } catch (err) {
    toast.show(err instanceof Error ? err.message : 'Не удалось позвать', 'error')
  }
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
  'слишком часто, подождите секунду': 'Не так быстро — подождите секунду',
  'не хватает фишек на ставку': 'Не хватает фишек на ставку этого стола',
  'вы уже играете за другим столом': 'Вы уже играете за другим столом',
  'не удалось начать партию, попробуйте ещё раз': 'Не удалось начать партию, попробуйте ещё раз',
  'касса недоступна, попробуйте позже': 'Касса недоступна, попробуйте позже',
}

watch(lastError, (e) => {
  if (!e) return
  launching.value = null
  sound.playError()
  toast.show(ERRORS[e.text] ?? e.text, 'error')
})

watch(
  () => [pairs.value.length, pairs.value.filter((p) => p.defend).length, state.value?.cardsLeft],
  (next, prev) => {
    if (flightsEnabled || !prev || !state.value) return
    if (next[0] !== prev[0] || next[1] !== prev[1]) sound.playCardDrop()
    else if (next[2] !== prev[2]) sound.playCardDeal()
  },
)

watch(state, (s, prev) => {
  if (!s) return
  if (!prev && !gameOver.value) sound.playShuffle()
  launching.value = null
  if (selected.value && !(s.myHand ?? []).some((c) => sameCard(c, selected.value!))) selected.value = null
  if (prev && s.defenderId !== prev.defenderId) selected.value = null
})

const iVotedRematch = computed(() => !!rematch.value?.votes.includes(me.value))
const rivalWantsRematch = computed(() => (rematch.value?.votes.length ?? 0) > (iVotedRematch.value ? 1 : 0))
const rematchOpen = computed(() => !watching && !!gameOver.value && status.value === 'open' && !rematchCancelled.value && !rematchTable.value)

function voteRematch() {
  sound.playClick()
  socket.rematchVote()
}

function declineRematch() {
  sound.playClick()
  socket.rematchDecline()
}

watch(rivalWantsRematch, (wants) => {
  if (!wants || iVotedRematch.value) return
  resultOpen.value = true
  sound.playYourTurn()
})

watch(rematchCancelled, (reason) => {
  if (reason) toast.show(`Реванша не будет: ${reason}`)
})

watch(rematchTable, (id) => {
  if (!id) return
  if (watching) {
    if (!waitingInfo.value?.private) router.push({ name: 'table', params: { id }, query: { watch: '1' } })
    return
  }
  sound.playShuffle()
  router.push({ name: 'table', params: { id } })
})

watch(gameOver, (g) => {
  if (!g) return
  resultOpen.value = true
  if (watching) return
  if (g.isDraw) sound.playSuccess()
  else if (g.loserId === me.value) sound.playLose()
  else sound.playChipsWin()
})

watch(
  () => chat.value.length,
  (len, prevLen) => {
    for (const entry of chat.value.slice(prevLen ?? 0, len)) {
      if (entry.system) continue
      showBubble(entry)
      if (entry.userId !== me.value) {
        if (!chatOpen.value) unread.value++
        sound.playMessage()
      }
    }
  },
)

watch(chatOpen, (open) => {
  if (open) unread.value = 0
})

watch(wide, (isWide) => {
  if (!isWide) chatOpen.value = false
})

function showBubble(entry: ChatEntry) {
  bubbles.value = { ...bubbles.value, [entry.userId]: entry }
  const prev = bubbleTimers.get(entry.userId)
  if (prev) clearTimeout(prev)
  bubbleTimers.set(
    entry.userId,
    window.setTimeout(() => {
      const next = { ...bubbles.value }
      if (next[entry.userId]?.id === entry.id) delete next[entry.userId]
      bubbles.value = next
    }, BUBBLE_MS),
  )
}

async function pollWaiting() {
  if (state.value && waitingInfo.value) return
  try {
    waitingInfo.value = await getTable(roomId)
  } catch {
    return
  }
  if (!state.value && !waitingInfo.value.matchmaking) refreshFriends()
}

onMounted(() => {
  tickTimer = window.setInterval(() => (tick.value = Date.now()), 250)
  pollWaiting()
  waitTimer = window.setInterval(pollWaiting, 2500)
})

onUnmounted(() => {
  if (tickTimer) clearInterval(tickTimer)
  if (launchTimer) clearTimeout(launchTimer)
  if (waitTimer) clearInterval(waitTimer)
  bubbleTimers.forEach((t) => clearTimeout(t))
})
</script>

<template>
  <div ref="gameEl" class="game" :class="{ 'chat-docked': wide && chatOpen }" :data-theme="settings.theme" :data-felt="settings.felt">
    <div class="glow" aria-hidden="true"></div>

    <div class="main">
      <GameTopBar
        :mode-label="modeLabel"
        :is-transfer="isPerevodnoy"
        :deck="deckSize"
        :stake="stake"
        :status="status"
        :sound-on="settings.soundOn"
        :chat-open="chatOpen"
        :unread="unread"
        :can-surrender="canAct"
        :waiting="!state && !watching"
        :copied="copied"
        :watchers="watchers"
        :watching="watching"
        @back="router.push('/lobby')"
        @sound="toggleSound"
        @chat="toggleChat"
        @surrender="confirmSurrender = true"
        @copy="copyLink"
      />

      <section class="arena" aria-label="Стол">
        <div class="oval">
          <span class="rim" aria-hidden="true"></span>

          <div class="seats">
            <TableSeat
              v-for="s in seats"
              :key="s.key"
              class="seat-pos"
              :data-fly-seat="s.player?.id"
              :user-id="s.player?.id"
              :style="{ '--x': `${s.x}%`, '--y': `${s.y}%` }"
              :name="s.player?.name ?? s.waitingName"
              :hand-count="s.player?.handCount ?? 0"
              :role="s.player ? roleOf(s.player.id) : null"
              :out="!!s.player && !s.player.isActive"
              :offline="!!s.player && offline.has(s.player.id)"
              :turn-fraction="s.player && actorId === s.player.id && turnLeft > 0 ? turnFraction : null"
              :bubble="s.player ? bubbles[s.player.id] ?? null : null"
            />
          </div>

          <div class="felt-content">
            <BattleArea
              v-if="state"
              :pairs="pairs"
              :trump-card="state.trumpCard"
              :trump-suit="state.trumpSuit"
              :cards-left="state.cardsLeft"
              :discard-size="state.discardSize"
              :targets="defendTargets"
              :hint="battleHint"
              @pair="onPairClick"
            />

            <div v-else-if="searching" class="waiting searching" role="status">
              <div class="radar" aria-hidden="true">
                <span class="pulse p1"></span>
                <span class="pulse p2"></span>
                <span class="pulse p3"></span>
                <GameCard face-down size="sm" class="radar-card" />
              </div>
              <h1>Ищем соперника</h1>
              <p v-if="waitingInfo">{{ modeLabel }} · {{ waitingInfo.deck }} карт · ставка {{ stake.toLocaleString('ru-RU') }}</p>
              <p v-else>Подбираем стол с вашими настройками</p>
              <span class="elapsed" aria-hidden="true">{{ elapsedText }}</span>
              <button type="button" class="felt-btn ghost" @click="cancelSearch">Отменить поиск</button>
            </div>

            <div v-else class="waiting">
              <div v-if="!showCall" class="waiting-cards" aria-hidden="true">
                <GameCard v-for="i in 3" :key="i" face-down size="sm" :class="`wc wc${i}`" />
              </div>
              <h1>{{ status === 'closed' ? 'Не удалось подключиться' : watching ? 'Партия ещё не началась' : 'Ждём соперников' }}</h1>
              <p v-if="status === 'closed'">Стол закрыт или сервер недоступен. Вернитесь в лобби и выберите другой стол.</p>
              <p v-else-if="waitingInfo">
                За столом {{ Math.max(waitingInfo.players, 1) }} из {{ waitingInfo.maxPlayers }}.
                {{ waitingInfo.private ? 'Стол закрытый: сядут только позванные и те, у кого есть ссылка.' : 'Партия начнётся сама, когда все сядут.' }}
              </p>
              <p v-else>Партия начнётся сама, когда все сядут.</p>
              <div v-if="showCall" class="call">
                <span class="call-label">Позвать друзей · свободно мест: {{ seatsLeft }}</span>
                <div v-if="callable.length" class="call-list">
                  <button
                    v-for="f in callable"
                    :key="f.id"
                    type="button"
                    class="call-friend"
                    :class="{ on: isCalled(f.id) }"
                    :disabled="isCalled(f.id)"
                    @click="callFriend(f)"
                  >
                    <span class="avatar" :style="{ '--tone': avatarTone(f.username) }"><AvatarGlyph :name="f.username" :user-id="f.id" /></span>
                    {{ f.username }}
                    <span class="call-state">{{ isCalled(f.id) ? 'позван' : 'позвать' }}</span>
                  </button>
                </div>
                  <span v-else class="call-none">Друзей в лобби сейчас нет. Позовите по ссылке сверху.</span>
              </div>
              <button v-if="status === 'closed'" type="button" class="felt-btn" @click="router.push('/lobby')">В лобби</button>
            </div>
          </div>
        </div>
      </section>

      <section class="dock" :class="{ 'my-turn': myTurn }">
        <div class="action-row">
          <TableSeat
            v-if="watching"
            class="me-seat"
            :data-fly-seat="anchorId"
            :user-id="anchorId"
            bubble-side="up"
            :name="anchor?.name ?? null"
            :hand-count="anchor?.handCount ?? 0"
            :role="anchor ? roleOf(anchor.id) : null"
            :out="!!anchor && !anchor.isActive"
            :offline="!!anchor && offline.has(anchor.id)"
            :turn-fraction="anchor && actorId === anchor.id && turnLeft > 0 ? turnFraction : null"
            :bubble="anchor ? bubbles[anchor.id] ?? null : null"
          />
          <TableSeat
            v-else
            class="me-seat"
            me
            :data-fly-seat="me"
            bubble-side="up"
            :name="myName"
            :role="state && canAct ? roleOf(me) : null"
            :turn-fraction="myTurn && turnLeft > 0 ? turnFraction : null"
            :bubble="bubbles[me] ?? null"
          />

          <div class="status">
            <span v-if="roleText" class="role">{{ roleText }}</span>
            <span v-if="result && !resultOpen" class="status-text result-line" :class="result.tone">{{ result.title }}</span>
            <span v-else class="status-text">{{ state ? statusText : watching ? 'Вы зритель — партия начнётся, когда все сядут' : 'Садитесь поудобнее — ждём остальных' }}</span>
            <div v-if="myTurn && turnLeft > 0" class="timer" :class="{ urgent: turnSeconds <= 5 }">
              <div class="timer-bar" :style="{ transform: `scaleX(${turnFraction})` }"></div>
              <span class="timer-text">{{ turnSeconds }} с</span>
            </div>
          </div>

          <div class="actions">
            <template v-if="result && !resultOpen">
              <button type="button" class="bar-btn" @click="resultOpen = true">Итоги</button>
              <button type="button" class="bar-btn primary" @click="router.push('/lobby')">В лобби</button>
            </template>
            <button v-if="showTransfer" type="button" class="act" @click="doTransfer">Перевести</button>
            <button v-if="showTake" type="button" class="act warn" @click="doTake">Беру</button>
            <button v-if="showBeaten" type="button" class="act" @click="doPass">Бито</button>
            <button v-if="showEnough" type="button" class="act" @click="doPass">Хватит</button>
          </div>
        </div>

        <p v-if="watching" class="watch-note">Вы смотрите партию: карты игроков на руках не видны, ходить и писать в чат нельзя.</p>
        <div v-else ref="handEl" class="hand" :style="handStyle" data-fly-zone="hand">
          <TransitionGroup name="hand-slot">
            <div
              v-for="(card, i) in hand"
              :key="cardKey(card)"
              class="hand-slot"
              :class="{ launching: launching === cardKey(card) }"
              :style="{ zIndex: i, margin: `0 -${handOverlap}px` }"
            >
              <div class="fan" :style="fanStyle(i, hand.length)">
                <GameCard
                  :card="card"
                  size="lg"
                  :data-fly="cardKey(card)"
                  :playable="isPlayable(card)"
                  :selected="!!selected && sameCard(selected, card)"
                  @click="onHandClick(card)"
                />
              </div>
            </div>
          </TransitionGroup>
          <p v-if="state && !hand.length && iAmActive" class="hand-empty">У вас нет карт</p>
        </div>
      </section>
    </div>

    <aside v-if="wide && chatOpen" class="chat-col">
      <ChatBox class="chat-box" title="Чат стола" :placeholder="chatPlaceholder" :disabled="watching" :messages="chat" :my-id="me" @send="say">
        <template v-if="!watching" #tools><QuickChat @send="say" /></template>
      </ChatBox>
    </aside>

    <AnimatePresence>
      <OverlayPanel v-if="!wide && chatOpen" key="chat" side="right" :width="380" title="Чат стола" @close="chatOpen = false">
        <ChatBox class="chat-box sheet" title="" :placeholder="chatPlaceholder" :disabled="watching" :messages="chat" :my-id="me" @send="say">
          <template v-if="!watching" #tools><QuickChat @send="say" /></template>
        </ChatBox>
      </OverlayPanel>
    </AnimatePresence>

    <FlightLayer :flights="flights" @land="onLand" @reveal="flightReveal" @done="flightDone" />

    <Transition name="fade">
      <div v-if="confirmSurrender" class="modal" @click.self="confirmSurrender = false">
        <div class="dialog" role="dialog" aria-modal="true" aria-labelledby="surrender-title">
          <h2 id="surrender-title">Сдаться?</h2>
          <p>Вы выйдете из партии и будете считаться проигравшим.</p>
          <div class="dialog-actions">
            <button type="button" class="btn ghost" @click="confirmSurrender = false">Играть дальше</button>
            <button type="button" class="btn danger" @click="doSurrender">Сдаться</button>
          </div>
        </div>
      </div>
    </Transition>

    <Transition name="fade">
      <div v-if="result && resultOpen" class="modal" @click.self="resultOpen = false">
        <div class="dialog result" :class="result.tone" role="dialog" aria-modal="true" aria-labelledby="result-title">
          <button type="button" class="dialog-close" aria-label="Свернуть итоги" title="Свернуть" @click="resultOpen = false">
            <PhX :size="18" />
          </button>
          <div class="result-cards" aria-hidden="true">
            <GameCard :card="{ suit: 3, rank: 14 }" size="sm" class="rc-a" />
            <GameCard :card="{ suit: 2, rank: 12 }" size="sm" class="rc-b" />
          </div>
          <h2 id="result-title">{{ result.title }}</h2>
          <p>{{ result.text }}</p>
          <p v-if="payoutLine" class="payout-line" :class="{ plus: (myPayout ?? 0) > 0, minus: (myPayout ?? 0) < 0 }">{{ payoutLine }}</p>
          <ol v-if="standings.length" class="standings" aria-label="Итоги партии">
            <li v-for="(s, i) in standings" :key="s.id" class="standing" :class="[s.kind, { me: s.me }]" :style="{ '--i': i }">
              <span class="medal" :class="s.kind === 'place' && s.place <= 3 ? `m${s.place}` : ''">
                {{ s.kind === 'fool' ? '🃏' : s.kind === 'surrendered' ? '⚑' : s.place }}
              </span>
              <span class="standing-name">{{ s.name }}<small v-if="s.me"> · вы</small></span>
              <span class="standing-label">{{ STANDING_LABEL[s.kind](s) }}</span>
              <span v-if="s.payout !== null" class="standing-payout" :class="{ plus: s.payout > 0, minus: s.payout < 0 }">{{ chips(s.payout) }}</span>
            </li>
          </ol>
          <PostGameChat v-if="!watching" :messages="chat" :my-id="me" @send="say" />
          <p v-if="rematchCancelled" class="rematch-note">Реванша не будет: {{ rematchCancelled }}</p>
          <p v-else-if="rematchOpen && rivalWantsRematch && !iVotedRematch" class="rematch-note hot">Соперник хочет реванш</p>
          <div class="dialog-actions center">
            <template v-if="rematchOpen">
              <button v-if="rivalWantsRematch && !iVotedRematch" type="button" class="btn ghost" @click="declineRematch">Нет, спасибо</button>
              <button v-else type="button" class="btn ghost" @click="router.push('/lobby')">В лобби</button>
              <button type="button" class="btn primary" :disabled="iVotedRematch" @click="voteRematch">
                {{ iVotedRematch ? `Ждём соперника… ${rematch?.votes.length ?? 1} из ${rematch?.needed ?? 2}` : 'Реванш' }}
              </button>
            </template>
            <template v-else>
              <button type="button" class="btn ghost" @click="resultOpen = false">Остаться за столом</button>
              <button type="button" class="btn primary" @click="router.push('/lobby')">В лобби</button>
            </template>
          </div>
        </div>
      </div>
    </Transition>

    <ToastStack />
  </div>
</template>

<style scoped>
.game {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  height: 100dvh;
  overflow: hidden;
  background: var(--color-paper);
  color: var(--color-ink);
}

.game.chat-docked {
  grid-template-columns: minmax(0, 1fr) 360px;
}

.glow {
  position: absolute;
  top: -35vh;
  left: 15%;
  width: 70vw;
  height: 90vh;
  background: radial-gradient(ellipse at 50% 40%, var(--color-glow), transparent 65%);
  pointer-events: none;
}

.main {
  position: relative;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}

.arena {
  flex: 1;
  min-height: 0;
  display: grid;
  place-items: center;
  padding: 76px 96px 30px;
  container-type: size;
}

.oval {
  position: relative;
  width: min(100cqw, 210cqh);
  aspect-ratio: 2.1;
  border-radius: 50%;
  background:
    radial-gradient(ellipse 70% 60% at 50% 35%, rgba(255, 255, 255, 0.12), transparent 70%),
    radial-gradient(ellipse at 50% 50%, var(--color-felt), var(--color-felt-deep));
  box-shadow:
    0 0 0 10px var(--color-paper-raised),
    0 0 0 11px var(--color-line-strong),
    inset 0 0 60px color-mix(in srgb, var(--color-felt-deep) 70%, transparent),
    0 50px 80px -40px rgba(0, 0, 0, 0.55);
}

.oval::before {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: inherit;
  background-image: radial-gradient(rgba(255, 255, 255, 0.05) 1px, transparent 1px);
  background-size: 4px 4px;
  pointer-events: none;
}

.rim {
  position: absolute;
  inset: 22px;
  border-radius: 50%;
  border: 1.5px dashed var(--color-felt-line);
  pointer-events: none;
}

.seats {
  position: absolute;
  inset: 0;
  z-index: 3;
  pointer-events: none;
}

.seat-pos {
  position: absolute;
  left: var(--x);
  top: var(--y);
  translate: -50% -62%;
  pointer-events: auto;
}

.felt-content {
  position: absolute;
  inset: 18% 9% 12%;
  z-index: 2;
  display: grid;
  grid-template: minmax(0, 1fr) / minmax(0, 1fr);
  place-items: center;
}

.waiting {
  display: flex;
  flex-direction: column;
  align-items: center;
  max-width: 420px;
  text-align: center;
  color: var(--color-on-felt);
}

.waiting-cards {
  position: relative;
  width: 120px;
  height: 90px;
  margin-bottom: 10px;
}

.wc {
  position: absolute;
  left: 50%;
  top: 0;
  margin-left: -29px;
  animation: shuffle 2.4s ease-in-out infinite;
}

.wc1 {
  rotate: -14deg;
  translate: -24px 6px;
}

.wc2 {
  animation-delay: 0.3s;
}

.wc3 {
  rotate: 14deg;
  translate: 24px 6px;
  animation-delay: 0.6s;
}

@keyframes shuffle {
  50% {
    transform: translateY(-8px);
  }
}

.waiting h1 {
  margin: 0;
  font-family: var(--font-card);
  font-size: clamp(28px, 3.4vw, 42px);
  font-style: italic;
  font-weight: 400;
}

.watch-note {
  margin: 0;
  padding: 18px 20px 22px;
  font-size: 14px;
  text-align: center;
  color: var(--color-ink-muted);
}

.waiting p {
  margin: 8px 0 0;
  font-size: 15px;
  line-height: 1.45;
  color: var(--color-on-felt-muted);
}

.call {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
}

.call-label,
.call-none {
  font-size: 13px;
  color: var(--color-on-felt-muted);
}

.call-list {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}

.call-friend {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 14px 0 5px;
  border: 0;
  border-radius: 999px;
  background: var(--color-felt-well);
  box-shadow: inset 0 0 0 1px var(--color-felt-line);
  color: var(--color-on-felt);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.call-friend:hover:not(:disabled) {
  box-shadow: inset 0 0 0 1.5px var(--color-gold);
}

.call-friend:disabled {
  cursor: default;
}

.call-friend .avatar {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-gold) var(--tone, 52%), #000);
  color: #fff;
  font-size: 13px;
  font-weight: 500;
}

.call-state {
  font-size: 12px;
  color: var(--color-gold);
}

.call-friend.on .call-state {
  color: var(--color-on-felt-faint);
}

.radar {
  position: relative;
  display: grid;
  place-items: center;
  width: 150px;
  height: 150px;
  margin-bottom: 8px;
}

.pulse {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  border: 2px solid var(--color-on-felt-faint);
  opacity: 0;
  animation: radar 2.4s cubic-bezier(0.2, 0.8, 0.2, 1) infinite;
}

.p2 {
  animation-delay: 0.8s;
}

.p3 {
  animation-delay: 1.6s;
}

@keyframes radar {
  0% {
    opacity: 0.9;
    transform: scale(0.3);
  }
  100% {
    opacity: 0;
    transform: scale(1);
  }
}

.radar-card {
  position: relative;
  rotate: -6deg;
  animation: shuffle 2.4s ease-in-out infinite;
}

.elapsed {
  margin-top: 10px;
  font-size: 24px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.02em;
}

.felt-btn.ghost {
  background: var(--color-felt-well);
  box-shadow: inset 0 0 0 1px var(--color-felt-line);
  color: var(--color-on-felt);
}

.felt-btn.ghost:hover {
  box-shadow: inset 0 0 0 1.5px var(--color-on-felt-faint);
}

@media (prefers-reduced-motion: reduce) {
  .pulse,
  .radar-card {
    animation: none;
  }

  .pulse.p1 {
    opacity: 0.5;
    transform: scale(0.7);
  }
}

.felt-btn {
  height: 44px;
  margin-top: 16px;
  padding: 0 22px;
  border: 0;
  border-radius: 999px;
  background: var(--color-on-felt);
  color: var(--color-on-felt-invert);
  font: inherit;
  cursor: pointer;
}

.dock {
  position: relative;
  z-index: 4;
  padding: 0 24px 10px;
}

.action-row {
  display: grid;
  grid-template-columns: 140px minmax(0, 1fr) auto;
  align-items: center;
  gap: 18px;
  max-width: 1100px;
  margin: 0 auto;
}

.me-seat {
  justify-self: center;
}

.status {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.role {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-gold-ink);
}

.status-text {
  font-size: 18px;
  line-height: 1.3;
}

.timer {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: 360px;
  margin-top: 4px;
}

.timer::before {
  content: '';
  position: absolute;
  left: 0;
  right: 48px;
  height: 6px;
  border-radius: 3px;
  background: var(--color-paper-sunken);
}

.timer-bar {
  position: relative;
  flex: 1;
  height: 6px;
  margin-right: 48px;
  border-radius: 3px;
  background: var(--color-gold);
  transform-origin: left;
  transition: transform 0.25s linear;
}

.timer-text {
  position: absolute;
  right: 0;
  font-size: 13px;
  color: var(--color-ink-faint);
  font-variant-numeric: tabular-nums;
}

.timer.urgent .timer-bar {
  background: var(--color-red);
}

.timer.urgent .timer-text {
  color: var(--color-red);
}

.actions {
  display: flex;
  gap: 8px;
}

.act {
  height: 52px;
  padding: 0 28px;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font: inherit;
  font-size: 17px;
  font-weight: 500;
  cursor: pointer;
  box-shadow: 0 14px 26px -16px rgba(0, 0, 0, 0.7);
  animation: pop 0.3s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.act:hover {
  background: var(--color-ink);
}

.act.warn {
  background: var(--color-red);
  color: #fff;
}

.act.warn:hover {
  background: var(--color-red-strong);
}

@keyframes pop {
  from {
    transform: scale(0.8);
    opacity: 0;
  }
}

.hand {
  display: flex;
  justify-content: center;
  align-items: flex-end;
  min-height: 184px;
  padding: 30px 12px 14px;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
}

.hand::-webkit-scrollbar {
  display: none;
}

.fan {
  transform-origin: 50% 160%;
  transition: transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.hand-slot:hover {
  z-index: 50 !important;
}

.hand-slot-move {
  transition: transform 0.42s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.hand-slot.launching :deep(.game-card) {
  transform: translateY(-30px) scale(1.05);
  transition-duration: 0.16s;
}

.hand-slot:hover :deep(.game-card:not(.selected)) {
  transform: translateY(-16px);
}

.hand-slot:hover :deep(.game-card:not(.selected) .face--front) {
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.9) inset,
    0 0 0 1px rgba(160, 150, 132, 0.35),
    0 22px 30px -18px rgba(40, 32, 20, 0.5);
}

.hand-empty {
  margin: auto;
  color: var(--color-ink-faint);
}

.chat-col {
  position: relative;
  min-height: 0;
  padding: 16px;
  border-left: 1px solid var(--color-line);
  background: color-mix(in srgb, var(--color-paper-sunken) 60%, transparent);
}

.chat-box {
  height: 100%;
  padding: 14px 12px;
  box-sizing: border-box;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.chat-box.sheet {
  padding: 0;
  border-radius: 0;
  background: none;
  box-shadow: none;
}

.modal {
  position: fixed;
  inset: 0;
  z-index: 200;
  display: grid;
  place-items: center;
  padding: 16px;
  background: var(--color-scrim);
  backdrop-filter: blur(3px);
}

.dialog {
  position: relative;
  width: min(440px, 100%);
  max-height: calc(100dvh - 32px);
  overflow-y: auto;
  padding: 28px;
  box-sizing: border-box;
  border-radius: 28px;
  background: var(--color-paper-raised);
  box-shadow: var(--shadow-panel);
}

.dialog h2 {
  margin: 0;
  font-family: var(--font-card);
  font-size: 34px;
  font-style: italic;
  font-weight: 400;
}

.dialog p {
  margin: 8px 0 0;
  font-size: 15px;
  line-height: 1.5;
  color: var(--color-ink-subtle);
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 22px;
}

.dialog-actions.center {
  justify-content: center;
}

.rematch-note {
  margin: 18px 0 0;
  color: var(--color-ink-faint);
  font-size: 14px;
  text-align: center;
}

.rematch-note.hot {
  color: var(--color-gold-ink);
  font-weight: 500;
}

.btn {
  height: 46px;
  padding: 0 22px;
  border: 0;
  border-radius: 999px;
  font: inherit;
  font-size: 15px;
  white-space: nowrap;
  cursor: pointer;
}

@media (max-width: 560px) {
  .dialog-actions {
    flex-wrap: wrap;
  }

  .dialog-actions .btn {
    flex: 1 1 auto;
    padding: 0 14px;
    font-size: 14px;
  }
}

.btn.ghost {
  background: var(--color-paper-hover);
  color: var(--color-ink);
}

.btn.primary {
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

.btn.danger {
  background: var(--color-red);
  color: #fff;
}

.result {
  text-align: center;
}

.dialog-close {
  position: absolute;
  top: 14px;
  right: 14px;
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  cursor: pointer;
}

.dialog-close:hover {
  color: var(--color-ink);
}

.bar-btn {
  height: 46px;
  padding: 0 22px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-hover);
  color: var(--color-ink);
  font: inherit;
  font-size: 15px;
  cursor: pointer;
}

.bar-btn.primary {
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

.result-line {
  font-family: var(--font-card);
  font-size: 26px;
  font-style: italic;
}

.result-line.win {
  color: var(--color-gold-ink);
}

.result-line.lose {
  color: var(--color-red);
}

.result-cards {
  position: relative;
  height: 90px;
  margin-bottom: 6px;
}

.rc-a,
.rc-b {
  position: absolute;
  left: 50%;
  top: 0;
}

.rc-a {
  translate: -80% 0;
  rotate: -12deg;
}

.rc-b {
  translate: -20% 6px;
  rotate: 10deg;
}

.standings {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 18px 0 0;
  padding: 6px;
  border-radius: 18px;
  background: var(--color-paper-sunken);
  list-style: none;
  text-align: left;
}

.standing {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 12px;
  animation: standing-in 0.4s cubic-bezier(0.2, 0.8, 0.2, 1) both;
  animation-delay: calc(0.25s + var(--i) * 0.08s);
}

.standing.me {
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
}

@keyframes standing-in {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
}

.medal {
  display: grid;
  place-items: center;
  flex: none;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.medal.m1 {
  background: var(--color-gold);
  color: #fff;
}

.medal.m2 {
  background: #b9bec6;
  color: #fff;
}

.medal.m3 {
  background: #c79a6b;
  color: #fff;
}

.fool .medal {
  background: var(--color-red-tint);
  font-size: 15px;
}

.surrendered .medal {
  color: var(--color-ink-faint);
}

.standing-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
}

.standing-name small {
  font-size: 13px;
  color: var(--color-ink-faint);
}

.standing-label {
  font-size: 13px;
  color: var(--color-ink-faint);
  white-space: nowrap;
}

.fool .standing-label {
  color: var(--color-red);
}

.standing-payout {
  min-width: 56px;
  text-align: right;
  font-size: 14px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--color-ink-muted);
}

.standing-payout.plus,
.payout-line.plus {
  color: var(--color-green);
}

.standing-payout.minus,
.payout-line.minus {
  color: var(--color-red);
}

.payout-line {
  margin-top: 4px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.surrendered .standing-name,
.surrendered .standing-label {
  color: var(--color-ink-ghost);
}

@media (prefers-reduced-motion: reduce) {
  .standing {
    animation: none;
  }
}

.result.win h2 {
  color: var(--color-gold-ink);
}

.result.lose h2 {
  color: var(--color-red);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

@media (max-width: 1100px) {
  .arena {
    padding: 70px 70px 24px;
  }

  .hand-slot :deep(.game-card) {
    --w: 88px;
  }
}

@media (max-width: 767px) {
  .arena {
    padding: 8px 10px;
    place-items: stretch;
    container-type: normal;
  }

  .oval {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    aspect-ratio: auto;
    border-radius: 36px;
    box-shadow:
      0 0 0 6px var(--color-paper-raised),
      0 0 0 7px var(--color-line-strong),
      inset 0 0 40px color-mix(in srgb, var(--color-felt-deep) 70%, transparent);
  }

  .rim {
    inset: 10px;
    border-radius: 28px;
  }

  .seats {
    position: relative;
    display: flex;
    justify-content: space-around;
    padding: 12px 6px 0;
  }

  .seat-pos {
    position: relative;
    left: auto;
    top: auto;
    translate: none;
    width: auto;
    max-width: 96px;
  }

  .seat-pos :deep(.backs) {
    display: none;
  }

  .felt-content {
    position: relative;
    inset: auto;
    flex: 1;
    padding: 8px 12px 14px;
  }

  .dock {
    padding: 0 10px 8px;
  }

  .action-row {
    grid-template-columns: minmax(0, 1fr);
    gap: 10px;
  }

  .me-seat {
    display: none;
  }

  .actions {
    grid-column: 1 / -1;
    justify-content: center;
  }

  .actions:empty {
    display: none;
  }

  .act {
    height: 46px;
    padding: 0 22px;
    font-size: 16px;
  }

  .status-text {
    font-size: 15px;
  }

  .hand {
    min-height: 120px;
    padding-top: 22px;
  }

  .hand-slot :deep(.game-card) {
    --w: 66px;
  }
}

@media (max-height: 820px) and (min-width: 768px) {
  .arena {
    padding-top: 62px;
    padding-bottom: 14px;
  }

  .me-seat {
    zoom: 0.8;
  }

  .hand {
    min-height: 146px;
    padding-top: 22px;
  }

  .hand-slot :deep(.game-card) {
    --w: 84px;
  }
}
</style>
