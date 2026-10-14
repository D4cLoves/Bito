<script setup lang="ts">
import type { GameHistoryEntry } from '../../types/social'

defineProps<{ games: GameHistoryEntry[] }>()

function when(iso: string) {
  const d = new Date(iso)
  const today = new Date()
  const time = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  if (d.toDateString() === today.toDateString()) return `сегодня, ${time}`
  const yesterday = new Date(today)
  yesterday.setDate(today.getDate() - 1)
  if (d.toDateString() === yesterday.toDateString()) return `вчера, ${time}`
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
}

const outcomeOf = (g: GameHistoryEntry) => g.outcome ?? (g.payout > 0 ? 'win' : g.payout < 0 ? 'lose' : 'draw')
const isLoss = (g: GameHistoryEntry) => outcomeOf(g) === 'lose'

function placeText(g: GameHistoryEntry) {
  const o = outcomeOf(g)
  if (o === 'draw') return 'ничья'
  if (o === 'lose') return 'дурак'
  return `${g.place} место`
}

function payoutText(p: number) {
  if (p === 0) return '0'
  return `${p > 0 ? '+' : '−'}${Math.abs(p).toLocaleString('ru-RU')}`
}
</script>

<template>
  <section class="history" aria-labelledby="history-title">
    <h3 id="history-title">История партий</h3>

    <ol v-if="games.length" class="list">
      <li v-for="g in games" :key="g.id" class="row" :class="{ loss: isLoss(g) }">
        <span class="suit" :class="{ red: g.mode === 'perevodnoy' }" aria-hidden="true">{{ g.mode === 'perevodnoy' ? '♥' : '♠' }}</span>
        <span class="what">
          <b>{{ g.mode === 'perevodnoy' ? 'Переводной' : 'Подкидной' }}</b>
          <small>{{ when(g.finishedAt) }} · {{ g.deck }} карт · {{ g.players }} игр. · ставка {{ g.stake.toLocaleString('ru-RU') }}</small>
        </span>
        <span class="place">{{ placeText(g) }}</span>
        <span class="payout">{{ payoutText(g.payout) }}</span>
      </li>
    </ol>
    <p v-else class="empty">Сыгранные партии появятся здесь.</p>
  </section>
</template>

<style scoped>
h3 {
  margin: 0 0 10px;
  font-size: 18px;
  font-weight: 500;
  letter-spacing: -0.01em;
}

.list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 6px;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  list-style: none;
}

.row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px 12px;
  border-radius: 16px;
}

.row + .row {
  border-top: 1px solid var(--color-line);
}

.suit {
  display: grid;
  place-items: center;
  flex: none;
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: var(--color-paper-hover);
  font-family: var(--font-card);
  font-size: 20px;
}

.suit.red {
  color: var(--color-red);
}

.what {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.what b {
  font-size: 15px;
  font-weight: 500;
}

.what small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: var(--color-ink-faint);
}

.place {
  padding: 3px 10px;
  border-radius: 999px;
  background: var(--color-paper-hover);
  font-size: 12px;
  color: var(--color-ink-muted);
  white-space: nowrap;
}

.loss .place {
  background: var(--color-red-tint);
  color: var(--color-red-strong);
}

.payout {
  min-width: 64px;
  font-size: 16px;
  font-weight: 500;
  text-align: right;
  color: var(--color-green);
  font-variant-numeric: tabular-nums;
}

.loss .payout {
  color: var(--color-red);
}

.empty {
  margin: 0;
  padding: 24px;
  border-radius: 20px;
  border: 1.5px dashed var(--color-line-stronger);
  color: var(--color-ink-faint);
  text-align: center;
}

@media (max-width: 560px) {
  .place {
    display: none;
  }
}
</style>
