<script setup lang="ts">
import { computed, ref } from 'vue'
import { motion, AnimatePresence } from 'motion-v'
import { PhPlus } from '@phosphor-icons/vue'
import TableCard from './TableCard.vue'
import PickField from './PickField.vue'
import { plural } from '../../composables/useLanding'
import { canAfford, sortTables, stakeOf, type TableSort } from '../../composables/tableList'
import type { GameMode, TableInfo } from '../../types/game'
import type { PickOption } from '../../types/ui'

type PlayersFilter = 'all' | 3 | 4 | 5 | 6
type ModeFilter = 'any' | GameMode
type StakeFilter = 'affordable' | 'all' | 100 | 250 | 500 | 1000

const props = defineProps<{ tables: TableInfo[]; loaded: boolean; myId?: string; now: number; balance: number | null }>()
defineEmits<{ (e: 'join', table: TableInfo): void; (e: 'create'): void }>()

const PLAYERS: { id: PlayersFilter; label: string }[] = [
  { id: 'all', label: 'Все' },
  { id: 3, label: '3' },
  { id: 4, label: '4' },
  { id: 5, label: '5' },
  { id: 6, label: '6' },
]

const MODES: { id: ModeFilter; label: string; suit?: string }[] = [
  { id: 'any', label: 'Любой' },
  { id: 'podkidnoy', label: 'Подкидной', suit: '♠' },
  { id: 'perevodnoy', label: 'Переводной', suit: '♥' },
]

const STAKE_FILTERS: { id: StakeFilter; label: string }[] = [
  { id: 'affordable', label: 'По карману' },
  { id: 'all', label: 'Все' },
  { id: 100, label: '100' },
  { id: 250, label: '250' },
  { id: 500, label: '500' },
  { id: 1000, label: '1K и выше' },
]

const SORTS: PickOption<TableSort>[] = [
  { value: 'full', label: 'Почти полные', hint: 'партия начнётся быстрее' },
  { value: 'new', label: 'Новые' },
  { value: 'cheap', label: 'Ставка по возрастанию' },
  { value: 'rich', label: 'Ставка по убыванию' },
]

const players = ref<PlayersFilter>('all')
const mode = ref<ModeFilter>('any')
const stake = ref<StakeFilter>('affordable')
const sort = ref<TableSort>('full')

function matchesStake(t: TableInfo) {
  if (stake.value === 'all') return true
  if (stake.value === 'affordable') return canAfford(t, props.balance)
  if (stake.value === 1000) return stakeOf(t) >= 1000
  return stakeOf(t) === stake.value
}

const visible = computed(() =>
  sortTables(
    props.tables.filter(
      (t) => (players.value === 'all' || t.maxPlayers === players.value) && (mode.value === 'any' || t.mode === mode.value) && matchesStake(t),
    ),
    sort.value,
    props.balance,
  ),
)

const filtered = computed(() => players.value !== 'all' || mode.value !== 'any' || stake.value !== 'affordable')

function reset() {
  players.value = 'all'
  mode.value = 'any'
  stake.value = 'affordable'
}

const spring = { type: 'spring', stiffness: 380, damping: 30 } as const
</script>

<template>
  <div class="all-tables">
    <div class="filters">
      <div class="filter">
        <span id="f-players" class="flabel">Игроков</span>
        <div class="seg" role="radiogroup" aria-labelledby="f-players">
          <button
            v-for="p in PLAYERS"
            :key="p.id"
            type="button"
            role="radio"
            :aria-checked="players === p.id"
            :class="{ on: players === p.id }"
            @click="players = p.id"
          >
            {{ p.label }}
          </button>
        </div>
      </div>

      <div class="filter">
        <span id="f-mode" class="flabel">Режим</span>
        <div class="seg" role="radiogroup" aria-labelledby="f-mode">
          <button
            v-for="m in MODES"
            :key="m.id"
            type="button"
            role="radio"
            :aria-checked="mode === m.id"
            :class="{ on: mode === m.id }"
            @click="mode = m.id"
          >
            <span v-if="m.suit" class="suit" :class="{ red: m.id === 'perevodnoy' }" aria-hidden="true">{{ m.suit }}</span>
            {{ m.label }}
          </button>
        </div>
      </div>

      <div class="filter">
        <span id="f-stake" class="flabel">Ставка</span>
        <div class="seg" role="radiogroup" aria-labelledby="f-stake">
          <button
            v-for="s in STAKE_FILTERS"
            :key="s.id"
            type="button"
            role="radio"
            :aria-checked="stake === s.id"
            :class="{ on: stake === s.id }"
            @click="stake = s.id"
          >
            {{ s.label }}
          </button>
        </div>
      </div>
    </div>

    <div class="bar">
      <span class="found">
        {{ loaded ? `Найдено ${visible.length} ${plural(visible.length, ['стол', 'стола', 'столов'])}` : 'Загружаем столы…' }}
      </span>
      <PickField v-model="sort" class="sort" tone="paper" label="Сначала" :options="SORTS" align="end" />
    </div>

    <div v-if="!loaded" class="grid" aria-busy="true">
      <span v-for="i in 6" :key="i" class="skeleton"></span>
    </div>

    <ul v-else-if="visible.length" class="grid">
      <AnimatePresence :initial="false">
        <motion.li
          v-for="t in visible"
          :key="t.id"
          layout
          :initial="{ opacity: 0, scale: 0.94 }"
          :animate="{ opacity: 1, scale: 1 }"
          :exit="{ opacity: 0, scale: 0.94 }"
          :transition="spring"
        >
          <TableCard :table="t" :mine="t.createdBy === myId" :can-afford="canAfford(t, balance)" :now="now" @join="$emit('join', t)" />
        </motion.li>
      </AnimatePresence>
    </ul>

    <div v-else class="empty">
      <b>{{ tables.length ? 'По этим фильтрам столов нет' : 'Пока никто не собирает компанию' }}</b>
      <p>Создайте стол с нужными настройками — он появится в списке, и соперники подсядут.</p>
      <div class="empty-actions">
        <button v-if="filtered && tables.length" type="button" class="btn ghost" @click="reset">Сбросить фильтры</button>
        <button type="button" class="btn primary" @click="$emit('create')">
          <PhPlus :size="16" weight="bold" aria-hidden="true" />
          Создать стол
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.all-tables {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.filters {
  display: flex;
  flex-wrap: wrap;
  gap: 14px 24px;
}

.filter {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.flabel {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-ink-faint);
}

.seg {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.seg button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 14px;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 0.2s,
    color 0.2s;
}

.seg button:hover:not(.on) {
  color: var(--color-ink);
}

.seg button.on {
  background: var(--color-ink-soft);
  box-shadow: none;
  color: var(--color-paper);
}

.suit {
  font-family: var(--font-card);
  font-size: 16px;
  line-height: 1;
}

.suit.red {
  color: var(--color-red);
}

.on .suit.red {
  color: var(--color-red-soft);
}

.bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-top: 14px;
  border-top: 1px solid var(--color-line);
}

.found {
  font-size: 14px;
  color: var(--color-ink-muted);
  font-variant-numeric: tabular-nums;
}

.sort {
  width: 240px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.grid li {
  display: flex;
}

.grid li > * {
  flex: 1;
}

.skeleton {
  height: 262px;
  border-radius: 24px;
  background: linear-gradient(100deg, var(--color-paper-sunken) 30%, var(--color-paper-hover) 50%, var(--color-paper-sunken) 70%);
  background-size: 200% 100%;
  animation: shimmer 1.4s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 48px 20px;
  border-radius: 24px;
  border: 1.5px dashed var(--color-line-heavy);
  text-align: center;
}

.empty b {
  font-size: 17px;
  font-weight: 500;
}

.empty p {
  max-width: 380px;
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-ink-faint);
}

.empty-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  margin-top: 10px;
}

.btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  padding: 0 18px;
  border: 0;
  border-radius: 999px;
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
}

.btn.ghost {
  background: var(--color-paper-hover);
  color: var(--color-ink);
}

.btn.primary {
  background: var(--color-gold);
  color: var(--color-on-gold);
}

@media (max-width: 560px) {
  .all-tables {
    gap: 14px;
  }

  .filters {
    gap: 12px;
  }

  .filter {
    gap: 6px;
    min-width: 0;
    max-width: 100%;
  }

  .seg {
    flex-wrap: nowrap;
    overflow-x: auto;
    margin: 0 -16px;
    padding: 0 16px;
    scrollbar-width: none;
    overscroll-behavior-x: contain;
  }

  .seg::-webkit-scrollbar {
    display: none;
  }

  .seg button {
    flex: none;
    height: 34px;
    padding: 0 13px;
  }

  .bar {
    padding-top: 12px;
  }

  .sort {
    width: 180px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton {
    animation: none;
  }
}
</style>
