<script setup lang="ts">
import { computed, ref } from 'vue'
import { motion } from 'motion-v'
import { PhCheck } from '@phosphor-icons/vue'
import CardBack from '../client/CardBack.vue'
import ChipIcon from '../client/ChipIcon.vue'
import { CARD_BACKS, FELTS, THEMES, type CardBackId, type CosmeticKind, type FeltId, type ThemeId } from '../../constants/cosmetics'
import { useSettingsStore } from '../../stores/settings'

type Filter = 'all' | CosmeticKind

interface ShopItem {
  kind: CosmeticKind
  id: string
  name: string
  price: number
  note?: string
}

defineProps<{ balance: number | null }>()
const emit = defineEmits<{ (e: 'buy', item: ShopItem): void }>()

const settings = useSettingsStore()

const FILTERS: { id: Filter; label: string }[] = [
  { id: 'all', label: 'Всё' },
  { id: 'back', label: 'Рубашки' },
  { id: 'felt', label: 'Сукно' },
  { id: 'theme', label: 'Темы' },
]

const KIND_LABEL: Record<CosmeticKind, string> = { back: 'Рубашка', felt: 'Сукно', theme: 'Тема' }

const ITEMS: ShopItem[] = [
  ...CARD_BACKS.map((i) => ({ ...i, kind: 'back' as const })),
  ...FELTS.map((i) => ({ ...i, kind: 'felt' as const })),
  ...THEMES.map((i) => ({ ...i, kind: 'theme' as const })),
].filter((i) => i.price > 0)

const filter = ref<Filter>('all')
const visible = computed(() => (filter.value === 'all' ? ITEMS : ITEMS.filter((i) => i.kind === filter.value)))

function isActive(item: ShopItem) {
  if (item.kind === 'back') return settings.cardBack === item.id
  if (item.kind === 'felt') return settings.felt === item.id
  return settings.theme === item.id
}

function wear(item: ShopItem) {
  if (item.kind === 'back') settings.cardBack = item.id as CardBackId
  else if (item.kind === 'felt') settings.felt = item.id as FeltId
  else settings.theme = item.id as ThemeId
}

const spring = { type: 'spring', stiffness: 380, damping: 30 } as const
</script>

<template>
  <div class="shop">
    <div class="top">
      <p class="intro">Рубашки, сукно и темы за фишки. Купленное сразу появится в настройках.</p>
      <span v-if="balance !== null" class="wallet"><ChipIcon :size="18" />{{ balance.toLocaleString('ru-RU') }}</span>
    </div>

    <div class="filters" role="radiogroup" aria-label="Раздел магазина">
      <button
        v-for="f in FILTERS"
        :key="f.id"
        type="button"
        role="radio"
        :aria-checked="filter === f.id"
        :class="{ on: filter === f.id }"
        @click="filter = f.id"
      >
        <motion.span v-if="filter === f.id" layout-id="shop-filter" class="pill" :transition="spring" />
        <span class="txt">{{ f.label }}</span>
      </button>
    </div>

    <ul class="grid">
      <li v-for="item in visible" :key="`${item.kind}:${item.id}`" class="item" :class="{ owned: settings.isOwned(item.kind, item.id) }">
        <span class="preview">
          <span v-if="item.kind === 'back'" class="back"><CardBack :variant="item.id as CardBackId" /></span>
          <span v-else-if="item.kind === 'felt'" class="felt" :data-felt="item.id"></span>
          <span v-else class="theme" :data-theme="item.id">
            <i class="t-rail"></i>
            <span class="t-main"><i class="t-felt"></i><i class="t-row"></i></span>
          </span>
        </span>
        <span class="meta">
          <small>{{ KIND_LABEL[item.kind] }}</small>
          <b>{{ item.name }}</b>
        </span>

        <button v-if="!settings.isOwned(item.kind, item.id)" type="button" class="buy" @click="emit('buy', item)">
          <ChipIcon :size="16" />
          {{ item.price.toLocaleString('ru-RU') }}
        </button>
        <button v-else-if="isActive(item)" type="button" class="buy worn" disabled>
          <PhCheck :size="15" weight="bold" />
          Надето
        </button>
        <button v-else type="button" class="buy wear" @click="wear(item)">Надеть</button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.top {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}

.intro {
  max-width: 460px;
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-ink-subtle);
}

.wallet {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
  height: 36px;
  padding: 0 14px 0 9px;
  border-radius: 999px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.filters {
  display: inline-flex;
  margin-bottom: 18px;
  padding: 4px;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.filters button {
  position: relative;
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

.filters button.on {
  color: var(--color-paper);
}

.pill {
  position: absolute;
  inset: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
}

.txt {
  position: relative;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
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
  transition:
    transform 0.3s cubic-bezier(0.22, 1, 0.36, 1),
    box-shadow 0.3s;
}

.item:hover {
  transform: translateY(-3px);
  box-shadow:
    inset 0 0 0 1px var(--color-line-stronger),
    0 18px 30px -22px rgba(0, 0, 0, 0.5);
}

.preview {
  display: grid;
  place-items: center;
  height: 140px;
  border-radius: 14px;
  background: var(--color-paper-sunken);
  overflow: hidden;
}

.back {
  width: 70px;
  height: 100px;
  transition: transform 0.35s cubic-bezier(0.22, 1, 0.36, 1);
}

.item:hover .back {
  transform: rotate(-6deg) translateY(-4px);
}

.felt {
  width: 82%;
  aspect-ratio: 2 / 1;
  border-radius: 999px;
  background: radial-gradient(ellipse at 50% 30%, var(--color-felt), var(--color-felt-deep));
  box-shadow:
    inset 0 0 0 4px var(--color-felt-well),
    inset 0 0 0 5px var(--color-felt-line),
    0 10px 20px -12px rgba(0, 0, 0, 0.5);
}

.theme {
  display: flex;
  gap: 5px;
  width: 82%;
  height: 92px;
  padding: 6px;
  box-sizing: border-box;
  border-radius: 12px;
  background: var(--color-paper);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
}

.t-rail {
  width: 12px;
  border-radius: 4px;
  background: var(--color-paper-sunken);
}

.t-main {
  display: flex;
  flex-direction: column;
  gap: 5px;
  flex: 1;
}

.t-felt {
  flex: 1;
  border-radius: 6px;
  background: var(--color-gold-tint);
  box-shadow: inset 0 0 0 1px var(--color-gold);
}

.t-row {
  height: 18px;
  border-radius: 4px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
}

.meta {
  display: flex;
  flex-direction: column;
  padding: 0 4px;
}

.meta small {
  font-size: 11px;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-ink-faint);
}

.meta b {
  font-size: 15px;
  font-weight: 500;
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

.buy:hover:not(:disabled) {
  box-shadow: inset 0 0 0 1px var(--color-gold);
}

.buy.wear {
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

.buy.worn {
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  cursor: default;
}
</style>
