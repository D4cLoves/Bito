<script setup lang="ts">
import type { CardBackId } from '../../stores/settings'

defineProps<{ variant: CardBackId }>()
</script>

<template>
  <div class="card-back" :class="variant" aria-hidden="true">
    <span class="inner">
      <span class="mark">{{ variant === 'bito' ? 'B' : '' }}</span>
    </span>
  </div>
</template>

<style scoped>
.card-back {
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  padding: 7%;
  border-radius: 10px;
  background: #fbfaf6;
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.12),
    0 10px 24px -12px rgba(0, 0, 0, 0.6);
}

.inner {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  border-radius: 6px;
}

.mark {
  font-family: var(--font-card);
  font-size: 1.6em;
  font-style: italic;
  line-height: 1;
}

.classic .inner {
  background:
    repeating-linear-gradient(45deg, rgba(255, 255, 255, 0.18) 0 2px, transparent 2px 9px),
    repeating-linear-gradient(-45deg, rgba(255, 255, 255, 0.18) 0 2px, transparent 2px 9px),
    #9b2f2a;
}

.midnight .inner {
  background:
    linear-gradient(45deg, transparent 46%, rgba(214, 177, 108, 0.7) 47% 53%, transparent 54%) center / 60% 60% no-repeat,
    linear-gradient(-45deg, transparent 46%, rgba(214, 177, 108, 0.7) 47% 53%, transparent 54%) center / 60% 60% no-repeat,
    radial-gradient(circle at 50% 50%, #2b4166, #14223a 70%);
}

.bito .inner {
  background: #e7e1d4;
  box-shadow: inset 0 0 0 1px rgba(160, 150, 132, 0.6);
  color: #57534b;
}

.grove .inner {
  background:
    radial-gradient(circle at 25% 25%, rgba(191, 224, 145, 0.35) 0 3px, transparent 4px) 0 0 / 12px 12px,
    radial-gradient(circle at 75% 75%, rgba(191, 224, 145, 0.2) 0 2px, transparent 3px) 0 0 / 12px 12px,
    #1f4630;
}
</style>
