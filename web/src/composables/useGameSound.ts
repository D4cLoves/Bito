const MUTED_KEY = 'bito_sound_muted'
const BASE = '/sounds/'

const SAMPLES = {
  place: ['place-1', 'place-2', 'place-3', 'place-4'],
  slide: ['slide-1', 'slide-2', 'slide-3', 'slide-4'],
  shove: ['shove-1', 'shove-2'],
  fan: ['fan-1', 'fan-2'],
  pick: ['pick'],
  shuffle: ['shuffle'],
  chipCollide: ['chip-collide-1', 'chip-collide-2', 'chip-collide-3', 'chip-collide-4'],
  chipStack: ['chip-stack-1', 'chip-stack-2', 'chip-stack-3'],
  chipLay: ['chip-lay-1', 'chip-lay-2', 'chip-lay-3'],
  chipsRattle: ['chips-rattle'],
  tap: ['tap'],
  confirm: ['confirm'],
  turn: ['turn'],
  win: ['win'],
  lose: ['lose'],
  error: ['error'],
  message: ['message'],
} as const

type Group = keyof typeof SAMPLES

interface Voice {
  gain: number
  rate?: [number, number]
  lowpass?: number
  room?: number
  at?: number
}

interface Engine {
  ctx: AudioContext
  dry: GainNode
  wet: GainNode
  buffers: Map<string, AudioBuffer>
}

let engine: Engine | null = null
let raw: Promise<Map<string, ArrayBuffer>> | null = null
const lastPlayed = new Map<Group, number>()
const lastIndex = new Map<Group, number>()

function fetchAll() {
  if (raw) return raw
  const names: string[] = [...new Set(Object.values(SAMPLES).flat())]
  const loaded = new Map<string, ArrayBuffer>()
  raw = Promise.all(
    names.map(async (name) => {
      try {
        const res = await fetch(`${BASE}${name}.ogg`)
        if (res.ok) loaded.set(name, await res.arrayBuffer())
      } catch {
        return
      }
    }),
  ).then(() => loaded)
  return raw
}

function createEngine(): Engine | null {
  if (typeof window === 'undefined') return null
  const Ctx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext
  if (!Ctx) return null
  const ctx = new Ctx()

  const comp = ctx.createDynamicsCompressor()
  comp.threshold.value = -14
  comp.knee.value = 10
  comp.ratio.value = 2.5
  comp.attack.value = 0.005
  comp.release.value = 0.2

  const master = ctx.createGain()
  master.gain.value = 0.85
  master.connect(comp)
  comp.connect(ctx.destination)

  const dry = ctx.createGain()
  dry.connect(master)

  const reverb = ctx.createConvolver()
  reverb.buffer = roomImpulse(ctx, 0.7)
  const wet = ctx.createGain()
  wet.connect(reverb)
  reverb.connect(master)

  const e: Engine = { ctx, dry, wet, buffers: new Map() }
  fetchAll().then((map) =>
    map.forEach((data, name) => {
      ctx.decodeAudioData(data.slice(0)).then((buf) => e.buffers.set(name, buf)).catch(() => {})
    }),
  )
  return e
}

function roomImpulse(ctx: AudioContext, seconds: number) {
  const len = Math.floor(ctx.sampleRate * seconds)
  const buf = ctx.createBuffer(2, len, ctx.sampleRate)
  for (let ch = 0; ch < 2; ch++) {
    const d = buf.getChannelData(ch)
    for (let i = 0; i < len; i++) d[i] = (Math.random() * 2 - 1) * Math.pow(1 - i / len, 4)
  }
  return buf
}

function getEngine(): Engine | null {
  if (!engine) engine = createEngine()
  if (engine && engine.ctx.state === 'suspended') engine.ctx.resume().catch(() => {})
  return engine
}

if (typeof window !== 'undefined') {
  fetchAll()
  const unlock = () => {
    getEngine()
    window.removeEventListener('pointerdown', unlock)
    window.removeEventListener('keydown', unlock)
  }
  window.addEventListener('pointerdown', unlock, { passive: true })
  window.addEventListener('keydown', unlock, { passive: true })
}

function pickName(group: Group) {
  const list = SAMPLES[group]
  if (list.length === 1) return list[0]
  const prev = lastIndex.get(group) ?? -1
  let i = Math.floor(Math.random() * list.length)
  if (i === prev) i = (i + 1) % list.length
  lastIndex.set(group, i)
  return list[i]
}

function voice(e: Engine, group: Group, v: Voice) {
  const buf = e.buffers.get(pickName(group))
  if (!buf) return
  const t = e.ctx.currentTime + (v.at ?? 0)
  const [lo, hi] = v.rate ?? [0.96, 1.04]

  const src = e.ctx.createBufferSource()
  src.buffer = buf
  src.playbackRate.value = lo + Math.random() * (hi - lo)

  const gain = e.ctx.createGain()
  gain.gain.value = v.gain * (0.88 + Math.random() * 0.24)

  let tail: AudioNode = src
  if (v.lowpass) {
    const lp = e.ctx.createBiquadFilter()
    lp.type = 'lowpass'
    lp.frequency.value = v.lowpass
    lp.Q.value = 0.5
    src.connect(lp)
    tail = lp
  }
  tail.connect(gain)
  gain.connect(e.dry)
  if (v.room) {
    const send = e.ctx.createGain()
    send.gain.value = v.room
    gain.connect(send)
    send.connect(e.wet)
  }
  src.start(t)
}

function throttled(group: Group, gapMs: number) {
  const now = performance.now()
  if (now - (lastPlayed.get(group) ?? -Infinity) < gapMs) return true
  lastPlayed.set(group, now)
  return false
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

  function play(group: Group, v: Voice, gapMs = 0) {
    if (isMuted()) return
    if (gapMs && throttled(group, gapMs)) return
    const e = getEngine()
    if (e) voice(e, group, v)
  }

  const playCardDrop = () => play('place', { gain: 0.5, rate: [0.92, 1.04], lowpass: 7000, room: 0.12 })
  const playCardDeal = () => play('slide', { gain: 0.38, rate: [0.94, 1.08], lowpass: 7000, room: 0.08 })
  const playCardSweep = () => play('shove', { gain: 0.42, rate: [0.95, 1.05], lowpass: 6000, room: 0.12 }, 280)
  const playCardTake = () => play('fan', { gain: 0.5, rate: [0.97, 1.05], room: 0.1 }, 350)
  const playCardPick = () => play('pick', { gain: 0.16, rate: [0.95, 1.1] })
  const playShuffle = () => play('shuffle', { gain: 0.63, rate: [0.98, 1.02], lowpass: 6500, room: 0.12 }, 1500)

  const playClick = () => play('tap', { gain: 0.15, rate: [0.97, 1.03], lowpass: 3500 })
  const playChip = () => play('chipLay', { gain: 0.24, rate: [0.9, 1.0], lowpass: 3800, room: 0.1 })

  function playChips() {
    play('chipStack', { gain: 0.48, rate: [0.88, 0.96], lowpass: 3600, room: 0.12 })
    play('chipCollide', { gain: 0.35, rate: [0.88, 0.96], lowpass: 3600, room: 0.12, at: 0.09 })
  }

  const playMessage = () => play('message', { gain: 0.15, rate: [0.98, 1.02], lowpass: 3000, room: 0.1 })
  const playSuccess = () => play('confirm', { gain: 0.15, rate: [1, 1], lowpass: 4000, room: 0.15 })
  const playError = () => play('error', { gain: 0.24, rate: [1, 1], lowpass: 2500 }, 250)
  const playYourTurn = () => play('turn', { gain: 0.34, rate: [1, 1], lowpass: 4500, room: 0.2 })

  function playChipsWin() {
    play('chipsRattle', { gain: 0.5, rate: [0.9, 0.95], lowpass: 3800, room: 0.15 })
    play('win', { gain: 0.24, rate: [1, 1], lowpass: 4500, room: 0.25, at: 0.35 })
  }

  function playLose() {
    play('fan', { gain: 0.32, rate: [0.9, 0.95], room: 0.12 })
    play('lose', { gain: 0.16, rate: [0.94, 0.94], lowpass: 2500, room: 0.25, at: 0.3 })
  }

  return {
    isMuted,
    setMuted,
    toggleMuted,
    playCardDrop,
    playCardDeal,
    playCardSweep,
    playCardTake,
    playCardPick,
    playShuffle,
    playClick,
    playChip,
    playChips,
    playMessage,
    playSuccess,
    playError,
    playChipsWin,
    playLose,
    playYourTurn,
  }
}
