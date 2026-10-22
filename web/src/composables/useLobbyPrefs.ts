import { ref, watch } from 'vue'
import { MIN_STAKE, STAKES } from '../constants/economy'
import type { AnyDeck, AnyMode, DeckSize, GameMode } from '../types/game'

const QUICK_KEY = 'bito_quick_play'
const CREATE_KEY = 'bito_create_table'
const HAND = 6

function read(key: string): Record<string, unknown> {
  try {
    const value = JSON.parse(localStorage.getItem(key) ?? '{}')
    return value && typeof value === 'object' ? value : {}
  } catch {
    return {}
  }
}

function write(key: string, value: unknown) {
  try {
    localStorage.setItem(key, JSON.stringify(value))
  } catch {
    return
  }
}

const isStake = (v: unknown): v is number => typeof v === 'number' && (STAKES as readonly number[]).includes(v)
const isMode = (v: unknown): v is GameMode => v === 'podkidnoy' || v === 'perevodnoy'
const isDeck = (v: unknown): v is DeckSize => v === 24 || v === 36 || v === 52

export const fitsDeck = (players: number, deck: number) => players * HAND <= deck

export function useQuickPrefs() {
  const saved = read(QUICK_KEY)
  const stake = ref(isStake(saved.stake) ? saved.stake : MIN_STAKE)
  const mode = ref<AnyMode>(saved.mode === 'any' || isMode(saved.mode) ? (saved.mode as AnyMode) : 'any')
  const deck = ref<AnyDeck>(saved.deck === 0 || isDeck(saved.deck) ? (saved.deck as AnyDeck) : 0)

  watch([stake, mode, deck], () => write(QUICK_KEY, { stake: stake.value, mode: mode.value, deck: deck.value }))

  return { stake, mode, deck }
}

export function useCreatePrefs() {
  const saved = read(CREATE_KEY)
  const stake = ref(isStake(saved.stake) ? saved.stake : MIN_STAKE)
  const mode = ref<GameMode>(isMode(saved.mode) ? saved.mode : 'podkidnoy')
  const deck = ref<DeckSize>(isDeck(saved.deck) ? saved.deck : 36)
  const raw = typeof saved.players === 'number' ? saved.players : 3
  const players = ref(raw >= 3 && raw <= 6 && fitsDeck(raw, deck.value) ? raw : 3)

  watch([stake, mode, deck, players], () =>
    write(CREATE_KEY, { stake: stake.value, mode: mode.value, deck: deck.value, players: players.value }),
  )

  return { stake, mode, deck, players }
}

export function bestAffordable(balance: number | null, wanted: number): number {
  if (balance === null || wanted <= balance) return wanted
  const fit = [...STAKES].reverse().find((s) => s <= balance)
  return fit ?? MIN_STAKE
}
