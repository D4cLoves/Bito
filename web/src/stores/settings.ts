import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import {
  CARD_BACKS,
  CATALOG,
  FELTS,
  THEMES,
  isKnown,
  type CardBackId,
  type CosmeticKind,
  type FeltId,
  type ThemeId,
} from '../constants/cosmetics'
import { buyItem, getShop, saveLook, type Look } from '../api/shop'

export type { CardBackId, FeltId, ThemeId } from '../constants/cosmetics'
export type InvitePolicy = 'all' | 'friends' | 'nobody'

interface StoredSettings {
  theme: ThemeId
  felt: FeltId
  cardBack: CardBackId
  invites: InvitePolicy
  music: boolean
}

const KEY = 'bito_settings'
const SOUND_KEY = 'bito_sound_muted'

const DEFAULTS: StoredSettings = {
  theme: 'paper',
  felt: 'linen',
  cardBack: 'classic',
  invites: 'all',
  music: true,
}

function read(): StoredSettings {
  let raw: Partial<StoredSettings> = {}
  try {
    raw = JSON.parse(localStorage.getItem(KEY) ?? '{}')
  } catch {
    raw = {}
  }
  return {
    ...DEFAULTS,
    ...raw,
    theme: isKnown(THEMES, raw.theme) ? raw.theme : DEFAULTS.theme,
    felt: isKnown(FELTS, raw.felt) ? raw.felt : DEFAULTS.felt,
    cardBack: isKnown(CARD_BACKS, raw.cardBack) ? raw.cardBack : DEFAULTS.cardBack,
  }
}

export const useSettingsStore = defineStore('settings', () => {
  const saved = read()

  const theme = ref<ThemeId>(saved.theme)
  const felt = ref<FeltId>(saved.felt)
  const cardBack = ref<CardBackId>(saved.cardBack)
  const invites = ref<InvitePolicy>(saved.invites)
  const music = ref(saved.music)
  const soundOn = ref(localStorage.getItem(SOUND_KEY) !== 'true')
  const owned = ref(new Set<string>())
  const prices = ref(new Map<string, number>())
  const shopLoaded = ref(false)

  function priceOf(kind: CosmeticKind, id: string) {
    return prices.value.get(`${kind}:${id}`) ?? CATALOG[kind].find((i) => i.id === id)?.price ?? 0
  }

  function isOwned(kind: CosmeticKind, id: string) {
    const item = CATALOG[kind].find((i) => i.id === id)
    return !!item && (priceOf(kind, id) === 0 || owned.value.has(`${kind}:${id}`))
  }

  function isActive(kind: CosmeticKind, id: string) {
    if (kind === 'back') return cardBack.value === id
    if (kind === 'felt') return felt.value === id
    return theme.value === id
  }

  function apply(kind: CosmeticKind, id: string) {
    if (kind === 'back' && isKnown(CARD_BACKS, id)) cardBack.value = id
    else if (kind === 'felt' && isKnown(FELTS, id)) felt.value = id
    else if (kind === 'theme' && isKnown(THEMES, id)) theme.value = id
  }

  function look(): Look {
    return { theme: theme.value, felt: felt.value, back: cardBack.value }
  }

  function pushLook() {
    saveLook(look()).catch(() => loadShop().catch(() => {}))
  }

  function select(kind: CosmeticKind, id: string) {
    if (!isOwned(kind, id)) return false
    apply(kind, id)
    if (shopLoaded.value) pushLook()
    return true
  }

  async function loadShop() {
    const state = await getShop()
    prices.value = new Map(state.items.map((i) => [`${i.kind}:${i.id}`, i.price]))
    owned.value = new Set(state.owned)
    shopLoaded.value = true

    if (state.lookSaved) {
      apply('theme', state.look.theme)
      apply('felt', state.look.felt)
      apply('back', state.look.back)
      return
    }
    if (!isOwned('theme', theme.value)) theme.value = DEFAULTS.theme
    if (!isOwned('felt', felt.value)) felt.value = DEFAULTS.felt
    if (!isOwned('back', cardBack.value)) cardBack.value = DEFAULTS.cardBack
    pushLook()
  }

  async function buy(kind: CosmeticKind, id: string) {
    const result = await buyItem(`${kind}:${id}`)
    owned.value = new Set(result.owned)
    return result.balance
  }

  watch([theme, felt, cardBack, invites, music], () => {
    const value: StoredSettings = {
      theme: theme.value,
      felt: felt.value,
      cardBack: cardBack.value,
      invites: invites.value,
      music: music.value,
    }
    localStorage.setItem(KEY, JSON.stringify(value))
  })

  watch(soundOn, (on) => localStorage.setItem(SOUND_KEY, on ? 'false' : 'true'))

  return { theme, felt, cardBack, invites, music, soundOn, owned, shopLoaded, priceOf, isOwned, isActive, select, loadShop, buy }
})
