import { MIN_STAKE } from '../constants/economy'
import type { DeckSize, GameMode, TableInfo } from '../types/game'

export type TableSort = 'full' | 'new' | 'cheap' | 'rich'

export interface TablePreset {
  mode: GameMode
  deck: DeckSize
  players: number
  stake: number
}

export const TABLE_PRESETS: TablePreset[] = [
  { mode: 'podkidnoy', deck: 36, players: 3, stake: 100 },
  { mode: 'perevodnoy', deck: 36, players: 4, stake: 250 },
  { mode: 'podkidnoy', deck: 36, players: 6, stake: 500 },
  { mode: 'perevodnoy', deck: 52, players: 5, stake: 1000 },
]

export const stakeOf = (t: TableInfo) => t.stake ?? MIN_STAKE
export const canAfford = (t: TableInfo, balance: number | null) => balance === null || stakeOf(t) <= balance
const fullness = (t: TableInfo) => t.players / t.maxPlayers
const created = (t: TableInfo) => new Date(t.createdAt).getTime()

const ORDER: Record<TableSort, (a: TableInfo, b: TableInfo) => number> = {
  full: (a, b) => fullness(b) - fullness(a) || created(a) - created(b),
  new: (a, b) => created(b) - created(a),
  cheap: (a, b) => stakeOf(a) - stakeOf(b) || fullness(b) - fullness(a),
  rich: (a, b) => stakeOf(b) - stakeOf(a) || fullness(b) - fullness(a),
}

export function sortTables(list: TableInfo[], sort: TableSort, balance: number | null): TableInfo[] {
  return [...list].sort((a, b) => Number(canAfford(b, balance)) - Number(canAfford(a, balance)) || ORDER[sort](a, b))
}
