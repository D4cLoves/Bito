import type { Friend, FriendRequest, GameHistoryEntry } from '../types/social'

const ago = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString()

export const MOCK_FRIENDS: Friend[] = [
  { id: 'f1', username: 'Миша', status: 'playing' },
  { id: 'f2', username: 'Оля_К', status: 'online' },
  { id: 'f3', username: 'Тузовый', status: 'online' },
  { id: 'f4', username: 'Петрович', status: 'offline', lastSeen: ago(45) },
  { id: 'f5', username: 'Лиса', status: 'offline', lastSeen: ago(60 * 26) },
]

export const MOCK_REQUESTS: FriendRequest[] = [{ id: 'r1', username: 'Козырь77' }]

export const MOCK_HISTORY: GameHistoryEntry[] = [
  { id: 'g1', finishedAt: ago(35), mode: 'perevodnoy', deck: 36, players: 4, stake: 500, place: 1, payout: 238 },
  { id: 'g2', finishedAt: ago(80), mode: 'podkidnoy', deck: 36, players: 2, stake: 250, place: 2, payout: -250 },
  { id: 'g3', finishedAt: ago(60 * 5), mode: 'podkidnoy', deck: 24, players: 3, stake: 100, place: 2, payout: 31 },
  { id: 'g4', finishedAt: ago(60 * 27), mode: 'perevodnoy', deck: 52, players: 5, stake: 1000, place: 1, payout: 380 },
  { id: 'g5', finishedAt: ago(60 * 50), mode: 'podkidnoy', deck: 36, players: 2, stake: 100, place: 1, payout: 95 },
]

export const MOCK_OWNED: string[] = ['felt:wine', 'back:grove']
