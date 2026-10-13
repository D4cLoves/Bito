const MUTED_KEY = 'bito_sound_muted'

interface Engine {
  ctx: AudioContext
  dry: GainNode
  wet: GainNode
  noise: AudioBuffer
}

interface NoiseOpts {
  at?: number
  dur: number
  gain: number
  type: BiquadFilterType
  freq: number
  freqEnd?: number
  q?: number
  attack?: number
  room?: number
}

interface NoteOpts {
  at?: number
  freq: number
  gain: number
  decay: number
  room?: number
}

let engine: Engine | null = null

function createEngine(): Engine | null {
  if (typeof window === 'undefined') return null
  const Ctx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
  if (!Ctx) return null
  const ctx = new Ctx()

  const comp = ctx.createDynamicsCompressor()
  comp.threshold.value = -18
  comp.knee.value = 12
  comp.ratio.value = 3
  comp.attack.value = 0.003
  comp.release.value = 0.15

  const master = ctx.createGain()
  master.gain.value = 0.9
  master.connect(comp)
  comp.connect(ctx.destination)

  const dry = ctx.createGain()
  dry.connect(master)

  const reverb = ctx.createConvolver()
  reverb.buffer = roomImpulse(ctx, 0.9)
  const wet = ctx.createGain()
  wet.connect(reverb)
  reverb.connect(master)

  const noise = ctx.createBuffer(1, ctx.sampleRate * 2, ctx.sampleRate)
  const data = noise.getChannelData(0)
  for (let i = 0; i < data.length; i++) data[i] = Math.random() * 2 - 1

  return { ctx, dry, wet, noise }
}

function roomImpulse(ctx: AudioContext, seconds: number) {
  const len = Math.floor(ctx.sampleRate * seconds)
  const buf = ctx.createBuffer(2, len, ctx.sampleRate)
  for (let ch = 0; ch < 2; ch++) {
    const d = buf.getChannelData(ch)
    for (let i = 0; i < len; i++) d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 3.5)
  }
  return buf
}

function getEngine(): Engine | null {
  if (!engine) engine = createEngine()
  if (engine && engine.ctx.state === 'suspended') engine.ctx.resume().catch(() => {})
  return engine
}

if (typeof window !== 'undefined') {
  const unlock = () => {
    if (engine && engine.ctx.state === 'suspended') engine.ctx.resume().catch(() => {})
  }
  window.addEventListener('click', unlock, { once: true, passive: true })
  window.addEventListener('keydown', unlock, { once: true, passive: true })
  window.addEventListener('touchstart', unlock, { once: true, passive: true })
}

const vary = (value: number, spread: number) => value * (1 + (Math.random() * 2 - 1) * spread)

function route(e: Engine, node: AudioNode, room: number) {
  node.connect(e.dry)
  if (room > 0) {
    const send = e.ctx.createGain()
    send.gain.value = room
    node.connect(send)
    send.connect(e.wet)
  }
}

function noiseBurst(e: Engine, o: NoiseOpts) {
  const t = e.ctx.currentTime + (o.at ?? 0)
  const attack = o.attack ?? 0.002

  const src = e.ctx.createBufferSource()
  src.buffer = e.noise

  const filter = e.ctx.createBiquadFilter()
  filter.type = o.type
  filter.Q.value = o.q ?? 1
  filter.frequency.setValueAtTime(o.freq, t)
  if (o.freqEnd) filter.frequency.exponentialRampToValueAtTime(o.freqEnd, t + o.dur)

  const env = e.ctx.createGain()
  env.gain.setValueAtTime(0.0001, t)
  env.gain.linearRampToValueAtTime(o.gain, t + attack)
  env.gain.exponentialRampToValueAtTime(0.0001, t + o.dur)

  src.connect(filter)
  filter.connect(env)
  route(e, env, o.room ?? 0)
  src.start(t, Math.random() * 1.5, o.dur + 0.05)
}

function marimba(e: Engine, o: NoteOpts) {
  const t = e.ctx.currentTime + (o.at ?? 0)
  const out = e.ctx.createGain()
  out.gain.value = o.gain
  route(e, out, o.room ?? 0.25)

  const partials: [number, number, number][] = [
    [1, 1, o.decay],
    [3.93, 0.28, o.decay * 0.22],
    [9.2, 0.06, o.decay * 0.08],
  ]
  for (const [ratio, level, decay] of partials) {
    const osc = e.ctx.createOscillator()
    osc.type = 'sine'
    osc.frequency.value = o.freq * ratio
    const env = e.ctx.createGain()
    env.gain.setValueAtTime(0.0001, t)
    env.gain.linearRampToValueAtTime(level, t + 0.004)
    env.gain.exponentialRampToValueAtTime(0.0001, t + decay)
    osc.connect(env)
    env.connect(out)
    osc.start(t)
    osc.stop(t + decay + 0.02)
  }
}

function thump(e: Engine, at: number, freq: number, gain: number, dur: number) {
  const t = e.ctx.currentTime + at
  const osc = e.ctx.createOscillator()
  osc.type = 'sine'
  osc.frequency.setValueAtTime(freq, t)
  osc.frequency.exponentialRampToValueAtTime(freq * 0.6, t + dur)
  const env = e.ctx.createGain()
  env.gain.setValueAtTime(0.0001, t)
  env.gain.linearRampToValueAtTime(gain, t + 0.002)
  env.gain.exponentialRampToValueAtTime(0.0001, t + dur)
  osc.connect(env)
  route(e, env, 0)
  osc.start(t)
  osc.stop(t + dur + 0.02)
}

function chip(e: Engine, at: number, gain: number) {
  const t = e.ctx.currentTime + at
  const base = vary(3400, 0.18)
  for (const ratio of [1, 1.47, 2.09]) {
    const osc = e.ctx.createOscillator()
    osc.type = 'sine'
    osc.frequency.value = base * ratio
    const env = e.ctx.createGain()
    env.gain.setValueAtTime(0.0001, t)
    env.gain.linearRampToValueAtTime(gain / ratio, t + 0.001)
    env.gain.exponentialRampToValueAtTime(0.0001, t + vary(0.07, 0.3))
    osc.connect(env)
    route(e, env, 0.15)
    osc.start(t)
    osc.stop(t + 0.12)
  }
  noiseBurst(e, { at, dur: 0.012, gain: gain * 1.4, type: 'bandpass', freq: vary(5200, 0.15), q: 3 })
}

export function useGameSound() {
  const isMuted = () => localStorage.getItem(MUTED_KEY) === 'true'

  const setMuted = (muted: boolean) => {
    localStorage.setItem(MUTED_KEY, muted ? 'true' : 'false')
  }

  const toggleMuted = () => {
    const next = !isMuted()
    setMuted(next)
    return next
  }

  function play(fn: (e: Engine) => void) {
    if (isMuted()) return
    const e = getEngine()
    if (e) fn(e)
  }

  const playCardDrop = () =>
    play((e) => {
      noiseBurst(e, { dur: 0.018, gain: vary(0.12, 0.2), type: 'highpass', freq: vary(3200, 0.15), room: 0.05 })
      noiseBurst(e, { dur: vary(0.07, 0.15), gain: vary(0.32, 0.15), type: 'bandpass', freq: vary(1100, 0.2), q: 0.8, room: 0.08 })
      thump(e, 0, vary(210, 0.1), 0.07, 0.05)
    })

  const playCardDeal = () =>
    play((e) => {
      noiseBurst(e, { dur: vary(0.1, 0.15), gain: vary(0.16, 0.2), type: 'bandpass', freq: vary(3400, 0.1), freqEnd: 1500, q: 1.3, attack: 0.025, room: 0.06 })
      noiseBurst(e, { at: 0.085, dur: 0.03, gain: vary(0.1, 0.2), type: 'bandpass', freq: vary(1300, 0.15), q: 0.9 })
    })

  const playCardSweep = () =>
    play((e) => {
      noiseBurst(e, { dur: vary(0.17, 0.1), gain: vary(0.16, 0.15), type: 'lowpass', freq: 2600, freqEnd: 700, q: 0.7, attack: 0.04, room: 0.1 })
    })

  const playCardTake = () =>
    play((e) => {
      noiseBurst(e, { dur: vary(0.12, 0.1), gain: vary(0.22, 0.15), type: 'bandpass', freq: vary(1300, 0.1), freqEnd: 2800, q: 1.1, attack: 0.03, room: 0.06 })
    })

  const playCardPick = () =>
    play((e) => {
      noiseBurst(e, { dur: 0.03, gain: 0.1, type: 'bandpass', freq: vary(2600, 0.1), freqEnd: 3800, q: 1.5, attack: 0.006 })
    })

  const playClick = () =>
    play((e) => {
      marimba(e, { freq: vary(1180, 0.02), gain: 0.07, decay: 0.06, room: 0 })
      noiseBurst(e, { dur: 0.006, gain: 0.04, type: 'highpass', freq: 4000 })
    })

  const playMessage = () =>
    play((e) => {
      const t = e.ctx.currentTime
      const osc = e.ctx.createOscillator()
      osc.type = 'sine'
      osc.frequency.setValueAtTime(620, t)
      osc.frequency.exponentialRampToValueAtTime(980, t + 0.05)
      const env = e.ctx.createGain()
      env.gain.setValueAtTime(0.0001, t)
      env.gain.linearRampToValueAtTime(0.2, t + 0.008)
      env.gain.exponentialRampToValueAtTime(0.0001, t + 0.12)
      osc.connect(env)
      route(e, env, 0.1)
      osc.start(t)
      osc.stop(t + 0.14)
    })

  const playSuccess = () =>
    play((e) => {
      marimba(e, { freq: 659.25, gain: 0.16, decay: 0.5 })
      marimba(e, { at: 0.09, freq: 987.77, gain: 0.15, decay: 0.7 })
    })

  const playError = () =>
    play((e) => {
      marimba(e, { freq: 392, gain: 0.085, decay: 0.25, room: 0.1 })
      marimba(e, { at: 0.08, freq: 349.23, gain: 0.08, decay: 0.3, room: 0.1 })
    })

  const playChipsWin = () =>
    play((e) => {
      let at = 0
      for (let i = 0; i < 7; i++) {
        chip(e, at, vary(0.07, 0.25))
        at += vary(0.075 - i * 0.006, 0.25)
      }
      const chord = [523.25, 659.25, 783.99, 1046.5]
      chord.forEach((freq, i) => marimba(e, { at: 0.12 + i * 0.075, freq, gain: 0.14, decay: 0.9, room: 0.35 }))
    })

  const playLose = () =>
    play((e) => {
      const notes = [392, 329.63, 261.63]
      notes.forEach((freq, i) => marimba(e, { at: i * 0.16, freq, gain: 0.15, decay: 0.8, room: 0.4 }))
    })

  const playYourTurn = () =>
    play((e) => {
      marimba(e, { freq: 880, gain: 0.2, decay: 0.7, room: 0.3 })
      marimba(e, { at: 0.13, freq: 1318.51, gain: 0.2, decay: 0.9, room: 0.3 })
    })

  return {
    isMuted,
    setMuted,
    toggleMuted,
    playCardDrop,
    playCardDeal,
    playCardSweep,
    playCardTake,
    playCardPick,
    playClick,
    playMessage,
    playSuccess,
    playError,
    playChipsWin,
    playLose,
    playYourTurn,
  }
}
