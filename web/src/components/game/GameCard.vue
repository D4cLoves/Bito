<script setup lang="ts">
import { computed } from 'vue'
import PaperCard from '../landing/PaperCard.vue'
import { getRankLabel, getSuitSymbol, type Card } from '../../types/game'

const props = withDefaults(
  defineProps<{
    card?: Card
    faceDown?: boolean
    size?: 'xs' | 'sm' | 'md' | 'lg'
    selected?: boolean
    playable?: boolean
    dimmed?: boolean
    trump?: boolean
  }>(),
  { size: 'md' },
)

defineEmits<{ (e: 'click'): void }>()

const rank = computed(() => (props.card ? getRankLabel(props.card.rank) : ''))
const suit = computed(() => (props.card ? (getSuitSymbol(props.card.suit) as '♠' | '♣' | '♥' | '♦') : undefined))
</script>

<template>
  <button
    type="button"
    class="game-card"
    :class="[`s-${size}`, { selected, playable, dimmed, trump, down: faceDown || !card }]"
    :tabindex="playable ? 0 : -1"
    :aria-label="card && !faceDown ? `${rank}${suit}` : 'карта рубашкой вверх'"
    @click="$emit('click')"
  >
    <span class="flip">
      <PaperCard :rank="rank" :suit="suit" />
    </span>
  </button>
</template>

<style scoped>
.game-card {
  --w: 92px;
  position: relative;
  flex: none;
  width: var(--w);
  height: calc(var(--w) * 1.42);
  padding: 0;
  border: 0;
  background: none;
  perspective: 900px;
  cursor: default;
  transition:
    transform 0.28s cubic-bezier(0.2, 0.8, 0.2, 1),
    filter 0.2s,
    opacity 0.2s;
}

.s-xs {
  --w: 34px;
}
.s-sm {
  --w: 58px;
}
.s-lg {
  --w: 104px;
}

.s-xs :deep(.pip) {
  font-size: 0;
}
.s-xs :deep(.index) {
  display: none;
}
.s-xs :deep(.back-mark) {
  width: 14px;
  height: 14px;
  font-size: 0;
}
.s-sm :deep(.pip) {
  font-size: 24px;
}
.s-sm :deep(.index) {
  font-size: 13px;
}
.s-sm :deep(.index i) {
  font-size: 11px;
}
.s-sm :deep(.back-mark) {
  width: 18px;
  height: 18px;
  font-size: 10px;
}

.flip {
  display: block;
  width: 100%;
  height: 100%;
  transform-style: preserve-3d;
  transition: transform 0.5s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.down .flip {
  transform: rotateY(180deg);
}

.playable {
  cursor: pointer;
}

.playable:hover {
  transform: translateY(-14px);
}

.playable:focus-visible {
  outline: 2px solid var(--color-ink-soft);
  outline-offset: 4px;
  border-radius: 10px;
}

.selected,
.selected.playable:hover {
  transform: translateY(-26px);
}

.selected :deep(.face--front) {
  box-shadow:
    0 0 0 2px var(--color-ink-soft),
    0 18px 30px -14px rgba(54, 50, 44, 0.55);
}

.trump :deep(.face--front) {
  box-shadow:
    0 0 0 2px var(--color-gold),
    0 10px 20px -12px rgba(120, 90, 40, 0.5);
}

.dimmed {
  filter: saturate(0.4);
  opacity: 0.55;
}
</style>
