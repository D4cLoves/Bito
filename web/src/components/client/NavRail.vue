<script setup lang="ts">
import { computed } from 'vue'
import { motion } from 'motion-v'
import {
  PhCardsThree,
  PhUserCircle,
  PhWallet,
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

const ITEMS: { id: PanelId | 'home'; label: string; icon: typeof PhCardsThree }[] = [
  { id: 'home', label: 'Играть', icon: PhCardsThree },
  { id: 'profile', label: 'Профиль', icon: PhUserCircle },
  { id: 'wallet', label: 'Кошелёк', icon: PhWallet },
  { id: 'shop', label: 'Магазин', icon: PhStorefront },
  { id: 'settings', label: 'Настройки', icon: PhGearSix },
]

const OVER_LOBBY: (PanelId | null)[] = [null, 'friends', 'tables']
const current = computed(() => (OVER_LOBBY.includes(props.active) ? 'home' : props.active))

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
  height: calc(100% - 28px);
  margin: 14px 0 14px 14px;
  padding: 14px 0;
  box-sizing: border-box;
  border-radius: 30px;
  background: var(--color-paper-raised);
  box-shadow:
    inset 0 0 0 1px var(--color-line),
    0 24px 48px -36px rgba(0, 0, 0, 0.5);
}

.brand {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  margin-bottom: 14px;
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
  fill: var(--color-paper-raised);
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
  width: 64px;
  height: 60px;
  border: 0;
  border-radius: 18px;
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
  color: var(--color-gold-ink);
}

.item.small {
  width: 48px;
  height: 48px;
}

.pill {
  position: absolute;
  inset: 0;
  border-radius: 18px;
  background: var(--color-gold-tint);
}

.icon,
.label {
  position: relative;
}

.item.on .icon {
  color: var(--color-gold-ink);
}

.label {
  font-size: 11px;
  letter-spacing: 0.01em;
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
    height: auto;
    margin: 0;
    padding: 6px 4px calc(6px + env(safe-area-inset-bottom));
    border-radius: 0;
    box-shadow: inset 0 1px 0 var(--color-line);
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
    inset: 4px 6px;
  }
}
</style>
