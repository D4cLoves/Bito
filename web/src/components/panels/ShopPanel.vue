<script setup lang="ts">
import CardBack from '../client/CardBack.vue'
import ChipIcon from '../client/ChipIcon.vue'
import type { CardBackId, FeltId, ThemeId } from '../../stores/settings'

type Item =
  | { id: string; kind: 'back'; name: string; price: number; back: CardBackId }
  | { id: string; kind: 'felt'; name: string; price: number; felt: FeltId }
  | { id: string; kind: 'theme'; name: string; price: number; theme: ThemeId }

const ITEMS: Item[] = [
  { id: 'i1', kind: 'back', name: 'Рубашка «Полночь»', price: 1500, back: 'midnight' },
  { id: 'i2', kind: 'back', name: 'Рубашка «Роща»', price: 1500, back: 'grove' },
  { id: 'i3', kind: 'felt', name: 'Сукно «Бордо»', price: 2500, felt: 'wine' },
  { id: 'i4', kind: 'felt', name: 'Сукно «Графит»', price: 2500, felt: 'graphite' },
  { id: 'i5', kind: 'theme', name: 'Тема «Лес»', price: 4000, theme: 'forest' },
  { id: 'i6', kind: 'theme', name: 'Тема «Бумага»', price: 4000, theme: 'paper' },
]

defineEmits<{ (e: 'buy'): void }>()
</script>

<template>
  <div class="shop">
    <p class="intro">Косметика за фишки: рубашки, сукно и темы. Покупка спишет фишки с баланса, предмет появится в настройках.</p>

    <ul class="grid">
      <li v-for="item in ITEMS" :key="item.id" class="item">
        <span class="preview">
          <span v-if="item.kind === 'back'" class="back"><CardBack :variant="item.back" /></span>
          <span v-else-if="item.kind === 'felt'" class="felt" :data-felt="item.felt"></span>
          <span v-else class="theme" :data-theme="item.theme"><i></i><i></i><i></i></span>
        </span>
        <span class="name">{{ item.name }}</span>
        <button type="button" class="buy" @click="$emit('buy')">
          <ChipIcon :size="16" />
          {{ item.price.toLocaleString('ru-RU') }}
        </button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.intro {
  max-width: 520px;
  margin: 0 0 20px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-ink-subtle);
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.item {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px 10px 12px;
  border-radius: 20px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.preview {
  display: grid;
  place-items: center;
  height: 130px;
  border-radius: 14px;
  background: var(--color-paper-sunken);
}

.back {
  width: 64px;
  height: 90px;
}

.felt {
  width: 80%;
  aspect-ratio: 2 / 1;
  border-radius: 999px;
  background: radial-gradient(ellipse at 50% 30%, var(--color-felt), var(--color-felt-deep));
  box-shadow: inset 0 0 0 4px rgba(0, 0, 0, 0.25);
}

.theme {
  display: flex;
  gap: 5px;
  width: 80%;
  height: 80px;
  padding: 6px;
  box-sizing: border-box;
  border-radius: 12px;
  background: var(--color-paper);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
}

.theme i {
  flex: 1;
  border-radius: 6px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
}

.theme i:nth-child(2) {
  flex: 2;
  background: var(--color-gold-tint);
  box-shadow: inset 0 0 0 1px var(--color-gold);
}

.name {
  padding: 0 4px;
  font-size: 14px;
}

.buy {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 38px;
  border: 0;
  border-radius: 999px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
}

.buy:hover {
  box-shadow: inset 0 0 0 1px var(--color-gold);
}
</style>
