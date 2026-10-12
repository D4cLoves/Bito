<script setup lang="ts">
import { computed } from 'vue'
import { motion } from 'motion-v'
import {
  PhCardsThree,
  PhUserCircle,
  PhTrophy,
  PhStorefront,
  PhGearSix,
  PhSpeakerHigh,
  PhSpeakerSlash,
  PhSignOut,
} from '@phosphor-icons/vue'
import type { PanelId } from '../../composables/usePanel'

const props = defineProps<{ active: PanelId | null; soundOn: boolean }>()
const emit = defineEmits<{
  (e: 'home'): void
  (e: 'open', id: PanelId): void
  (e: 'toggle-sound'): void
  (e: 'logout'): void
}>()

const ITEMS: { id: PanelId | 'home'; label: string; icon: typeof PhCardsThree; soon?: boolean }[] = [
  { id: 'home', label: 'Играть', icon: PhCardsThree },
  { id: 'profile', label: 'Профиль', icon: PhUserCircle },
  { id: 'leaderboard', label: 'Рейтинг', icon: PhTrophy, soon: true },
  { id: 'shop', label: 'Магазин', icon: PhStorefront, soon: true },
  { id: 'settings', label: 'Настройки', icon: PhGearSix },
]

const current = computed(() => (props.active && props.active !== 'social' ? props.active : 'home'))

function pick(id: PanelId | 'home') {
  if (id === 'home') emit('home')
  else emit('open', id)
}

const spring = { type: 'spring', stiffness: 420, damping: 34 } as const
</script>

<template>
  <nav class="rail" aria-label="Главное меню">
    <button type="button" class="brand" aria-label="Bito — лобби" @click="emit('home')">
      <svg viewBox="0 0 40 38" aria-hidden="true">
        <rect x="17.5" y="4.5" width="18" height="26" rx="3.2" transform="rotate(12 26.5 17.5)" fill="none" stroke="currentColor" stroke-width="2" />
        <rect x="4" y="5.5" width="18" height="26" rx="3.2" transform="rotate(-14 13 18.5)" class="brand-front" stroke="currentColor" stroke-width="2" />
        <path
          transform="rotate(-14 13 18.5)"
          d="M13 13.2c-1.9 2.3-4.1 3.9-4.1 6 0 1.4 1 2.3 2.2 2.3.8 0 1.4-.4 1.6-.9-.1 1-.5 1.9-1.2 2.6h3c-.7-.7-1.1-1.6-1.2-2.6.2.5.8.9 1.6.9 1.2 0 2.2-.9 2.2-2.3 0-2.1-2.2-3.7-4.1-6z"
          fill="currentColor"
        />
      </svg>
    </button>

    <ul class="items">
      <li v-for="item in ITEMS" :key="item.id">
        <button
          type="button"
          class="item"
          :class="{ on: current === item.id }"
          :aria-current="current === item.id ? 'page' : undefined"
          @click="pick(item.id)"
        >
          <motion.span v-if="current === item.id" layout-id="rail-pill" class="pill" :transition="spring" />
          <component :is="item.icon" :size="24" :weight="current === item.id ? 'fill' : 'regular'" class="icon" aria-hidden="true" />
          <span class="label">{{ item.label }}</span>
          <span v-if="item.soon" class="soon" aria-label="скоро"></span>
        </button>
      </li>
    </ul>

    <div class="bottom">
      <button
        type="button"
        class="item small"
        :aria-label="soundOn ? 'Выключить звук' : 'Включить звук'"
        :title="soundOn ? 'Выключить звук' : 'Включить звук'"
        @click="emit('toggle-sound')"
      >
        <PhSpeakerHigh v-if="soundOn" :size="20" class="icon" />
        <PhSpeakerSlash v-else :size="20" class="icon" />
      </button>
      <button type="button" class="item small" aria-label="Выйти" title="Выйти" @click="emit('logout')">
        <PhSignOut :size="20" class="icon" />
      </button>
    </div>
  </nav>
</template>

<style scoped>
.rail {
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  padding: 16px 0;
  box-sizing: border-box;
  border-right: 1px solid var(--color-line);
  background: var(--color-paper-sunken);
}

.brand {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  margin-bottom: 18px;
  border: 0;
  border-radius: 16px;
  background: transparent;
  color: var(--color-gold);
  cursor: pointer;
}

.brand svg {
  width: 34px;
  height: 32px;
}

.brand-front {
  fill: var(--color-paper-sunken);
}

.items {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.item {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: 68px;
  height: 62px;
  border: 0;
  border-radius: 16px;
  background: transparent;
  color: var(--color-ink-faint);
  font: inherit;
  cursor: pointer;
  transition: color 0.2s;
}

.item:hover {
  color: var(--color-ink);
}

.item.on {
  color: var(--color-ink);
}

.item.small {
  width: 48px;
  height: 48px;
}

.pill {
  position: absolute;
  inset: 0;
  border-radius: 16px;
  background: var(--color-paper-hover);
  box-shadow: inset 3px 0 0 var(--color-gold);
}

.icon,
.label {
  position: relative;
}

.item.on .icon {
  color: var(--color-gold);
}

.label {
  font-size: 11px;
  letter-spacing: 0.01em;
}

.soon {
  position: absolute;
  top: 10px;
  right: 16px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-ink-ghost);
}

.bottom {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  margin-top: auto;
}

@media (max-width: 767px) {
  .rail {
    flex-direction: row;
    justify-content: space-around;
    padding: 6px 4px calc(6px + env(safe-area-inset-bottom));
    border-right: 0;
    border-top: 1px solid var(--color-line);
  }

  .brand,
  .bottom {
    display: none;
  }

  .items {
    flex-direction: row;
    justify-content: space-around;
    width: 100%;
    gap: 0;
  }

  .items li {
    flex: 1;
  }

  .item {
    width: 100%;
    height: 56px;
  }

  .pill {
    box-shadow: inset 0 -3px 0 var(--color-gold);
  }
}
</style>
