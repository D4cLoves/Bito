<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import GameCard from './GameCard.vue'
import { HANDOFF_MS, paramsFor, tossMotion } from './tossMotion'
import type { Flight, FlightKind } from '../../composables/useCardFlights'

const props = defineProps<{ flight: Flight }>()
const emit = defineEmits<{
  (e: 'land', kind: FlightKind): void
  (e: 'reveal', id: number): void
  (e: 'done', id: number): void
}>()

const el = ref<HTMLElement | null>(null)
const shade = ref<HTMLElement | null>(null)

const width = computed(() => Math.max(props.flight.from.w, props.flight.to.w))
const size = computed(() => (width.value < 46 ? 'xs' : width.value < 80 ? 'sm' : width.value < 100 ? 'md' : 'lg'))
const startsDown = computed(() => props.flight.faceDown && !props.flight.flip)

function at(delay: number, fn: () => void) {
  const timer = el.value?.animate([], { delay: Math.max(0, delay), duration: 0 })
  timer?.finished.then(fn, () => {})
}

onMounted(() => {
  const f = props.flight
  const motion = tossMotion(f, paramsFor(f), width.value)
  const timing: KeyframeAnimationOptions = { duration: motion.duration, delay: f.delay, easing: 'linear', fill: 'both' }
  const animation = el.value?.animate(motion.card, timing)
  shade.value?.animate(motion.shadow, timing)

  if (motion.flip) {
    const flipEl = el.value?.querySelector<HTMLElement>('.flip')
    flipEl?.animate(
      { transform: f.faceDown ? ['rotateY(180deg)', 'rotateY(0deg)'] : ['rotateY(0deg)', 'rotateY(180deg)'] },
      { delay: f.delay + motion.flip.delay, duration: motion.flip.duration, easing: 'cubic-bezier(0.45, 0.05, 0.55, 0.95)', fill: 'both' },
    )
  }

  at(f.delay + motion.contact, () => emit('land', f.kind))
  if (!f.fade) at(f.delay + motion.duration - HANDOFF_MS, () => emit('reveal', f.id))

  if (!animation) return emit('done', f.id)
  animation.finished.then(
    () => emit('done', f.id),
    () => emit('done', f.id),
  )
})
</script>

<template>
  <div ref="el" class="flying" :style="{ left: `${flight.to.x}px`, top: `${flight.to.y}px`, '--cw': `${width}px` }">
    <div ref="shade" class="shade"></div>
    <GameCard :card="flight.card ?? undefined" :size="size" :face-down="startsDown" class="clone" :style="{ '--w': `${width}px` }" />
  </div>
</template>

<style scoped>
.flying {
  position: absolute;
  width: var(--cw);
  height: calc(var(--cw) * 1.42);
  margin-left: calc(var(--cw) / -2);
  margin-top: calc(var(--cw) * -0.71);
  will-change: transform, opacity;
}

.shade {
  position: absolute;
  inset: 4%;
  border-radius: 9% / 6%;
  background: rgba(0, 0, 0, 0.14);
  box-shadow: 0 0 10px 4px rgba(0, 0, 0, 0.16);
  will-change: transform, opacity;
}

.clone {
  position: absolute;
  inset: 0;
}

.clone :deep(.flip) {
  transition: none;
}

.clone :deep(.face) {
  box-shadow: 0 0 0 1px rgba(160, 150, 132, 0.35);
}
</style>
