/**
 * Procedural Web Audio API Sound FX Engine for BITO: Durak Online
 * Zero external MP3 files, zero latency, pure synthesized acoustic tactile audio.
 */

let audioCtx: AudioContext | null = null

function getAudioContext(): AudioContext | null {
  if (typeof window === 'undefined') return null
  if (!audioCtx) {
    const AudioContextClass = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
    if (AudioContextClass) {
      audioCtx = new AudioContextClass()
    }
  }
  if (audioCtx && audioCtx.state === 'suspended') {
    audioCtx.resume().catch(() => {})
  }
  return audioCtx
}

// Global user interaction listener to unlock AudioContext
if (typeof window !== 'undefined') {
  const unlock = () => {
    if (audioCtx && audioCtx.state === 'suspended') {
      audioCtx.resume().catch(() => {})
    }
  }
  window.addEventListener('click', unlock, { once: true, passive: true })
  window.addEventListener('keydown', unlock, { once: true, passive: true })
  window.addEventListener('touchstart', unlock, { once: true, passive: true })
}

export function useGameSound() {
  const isMuted = () => localStorage.getItem('bito_sound_muted') === 'true'

  const setMuted = (muted: boolean) => {
    localStorage.setItem('bito_sound_muted', muted ? 'true' : 'false')
  }

  const toggleMuted = () => {
    const next = !isMuted()
    setMuted(next)
    return next
  }

  /**
   * Card Slap on Felt:
   * Short shaped noise burst passed through a steep low-pass filter with a low sine thump.
   */
  function playCardDrop() {
    if (isMuted()) return
    const ctx = getAudioContext()
    if (!ctx) return

    const now = ctx.currentTime

    // 1. Felt thump (low sine)
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()
    osc.type = 'sine'
    osc.frequency.setValueAtTime(140, now)
    osc.frequency.exponentialRampToValueAtTime(35, now + 0.08)

    gain.gain.setValueAtTime(0.35, now)
    gain.gain.exponentialRampToValueAtTime(0.001, now + 0.08)

    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.start(now)
    osc.stop(now + 0.08)

    // 2. Card snap friction (filtered noise)
    const bufferSize = ctx.sampleRate * 0.04
    const buffer = ctx.createBuffer(1, bufferSize, ctx.sampleRate)
    const output = buffer.getChannelData(0)
    for (let i = 0; i < bufferSize; i++) {
      output[i] = (Math.random() * 2 - 1) * Math.exp(-i / (bufferSize * 0.25))
    }

    const whiteNoise = ctx.createBufferSource()
    whiteNoise.buffer = buffer

    const filter = ctx.createBiquadFilter()
    filter.type = 'bandpass'
    filter.frequency.setValueAtTime(800, now)
    filter.Q.setValueAtTime(1.5, now)

    const noiseGain = ctx.createGain()
    noiseGain.gain.setValueAtTime(0.2, now)
    noiseGain.gain.exponentialRampToValueAtTime(0.001, now + 0.04)

    whiteNoise.connect(filter)
    filter.connect(noiseGain)
    noiseGain.connect(ctx.destination)

    whiteNoise.start(now)
    whiteNoise.stop(now + 0.04)
  }

  /**
   * Card Deal Swipe:
   * Whispering card friction glide.
   */
  function playCardDeal() {
    if (isMuted()) return
    const ctx = getAudioContext()
    if (!ctx) return

    const now = ctx.currentTime
    const bufferSize = ctx.sampleRate * 0.06
    const buffer = ctx.createBuffer(1, bufferSize, ctx.sampleRate)
    const data = buffer.getChannelData(0)
    for (let i = 0; i < bufferSize; i++) {
      data[i] = (Math.random() * 2 - 1) * (1 - i / bufferSize)
    }

    const noise = ctx.createBufferSource()
    noise.buffer = buffer

    const filter = ctx.createBiquadFilter()
    filter.type = 'bandpass'
    filter.frequency.setValueAtTime(1400, now)
    filter.frequency.exponentialRampToValueAtTime(500, now + 0.06)
    filter.Q.setValueAtTime(2.0, now)

    const gain = ctx.createGain()
    gain.gain.setValueAtTime(0.12, now)
    gain.gain.exponentialRampToValueAtTime(0.001, now + 0.06)

    noise.connect(filter)
    filter.connect(gain)
    gain.connect(ctx.destination)

    noise.start(now)
    noise.stop(now + 0.06)
  }

  /**
   * Chips / Coins Clink:
   * Harmonic sinusoidal chimes.
   */
  function playChipsWin() {
    if (isMuted()) return
    const ctx = getAudioContext()
    if (!ctx) return

    const now = ctx.currentTime
    const frequencies = [1760, 2217, 2637, 3520] // A6, C#7, E7, A7

    frequencies.forEach((freq, idx) => {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      const startTime = now + idx * 0.035

      osc.type = 'sine'
      osc.frequency.setValueAtTime(freq, startTime)

      gain.gain.setValueAtTime(0.15, startTime)
      gain.gain.exponentialRampToValueAtTime(0.0001, startTime + 0.35)

      osc.connect(gain)
      gain.connect(ctx.destination)
      osc.start(startTime)
      osc.stop(startTime + 0.35)
    })
  }

  /**
   * Tactile Micro-Click:
   * Premium subtle UI tap.
   */
  function playClick() {
    if (isMuted()) return
    const ctx = getAudioContext()
    if (!ctx) return

    const now = ctx.currentTime
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()

    osc.type = 'triangle'
    osc.frequency.setValueAtTime(320, now)
    osc.frequency.exponentialRampToValueAtTime(120, now + 0.02)

    gain.gain.setValueAtTime(0.08, now)
    gain.gain.exponentialRampToValueAtTime(0.001, now + 0.02)

    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.start(now)
    osc.stop(now + 0.02)
  }

  /**
   * Success Chime:
   * Major interval bell.
   */
  function playSuccess() {
    if (isMuted()) return
    const ctx = getAudioContext()
    if (!ctx) return

    const now = ctx.currentTime
    const notes = [523.25, 659.25, 783.99, 1046.5] // C5, E5, G5, C6
    notes.forEach((freq, i) => {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      const start = now + i * 0.06

      osc.type = 'sine'
      osc.frequency.setValueAtTime(freq, start)

      gain.gain.setValueAtTime(0.12, start)
      gain.gain.exponentialRampToValueAtTime(0.001, start + 0.25)

      osc.connect(gain)
      gain.connect(ctx.destination)
      osc.start(start)
      osc.stop(start + 0.25)
    })
  }

  /**
   * Soft Error Warning:
   * Low discordant tap.
   */
  function playError() {
    if (isMuted()) return
    const ctx = getAudioContext()
    if (!ctx) return

    const now = ctx.currentTime
    const osc = ctx.createOscillator()
    const gain = ctx.createGain()

    osc.type = 'sawtooth'
    osc.frequency.setValueAtTime(180, now)
    osc.frequency.exponentialRampToValueAtTime(90, now + 0.12)

    gain.gain.setValueAtTime(0.08, now)
    gain.gain.exponentialRampToValueAtTime(0.001, now + 0.12)

    osc.connect(gain)
    gain.connect(ctx.destination)
    osc.start(now)
    osc.stop(now + 0.12)
  }

  return {
    isMuted,
    setMuted,
    toggleMuted,
    playCardDrop,
    playCardDeal,
    playChipsWin,
    playClick,
    playSuccess,
    playError,
  }
}
