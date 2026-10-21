<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { PhCheck } from '@phosphor-icons/vue'
import ChipIcon from '../client/ChipIcon.vue'
import CosmeticTile from '../cosmetics/CosmeticTile.vue'
import CosmeticFilter from '../cosmetics/CosmeticFilter.vue'
import { ALL_ITEMS, type CatalogEntry, type CosmeticFilterValue } from '../../constants/cosmetics'
import { useSettingsStore } from '../../stores/settings'
import { useGameSound } from '../../composables/useGameSound'

const props = defineProps<{ balance: number | null; buying: string | null }>()
const emit = defineEmits<{ (e: 'buy', item: CatalogEntry): void }>()

const settings = useSettingsStore()
const sound = useGameSound()

const ITEMS = ALL_ITEMS.filter((i) => i.price > 0)

const filter = ref<CosmeticFilterValue>('all')
const visible = computed(() => (filter.value === 'all' ? ITEMS : ITEMS.filter((i) => i.kind === filter.value)))
const counts = computed(() => ({
  all: ITEMS.length,
  back: ITEMS.filter((i) => i.kind === 'back').length,
  felt: ITEMS.filter((i) => i.kind === 'felt').length,
  theme: ITEMS.filter((i) => i.kind === 'theme').length,
}))

const confirming = ref<string | null>(null)
let confirmTimer: number | null = null

const keyOf = (item: CatalogEntry) => `${item.kind}:${item.id}`
const priceOf = (item: CatalogEntry) => settings.priceOf(item.kind, item.id)
const tooPoor = (item: CatalogEntry) => props.balance !== null && priceOf(item) > props.balance

function wear(item: CatalogEntry) {
  if (settings.select(item.kind, item.id)) sound.playClick()
}

function buy(item: CatalogEntry) {
  if (props.buying || tooPoor(item)) return
  if (confirmTimer) clearTimeout(confirmTimer)
  if (confirming.value !== keyOf(item)) {
    sound.playClick()
    confirming.value = keyOf(item)
    confirmTimer = window.setTimeout(() => (confirming.value = null), 3500)
    return
  }
  confirming.value = null
  emit('buy', item)
}

onUnmounted(() => {
  if (confirmTimer) clearTimeout(confirmTimer)
})
</script>

<template>
  <div class="shop">
    <div class="top">
      <CosmeticFilter v-model="filter" :counts="counts" layout-id="shop-filter" />
      <span v-if="balance !== null" class="wallet"><ChipIcon :size="18" />{{ balance.toLocaleString('ru-RU') }}</span>
    </div>

    <ul class="grid">
      <li v-for="item in visible" :key="`${item.kind}:${item.id}`">
        <CosmeticTile :kind="item.kind" :id="item.id" :name="item.name" :active="settings.isActive(item.kind, item.id)">
          <button
            v-if="!settings.isOwned(item.kind, item.id)"
            type="button"
            class="act buy"
            :class="{ confirm: confirming === keyOf(item), poor: tooPoor(item) }"
            :disabled="buying === keyOf(item) || tooPoor(item)"
            :title="tooPoor(item) ? 'Не хватает фишек' : ''"
            @click="buy(item)"
          >
            <template v-if="buying === keyOf(item)">Покупаем…</template>
            <template v-else-if="confirming === keyOf(item)">Купить за {{ priceOf(item).toLocaleString('ru-RU') }}?</template>
            <template v-else>
              <ChipIcon :size="16" />
              {{ priceOf(item).toLocaleString('ru-RU') }}
            </template>
          </button>
          <span v-else-if="settings.isActive(item.kind, item.id)" class="act worn"><PhCheck :size="14" weight="bold" />Надето</span>
          <button v-else type="button" class="act wear" @click="wear(item)">Надеть</button>
        </CosmeticTile>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.wallet {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
  height: 40px;
  padding: 0 14px 0 10px;
  border-radius: 999px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 16px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.act {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex: none;
  height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.buy {
  padding-left: 10px;
  background: var(--gradient-gold);
  color: var(--color-on-gold);
  box-shadow: 0 8px 16px -10px rgba(120, 80, 20, 0.7);
  cursor: pointer;
  transition: transform 0.2s cubic-bezier(0.22, 1, 0.36, 1);
}

.buy:hover {
  transform: translateY(-1px) scale(1.04);
}

.buy.confirm {
  padding-left: 14px;
  background: var(--color-ink);
  color: var(--color-paper);
}

.buy:disabled {
  cursor: default;
  transform: none;
}

.buy.poor {
  background: var(--color-paper-sunken);
  color: var(--color-ink-faint);
  box-shadow: none;
}

.buy :deep(.chip) {
  color: var(--color-gold-ink);
}

.wear {
  background: var(--color-ink-soft);
  color: var(--color-paper);
  cursor: pointer;
}

.worn {
  padding: 0 10px;
  color: var(--color-gold-ink);
}

@media (max-width: 560px) {
  .top {
    flex-direction: column-reverse;
    align-items: stretch;
  }

  .wallet {
    align-self: flex-end;
  }

  .grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
}
</style>
