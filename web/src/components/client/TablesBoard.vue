<script setup lang="ts">
import { computed, ref } from 'vue'
import { motion, AnimatePresence } from 'motion-v'
import { PhArrowsClockwise, PhPlus } from '@phosphor-icons/vue'
import TableTile from './TableTile.vue'
import { MIN_STAKE } from '../../constants/economy'
import { plural } from '../../composables/useLanding'
import type { GameMode, TableInfo } from '../../types/game'

type ModeFilter = 'all' | GameMode

const props = defineProps<{
  tables: TableInfo[]
  loaded: boolean
  refreshing: boolean
  error: string
  myId?: string
  now: number
  stake: number
  balance: number | null
}>()

defineEmits<{ (e: 'join', table: TableInfo): void; (e: 'refresh'): void; (e: 'create'): void }>()

const MODES: { id: ModeFilter; label: string }[] = [
  { id: 'all', label: 'Все' },
  { id: 'podkidnoy', label: 'Подкидной' },
  { id: 'perevodnoy', label: 'Переводной' },
]

const mode = ref<ModeFilter>('all')
const onlyMyStake = ref(false)

const stakeOf = (t: TableInfo) => t.stake ?? MIN_STAKE
const canAfford = (t: TableInfo) => props.balance === null || stakeOf(t) <= props.balance

const visible = computed(() =>
  props.tables.filter((t) => (mode.value === 'all' || t.mode === mode.value) && (!onlyMyStake.value || stakeOf(t) === props.stake)),
)

const spring = { type: 'spring', stiffness: 380, damping: 30 } as const
</script>

<template>
  <section class="board" aria-labelledby="board-title">
    <div class="head">
      <div class="title">
        <h2 id="board-title">Открытые столы</h2>
        <span class="count">{{ tables.length }} {{ plural(tables.length, ['стол', 'стола', 'столов']) }}</span>
      </div>

      <div class="tools">
        <div class="seg" role="radiogroup" aria-label="Режим">
          <button
            v-for="m in MODES"
            :key="m.id"
            type="button"
            role="radio"
            :aria-checked="mode === m.id"
            :class="{ on: mode === m.id }"
            @click="mode = m.id"
          >
            <motion.span v-if="mode === m.id" layout-id="board-mode" class="pill" :transition="spring" />
            <span class="txt">{{ m.label }}</span>
          </button>
        </div>

        <button type="button" class="toggle" :class="{ on: onlyMyStake }" :aria-pressed="onlyMyStake" @click="onlyMyStake = !onlyMyStake">
          Ставка {{ stake.toLocaleString('ru-RU') }}
        </button>

        <button
          type="button"
          class="icon-btn"
          :class="{ spin: refreshing }"
          aria-label="Обновить список"
          title="Обновить список"
          @click="$emit('refresh')"
        >
          <PhArrowsClockwise :size="18" weight="bold" />
        </button>
      </div>
    </div>

    <p v-if="error" class="error" role="alert">{{ error }}</p>

    <div v-if="!loaded" class="grid" aria-busy="true">
      <span v-for="i in 4" :key="i" class="skeleton"></span>
    </div>

    <ul v-else-if="visible.length" class="grid">
      <AnimatePresence :initial="false">
        <motion.li
          v-for="t in visible"
          :key="t.id"
          layout
          :initial="{ opacity: 0, scale: 0.92 }"
          :animate="{ opacity: 1, scale: 1 }"
          :exit="{ opacity: 0, scale: 0.92 }"
          :transition="spring"
        >
          <TableTile :table="t" :mine="t.createdBy === myId" :can-afford="canAfford(t)" :now="now" @join="$emit('join', t)" />
        </motion.li>
      </AnimatePresence>
    </ul>

    <div v-else class="empty">
      <p class="empty-title">{{ tables.length ? 'По этому фильтру никого' : 'Пока никто не ждёт' }}</p>
      <p class="empty-text">Соберите свой стол — он появится здесь, и соперники смогут подсесть.</p>
      <button type="button" class="create" @click="$emit('create')">
        <PhPlus :size="16" weight="bold" aria-hidden="true" />
        Собрать стол
      </button>
    </div>
  </section>
</template>

<style scoped>
.board {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.title {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  letter-spacing: -0.02em;
}

.count {
  font-size: 13px;
  color: var(--color-ink-faint);
}

.tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.seg {
  display: flex;
  padding: 4px;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.seg button {
  position: relative;
  height: 34px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.seg button.on {
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

.toggle {
  height: 42px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 14px;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition:
    background-color 0.2s,
    color 0.2s;
}

.toggle.on {
  background: var(--color-gold-tint);
  box-shadow: inset 0 0 0 1px var(--color-gold);
  color: var(--color-gold-ink);
}

.icon-btn {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink-muted);
  cursor: pointer;
}

.icon-btn:hover {
  color: var(--color-ink);
}

.icon-btn.spin svg {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.error {
  margin: 0;
  padding: 11px 15px;
  border-radius: 14px;
  background: var(--color-red-tint);
  color: var(--color-red-strong);
  font-size: 14px;
}

.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(210px, 1fr));
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.skeleton {
  height: 214px;
  border-radius: 22px;
  background: linear-gradient(90deg, var(--color-paper-raised) 0%, var(--color-paper-hover) 50%, var(--color-paper-raised) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
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
  padding: 40px 20px;
  border-radius: 24px;
  border: 1.5px dashed var(--color-line-stronger);
  text-align: center;
}

.empty-title {
  margin: 0;
  font-size: 17px;
}

.empty-text {
  max-width: 360px;
  margin: 6px 0 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-ink-faint);
}

.create {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  margin-top: 16px;
  padding: 0 18px;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}
</style>
