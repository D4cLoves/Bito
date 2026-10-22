<script setup lang="ts">
import { computed } from 'vue'
import { PhLockSimple } from '@phosphor-icons/vue'
import TableTop from './TableTop.vue'
import { MIN_STAKE, bestPayout } from '../../constants/economy'
import type { TableInfo } from '../../types/game'

const props = defineProps<{ table: TableInfo; mine: boolean; canAfford: boolean; now: number; me?: string }>()
defineEmits<{ (e: 'join'): void }>()

const stake = computed(() => props.table.stake ?? MIN_STAKE)
const isTransfer = computed(() => props.table.mode === 'perevodnoy')
const free = computed(() => Math.max(0, props.table.maxPlayers - props.table.players))
const almost = computed(() => free.value === 1)
const win = computed(() => bestPayout(stake.value, props.table.maxPlayers))

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
  <article class="table-card" :class="{ locked: !canAfford && !mine, mine }">
    <span v-if="mine" class="tag gold">ваш</span>
    <span v-else-if="almost" class="tag hot">ещё 1 место</span>

    <TableTop :seats="table.maxPlayers" :taken="table.players" :stake="stake" :me="mine ? me : undefined" />

    <div class="info">
      <span class="title">
        <span class="suit" aria-hidden="true">{{ isTransfer ? '♥' : '♠' }}</span>
        {{ isTransfer ? 'Переводной' : 'Подкидной' }} · {{ table.deck }}
      </span>
      <span class="meta">{{ table.players }} из {{ table.maxPlayers }} · {{ ago }}</span>
      <span class="win">выигрыш до +{{ win.toLocaleString('ru-RU') }}</span>
    </div>

    <span v-if="mine" class="join ghost">Ждём игроков…</span>
    <button v-else type="button" class="join" :disabled="!canAfford" :aria-label="`Сесть: ${label}`" @click="$emit('join')">
      <PhLockSimple v-if="!canAfford" :size="14" weight="bold" aria-hidden="true" />
      {{ canAfford ? 'Сесть' : 'Мало фишек' }}
    </button>
  </article>
</template>

<style scoped>
.table-card {
  position: relative;
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 10px 10px 14px;
  box-sizing: border-box;
  border-radius: 26px;
  background: var(--color-paper-raised);
  box-shadow:
    inset 0 0 0 1px var(--color-line),
    0 24px 40px -30px rgba(0, 0, 0, 0.7);
  transition:
    transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1),
    box-shadow 0.25s;
}

.table-card:hover {
  transform: translateY(-3px);
  box-shadow:
    inset 0 0 0 1px var(--color-line-stronger),
    0 24px 40px -24px rgba(0, 0, 0, 0.75);
}

.table-card.mine {
  box-shadow:
    inset 0 0 0 1.5px var(--color-gold),
    0 24px 40px -30px rgba(0, 0, 0, 0.7);
}

.table-card.locked {
  opacity: 0.62;
}

.tag {
  position: absolute;
  top: 14px;
  right: 14px;
  z-index: 3;
  padding: 3px 9px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 500;
}

.tag.gold {
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.tag.hot {
  background: var(--color-red-tint);
  color: var(--color-red);
}

.info {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 4px 8px 12px;
}

.title {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
  font-size: 15px;
  font-weight: 500;
  white-space: nowrap;
}

.suit {
  color: var(--color-gold);
}

.meta {
  color: var(--color-ink-faint);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.win {
  color: var(--color-green);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.join {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 42px;
  margin: auto 2px 0;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink);
  color: var(--color-paper);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition:
    filter 0.2s,
    transform 0.15s;
}

.join:hover:not(:disabled) {
  filter: brightness(1.06);
}

.join:active:not(:disabled) {
  transform: scale(0.98);
}

.join:disabled {
  background: none;
  color: var(--color-red);
  cursor: not-allowed;
}

.join.ghost {
  background: none;
  box-shadow: inset 0 0 0 1.5px var(--color-line-heavy);
  color: var(--color-ink-muted);
  cursor: default;
}
</style>
