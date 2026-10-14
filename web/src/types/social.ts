import type { DeckSize, GameMode } from './game'

export type FriendStatus = 'online' | 'playing' | 'offline'

export interface Friend {
  id: string
  username: string
  status: FriendStatus
  lastSeen?: string
}

export interface FriendRequest {
  id: string
  username: string
}

export interface ChatEntry {
  id: string
  userId: string
  username: string
  text: string
  at: string
  system?: boolean
}

export type GameOutcome = 'win' | 'lose' | 'draw'

export interface GameHistoryEntry {
  id: string
  finishedAt: string
  mode: GameMode
  deck: DeckSize
  players: number
  stake: number
  place: number
  payout: number
  outcome?: GameOutcome
}

export interface ActiveTable {
  id: string
  mode: GameMode
  stake: number
  players: number
}
