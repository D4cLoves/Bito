import { onUnmounted, ref, shallowRef } from 'vue'
import { freshAccessToken } from '../api/http'
import type { Card, ChatMessage, GameOver, GameState } from '../types/game'
import type { ChatEntry } from '../types/social'

export type SocketStatus = 'connecting' | 'open' | 'reconnecting' | 'closed'

const MAX_RETRIES = 8
const CHAT_HISTORY = 100
const FATAL_ERRORS = new Set([
  'не хватает фишек на ставку',
  'вы уже играете за другим столом',
  'не удалось начать партию, попробуйте ещё раз',
  'касса недоступна, попробуйте позже',
  'партия уже идет, стол заполнен',
  'стол уже заполнен',
  'партия уже завершена',
])

interface ServerMessage {
  type: string
  payload?: unknown
}

export function useGameSocket(roomId: string) {
  const status = ref<SocketStatus>('connecting')
  const state = shallowRef<GameState | null>(null)
  const gameOver = shallowRef<GameOver | null>(null)
  const offline = ref<Set<string>>(new Set())
  const chat = ref<ChatEntry[]>([])
  const lastError = ref<{ text: string; at: number } | null>(null)
  const stateReceivedAt = ref(0)

  let ws: WebSocket | null = null
  let retries = 0
  let retryTimer: number | null = null
  let stopped = false
  let chatSeq = 0

  function pushChat(entry: Omit<ChatEntry, 'id' | 'at'>) {
    chat.value = [...chat.value.slice(-(CHAT_HISTORY - 1)), { ...entry, id: `c${++chatSeq}`, at: new Date().toISOString() }]
  }

  function system(text: string) {
    pushChat({ userId: 'system', username: '', text, system: true })
  }

  function nameOf(id: string) {
    return state.value?.players.find((p) => p.id === id)?.name ?? 'Соперник'
  }

  async function connect() {
    const hasSession = await freshAccessToken()
    if (stopped) return
    if (!hasSession) {
      status.value = 'closed'
      lastError.value = { text: 'Сессия истекла. Войдите заново.', at: Date.now() }
      return
    }

    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${proto}://${location.host}/ws?room_id=${encodeURIComponent(roomId)}`
    const socket = new WebSocket(url)
    ws = socket

    socket.onopen = () => {
      retries = 0
      status.value = 'open'
    }

    socket.onmessage = (ev) => {
      let msg: ServerMessage
      try {
        msg = JSON.parse(ev.data)
      } catch {
        return
      }
      handle(msg)
    }

    socket.onclose = () => {
      if (ws !== socket) return
      ws = null
      if (stopped || gameOver.value) {
        status.value = 'closed'
        return
      }
      if (retries >= MAX_RETRIES) {
        status.value = 'closed'
        return
      }
      status.value = 'reconnecting'
      const delay = Math.min(500 * 2 ** retries, 4000)
      retries++
      retryTimer = window.setTimeout(connect, delay)
    }
  }

  function handle(msg: ServerMessage) {
    switch (msg.type) {
      case 'game_state':
        if (!state.value) system('Партия началась. Удачи!')
        state.value = msg.payload as GameState
        stateReceivedAt.value = Date.now()
        break
      case 'game_over':
        if (!gameOver.value) system('Партия окончена')
        gameOver.value = msg.payload as GameOver
        break
      case 'error': {
        const text = (msg.payload as { message?: string })?.message ?? 'Ошибка'
        lastError.value = { text, at: Date.now() }
        if (FATAL_ERRORS.has(text)) {
          stopped = true
          status.value = 'closed'
        }
        break
      }
      case 'player_disconnected': {
        const id = (msg.payload as { playerId: string }).playerId
        offline.value = new Set(offline.value).add(id)
        system(`${nameOf(id)} потерял связь`)
        break
      }
      case 'player_reconnected': {
        const id = (msg.payload as { playerId: string }).playerId
        const next = new Set(offline.value)
        next.delete(id)
        offline.value = next
        system(`${nameOf(id)} вернулся`)
        break
      }
      case 'invite_declined': {
        const name = (msg.payload as { username?: string })?.username ?? 'Друг'
        system(`${name} не принял приглашение`)
        break
      }
      case 'chat': {
        const m = msg.payload as ChatMessage
        pushChat({ userId: m.senderId, username: m.senderName, text: m.message })
        break
      }
    }
  }

  function send(type: string, payload?: unknown) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      lastError.value = { text: 'Нет соединения с сервером', at: Date.now() }
      return
    }
    ws.send(JSON.stringify(payload === undefined ? { type } : { type, payload }))
  }

  const actions = {
    attack: (card: Card) => send('attack', { attackCard: card }),
    defend: (attackCard: Card, defendCard: Card) => send('defend', { attackCard, defendCard }),
    transfer: (card: Card) => send('transfer', { card }),
    take: () => send('take'),
    pass: () => send('pass'),
    surrender: () => send('surrender'),
    say: (message: string) => send('chat', { message }),
  }

  function disconnect() {
    stopped = true
    if (retryTimer) clearTimeout(retryTimer)
    ws?.close()
    ws = null
  }

  connect()
  onUnmounted(disconnect)

  return { status, state, gameOver, offline, chat, lastError, stateReceivedAt, ...actions, disconnect }
}
