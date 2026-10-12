<script setup lang="ts">
import { PhArrowBendUpRight, PhX } from '@phosphor-icons/vue'
import type { ActiveTable } from '../../types/social'

defineProps<{ table: ActiveTable }>()
defineEmits<{ (e: 'return'): void; (e: 'dismiss'): void }>()
</script>

<template>
  <div class="banner" role="status">
    <span class="pulse" aria-hidden="true"></span>
    <div class="text">
      <strong>У вас идёт партия</strong>
      <span>
        {{ table.mode === 'perevodnoy' ? 'Переводной' : 'Подкидной' }} · ставка {{ table.stake.toLocaleString('ru-RU') }} · {{ table.players }} игрока.
        Если не вернуться за 30 секунд, засчитается поражение.
      </span>
    </div>
    <button type="button" class="back" @click="$emit('return')">
      <PhArrowBendUpRight :size="16" weight="bold" aria-hidden="true" />
      Вернуться за стол
    </button>
    <button type="button" class="close" aria-label="Скрыть" @click="$emit('dismiss')">
      <PhX :size="16" />
    </button>
  </div>
</template>

<style scoped>
.banner {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 12px 12px 18px;
  border-radius: 20px;
  background: var(--color-red-tint);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-red) 35%, transparent);
}

.pulse {
  flex: none;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-red);
  animation: pulse 1.6s ease-in-out infinite;
}

@keyframes pulse {
  50% {
    box-shadow: 0 0 0 7px color-mix(in srgb, var(--color-red) 0%, transparent);
  }
  0%,
  100% {
    box-shadow: 0 0 0 0 color-mix(in srgb, var(--color-red) 45%, transparent);
  }
}

.text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

strong {
  font-size: 15px;
  font-weight: 500;
  color: var(--color-ink);
}

.text span {
  font-size: 13px;
  color: var(--color-ink-subtle);
}

.back {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--color-red);
  color: #fff;
  font: inherit;
  font-size: 14px;
  white-space: nowrap;
  cursor: pointer;
}

.close {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--color-ink-faint);
  cursor: pointer;
}

.close:hover {
  color: var(--color-ink);
}

@media (max-width: 640px) {
  .banner {
    flex-wrap: wrap;
  }

  .back {
    order: 3;
    width: 100%;
    justify-content: center;
  }
}
</style>
