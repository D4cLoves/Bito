import { onUnmounted, ref, shallowRef } from 'vue'
import { freshAccessToken } from '../api/http'
import type { Card, ChatMessage, GameOver, GameState } from '../types/game'

export type SocketStatus = 'connecting' | 'open' | 'reconnecting' | 'closed'

const MAX_RETRIES = 8

interface ServerMessage {
  type: string
  payload?: unknown
}

export function useGameSocket(roomId: string) {
  const status = ref<SocketStatus>('connecting')
  const state = shallowRef<GameState | null>(null)
  const gameOver = shallowRef<GameOver | null>(null)
  const offline = ref<Set<string>>(new Set())
  const chat = ref<ChatMessage[]>([])
  const lastError = ref<{ text: string; at: number } | null>(null)
  const stateReceivedAt = ref(0)

  let ws: WebSocket | null = null
  let retries = 0
  let retryTimer: number | null = null
  let stopped = false

  async function connect() {
    const token = await freshAccessToken()
    if (stopped) return
    if (!token) {
      status.value = 'closed'
      lastError.value = { text: 'Сессия истекла. Войдите заново.', at: Date.now() }
      return
    }

    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const url = `${proto}://${location.host}/ws?room_id=${encodeURIComponent(roomId)}&token=${encodeURIComponent(token)}`
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
        state.value = msg.payload as GameState
        stateReceivedAt.value = Date.now()
        break
      case 'game_over':
        gameOver.value = msg.payload as GameOver
        break
      case 'error': {
        const text = (msg.payload as { message?: string })?.message ?? 'Ошибка'
        lastError.value = { text, at: Date.now() }
        break
      }
      case 'player_disconnected': {
        const id = (msg.payload as { playerId: string }).playerId
        offline.value = new Set(offline.value).add(id)
        break
      }
      case 'player_reconnected': {
        const id = (msg.payload as { playerId: string }).playerId
        const next = new Set(offline.value)
        next.delete(id)
        offline.value = next
        break
      }
      case 'chat':
        chat.value = [...chat.value.slice(-49), msg.payload as ChatMessage]
        break
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
