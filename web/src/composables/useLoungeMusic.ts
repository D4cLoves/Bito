import { computed, onBeforeUnmount, watch } from 'vue'
import { useSettingsStore } from '../stores/settings'

const TRACKS = ['bossa-antigua', 'lobby-time', 'smooth-lovin', 'airport-lounge']
const VOLUME = 0.18
const FADE_MS = 1200

function shuffled<T>(list: T[]) {
  const out = [...list]
  for (let i = out.length - 1; i > 0; i--) {
    const j = Math.floor(Math.random() * (i + 1))
    ;[out[i], out[j]] = [out[j], out[i]]
  }
  return out
}

export function useLoungeMusic() {
  const settings = useSettingsStore()
  const enabled = computed(() => settings.music && settings.soundOn)

  const audio = new Audio()
  audio.preload = 'none'
  audio.volume = 0
  let queue: string[] = []
  let fadeTimer: number | null = null
  let waitingGesture = false

  function nextTrack() {
    if (!queue.length) queue = shuffled(TRACKS)
    audio.src = `/music/${queue.shift()}.mp3`
  }

  function fadeTo(target: number, done?: () => void) {
    if (fadeTimer) clearInterval(fadeTimer)
    const from = audio.volume
    const start = performance.now()
    fadeTimer = window.setInterval(() => {
      const k = Math.min(1, (performance.now() - start) / FADE_MS)
      audio.volume = from + (target - from) * k
      if (k < 1) return
      if (fadeTimer) clearInterval(fadeTimer)
      fadeTimer = null
      done?.()
    }, 50)
  }

  function onGesture() {
    removeGesture()
    if (enabled.value) start()
  }

  function removeGesture() {
    if (!waitingGesture) return
    waitingGesture = false
    window.removeEventListener('pointerdown', onGesture)
    window.removeEventListener('keydown', onGesture)
  }

  function start() {
    if (!audio.src) nextTrack()
    audio
      .play()
      .then(() => fadeTo(VOLUME))
      .catch(() => {
        if (waitingGesture) return
        waitingGesture = true
        window.addEventListener('pointerdown', onGesture)
        window.addEventListener('keydown', onGesture)
      })
  }

  function stop() {
    removeGesture()
    fadeTo(0, () => audio.pause())
  }

  audio.addEventListener('ended', () => {
    nextTrack()
    audio.play().catch(() => {})
  })

  watch(enabled, (on) => (on ? start() : stop()), { immediate: true })

  onBeforeUnmount(() => {
    removeGesture()
    if (fadeTimer) clearInterval(fadeTimer)
    audio.pause()
    audio.removeAttribute('src')
    audio.load()
  })
}
