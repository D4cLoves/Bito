<script setup lang="ts">
import { computed } from 'vue'
import { formatChips } from '../../constants/economy'

const props = withDefaults(defineProps<{ value: number; size?: number; label?: boolean }>(), { size: 40, label: true })

const TONES = [100, 250, 500, 1000, 2500, 5000]
const tone = computed(() => `v${TONES.includes(props.value) ? props.value : 100}`)
</script>

<template>
  <span class="stake-chip" :class="tone" :style="{ '--size': `${size}px` }" aria-hidden="true">
    <span class="face">{{ label ? formatChips(value) : '' }}</span>
  </span>
</template>

<style scoped>
.stake-chip {
  --chip: #e9e4d8;
  --chip-ink: #2b2824;

  position: relative;
  display: inline-grid;
  place-items: center;
  flex: none;
  width: var(--size);
  height: var(--size);
  border-radius: 50%;
  background:
    repeating-conic-gradient(from 0deg, rgba(255, 255, 255, 0.85) 0 10deg, transparent 10deg 45deg),
    var(--chip);
  box-shadow:
    0 calc(var(--size) * 0.07) 0 rgba(0, 0, 0, 0.32),
    0 calc(var(--size) * 0.16) calc(var(--size) * 0.3) calc(var(--size) * -0.14) rgba(0, 0, 0, 0.55);
}

.face {
  display: grid;
  place-items: center;
  width: 68%;
  height: 68%;
  border-radius: 50%;
  background: var(--chip);
  box-shadow: inset 0 0 0 calc(var(--size) * 0.03) rgba(255, 255, 255, 0.55);
  color: var(--chip-ink);
  font-size: calc(var(--size) * 0.23);
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.v250 {
  --chip: #b6463c;
  --chip-ink: #fff;
}

.v500 {
  --chip: #2f7a55;
  --chip-ink: #fff;
}

.v1000 {
  --chip: #23232a;
  --chip-ink: #f3eee4;
}

.v2500 {
  --chip: #6a4a9a;
  --chip-ink: #fff;
}

.v5000 {
  --chip: #d4ab5c;
  --chip-ink: #3a2a0c;
}
</style>
