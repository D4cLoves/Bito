<script setup lang="ts">
import { computed, ref } from 'vue'
import { motion, useMotionValue, useSpring, useTransform, useReducedMotion } from 'motion-v'
import { useCountUp } from '../../composables/useCountUp'
import { memberSince, rankTitle } from '../../composables/usePlayerRank'
import type { Profile } from '../../types/user'

const props = defineProps<{ profile: Profile | null }>()

const reduced = useReducedMotion()
const el = ref<HTMLElement | null>(null)
const px = useMotionValue(0)
const py = useMotionValue(0)
const sx = useSpring(px, { stiffness: 160, damping: 18 })
const sy = useSpring(py, { stiffness: 160, damping: 18 })
const rotateY = useTransform(sx, [-0.5, 0.5], [-14, 14])
const rotateX = useTransform(sy, [-0.5, 0.5], [12, -12])
const glareX = useTransform(sx, [-0.5, 0.5], ['20%', '80%'])
const glareY = useTransform(sy, [-0.5, 0.5], ['15%', '85%'])

function onMove(e: PointerEvent) {
  if (reduced.value || !el.value) return
  const r = el.value.getBoundingClientRect()
  px.set((e.clientX - r.left) / r.width - 0.5)
  py.set((e.clientY - r.top) / r.height - 0.5)
}

function onLeave() {
  px.set(0)
  py.set(0)
}

const elo = useCountUp(computed(() => props.profile?.stats.elo), 1200)
const initial = computed(() => (props.profile?.username?.[0] ?? '?').toUpperCase())
const title = computed(() => rankTitle(props.profile?.stats.elo ?? 0))
const since = computed(() => memberSince(props.profile?.createdAt))
</script>

<template>
  <div ref="el" class="stage rise" @pointermove="onMove" @pointerleave="onLeave">
    <motion.div
      class="card"
      :style="{ rotateX, rotateY }"
    >
      <div v-if="!profile" class="sk" aria-busy="true"></div>
      <template v-else>
        <span class="corner tl"><b>{{ initial }}</b><i>♠</i></span>
        <span class="corner br"><b>{{ initial }}</b><i>♠</i></span>

        <div class="face">
          <span class="label">Рейтинг</span>
          <span class="elo">{{ elo }}</span>
          <span class="rank">{{ title }}</span>
          <span class="rule" aria-hidden="true"></span>
          <span class="name">{{ profile.username }}</span>
          <span class="since">в Bito с {{ since }}</span>
        </div>

        <motion.span class="glare" :style="{ '--gx': glareX, '--gy': glareY }" aria-hidden="true" />
      </template>
    </motion.div>
  </div>
</template>

<style scoped>
.stage {
  perspective: 1100px;
  display: grid;
  place-items: center;
  padding: 10px 0;
}

.card {
  position: relative;
  width: min(320px, 100%);
  aspect-ratio: 5 / 7;
  border-radius: 24px;
  background: linear-gradient(160deg, var(--color-paper-raised), #f2efe7);
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.9) inset,
    0 0 0 1px rgba(160, 150, 132, 0.4),
    0 40px 70px -36px rgba(70, 62, 48, 0.6);
  color: var(--color-card-ink);
  transform-style: preserve-3d;
  overflow: hidden;
}

.card::before {
  content: '';
  position: absolute;
  inset: 12px;
  border: 1px solid rgba(160, 150, 132, 0.3);
  border-radius: 16px;
  pointer-events: none;
}

.sk {
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, var(--color-paper-sunken) 0%, var(--color-paper-raised) 50%, var(--color-paper-sunken) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

.corner {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  font-family: var(--font-card);
  line-height: 1;
}

.corner b {
  font-size: 30px;
  font-weight: 400;
}

.corner i {
  margin-top: 3px;
  font-size: 22px;
  font-style: normal;
}

.tl {
  left: 24px;
  top: 22px;
}

.br {
  right: 24px;
  bottom: 22px;
  transform: rotate(180deg);
}

.face {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 0 40px;
  text-align: center;
  transform: translateZ(40px);
}

.label {
  font-size: 12px;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: var(--color-ink-faint);
}

.elo {
  margin-top: 4px;
  font-family: var(--font-card);
  font-size: 84px;
  line-height: 1;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}

.rank {
  margin-top: 8px;
  padding: 4px 12px;
  border-radius: 999px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
  font-size: 13px;
}

.rule {
  width: 40px;
  height: 1px;
  margin: 22px 0 18px;
  background: var(--color-line-heavy);
}

.name {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 22px;
  letter-spacing: -0.02em;
}

.since {
  margin-top: 4px;
  font-size: 13px;
  color: var(--color-ink-faint);
}

.glare {
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at var(--gx, 50%) var(--gy, 50%), rgba(255, 255, 255, 0.55), transparent 45%);
  mix-blend-mode: soft-light;
  pointer-events: none;
}
</style>
