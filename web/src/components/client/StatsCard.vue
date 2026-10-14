<script setup lang="ts">
import { computed, onMounted, ref, useId } from 'vue'
import { PhFire } from '@phosphor-icons/vue'
import { useCountUp } from '../../composables/useCountUp'
import { winRate } from '../../composables/usePlayerRank'
import type { Profile } from '../../types/user'
import type { GameHistoryEntry } from '../../types/social'

const props = defineProps<{ profile: Profile | null; history: GameHistoryEntry[] }>()
defineEmits<{ (e: 'profile'): void }>()

const R = 50
const RING = 2 * Math.PI * R
const gradientId = `stats-ring-${useId()}`

const stats = computed(() => props.profile?.stats)
const rate = computed(() => (stats.value ? winRate(stats.value.wins, stats.value.losses) : null))

const isWin = (g: GameHistoryEntry) => (g.outcome ? g.outcome === 'win' : g.payout > 0)

const streak = computed(() => {
  let n = 0
  for (const g of props.history) {
    if (!isWin(g)) break
    n++
  }
  return n
})

const games = useCountUp(computed(() => stats.value?.totalGames))
const wins = useCountUp(computed(() => stats.value?.wins))
const streakShown = useCountUp(streak)
const rateShown = useCountUp(computed(() => rate.value ?? 0), 1100)

const ready = ref(false)
onMounted(() => window.setTimeout(() => (ready.value = true), 200))
</script>

<template>
  <section class="stats" aria-labelledby="stats-title">
    <header class="head">
      <h2 id="stats-title">Статистика</h2>
      <button type="button" class="link" @click="$emit('profile')">Профиль</button>
    </header>

    <div class="rate">
      <svg viewBox="0 0 120 120" aria-hidden="true">
        <defs>
          <linearGradient :id="gradientId" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0" style="stop-color: var(--color-gold)" />
            <stop offset="1" style="stop-color: var(--color-green)" />
          </linearGradient>
        </defs>
        <circle class="track" cx="60" cy="60" :r="R" />
        <circle
          class="bar"
          cx="60"
          cy="60"
          :r="R"
          :stroke="`url(#${gradientId})`"
          :stroke-dasharray="RING"
          :style="{ strokeDashoffset: ready ? RING * (1 - (rate ?? 0) / 100) : RING }"
        />
      </svg>
      <div class="center">
        <b>{{ rate === null ? '—' : `${rateShown}%` }}</b>
        <span>{{ rate === null ? 'ещё нет партий' : 'побед' }}</span>
      </div>
    </div>

    <dl class="nums">
      <div>
        <dt>Игр</dt>
        <dd>{{ games }}</dd>
      </div>
      <div>
        <dt>Побед</dt>
        <dd>{{ wins }}</dd>
      </div>
      <div :class="{ hot: streak >= 2 }">
        <dt>Серия</dt>
        <dd>
          <PhFire v-if="streak >= 2" :size="16" weight="fill" aria-hidden="true" />
          {{ streakShown }}
        </dd>
      </div>
    </dl>
  </section>
</template>

<style scoped>
.stats {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px 18px 16px;
  border-radius: 26px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}

h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
}

.link {
  padding: 0;
  border: 0;
  background: none;
  color: var(--color-ink-faint);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.link:hover {
  color: var(--color-ink);
}

.rate {
  position: relative;
  width: 148px;
  height: 148px;
  margin: 0 auto;
}

.rate svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.track,
.bar {
  fill: none;
  stroke-width: 11;
}

.track {
  stroke: var(--color-paper-hover);
}

.bar {
  stroke-linecap: round;
  transition: stroke-dashoffset 1.1s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 2px;
}

.center b {
  font-size: 30px;
  font-weight: 500;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.center span {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.nums {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin: 0;
}

.nums div {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 10px 4px;
  border-radius: 16px;
  background: var(--color-paper-sunken);
}

.nums dt {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.nums dd {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  margin: 0;
  font-size: 19px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.nums .hot {
  background: var(--color-gold-tint);
}

.nums .hot dd {
  color: var(--color-gold-ink);
}

@media (prefers-reduced-motion: reduce) {
  .bar {
    transition: none;
  }
}
</style>
