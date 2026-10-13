<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import GameCard from './GameCard.vue'
import type { Flight, FlightKind } from '../../composables/useCardFlights'

interface Motion {
  duration: number
  easing: string
  flipAt: number
  frames: (f: Flight, dx: number, dy: number, s: number) => Keyframe[]
}

const props = defineProps<{ flight: Flight }>()
const emit = defineEmits<{ (e: 'done', id: number): void }>()

const el = ref<HTMLElement | null>(null)
const faceDown = ref(props.flight.faceDown)
let flipTimer = 0

const SHADOW_HIGH = 'drop-shadow(0 26px 20px rgba(0, 0, 0, 0.3))'
const SHADOW_MID = 'drop-shadow(0 12px 10px rgba(0, 0, 0, 0.26))'
const SHADOW_LOW = 'drop-shadow(0 3px 4px rgba(0, 0, 0, 0.18))'

function overshoot(dx: number, dy: number, px: number) {
  const dist = Math.hypot(dx, dy) || 1
  return Math.min(0.1, px / dist)
}

const MOTIONS: Record<FlightKind, Motion> = {
  throw: {
    duration: 680,
    easing: 'cubic-bezier(0.12, 0.72, 0.28, 1)',
    flipAt: 0.3,
    frames: (f, dx, dy, s) => {
      const k = overshoot(dx, dy, 34)
      const spin = dx > 0 ? -1 : 1
      return [
        { transform: `translate(${dx}px, ${dy}px) rotate(${f.from.angle - spin * 28}deg) scale(${s})`, filter: SHADOW_HIGH, offset: 0 },
        { transform: `translate(${-dx * k}px, ${-dy * k}px) rotate(${f.to.angle + spin * 14}deg) scale(1.04)`, filter: SHADOW_MID, offset: 0.56 },
        { transform: `translate(${dx * k * 0.25}px, ${dy * k * 0.25}px) rotate(${f.to.angle - spin * 3}deg) scale(1)`, filter: SHADOW_LOW, offset: 0.82 },
        { transform: `translate(0, 0) rotate(${f.to.angle}deg) scale(1)`, filter: SHADOW_LOW, offset: 1 },
      ]
    },
  },
  deal: {
    duration: 460,
    easing: 'cubic-bezier(0.2, 0.8, 0.3, 1)',
    flipAt: 0.45,
    frames: (f, dx, dy, s) => {
      const k = overshoot(dx, dy, 14)
      return [
        { transform: `translate(${dx}px, ${dy}px) rotate(${f.from.angle - 10}deg) scale(${s})`, filter: SHADOW_HIGH, opacity: 0, offset: 0 },
        { transform: `translate(${dx * 0.85}px, ${dy * 0.85}px) rotate(${f.from.angle - 6}deg) scale(${s})`, filter: SHADOW_HIGH, opacity: 1, offset: 0.12 },
        { transform: `translate(${-dx * k}px, ${-dy * k}px) rotate(${f.to.angle + 4}deg) scale(1.02)`, filter: SHADOW_MID, opacity: 1, offset: 0.72 },
        { transform: `translate(0, 0) rotate(${f.to.angle}deg) scale(1)`, filter: SHADOW_LOW, opacity: f.fade ? 0 : 1, offset: 1 },
      ]
    },
  },
  sweep: {
    duration: 560,
    easing: 'cubic-bezier(0.55, 0, 0.25, 1)',
    flipAt: 0.35,
    frames: (f, dx, dy, s) => [
      { transform: `translate(${dx}px, ${dy}px) rotate(${f.from.angle}deg) scale(${s})`, filter: SHADOW_LOW, opacity: 1, offset: 0 },
      { transform: `translate(${dx * 0.9}px, ${dy * 0.9 - 16}px) rotate(${f.from.angle + 8}deg) scale(${s * 1.04})`, filter: SHADOW_HIGH, opacity: 1, offset: 0.2 },
      { transform: `translate(0, 0) rotate(${f.to.angle}deg) scale(1)`, filter: SHADOW_LOW, opacity: 1, offset: 0.88 },
      { transform: `translate(0, 0) rotate(${f.to.angle}deg) scale(1)`, filter: SHADOW_LOW, opacity: f.fade ? 0 : 1, offset: 1 },
    ],
  },
  take: {
    duration: 520,
    easing: 'cubic-bezier(0.45, 0, 0.2, 1)',
    flipAt: 0.4,
    frames: (f, dx, dy, s) => [
      { transform: `translate(${dx}px, ${dy}px) rotate(${f.from.angle}deg) scale(${s})`, filter: SHADOW_LOW, opacity: 1, offset: 0 },
      { transform: `translate(${dx * 0.92}px, ${dy * 0.92 - 12}px) rotate(${f.from.angle - 6}deg) scale(${s * 1.04})`, filter: SHADOW_HIGH, opacity: 1, offset: 0.18 },
      { transform: `translate(0, 0) rotate(${f.to.angle}deg) scale(1)`, filter: SHADOW_LOW, opacity: 1, offset: 0.9 },
      { transform: `translate(0, 0) rotate(${f.to.angle}deg) scale(1)`, filter: SHADOW_LOW, opacity: f.fade ? 0 : 1, offset: 1 },
    ],
  },
}

onMounted(() => {
  const f = props.flight
  const motion = MOTIONS[f.kind]
  const dx = f.from.x - f.to.x
  const dy = f.from.y - f.to.y
  const s = f.from.w / f.to.w
  const animation = el.value?.animate(motion.frames(f, dx, dy, s), {
    duration: motion.duration,
    delay: f.delay,
    easing: motion.easing,
    fill: 'both',
  })
  if (f.flip) flipTimer = window.setTimeout(() => (faceDown.value = !faceDown.value), f.delay + motion.duration * motion.flipAt)
  if (!animation) return emit('done', f.id)
  animation.finished.then(
    () => emit('done', f.id),
    () => emit('done', f.id),
  )
})

onUnmounted(() => clearTimeout(flipTimer))
</script>

<template>
  <div ref="el" class="flying" :style="{ left: `${flight.to.x}px`, top: `${flight.to.y}px`, '--w': `${flight.to.w}px` }">
    <GameCard :card="flight.card ?? undefined" :face-down="faceDown" :style="{ '--w': `${flight.to.w}px` }" />
  </div>
</template>

<style scoped>
.flying {
  position: absolute;
  width: var(--w);
  height: calc(var(--w) * 1.42);
  margin-left: calc(var(--w) / -2);
  margin-top: calc(var(--w) * -0.71);
  will-change: transform, opacity;
}

.flying :deep(.flip) {
  transition-duration: 0.32s;
}
</style>
