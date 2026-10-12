<script setup lang="ts">
import { computed } from 'vue'
import { PhLockSimple } from '@phosphor-icons/vue'
import ChipIcon from './ChipIcon.vue'
import { MIN_STAKE } from '../../constants/economy'
import type { TableInfo } from '../../types/game'

const props = defineProps<{ table: TableInfo; mine: boolean; canAfford: boolean; now: number }>()
defineEmits<{ (e: 'join'): void }>()

const stake = computed(() => props.table.stake ?? MIN_STAKE)
const isTransfer = computed(() => props.table.mode === 'perevodnoy')

const seats = computed(() =>
  Array.from({ length: props.table.maxPlayers }, (_, i) => {
    const a = Math.PI / 2 + (i * 2 * Math.PI) / props.table.maxPlayers
    return { x: 50 + 44 * Math.cos(a), y: 50 + 40 * Math.sin(a), taken: i < props.table.players }
  }),
)

const ago = computed(() => {
  const s = Math.max(0, Math.round((props.now - new Date(props.table.createdAt).getTime()) / 1000))
  return s < 60 ? 'только что' : `${Math.round(s / 60)} мин`
})
</script>

<template>
  <article class="tile" :class="{ mine, locked: !canAfford }">
    <div class="felt" aria-hidden="true">
      <span class="felt-inner"></span>
      <span v-for="(s, i) in seats" :key="i" class="seat" :class="{ taken: s.taken }" :style="{ left: `${s.x}%`, top: `${s.y}%` }"></span>
      <span class="stake"><ChipIcon :size="16" />{{ stake.toLocaleString('ru-RU') }}</span>
    </div>

    <div class="info">
      <div class="title">
        <span class="suit" :class="{ red: isTransfer }">{{ isTransfer ? '♥' : '♠' }}</span>
        {{ isTransfer ? 'Переводной' : 'Подкидной' }}
        <span v-if="mine" class="tag">ваш</span>
      </div>
      <p class="meta">{{ table.deck }} карт · {{ table.players }}/{{ table.maxPlayers }} · {{ ago }}</p>
    </div>

    <button
      type="button"
      class="join"
      :disabled="!canAfford && !mine"
      :aria-label="`${isTransfer ? 'Переводной' : 'Подкидной'}, ставка ${stake}, ${table.players} из ${table.maxPlayers}`"
      @click="$emit('join')"
    >
      <PhLockSimple v-if="!canAfford && !mine" :size="14" weight="bold" aria-hidden="true" />
      {{ mine ? 'Вернуться' : canAfford ? 'Сесть' : 'Мало фишек' }}
    </button>
  </article>
</template>

<style scoped>
.tile {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px 12px 14px;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  transition:
    transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1),
    box-shadow 0.25s;
}

.tile:hover {
  transform: translateY(-3px);
  box-shadow:
    inset 0 0 0 1px var(--color-line-stronger),
    0 18px 30px -20px rgba(0, 0, 0, 0.6);
}

.tile.mine {
  box-shadow: inset 0 0 0 1.5px var(--color-gold);
}

.tile.locked {
  opacity: 0.6;
}

.felt {
  position: relative;
  aspect-ratio: 2 / 1;
  border-radius: 999px;
  background: radial-gradient(ellipse at 50% 30%, var(--color-felt), var(--color-felt-deep));
  box-shadow:
    inset 0 0 0 5px rgba(0, 0, 0, 0.25),
    inset 0 0 22px rgba(0, 0, 0, 0.4);
}

.felt-inner {
  position: absolute;
  inset: 14%;
  border-radius: 999px;
  border: 1px dashed rgba(255, 255, 255, 0.14);
}

.seat {
  position: absolute;
  width: 14px;
  height: 14px;
  translate: -50% -50%;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.35);
  box-shadow: inset 0 0 0 1.5px rgba(255, 255, 255, 0.35);
}

.seat.taken {
  background: #f3eee4;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.5);
}

.stake {
  position: absolute;
  left: 50%;
  top: 50%;
  translate: -50% -50%;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 10px 4px 5px;
  border-radius: 999px;
  background: rgba(0, 0, 0, 0.35);
  color: #f3eee4;
  font-size: 13px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.info {
  padding: 0 4px;
}

.title {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 15px;
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
  margin-left: auto;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
  font-size: 11px;
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
  height: 40px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-hover);
  color: var(--color-ink);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    color 0.2s;
}

.join:hover:not(:disabled) {
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

.join:disabled {
  cursor: not-allowed;
  color: var(--color-ink-faint);
}
</style>
