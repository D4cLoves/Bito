<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { PhCoins, PhCrown, PhHeartBreak, PhCards } from '@phosphor-icons/vue'
import { useCountUp } from '../../composables/useCountUp'
import { plural } from '../../composables/useLanding'
import { winRate } from '../../composables/usePlayerRank'
import type { Profile } from '../../types/user'

const props = defineProps<{ profile: Profile | null }>()

const RING = 2 * Math.PI * 30

const s = computed(() => props.profile?.stats)
const wins = useCountUp(computed(() => s.value?.wins))
const losses = useCountUp(computed(() => s.value?.losses))
const games = useCountUp(computed(() => s.value?.totalGames))
const balance = useCountUp(computed(() => props.profile?.wallet.balance), 1300)
const rate = computed(() => (s.value ? winRate(s.value.wins, s.value.losses) : null))

const ringReady = ref(false)
onMounted(() => window.setTimeout(() => (ringReady.value = true), 250))
</script>

<template>
  <div class="tiles">
    <div class="tile rate rise" style="--d: 2">
      <svg viewBox="0 0 72 72" class="ring-svg" aria-hidden="true">
        <circle cx="36" cy="36" r="30" class="track" />
        <circle
          cx="36"
          cy="36"
          r="30"
          class="bar"
          :stroke-dasharray="RING"
          :style="{ strokeDashoffset: ringReady ? RING * (1 - (rate ?? 0) / 100) : RING }"
        />
      </svg>
      <div>
        <span class="value">{{ rate === null ? '—' : `${rate}%` }}</span>
        <span class="label">побед в партиях</span>
      </div>
    </div>

    <div class="tile rise" style="--d: 3">
      <PhCrown :size="22" weight="duotone" class="icon gold" aria-hidden="true" />
      <span class="value">{{ wins }}</span>
      <span class="label">{{ plural(wins, ['победа', 'победы', 'побед']) }}</span>
    </div>

    <div class="tile rise" style="--d: 4">
      <PhHeartBreak :size="22" weight="duotone" class="icon red" aria-hidden="true" />
      <span class="value">{{ losses }}</span>
      <span class="label">{{ plural(losses, ['поражение', 'поражения', 'поражений']) }}</span>
    </div>

    <div class="tile rise" style="--d: 5">
      <PhCards :size="22" weight="duotone" class="icon" aria-hidden="true" />
      <span class="value">{{ games }}</span>
      <span class="label">{{ plural(games, ['партия', 'партии', 'партий']) }} сыграно</span>
    </div>

    <div class="tile wallet rise" style="--d: 6">
      <PhCoins :size="24" weight="duotone" class="icon" aria-hidden="true" />
      <div>
        <span class="value">{{ balance.toLocaleString('ru-RU') }}</span>
        <span class="label">{{ plural(balance, ['фишка', 'фишки', 'фишек']) }} на счету</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tiles {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.tile {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 18px;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: 0 0 0 1px rgba(214, 208, 196, 0.9);
}

.rate,
.wallet {
  flex-direction: row;
  align-items: center;
  gap: 16px;
}

.rate {
  grid-column: span 2;
}

.wallet {
  grid-column: span 3;
  justify-content: flex-start;
  background: var(--color-gold-tint);
  box-shadow: none;
  color: var(--color-gold-ink);
}

.wallet .label {
  color: var(--color-gold-ink);
  opacity: 0.85;
}

.rate div,
.wallet div {
  display: flex;
  flex-direction: column;
}

.icon {
  margin-bottom: 8px;
  color: var(--color-ink-muted);
}

.wallet .icon {
  margin: 0;
  color: var(--color-gold-ink);
}

.icon.gold {
  color: var(--color-gold);
}

.icon.red {
  color: var(--color-red);
}

.value {
  font-size: 32px;
  font-weight: 500;
  letter-spacing: -0.03em;
  line-height: 1.1;
  font-variant-numeric: tabular-nums;
}

.label {
  font-size: 14px;
  color: var(--color-ink-faint);
}

.ring-svg {
  flex: none;
  width: 72px;
  height: 72px;
  transform: rotate(-90deg);
}

.track,
.bar {
  fill: none;
  stroke-width: 7;
}

.track {
  stroke: var(--color-paper-sunken);
}

.bar {
  stroke: var(--color-ink-soft);
  stroke-linecap: round;
  transition: stroke-dashoffset 1.2s cubic-bezier(0.2, 0.8, 0.2, 1);
}

@media (max-width: 560px) {
  .tiles {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .rate {
    grid-column: span 2;
  }

  .wallet {
    grid-column: span 1;
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }
}
</style>
