<script setup lang="ts">
import { computed } from 'vue'
import PaperCard from '../landing/PaperCard.vue'
import CardBack from '../client/CardBack.vue'
import { useSettingsStore } from '../../stores/settings'
import { getRankLabel, getSuitSymbol, type Card } from '../../types/game'

const props = withDefaults(
  defineProps<{
    card?: Card
    faceDown?: boolean
    size?: 'xs' | 'sm' | 'md' | 'lg'
    selected?: boolean
    playable?: boolean
  }>(),
  { size: 'md' },
)

defineEmits<{ (e: 'click'): void }>()

const settings = useSettingsStore()

const rank = computed(() => (props.card ? getRankLabel(props.card.rank) : ''))
const suit = computed(() => (props.card ? (getSuitSymbol(props.card.suit) as '♠' | '♣' | '♥' | '♦') : undefined))
</script>

<template>
  <button
    type="button"
    class="game-card"
    :class="[`s-${size}`, { selected, playable, down: faceDown || !card }]"
    :tabindex="playable ? 0 : -1"
    :aria-label="card && !faceDown ? `${rank}${suit}` : 'карта рубашкой вверх'"
    @click="$emit('click')"
  >
    <span class="flip">
      <PaperCard :rank="rank" :suit="suit">
        <template #back><CardBack :variant="settings.cardBack" /></template>
      </PaperCard>
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
  transition: transform 0.4s cubic-bezier(0.22, 1, 0.36, 1);
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

.playable:focus-visible {
  outline: 2px solid var(--color-gold);
  outline-offset: 4px;
  border-radius: 10px;
}

.selected {
  transform: translateY(-30px);
}

.selected :deep(.face--front) {
  box-shadow:
    0 0 0 1.5px color-mix(in srgb, var(--color-gold) 70%, transparent),
    0 26px 36px -18px rgba(40, 32, 20, 0.55);
}

:deep(.face--front) {
  transition: box-shadow 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}
</style>
