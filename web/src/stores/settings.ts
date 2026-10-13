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
import { MOCK_OWNED } from '../mocks/lobby'

export type { CardBackId, FeltId, ThemeId } from '../constants/cosmetics'
export type InvitePolicy = 'all' | 'friends' | 'nobody'

interface StoredSettings {
  theme: ThemeId
  felt: FeltId
  cardBack: CardBackId
  showChat: boolean
  invites: InvitePolicy
  music: boolean
}

const KEY = 'bito_settings'
const SOUND_KEY = 'bito_sound_muted'

const DEFAULTS: StoredSettings = {
  theme: 'paper',
  felt: 'linen',
  cardBack: 'classic',
  showChat: true,
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
  const showChat = ref(saved.showChat)
  const invites = ref<InvitePolicy>(saved.invites)
  const music = ref(saved.music)
  const soundOn = ref(localStorage.getItem(SOUND_KEY) !== 'true')
  const owned = ref(new Set<string>(MOCK_OWNED))

  function isOwned(kind: CosmeticKind, id: string) {
    const item = CATALOG[kind].find((i) => i.id === id)
    return !!item && (item.price === 0 || owned.value.has(`${kind}:${id}`))
  }

  function isActive(kind: CosmeticKind, id: string) {
    if (kind === 'back') return cardBack.value === id
    if (kind === 'felt') return felt.value === id
    return theme.value === id
  }

  function select(kind: CosmeticKind, id: string) {
    if (!isOwned(kind, id)) return false
    if (kind === 'back' && isKnown(CARD_BACKS, id)) cardBack.value = id
    else if (kind === 'felt' && isKnown(FELTS, id)) felt.value = id
    else if (kind === 'theme' && isKnown(THEMES, id)) theme.value = id
    return true
  }

  watch([theme, felt, cardBack, showChat, invites, music], () => {
    const value: StoredSettings = {
      theme: theme.value,
      felt: felt.value,
      cardBack: cardBack.value,
      showChat: showChat.value,
      invites: invites.value,
      music: music.value,
    }
    localStorage.setItem(KEY, JSON.stringify(value))
  })

  watch(soundOn, (on) => localStorage.setItem(SOUND_KEY, on ? 'false' : 'true'))

  return { theme, felt, cardBack, showChat, invites, music, soundOn, owned, isOwned, isActive, select }
})
