export type SuitType = 'spades' | 'clubs' | 'diamonds' | 'hearts'

export type CardRank = 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14

export interface Card {
  id?: string
  suit: SuitType | number
  rank: CardRank | number
}

export interface CardPair {
  attack: Card
  defend?: Card
}

export function normalizeSuit(suit: SuitType | number): SuitType {
  if (typeof suit === 'string') {
    return suit.toLowerCase() as SuitType
  }
  switch (suit) {
    case 0:
      return 'spades'
    case 1:
      return 'clubs'
    case 2:
      return 'hearts'
    case 3:
      return 'diamonds'
    default:
      return 'spades'
  }
}

export function getRankLabel(rank: CardRank | number): string {
  switch (rank) {
    case 11:
      return 'J'
    case 12:
      return 'Q'
    case 13:
      return 'K'
    case 14:
      return 'A'
    default:
      return String(rank)
  }
}

export function getSuitSymbol(suit: SuitType | number): string {
  const s = normalizeSuit(suit)
  switch (s) {
    case 'spades':
      return '♠'
    case 'clubs':
      return '♣'
    case 'hearts':
      return '♥'
    case 'diamonds':
      return '♦'
  }
}

export function isRedSuit(suit: SuitType | number): boolean {
  const s = normalizeSuit(suit)
  return s === 'hearts' || s === 'diamonds'
}

export type GameMode = 'podkidnoy' | 'perevodnoy'
export type DeckSize = 24 | 36 | 52

export interface TableInfo {
  id: string
  mode: GameMode
  deck: DeckSize
  maxPlayers: number
  players: number
  createdBy: string
  createdAt: string
}

export interface CreateTableRequest {
  mode: GameMode
  deck: DeckSize
  maxPlayers: number
}

export type GamePhase = 'dealing' | 'battle' | 'giving_more' | 'finished'

export interface PlayerView {
  id: string
  name: string
  handCount: number
  isActive: boolean
}

export interface GameState {
  phase: GamePhase
  trumpCard: Card
  trumpSuit: number
  cardsLeft: number
  table: CardPair[] | null
  discardSize: number
  attackerId: string
  defenderId: string
  myHand: Card[] | null
  players: PlayerView[]
  mode: GameMode
  turnLeftMs: number
  youPassed: boolean
}

export interface GameOver {
  winnerId?: string
  loserId?: string
  isDraw: boolean
}

export interface ChatMessage {
  senderId: string
  senderName: string
  message: string
}

export function sameCard(a: Card, b: Card): boolean {
  return a.suit === b.suit && a.rank === b.rank
}

export function cardKey(c: Card): string {
  return `${c.suit}-${c.rank}`
}

export function canBeat(attack: Card, defend: Card, trumpSuit: number): boolean {
  if (defend.suit === trumpSuit && attack.suit !== trumpSuit) return true
  return defend.suit === attack.suit && defend.rank > attack.rank
}
