<script setup lang="ts">
import { computed } from 'vue'
import { PhLockSimple } from '@phosphor-icons/vue'
import StakeChip from './StakeChip.vue'
import { MIN_STAKE } from '../../constants/economy'
import type { TableInfo } from '../../types/game'

const props = defineProps<{ table: TableInfo; mine: boolean; canAfford: boolean; now: number }>()
defineEmits<{ (e: 'join'): void }>()

const stake = computed(() => props.table.stake ?? MIN_STAKE)
const isTransfer = computed(() => props.table.mode === 'perevodnoy')
const free = computed(() => Math.max(0, props.table.maxPlayers - props.table.players))
const almost = computed(() => free.value === 1)

const seats = computed(() =>
  Array.from({ length: props.table.maxPlayers }, (_, i) => {
    const a = Math.PI / 2 + (i * 2 * Math.PI) / props.table.maxPlayers
    return { x: 50 + 46 * Math.cos(a), y: 50 + 42 * Math.sin(a), taken: i < props.table.players }
  }),
)

const ago = computed(() => {
  const s = Math.max(0, Math.round((props.now - new Date(props.table.createdAt).getTime()) / 1000))
  if (s < 60) return 'только что'
  const m = Math.round(s / 60)
  return m < 60 ? `${m} мин` : `${Math.round(m / 60)} ч`
})

const label = computed(
  () =>
    `${isTransfer.value ? 'Переводной' : 'Подкидной'}, ставка ${stake.value}, ${props.table.deck} карт, ${props.table.players} из ${props.table.maxPlayers}`,
)
</script>

<template>
  <article class="table-card" :class="{ locked: !canAfford, mine, almost }">
    <div class="felt" aria-hidden="true">
      <span class="felt-rim"></span>
      <span
        v-for="(s, i) in seats"
        :key="i"
        class="seat"
        :class="{ taken: s.taken }"
        :style="{ left: `${s.x}%`, top: `${s.y}%` }"
      ></span>
      <StakeChip class="stake" :value="stake" :size="46" />
    </div>

    <div class="info">
      <div class="title">
        <span class="suit" :class="{ red: isTransfer }" aria-hidden="true">{{ isTransfer ? '♥' : '♠' }}</span>
        <span class="name">{{ isTransfer ? 'Переводной' : 'Подкидной' }}</span>
        <span v-if="mine" class="tag gold">ваш</span>
        <span v-else-if="almost" class="tag hot">ещё 1 место</span>
      </div>
      <p class="meta">{{ table.deck }} карт · {{ table.players }} из {{ table.maxPlayers }} · {{ ago }}</p>
    </div>

    <button type="button" class="join" :disabled="!canAfford" :aria-label="`Сесть: ${label}`" @click="$emit('join')">
      <PhLockSimple v-if="!canAfford" :size="14" weight="bold" aria-hidden="true" />
      {{ canAfford ? 'Сесть' : 'Мало фишек' }}
    </button>
  </article>
</template>

<style scoped>
.table-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px 12px 14px;
  box-sizing: border-box;
  border-radius: 24px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  transition:
    transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1),
    box-shadow 0.25s;
}

.table-card:hover {
  transform: translateY(-3px);
  box-shadow:
    inset 0 0 0 1px var(--color-line-stronger),
    0 20px 34px -22px rgba(0, 0, 0, 0.6);
}

.table-card.mine {
  box-shadow: inset 0 0 0 1.5px var(--color-gold);
}

.table-card.almost:not(.mine) {
  box-shadow: inset 0 0 0 1.5px color-mix(in srgb, var(--color-red) 55%, transparent);
}

.table-card.locked {
  opacity: 0.62;
}

.felt {
  position: relative;
  aspect-ratio: 16 / 10;
  margin: 8px;
  border-radius: 999px;
  background:
    radial-gradient(ellipse at 50% 30%, rgba(255, 255, 255, 0.14), transparent 60%),
    radial-gradient(ellipse at 50% 60%, var(--color-felt), var(--color-felt-deep));
  box-shadow:
    inset 0 0 0 5px var(--color-felt-well),
    inset 0 0 0 6px var(--color-felt-line),
    0 12px 22px -14px rgba(0, 0, 0, 0.6);
}

.felt-rim {
  position: absolute;
  inset: 16%;
  border-radius: 999px;
  border: 1px dashed var(--color-felt-line);
}

.seat {
  position: absolute;
  width: 16px;
  height: 16px;
  translate: -50% -50%;
  border-radius: 50%;
  background: var(--color-paper-raised);
  box-shadow:
    inset 0 0 0 1.5px var(--color-line-heavy),
    0 0 0 2px var(--color-paper-raised);
}

.seat.taken {
  background: var(--color-ink-soft);
  box-shadow:
    0 0 0 2px var(--color-paper-raised),
    0 3px 6px rgba(0, 0, 0, 0.35);
}

.stake {
  position: absolute;
  left: 50%;
  top: 50%;
  translate: -50% -50%;
}

.info {
  padding: 0 4px;
}

.title {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  font-size: 15px;
}

.name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.suit {
  font-family: var(--font-card);
  font-size: 17px;
  line-height: 1;
}

.suit.red {
  color: var(--color-red);
}

.tag {
  flex: none;
  margin-left: auto;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  white-space: nowrap;
}

.tag.gold {
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.tag.hot {
  background: var(--color-red-tint);
  color: var(--color-red);
}

.meta {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--color-ink-faint);
  font-variant-numeric: tabular-nums;
}

.join {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 42px;
  margin-top: auto;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.15s;
}

.join:hover:not(:disabled) {
  background: var(--color-ink);
}

.join:active:not(:disabled) {
  transform: scale(0.98);
}

.join:disabled {
  background: var(--color-paper-hover);
  color: var(--color-ink-faint);
  cursor: not-allowed;
}
</style>
