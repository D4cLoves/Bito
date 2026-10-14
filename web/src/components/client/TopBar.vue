<script setup lang="ts">
import { computed } from 'vue'
import { PhChatsCircle, PhPlus } from '@phosphor-icons/vue'
import ChipIcon from './ChipIcon.vue'
import { useCountUp } from '../../composables/useCountUp'
import { TOP_UP_THRESHOLD } from '../../constants/economy'

const props = defineProps<{
  username: string
  balance: number | null
  online: number
  socialBadge: number
}>()

defineEmits<{ (e: 'profile'): void; (e: 'social'): void; (e: 'top-up'): void }>()

const shown = useCountUp(computed(() => props.balance ?? undefined), 900)
const canTopUp = computed(() => props.balance !== null && props.balance < TOP_UP_THRESHOLD)
const initial = computed(() => (props.username[0] ?? '?').toUpperCase())

function greetingFor(hour: number) {
  if (hour >= 5 && hour < 12) return 'Доброе утро'
  if (hour >= 12 && hour < 17) return 'Добрый день'
  if (hour >= 17 && hour < 23) return 'Добрый вечер'
  return 'Доброй ночи'
}

const greeting = greetingFor(new Date().getHours())
</script>

<template>
  <header class="top">
    <div class="where">
      <h1><span class="hello">{{ greeting }},</span> {{ username }}</h1>
      <span class="online"><i aria-hidden="true"></i>{{ online }} онлайн</span>
    </div>

    <div class="right">
      <button type="button" class="round social-btn" aria-label="Друзья и чат" title="Друзья и чат" @click="$emit('social')">
        <PhChatsCircle :size="22" />
        <span v-if="socialBadge" class="badge">{{ socialBadge }}</span>
      </button>

      <div class="wallet" :title="balance === null ? '' : `${balance.toLocaleString('ru-RU')} фишек`">
        <ChipIcon :size="22" />
        <span class="amount">{{ balance === null ? '—' : shown.toLocaleString('ru-RU') }}</span>
        <button
          type="button"
          class="top-up"
          :class="{ ready: canTopUp }"
          :aria-label="canTopUp ? 'Получить фишки' : 'Пополнение доступно, когда фишек меньше 100'"
          :title="canTopUp ? 'Получить фишки' : `Пополнение доступно, когда фишек меньше ${TOP_UP_THRESHOLD}`"
          @click="$emit('top-up')"
        >
          <PhPlus :size="14" weight="bold" />
        </button>
      </div>

      <button type="button" class="me compact-only" @click="$emit('profile')">
        <span class="avatar" aria-hidden="true">{{ initial }}</span>
        <span class="name">{{ username }}</span>
      </button>
    </div>
  </header>
</template>

<style scoped>
.top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 72px;
}

.where {
  display: flex;
  align-items: baseline;
  gap: 14px;
  min-width: 0;
}

h1 {
  margin: 0;
  overflow: hidden;
  font-family: var(--font-card);
  font-size: 30px;
  font-style: italic;
  font-weight: 400;
  letter-spacing: -0.02em;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hello {
  color: var(--color-ink-faint);
}

.online {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: 13px;
  color: var(--color-ink-faint);
  white-space: nowrap;
}

.online i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-green);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-green) 25%, transparent);
  animation: pulse 2.4s ease-in-out infinite;
}

@keyframes pulse {
  50% {
    box-shadow: 0 0 0 6px color-mix(in srgb, var(--color-green) 0%, transparent);
  }
}

.right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.round {
  position: relative;
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
  color: var(--color-ink-muted);
  cursor: pointer;
}

.round:hover {
  color: var(--color-ink);
}

.social-btn {
  display: none;
}

.badge {
  position: absolute;
  top: -2px;
  right: -2px;
  display: grid;
  place-items: center;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  box-sizing: border-box;
  border-radius: 999px;
  background: var(--color-red);
  color: #fff;
  font-size: 11px;
}

.wallet {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 5px 0 12px;
  box-sizing: border-box;
  border-radius: 999px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.amount {
  min-width: 48px;
  font-size: 16px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.top-up {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-gold) 20%, transparent);
  color: var(--color-gold-ink);
  cursor: pointer;
  opacity: 0.55;
  transition: opacity 0.2s, background-color 0.2s;
}

.top-up.ready {
  opacity: 1;
  background: var(--color-gold);
  color: var(--color-paper);
}

.top-up:hover {
  opacity: 1;
}

.me {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 44px;
  padding: 0 14px 0 5px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--color-ink);
  font: inherit;
  cursor: pointer;
  transition: background-color 0.2s;
}

.me:hover {
  background: var(--color-paper-hover);
}

.avatar {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font-size: 14px;
}

.name {
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
}

@media (min-width: 1280px) {
  .compact-only {
    display: none;
  }
}

@media (max-width: 1279px) {
  .social-btn {
    display: grid;
  }
}

@media (max-width: 767px) {
  .top {
    min-height: 60px;
  }

  h1 {
    font-size: 22px;
  }

  .online,
  .name {
    display: none;
  }

  .me {
    padding: 0 5px;
  }
}
</style>
