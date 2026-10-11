<script setup lang="ts">
import { computed, ref, toRef, watch } from 'vue'
import { Coins, Settings2 } from 'lucide-vue-next'
import { useCountUp } from '../../composables/useCountUp'
import { plural } from '../../composables/useLanding'
import type { Profile } from '../../types/user'

const props = defineProps<{ profile: Profile | null; error?: string }>()
defineEmits<{ (e: 'open-settings'): void; (e: 'retry'): void }>()

const RING = 2 * Math.PI * 44

const stats = computed(() => props.profile?.stats)
const elo = useCountUp(computed(() => stats.value?.elo))
const balance = useCountUp(computed(() => props.profile?.wallet.balance), 1100)
const games = toRef(() => stats.value?.totalGames ?? 0)

const winRate = computed(() => {
  const s = stats.value
  if (!s || s.wins + s.losses === 0) return null
  return Math.round((s.wins / (s.wins + s.losses)) * 100)
})

const ringReady = ref(false)
watch(
  () => props.profile,
  (p) => {
    if (p && !ringReady.value) requestAnimationFrame(() => requestAnimationFrame(() => (ringReady.value = true)))
  },
  { immediate: true },
)
const ringOffset = computed(() => (ringReady.value ? RING * (1 - (winRate.value ?? 0) / 100) : RING))
const initial = computed(() => (props.profile?.username?.[0] ?? '?').toUpperCase())

const MONTHS = ['января', 'февраля', 'марта', 'апреля', 'мая', 'июня', 'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря']
const since = computed(() => {
  if (!props.profile?.createdAt) return ''
  const d = new Date(props.profile.createdAt)
  return `${MONTHS[d.getMonth()]} ${d.getFullYear()}`
})

const rank = computed(() => {
  const e = stats.value?.elo ?? 0
  if (e >= 1600) return 'Мастер стола'
  if (e >= 1300) return 'Опытный'
  if (e >= 1100) return 'Игрок'
  return 'Новичок'
})
</script>

<template>
  <section class="profile-card" aria-label="Ваш профиль">
    <div v-if="!profile && !error" class="skeleton-wrap" aria-busy="true">
      <span class="sk sk-avatar"></span>
      <span class="sk sk-line"></span>
      <span class="sk sk-line short"></span>
      <span class="sk sk-block"></span>
    </div>

    <div v-else-if="!profile" class="failed">
      <p>{{ error }}</p>
      <button type="button" class="ghost" @click="$emit('retry')">Повторить</button>
    </div>

    <template v-else>
      <div class="head">
        <span class="avatar" aria-hidden="true">{{ initial }}</span>
        <div class="who">
          <h2 class="name">{{ profile.username }}</h2>
          <p class="meta">{{ rank }} · в Bito с {{ since }}</p>
        </div>
        <button type="button" class="icon-btn" aria-label="Настройки профиля" title="Настройки профиля" @click="$emit('open-settings')">
          <Settings2 :size="18" />
        </button>
      </div>

      <div class="rating">
        <div class="ring-wrap">
          <svg viewBox="0 0 100 100" class="ring-svg" aria-hidden="true">
            <circle cx="50" cy="50" r="44" class="track" />
            <circle
              cx="50"
              cy="50"
              r="44"
              class="bar"
              :stroke-dasharray="RING"
              :stroke-dashoffset="ringOffset"
            />
          </svg>
          <div class="ring-center">
            <span class="elo">{{ elo }}</span>
            <span class="elo-label">ELO</span>
          </div>
        </div>

        <dl class="numbers">
          <div>
            <dt>Побед</dt>
            <dd>{{ stats?.wins }}</dd>
          </div>
          <div>
            <dt>Поражений</dt>
            <dd>{{ stats?.losses }}</dd>
          </div>
          <div>
            <dt>Винрейт</dt>
            <dd>{{ winRate === null ? '—' : `${winRate}%` }}</dd>
          </div>
        </dl>
      </div>

      <p class="games">
        {{ games ? `${games} ${plural(games, ['партия', 'партии', 'партий'])} сыграно` : 'Первая партия ещё впереди' }}
      </p>

      <div class="wallet">
        <Coins :size="18" aria-hidden="true" />
        <span class="balance">{{ balance.toLocaleString('ru-RU') }}</span>
        <span class="chips">{{ plural(balance, ['фишка', 'фишки', 'фишек']) }}</span>
      </div>
    </template>
  </section>
</template>

<style scoped>
.profile-card {
  padding: 26px;
  border-radius: 28px;
  background: var(--color-paper-raised);
  box-shadow:
    0 0 0 1px rgba(214, 208, 196, 0.9),
    0 24px 48px -32px rgba(70, 62, 48, 0.35);
}

.head {
  display: flex;
  align-items: center;
  gap: 14px;
}

.avatar {
  display: grid;
  place-items: center;
  flex: none;
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font-family: var(--font-card);
  font-size: 24px;
}

.who {
  flex: 1;
  min-width: 0;
}

.name {
  margin: 0;
  font-size: 22px;
  font-weight: 500;
  letter-spacing: -0.02em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--color-ink-faint);
}

.icon-btn {
  display: grid;
  place-items: center;
  flex: none;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  color: var(--color-ink);
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.icon-btn:hover {
  background: var(--color-paper-hover);
  transform: rotate(45deg);
}

.rating {
  display: flex;
  align-items: center;
  gap: 22px;
  margin-top: 26px;
}

.ring-wrap {
  position: relative;
  flex: none;
  width: 124px;
  height: 124px;
}

.ring-svg {
  width: 100%;
  height: 100%;
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
  transition: stroke-dashoffset 1.1s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.ring-center {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}

.elo {
  font-size: 30px;
  font-weight: 500;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}

.elo-label {
  margin-top: 2px;
  font-size: 11px;
  letter-spacing: 0.16em;
  color: var(--color-ink-faint);
}

.numbers {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 0;
}

.numbers div {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--color-line);
}

.numbers div:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

dt {
  font-size: 14px;
  color: var(--color-ink-muted);
}

dd {
  margin: 0;
  font-size: 17px;
  font-variant-numeric: tabular-nums;
}

.games {
  margin: 18px 0 0;
  font-size: 14px;
  color: var(--color-ink-subtle);
}

.wallet {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 18px;
  padding: 14px 16px;
  border-radius: 18px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.balance {
  font-size: 20px;
  font-weight: 500;
  letter-spacing: -0.01em;
  font-variant-numeric: tabular-nums;
}

.chips {
  font-size: 14px;
}

.skeleton-wrap {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sk {
  display: block;
  border-radius: 14px;
  background: linear-gradient(90deg, var(--color-paper-sunken) 0%, var(--color-paper-raised) 50%, var(--color-paper-sunken) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

.sk-avatar {
  width: 56px;
  height: 56px;
  border-radius: 50%;
}

.sk-line {
  width: 70%;
  height: 18px;
}

.sk-line.short {
  width: 45%;
}

.sk-block {
  height: 160px;
  margin-top: 8px;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

.failed {
  text-align: center;
  color: var(--color-ink-muted);
}

.ghost {
  height: 40px;
  padding: 0 18px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  color: var(--color-ink);
  font: inherit;
  cursor: pointer;
}

@media (prefers-reduced-motion: reduce) {
  .bar,
  .icon-btn {
    transition: none;
  }
}
</style>
