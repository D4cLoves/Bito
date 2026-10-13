<script setup lang="ts">
import { motion } from 'motion-v'
import type { CosmeticFilterValue } from '../../constants/cosmetics'

const model = defineModel<CosmeticFilterValue>({ required: true })
const props = defineProps<{ counts?: Partial<Record<CosmeticFilterValue, number>>; layoutId: string }>()

const OPTIONS: { id: CosmeticFilterValue; label: string }[] = [
  { id: 'all', label: 'Всё' },
  { id: 'back', label: 'Рубашки' },
  { id: 'felt', label: 'Сукно' },
  { id: 'theme', label: 'Темы' },
]

const spring = { type: 'spring', stiffness: 380, damping: 30 } as const
</script>

<template>
  <div class="filter" role="radiogroup" aria-label="Раздел">
    <button
      v-for="o in OPTIONS"
      :key="o.id"
      type="button"
      role="radio"
      :aria-checked="model === o.id"
      :class="{ on: model === o.id }"
      @click="model = o.id"
    >
      <motion.span v-if="model === o.id" :layout-id="props.layoutId" class="pill" :transition="spring" />
      <span class="txt">{{ o.label }}</span>
      <span v-if="props.counts?.[o.id] !== undefined" class="count">{{ props.counts[o.id] }}</span>
    </button>
  </div>
</template>

<style scoped>
.filter {
  display: inline-flex;
  max-width: 100%;
  overflow-x: auto;
  padding: 4px;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  scrollbar-width: none;
}

button {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
  height: 36px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

button.on {
  color: var(--color-paper);
}

.pill {
  position: absolute;
  inset: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
}

.txt,
.count {
  position: relative;
}

.count {
  font-size: 12px;
  opacity: 0.6;
  font-variant-numeric: tabular-nums;
}
</style>
