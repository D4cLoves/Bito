<script setup lang="ts">
import { computed, ref } from 'vue'
import { motion, AnimatePresence } from 'motion-v'
import { PhArrowsClockwise, PhPlus } from '@phosphor-icons/vue'
import PaperCard from '../landing/PaperCard.vue'
import { plural } from '../../composables/useLanding'
import type { GameMode, TableInfo } from '../../types/game'

type Filter = 'all' | GameMode

const props = defineProps<{
  tables: TableInfo[]
  loaded: boolean
  refreshing: boolean
  error: string
  myId?: string
  now: number
}>()

defineEmits<{ (e: 'join', table: TableInfo): void; (e: 'refresh'): void; (e: 'create'): void }>()

const FILTERS: { id: Filter; label: string }[] = [
  { id: 'all', label: 'Все' },
  { id: 'podkidnoy', label: 'Подкидной' },
  { id: 'perevodnoy', label: 'Переводной' },
]

const filter = ref<Filter>('all')
const visible = computed(() => (filter.value === 'all' ? props.tables : props.tables.filter((t) => t.mode === filter.value)))
const countFor = (f: Filter) => (f === 'all' ? props.tables.length : props.tables.filter((t) => t.mode === f).length)

const spring = { type: 'spring', stiffness: 320, damping: 26 } as const

function fan(i: number, n: number) {
  const mid = (n - 1) / 2
  return { rotate: n > 1 ? (i - mid) * 2.5 : 0, y: Math.abs(i - mid) * 4 }
}

const suit = (m: GameMode) => (m === 'perevodnoy' ? '♥' : '♠')
const modeName = (m: GameMode) => (m === 'perevodnoy' ? 'Переводной' : 'Подкидной')

function ago(iso: string) {
  const s = Math.max(0, Math.round((props.now - new Date(iso).getTime()) / 1000))
  if (s < 60) return 'только что'
  const m = Math.round(s / 60)
  return `${m} мин назад`
}
</script>

<template>
  <section class="waiting" aria-labelledby="waiting-title">
    <div class="head">
      <div class="title">
        <h2 id="waiting-title">Ждут соперника</h2>
        <span class="live"><i aria-hidden="true"></i>{{ tables.length }} {{ plural(tables.length, ['стол', 'стола', 'столов']) }}</span>
      </div>

      <div class="tools">
        <div class="filters" role="tablist" aria-label="Фильтр по режиму">
          <button
            v-for="f in FILTERS"
            :key="f.id"
            type="button"
            role="tab"
            :aria-selected="filter === f.id"
            :class="{ on: filter === f.id }"
            @click="filter = f.id"
          >
            <motion.span v-if="filter === f.id" layout-id="filter-pill" class="pill" :transition="spring" />
            <span class="txt">{{ f.label }}</span>
            <span class="count">{{ countFor(f.id) }}</span>
          </button>
        </div>
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

    <div v-if="!loaded" class="hand" aria-busy="true">
      <span v-for="i in 4" :key="i" class="table-card skeleton" :style="{ rotate: `${(i - 2.5) * 2.5}deg` }"></span>
    </div>

    <ul v-else-if="visible.length" class="hand rise">
      <AnimatePresence :initial="false">
        <motion.li
          v-for="(t, i) in visible"
          :key="t.id"
          layout
          class="slot"
          :initial="{ opacity: 0, y: 60, rotate: 0 }"
          :animate="{ opacity: 1, ...fan(i, visible.length) }"
          :exit="{ opacity: 0, y: 40, scale: 0.9 }"
          :while-hover="{ y: -18, rotate: 0, scale: 1.04, zIndex: 20 }"
          :transition="{ ...spring, delay: i * 0.04 }"
        >
          <button type="button" class="table-card" :aria-label="`${modeName(t.mode)}, ${t.deck} карт, занято ${t.players} из ${t.maxPlayers}`" @click="$emit('join', t)">
            <span class="corner tl" :class="{ red: suit(t.mode) === '♥' }">
              <b>{{ t.deck }}</b><i>{{ suit(t.mode) }}</i>
            </span>
            <span class="corner br" :class="{ red: suit(t.mode) === '♥' }">
              <b>{{ t.deck }}</b><i>{{ suit(t.mode) }}</i>
            </span>
            <span v-if="t.createdBy === myId" class="mine">ваш</span>

            <span class="pips" :data-n="t.maxPlayers">
              <i v-for="s in t.maxPlayers" :key="s" :class="{ taken: s <= t.players }"></i>
            </span>

            <span class="mode">{{ modeName(t.mode) }}</span>
            <span class="meta">{{ t.players }}/{{ t.maxPlayers }} · {{ ago(t.createdAt) }}</span>
            <span class="cta">{{ t.createdBy === myId ? 'Вернуться' : 'Сесть' }}</span>
          </button>
        </motion.li>
      </AnimatePresence>
    </ul>

    <div v-else class="empty">
      <div class="empty-card" aria-hidden="true">
        <PaperCard />
      </div>
      <div>
        <p class="empty-title">{{ tables.length ? 'В этом режиме пока никого' : 'Пока никто не ждёт' }}</p>
        <p class="empty-text">Соберите свой стол — он появится здесь, и соперники смогут подсесть.</p>
        <button type="button" class="ghost" @click="$emit('create')">
          <PhPlus :size="16" weight="bold" aria-hidden="true" />
          Собрать стол
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
}

.title {
  display: flex;
  align-items: baseline;
  gap: 14px;
}

h2 {
  margin: 0;
  font-size: 30px;
  font-weight: 500;
  letter-spacing: -0.03em;
}

.live {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-ink-faint);
}

.live i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-green);
  box-shadow: 0 0 0 4px rgba(111, 154, 106, 0.18);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  50% {
    box-shadow: 0 0 0 7px rgba(111, 154, 106, 0);
  }
}

.tools {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;
}

.filters {
  display: flex;
  min-width: 0;
  padding: 4px;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: 0 0 0 1px rgba(214, 208, 196, 0.9);
  overflow-x: auto;
}

.filters button {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 38px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 14px;
  white-space: nowrap;
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

.txt,
.count {
  position: relative;
}

.count {
  min-width: 18px;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  color: var(--color-ink-muted);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
  text-align: center;
}

.filters button.on .count {
  background: rgba(245, 244, 239, 0.18);
  color: var(--color-paper);
}

.icon-btn {
  display: grid;
  place-items: center;
  flex: none;
  width: 46px;
  height: 46px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: 0 0 0 1px rgba(214, 208, 196, 0.9);
  color: var(--color-ink);
  cursor: pointer;
}

.icon-btn:hover {
  background: var(--color-paper-sunken);
}

.icon-btn.spin svg {
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.error {
  margin: 16px 0 0;
  padding: 10px 14px;
  border-radius: 14px;
  background: var(--color-red-tint);
  color: var(--color-red-strong);
  font-size: 14px;
}

.hand {
  display: flex;
  gap: 18px;
  margin: 0;
  padding: 40px 8px 30px;
  list-style: none;
  overflow-x: auto;
  scroll-snap-type: x proximity;
}

.slot {
  position: relative;
  flex: none;
  scroll-snap-align: start;
}

.table-card {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 172px;
  height: 244px;
  padding: 0;
  box-sizing: border-box;
  border: 0;
  border-radius: 16px;
  background: linear-gradient(160deg, var(--color-paper-raised), #f2efe7);
  box-shadow:
    0 1px 0 rgba(255, 255, 255, 0.9) inset,
    0 0 0 1px rgba(160, 150, 132, 0.35),
    0 22px 34px -22px rgba(70, 62, 48, 0.55);
  color: var(--color-card-ink);
  font: inherit;
  text-align: center;
  cursor: pointer;
}

.table-card::before {
  content: '';
  position: absolute;
  inset: 8px;
  border: 1px solid rgba(160, 150, 132, 0.25);
  border-radius: 10px;
  pointer-events: none;
}

.table-card.skeleton {
  flex: none;
  background: linear-gradient(90deg, var(--color-paper-sunken) 0%, var(--color-paper-raised) 50%, var(--color-paper-sunken) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

.corner {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  font-family: var(--font-card);
  line-height: 1;
}

.corner b {
  font-size: 19px;
  font-weight: 400;
}

.corner i {
  margin-top: 2px;
  font-size: 15px;
  font-style: normal;
}

.corner.red {
  color: var(--color-red);
}

.tl {
  left: 14px;
  top: 13px;
}

.br {
  right: 14px;
  bottom: 13px;
  transform: rotate(180deg);
}

.mine {
  position: absolute;
  right: 12px;
  top: 12px;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
  font-size: 11px;
}

.pips {
  display: grid;
  grid-template-columns: repeat(2, 16px);
  gap: 10px 18px;
  margin-top: 56px;
}

.pips[data-n='2'] {
  grid-template-columns: repeat(2, 16px);
}

.pips[data-n='3'],
.pips[data-n='5'],
.pips[data-n='6'] {
  grid-template-columns: repeat(3, 16px);
  gap: 10px 12px;
}

.pips i {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  box-shadow: inset 0 0 0 1.5px var(--color-line-heavy);
}

.pips i.taken {
  background: var(--color-ink-soft);
  box-shadow: none;
}

.mode {
  margin-top: auto;
  font-family: var(--font-card);
  font-size: 19px;
  font-style: italic;
}

.meta {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-ink-faint);
  font-variant-numeric: tabular-nums;
}

.cta {
  margin: 12px 0 34px;
  padding: 7px 18px;
  border-radius: 999px;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font-size: 13px;
  transition: background-color 0.2s;
}

.table-card:hover .cta {
  background: var(--color-ink);
}

.table-card:focus-visible {
  outline: 2px solid var(--color-ink-soft);
  outline-offset: 4px;
}

.empty {
  display: flex;
  align-items: center;
  gap: 28px;
  margin-top: 24px;
  padding: 28px;
  border-radius: 28px;
  background: rgba(250, 249, 245, 0.7);
  box-shadow: 0 0 0 1px rgba(214, 208, 196, 0.9);
}

.empty-card {
  flex: none;
  width: 96px;
  height: 136px;
  transform: rotateY(180deg) rotate(-6deg);
  transform-style: preserve-3d;
}

.empty-title {
  margin: 0;
  font-size: 19px;
}

.empty-text {
  max-width: 380px;
  margin: 6px 0 0;
  font-size: 15px;
  line-height: 1.5;
  color: var(--color-ink-subtle);
}

.ghost {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  margin-top: 14px;
  padding: 0 18px;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font: inherit;
  font-size: 15px;
  cursor: pointer;
}

.ghost:hover {
  background: var(--color-ink);
}

@media (max-width: 560px) {
  h2 {
    font-size: 24px;
  }

  .table-card {
    width: 150px;
    height: 214px;
  }

  .empty {
    flex-direction: column;
    text-align: center;
  }
}

@media (prefers-reduced-motion: reduce) {
  .live i {
    animation: none;
  }
}
</style>
