import { onUnmounted, ref, shallowRef } from 'vue'
import { freshAccessToken } from '../api/http'
import type { LobbyState, TableInfo } from '../types/game'
import type { SocketStatus } from './useGameSocket'

const MAX_DELAY_MS = 10000

export function useLobbySocket() {
  const status = ref<SocketStatus>('connecting')
  const tables = shallowRef<TableInfo[] | null>(null)
  const online = ref<number | null>(null)
  const friendsVersion = ref(0)

  let ws: WebSocket | null = null
  let retries = 0
  let retryTimer: number | null = null
  let stopped = false

  async function connect() {
    const hasSession = await freshAccessToken()
    if (stopped) return
    if (!hasSession) {
      status.value = 'closed'
      return
    }

    const proto = location.protocol === 'https:' ? 'wss' : 'ws'
    const socket = new WebSocket(`${proto}://${location.host}/ws/lobby`)
    ws = socket

    socket.onopen = () => {
      retries = 0
      status.value = 'open'
    }

    socket.onmessage = (ev) => {
      let msg: { type: string; payload?: LobbyState }
      try {
        msg = JSON.parse(ev.data)
      } catch {
        return
      }
      if (msg.type === 'friends_changed') {
        friendsVersion.value++
        return
      }
      if (msg.type !== 'lobby_state' || !msg.payload) return
      tables.value = msg.payload.tables
      online.value = msg.payload.online
    }

    socket.onclose = () => {
      if (ws !== socket) return
      ws = null
      if (stopped) {
        status.value = 'closed'
        return
      }
      status.value = 'reconnecting'
      const delay = Math.min(500 * 2 ** retries, MAX_DELAY_MS)
      retries++
      retryTimer = window.setTimeout(connect, delay)
    }
  }

  function disconnect() {
    stopped = true
    if (retryTimer) clearTimeout(retryTimer)
    ws?.close()
    ws = null
  }

  connect()
  onUnmounted(disconnect)

  return { status, tables, online, friendsVersion, disconnect }
}
