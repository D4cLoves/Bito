<script setup lang="ts">
import { computed } from 'vue'
import { PhArrowCounterClockwise, PhArrowRight, PhCrown, PhHandshake, PhHeartBreak } from '@phosphor-icons/vue'
import type { GameHistoryEntry, GameOutcome } from '../../types/social'

const props = defineProps<{ entry: GameHistoryEntry | null; loaded: boolean; now: number }>()
defineEmits<{ (e: 'again', entry: GameHistoryEntry): void; (e: 'history'): void }>()

const outcome = computed<GameOutcome | null>(() => {
  const g = props.entry
  if (!g) return null
  if (g.outcome) return g.outcome
  return g.payout > 0 ? 'win' : g.payout < 0 ? 'lose' : 'draw'
})

const duel = computed(() => props.entry?.players === 2)

const title = computed(() => {
  const g = props.entry
  if (!g) return ''
  if (outcome.value === 'draw') return 'Ничья'
  if (outcome.value === 'lose') return duel.value ? 'Поражение' : 'Проигрыш'
  return duel.value || g.place === 1 ? 'Победа' : `${g.place} место`
})

const meta = computed(() => {
  const g = props.entry
  if (!g) return ''
  const mode = g.mode === 'perevodnoy' ? 'Переводной' : 'Подкидной'
  const who = duel.value ? '1 на 1' : `${g.players} игроков`
  return `${mode} · ${g.deck} карт · ${who} · ${ago(g.finishedAt)}`
})

const payout = computed(() => {
  const p = props.entry?.payout ?? 0
  if (p > 0) return `+${p.toLocaleString('ru-RU')}`
  if (p < 0) return `−${Math.abs(p).toLocaleString('ru-RU')}`
  return 'ставка вернулась'
})

function ago(iso: string) {
  const min = Math.max(0, Math.round((props.now - new Date(iso).getTime()) / 60000))
  if (min < 1) return 'только что'
  if (min < 60) return `${min} мин назад`
  const h = Math.round(min / 60)
  if (h < 24) return `${h} ч назад`
  return new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
}
</script>

<template>
  <section class="last" aria-labelledby="last-title">
    <header class="head">
      <h2 id="last-title">Последняя партия</h2>
      <button v-if="entry" type="button" class="link" @click="$emit('history')">
        История
        <PhArrowRight :size="15" weight="bold" aria-hidden="true" />
      </button>
    </header>

    <div v-if="!loaded" class="card skeleton" aria-busy="true"></div>

    <div v-else-if="entry" class="card" :class="outcome">
      <span class="badge" :class="{ first: outcome === 'win' && entry.place === 1 }" aria-hidden="true">
        <PhCrown v-if="outcome === 'win' && entry.place === 1" :size="22" weight="fill" />
        <PhHeartBreak v-else-if="outcome === 'lose'" :size="22" weight="fill" />
        <PhHandshake v-else-if="outcome === 'draw'" :size="22" weight="fill" />
        <template v-else>{{ entry.place }}</template>
      </span>
      <span class="what">
        <b>{{ title }}</b>
        <small>{{ meta }}</small>
      </span>
      <span class="payout">{{ payout }}</span>
      <button type="button" class="again" @click="$emit('again', entry)">
        <PhArrowCounterClockwise :size="16" weight="bold" aria-hidden="true" />
        Ещё партия
      </button>
    </div>

    <div v-else class="card empty">
      <span class="badge" aria-hidden="true">?</span>
      <span class="what">
        <b>Пока ни одной партии</b>
        <small>Сыграйте первую — здесь появится её итог и выигрыш.</small>
      </span>
    </div>
  </section>
</template>

<style scoped>
.last {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  letter-spacing: -0.01em;
}

.link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0;
  border: 0;
  background: none;
  color: var(--color-ink-faint);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.link:hover {
  color: var(--color-ink);
}

.card {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 84px;
  padding: 14px 16px;
  box-sizing: border-box;
  border-radius: 24px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.card.win {
  background: linear-gradient(100deg, color-mix(in srgb, var(--color-green) 14%, var(--color-paper-raised)), var(--color-paper-raised) 55%);
}

.card.lose {
  background: linear-gradient(100deg, var(--color-red-tint), var(--color-paper-raised) 55%);
}

.skeleton {
  background: linear-gradient(100deg, var(--color-paper-sunken) 30%, var(--color-paper-hover) 50%, var(--color-paper-sunken) 70%);
  background-size: 200% 100%;
  animation: shimmer 1.4s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

.badge {
  display: grid;
  place-items: center;
  flex: none;
  width: 52px;
  height: 52px;
  border-radius: 18px;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  font-size: 20px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.badge.first {
  background: var(--color-gold);
  color: #3a2a0c;
}

.lose .badge {
  background: var(--color-red-tint-strong);
  color: var(--color-red);
}

.what {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  gap: 3px;
}

.what b {
  font-size: 17px;
  font-weight: 500;
}

.what small {
  overflow: hidden;
  font-size: 13px;
  color: var(--color-ink-faint);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.payout {
  flex: none;
  font-size: 22px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.01em;
}

.win .payout {
  color: var(--color-green);
}

.lose .payout {
  color: var(--color-red);
}

.draw .payout {
  font-size: 14px;
  color: var(--color-ink-faint);
}

.again {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: none;
  height: 42px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.2s;
}

.again:hover {
  background: var(--color-ink);
}

.empty .badge {
  font-size: 22px;
  color: var(--color-ink-ghost);
}

@media (max-width: 600px) {
  .card {
    flex-wrap: wrap;
  }

  .what {
    flex-basis: calc(100% - 68px);
  }

  .payout {
    margin-left: 68px;
  }

  .again {
    margin-left: auto;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton {
    animation: none;
  }
}
</style>
