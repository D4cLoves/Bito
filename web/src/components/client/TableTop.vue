<script setup lang="ts">
import { computed } from 'vue'
import { PhUser } from '@phosphor-icons/vue'
import StakeChip from './StakeChip.vue'
import { avatarHue, avatarInitial } from '../../composables/avatar'

const props = defineProps<{ seats: number; taken: number; stake: number; me?: string; muted?: boolean }>()

const list = computed(() =>
  Array.from({ length: props.seats }, (_, i) => {
    const a = Math.PI / 2 + (i * 2 * Math.PI) / props.seats
    return { x: 50 + 44 * Math.cos(a), y: 50 + 41 * Math.sin(a), taken: i < props.taken, me: i === 0 && !!props.me }
  }),
)
</script>

<template>
  <div class="top" :class="{ muted }" aria-hidden="true">
    <div class="rim">
      <div class="cloth">
        <StakeChip class="chip" :value="stake" :size="40" />
      </div>
    </div>
    <span
      v-for="(s, i) in list"
      :key="i"
      class="seat"
      :class="{ taken: s.taken, me: s.me }"
      :style="{ left: `${s.x}%`, top: `${s.y}%`, '--h': me ? avatarHue(me) : 0 }"
    >
      <template v-if="s.me && me">{{ avatarInitial(me) }}</template>
      <PhUser v-else-if="s.taken" :size="16" weight="fill" />
      <template v-else>+</template>
    </span>
  </div>
</template>

<style scoped>
.top {
  position: relative;
  height: var(--table-h, 180px);
}

.rim {
  position: absolute;
  inset: 22px 20px;
  padding: 8px;
  border-radius: 999px;
  background: linear-gradient(180deg, #7a5532, #4a3019);
  box-shadow:
    inset 0 2px 0 rgba(255, 255, 255, 0.18),
    inset 0 -3px 0 rgba(0, 0, 0, 0.35),
    0 14px 22px -12px rgba(0, 0, 0, 0.8);
}

.cloth {
  position: relative;
  display: grid;
  place-items: center;
  height: 100%;
  border-radius: 999px;
  background: radial-gradient(ellipse at 50% 35%, color-mix(in srgb, var(--color-felt) 75%, #fff 12%), var(--color-felt) 55%, var(--color-felt-deep));
  box-shadow: inset 0 0 18px rgba(0, 0, 0, 0.5);
}

.cloth::after {
  content: '';
  position: absolute;
  inset: 9px 16px;
  border: 1px dashed rgba(255, 255, 255, 0.14);
  border-radius: 999px;
}

.chip {
  position: relative;
  z-index: 1;
}

.seat {
  position: absolute;
  z-index: 2;
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  margin: -17px 0 0 -17px;
  border-radius: 50%;
  background: var(--color-paper-sunken);
  box-shadow:
    0 0 0 3px var(--color-paper-raised),
    inset 0 0 0 1.5px var(--color-line-heavy);
  color: var(--color-ink-ghost);
  font-size: 16px;
  line-height: 1;
}

.seat.taken {
  background: linear-gradient(160deg, #f3eee4, #d9d0bd);
  box-shadow:
    0 0 0 3px var(--color-paper-raised),
    0 6px 10px -4px rgba(0, 0, 0, 0.7);
  color: #5a5246;
}

.seat.me {
  background: hsl(var(--h) 46% 50%);
  box-shadow:
    0 0 0 2px var(--color-paper-raised),
    0 0 0 4px var(--color-gold);
  color: #fff;
  font-size: 13px;
  font-weight: 500;
}

.muted .rim,
.muted .seat {
  opacity: 0.62;
}
</style>
