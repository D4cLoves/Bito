import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeId = 'hall' | 'paper' | 'forest'
export type FeltId = 'linen' | 'green' | 'wine' | 'navy' | 'graphite'
export type CardBackId = 'classic' | 'midnight' | 'bito' | 'grove'
export type InvitePolicy = 'all' | 'friends' | 'nobody'

interface StoredSettings {
  theme: ThemeId
  felt: FeltId
  cardBack: CardBackId
  showChat: boolean
  invites: InvitePolicy
}

const KEY = 'bito_settings'
const SOUND_KEY = 'bito_sound_muted'

const DEFAULTS: StoredSettings = {
  theme: 'paper',
  felt: 'linen',
  cardBack: 'classic',
  showChat: true,
  invites: 'all',
}

function read(): StoredSettings {
  try {
    return { ...DEFAULTS, ...JSON.parse(localStorage.getItem(KEY) ?? '{}') }
  } catch {
    return { ...DEFAULTS }
  }
}

export const useSettingsStore = defineStore('settings', () => {
  const saved = read()

  const theme = ref<ThemeId>(saved.theme)
  const felt = ref<FeltId>(saved.felt)
  const cardBack = ref<CardBackId>(saved.cardBack)
  const showChat = ref(saved.showChat)
  const invites = ref<InvitePolicy>(saved.invites)
  const soundOn = ref(localStorage.getItem(SOUND_KEY) !== 'true')

  watch([theme, felt, cardBack, showChat, invites], () => {
    const value: StoredSettings = {
      theme: theme.value,
      felt: felt.value,
      cardBack: cardBack.value,
      showChat: showChat.value,
      invites: invites.value,
    }
    localStorage.setItem(KEY, JSON.stringify(value))
  })

  watch(soundOn, (on) => localStorage.setItem(SOUND_KEY, on ? 'false' : 'true'))

  return { theme, felt, cardBack, showChat, invites, soundOn }
})
