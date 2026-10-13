<script setup lang="ts">
import type { CardBackId } from '../../constants/cosmetics'

defineProps<{ variant: CardBackId }>()

const MARKS: Partial<Record<CardBackId, string>> = {
  bito: 'B',
  sakura: '✿',
  royal: '♛',
  deco: '◆',
}
</script>

<template>
  <div class="card-back" :class="variant" aria-hidden="true">
    <span class="inner">
      <span v-if="MARKS[variant]" class="mark">{{ MARKS[variant] }}</span>
    </span>
  </div>
</template>

<style scoped>
.card-back {
  container-type: inline-size;
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
  position: relative;
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  overflow: hidden;
  border-radius: 6px;
}

.mark {
  position: relative;
  display: grid;
  place-items: center;
  width: 42cqw;
  height: 42cqw;
  border-radius: 50%;
  font-family: var(--font-card);
  font-size: 22cqw;
  line-height: 1;
}

.classic .inner {
  background:
    repeating-linear-gradient(45deg, rgba(255, 255, 255, 0.18) 0 2px, transparent 2px 9px),
    repeating-linear-gradient(-45deg, rgba(255, 255, 255, 0.18) 0 2px, transparent 2px 9px),
    #9b2f2a;
}

.bito .inner {
  background:
    repeating-linear-gradient(45deg, transparent 0 6px, rgba(160, 148, 124, 0.18) 6px 7px),
    repeating-linear-gradient(-45deg, transparent 0 6px, rgba(160, 148, 124, 0.18) 6px 7px),
    #e7e1d4;
  box-shadow: inset 0 0 0 1px rgba(160, 150, 132, 0.6);
}

.bito .mark {
  background: #efeadf;
  box-shadow: inset 0 0 0 1px #b9ae98;
  color: #8a7f68;
  font-style: italic;
}

.checker .inner {
  background:
    conic-gradient(#7a1f2b 25%, #f3ead8 0 50%, #7a1f2b 0 75%, #f3ead8 0) 0 0 / 10px 10px;
  box-shadow: inset 0 0 0 3px #7a1f2b;
}

.grove .inner {
  background:
    radial-gradient(circle at 25% 25%, rgba(191, 224, 145, 0.35) 0 3px, transparent 4px) 0 0 / 12px 12px,
    radial-gradient(circle at 75% 75%, rgba(191, 224, 145, 0.2) 0 2px, transparent 3px) 0 0 / 12px 12px,
    #1f4630;
}

.emerald .inner {
  background:
    linear-gradient(45deg, #0d5340 25%, transparent 25% 75%, #0d5340 75%) 0 0 / 12px 12px,
    linear-gradient(45deg, #0d5340 25%, #157a5c 25% 75%, #0d5340 75%) 6px 6px / 12px 12px;
  box-shadow:
    inset 0 0 0 2px #d6b16c,
    inset 0 0 0 4px #0d5340;
}

.sakura .inner {
  background:
    radial-gradient(circle at 30% 30%, rgba(255, 255, 255, 0.9) 0 1.5px, transparent 2.5px) 0 0 / 14px 14px,
    radial-gradient(circle at 75% 70%, #e795ab 0 2.5px, transparent 3.5px) 0 0 / 14px 14px,
    linear-gradient(160deg, #fbe6ec, #f0c3d0);
  box-shadow: inset 0 0 0 1px rgba(201, 106, 133, 0.35);
}

.sakura .mark {
  background: rgba(255, 255, 255, 0.75);
  color: #c96a85;
}

.waves .inner {
  background:
    radial-gradient(circle at 50% 100%, #1f4f80 0 22%, #f2efe6 23% 30%, #1f4f80 31% 46%, #f2efe6 47% 54%, #1f4f80 55% 70%, transparent 71%) 0 0 / 20px 10px,
    radial-gradient(circle at 50% 100%, #1f4f80 0 22%, #f2efe6 23% 30%, #1f4f80 31% 46%, #f2efe6 47% 54%, #1f4f80 55% 70%, transparent 71%) 10px 5px / 20px 10px,
    #1f4f80;
}

.deco .inner {
  background:
    radial-gradient(circle at 50% 100%, transparent 0 28%, #c9a24a 29% 33%, transparent 34% 50%, #c9a24a 51% 55%, transparent 56% 72%, #c9a24a 73% 77%, transparent 78%) 0 0 / 18px 18px,
    #16161a;
  box-shadow:
    inset 0 0 0 2px #c9a24a,
    inset 0 0 0 4px #16161a,
    inset 0 0 0 5px rgba(201, 162, 74, 0.6);
}

.deco .mark {
  background: #16161a;
  box-shadow: 0 0 0 1.5px #c9a24a;
  color: #e3c06b;
  font-size: 16cqw;
}

.royal .inner {
  background:
    repeating-linear-gradient(45deg, rgba(214, 177, 108, 0.4) 0 1px, transparent 1px 8px),
    repeating-linear-gradient(-45deg, rgba(214, 177, 108, 0.4) 0 1px, transparent 1px 8px),
    radial-gradient(circle at 50% 50%, #50308a, #2c1752 75%);
  box-shadow: inset 0 0 0 2px #d6b16c;
}

.royal .mark {
  background: #2c1752;
  box-shadow: 0 0 0 1.5px #d6b16c;
  color: #e9c886;
}
</style>
