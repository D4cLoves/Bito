import type { Flight } from '../../composables/useCardFlights'

export interface TossParams {
  duration: number
  air: number
  launch: number
  lift: number
  arc: number
  arcMax: number
  skid: number
  skidMax: number
  friction: number
  spin: number
  wobble: number
  cushion: number
  flipStart: number
  flipEnd: number
  fadeIn: number
}

export interface TossMotion {
  duration: number
  contact: number
  card: Keyframe[]
  shadow: Keyframe[]
  flip: { delay: number; duration: number } | null
}

export const HANDOFF_MS = 90
const FADE_OUT = 0.22
const REF_DIST = 400

const TOSS = {
  throwMine: { duration: 780, air: 0.5, launch: 1.45, lift: 0.1, arc: 0.08, arcMax: 44, skid: 0.14, skidMax: 48, friction: 4.5, spin: 30, wobble: 2.5, cushion: 0.18, flipStart: 0, flipEnd: 0, fadeIn: 0 },
  throwOpponent: { duration: 900, air: 0.56, launch: 1.2, lift: 0.14, arc: 0.1, arcMax: 56, skid: 0.12, skidMax: 42, friction: 4.5, spin: 46, wobble: 2.5, cushion: 0.18, flipStart: 0.16, flipEnd: 0.48, fadeIn: 0.05 },
  dealMine: { duration: 580, air: 0.76, launch: 1.15, lift: 0.07, arc: 0.06, arcMax: 30, skid: 0.05, skidMax: 12, friction: 5.5, spin: 0, wobble: 1.2, cushion: 0.08, flipStart: 0.24, flipEnd: 0.68, fadeIn: 0 },
  dealOpponent: { duration: 470, air: 1, launch: 1.15, lift: 0.05, arc: 0.05, arcMax: 24, skid: 0, skidMax: 0, friction: 0, spin: 0, wobble: 0, cushion: 0, flipStart: 0, flipEnd: 0, fadeIn: 0 },
  sweep: { duration: 660, air: 0.62, launch: -0.5, lift: 0.04, arc: 0.04, arcMax: 18, skid: 0.1, skidMax: 22, friction: 6, spin: 0, wobble: 0, cushion: 0.14, flipStart: 0.34, flipEnd: 0.7, fadeIn: 0 },
  take: { duration: 580, air: 1, launch: 0, lift: 0.08, arc: 0.05, arcMax: 24, skid: 0, skidMax: 0, friction: 0, spin: 0, wobble: 0, cushion: 0, flipStart: 0.34, flipEnd: 0.66, fadeIn: 0 },
} satisfies Record<string, TossParams>

export function paramsFor(f: Flight): TossParams {
  if (f.kind === 'throw') return f.faceDown ? TOSS.throwOpponent : TOSS.throwMine
  if (f.kind === 'deal') return f.card ? TOSS.dealMine : TOSS.dealOpponent
  return TOSS[f.kind]
}

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v))
const n2 = (v: number) => v.toFixed(2)

function seeded(seed: number) {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = Math.imul(a ^ (a >>> 15), a | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

function frictionCurve(k: number) {
  if (k < 0.01) return { g: (s: number) => 1 - (1 - s) ** 2, v0: 2 }
  const e = Math.exp(-k)
  const norm = (1 - e) / k - e
  return { g: (s: number) => ((1 - Math.exp(-k * s)) / k - s * e) / norm, v0: (1 - e) / norm }
}

function hermite(t: number, v0: number, v1: number) {
  const t2 = t * t
  const t3 = t2 * t
  return (t3 - 2 * t2 + t) * v0 + (3 * t2 - 2 * t3) + (t3 - t2) * v1
}

export function tossMotion(f: Flight, p: TossParams, cardWidth: number): TossMotion {
  const rand = seeded(Math.imul(f.id + 1, 0x9e3779b1))
  const jitter = (amount: number) => 1 + (rand() * 2 - 1) * amount

  const sx = f.from.x - f.to.x
  const sy = f.from.y - f.to.y
  const dist = Math.hypot(sx, sy)
  const ux = dist > 1 ? -sx / dist : 0
  const uy = dist > 1 ? -sy / dist : -1
  const nx = -uy
  const ny = ux

  const T = p.duration * clamp((dist / REF_DIST) ** 0.3, 0.8, 1.3) * jitter(0.05)
  const L = Math.min(p.skidMax, p.skid * dist, dist * 0.4) * jitter(0.25)
  const skids = L > 0.5
  const tAir = skids ? T * p.air : T
  const tSkid = T - tAir
  const fr = frictionCurve(p.friction)
  const cushion = skids ? p.cushion : 0

  let v0 = p.launch
  let v1 = skids ? clamp((L * fr.v0 * tAir) / (tSkid * Math.max(1, dist - L)), 0.3, 1.6) : f.fade ? 0.7 : 0
  const over = Math.hypot(v0, v1) / 3
  if (over > 1) {
    v0 /= over
    v1 /= over
  }

  const spinDir = Math.abs(ux) > 0.25 ? Math.sign(ux) : rand() < 0.5 ? -1 : 1
  const base = ((((f.to.angle - f.from.angle) % 360) + 540) % 360) - 180
  const twist = spinDir * p.spin * jitter(0.3)
  const angleAt = (progress: number) => f.from.angle + base * progress + twist * 4 * progress * (1 - progress)

  const bow = Math.min(p.arcMax, p.arc * dist) * jitter(0.4) * (rand() < 0.75 ? spinDir : -spinDir)
  const s0 = f.from.w / cardWidth
  const s1 = f.to.w / cardWidth
  const k = dist > 1 ? L / dist : 0
  const tx = sx * k
  const ty = sy * k

  const card: Keyframe[] = []
  const shadow: Keyframe[] = []

  const frame = (t: number, x: number, y: number, deg: number, scale: number, z: number) => {
    const o = t / T
    let opacity = p.fadeIn ? o / p.fadeIn : 1
    if (f.fade) opacity = Math.min(opacity, (1 - o) / FADE_OUT)
    else opacity = Math.min(opacity, (T - t) / HANDOFF_MS + 0.15)
    card.push({
      offset: o,
      opacity: n2(clamp(opacity, 0, 1)),
      transform: `translate(${n2(x)}px, ${n2(y)}px) rotate(${n2(deg)}deg) scale(${scale.toFixed(4)})`,
    })
    const drop = (3 + 22 * z) / Math.max(scale, 0.2)
    const rad = (deg * Math.PI) / 180
    shadow.push({
      offset: o,
      opacity: n2(clamp(1 - 0.5 * z, 0.3, 1)),
      transform: `translate(${n2(drop * Math.sin(rad))}px, ${n2(drop * Math.cos(rad))}px) scale(${(1 + 0.08 * z).toFixed(3)})`,
    })
  }

  const N = clamp(Math.round(T / 16), 24, 64)
  const nAir = skids ? Math.round(N * p.air) : N
  for (let i = 0; i < nAir; i++) {
    const tau = i / nAir
    const h = hermite(tau, v0, v1)
    const side = 6.75 * tau * (1 - tau) ** 2 * bow
    const z = 4 * tau * (1 - tau) + cushion * tau
    const scale = (s0 + (s1 - s0) * clamp(h, 0, 1)) * (1 + p.lift * z)
    frame(tau * tAir, sx + (tx - sx) * h + nx * side, sy + (ty - sy) * h + ny * side, angleAt(clamp(h, 0, 1) * (1 - k)), scale, z)
  }
  for (let j = 0, nSkid = N - nAir; skids && j < nSkid; j++) {
    const s = j / nSkid
    const g = fr.g(s)
    const z = cushion * (1 - g)
    const wobble = spinDir * p.wobble * Math.sin(Math.PI * s) ** 2
    frame(tAir + s * tSkid, tx * (1 - g), ty * (1 - g), angleAt(1 - k + k * g) + wobble, s1 * (1 + p.lift * z), z)
  }
  frame(T, 0, 0, f.to.angle, s1, 0)

  const flip = f.flip ? { delay: T * p.flipStart, duration: T * Math.max(0.1, p.flipEnd - p.flipStart) } : null
  return { duration: T, contact: skids ? tAir : T * 0.92, card, shadow, flip }
}
