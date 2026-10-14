<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { PhArrowRight, PhCaretLeft, PhCaretRight, PhPlus } from '@phosphor-icons/vue'
import TableCard from './TableCard.vue'
import { plural } from '../../composables/useLanding'
import { canAfford, sortTables } from '../../composables/tableList'
import type { TableInfo } from '../../types/game'

const MAX_CARDS = 12

const props = defineProps<{
  tables: TableInfo[]
  loaded: boolean
  error: string
  myId?: string
  now: number
  balance: number | null
}>()

defineEmits<{ (e: 'join', table: TableInfo): void; (e: 'create'): void; (e: 'all'): void }>()

const track = ref<HTMLElement | null>(null)
const atStart = ref(true)
const atEnd = ref(true)

const shown = computed(() => sortTables(props.tables, 'full', props.balance).slice(0, MAX_CARDS))

function update() {
  const el = track.value
  if (!el) return
  atStart.value = el.scrollLeft <= 4
  atEnd.value = el.scrollLeft + el.clientWidth >= el.scrollWidth - 4
}

function scroll(dir: number) {
  const el = track.value
  if (!el) return
  el.scrollBy({ left: dir * el.clientWidth * 0.8, behavior: 'smooth' })
}

let observer: ResizeObserver | null = null

onMounted(() => {
  update()
  if (track.value) {
    observer = new ResizeObserver(update)
    observer.observe(track.value)
  }
})

onUnmounted(() => observer?.disconnect())

watch(
  () => shown.value.length,
  () => nextTick(update),
)
</script>

<template>
  <section class="carousel" aria-labelledby="companies-title">
    <header class="head">
      <h2 id="companies-title">Компании на 3–6</h2>
      <span class="count">{{ tables.length }} {{ plural(tables.length, ['стол', 'стола', 'столов']) }}</span>
      <div v-if="!(atStart && atEnd)" class="arrows">
        <button type="button" class="arrow" :disabled="atStart" aria-label="Прокрутить назад" @click="scroll(-1)">
          <PhCaretLeft :size="16" weight="bold" />
        </button>
        <button type="button" class="arrow" :disabled="atEnd" aria-label="Прокрутить вперёд" @click="scroll(1)">
          <PhCaretRight :size="16" weight="bold" />
        </button>
      </div>
      <button type="button" class="all" @click="$emit('all')">
        Все столы
        <PhArrowRight :size="16" weight="bold" aria-hidden="true" />
      </button>
    </header>

    <p v-if="error" class="error" role="alert">{{ error }}</p>

    <div class="track-wrap" :class="{ 'fade-start': !atStart, 'fade-end': !atEnd }">
      <div ref="track" class="track" role="list" @scroll.passive="update">
        <template v-if="!loaded">
          <span v-for="i in 3" :key="i" class="skeleton" role="listitem" aria-hidden="true"></span>
        </template>
        <template v-else>
          <div v-for="t in shown" :key="t.id" class="slot" role="listitem">
            <TableCard
              :table="t"
              :mine="t.createdBy === myId"
              :can-afford="canAfford(t, balance)"
              :now="now"
              @join="$emit('join', t)"
            />
          </div>
          <div class="slot" :class="{ wide: !shown.length }" role="listitem">
            <button type="button" class="create" @click="$emit('create')">
              <span class="plus" aria-hidden="true"><PhPlus :size="22" weight="bold" /></span>
              <b>Создать стол</b>
              <small v-if="shown.length">ставка, режим и 3–6 игроков</small>
              <small v-else>Пока никто не собирает компанию. Создайте стол — соперники подсядут.</small>
            </button>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<style scoped>
.carousel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}

.head {
  display: flex;
  align-items: center;
  gap: 12px;
}

h2 {
  margin: 0;
  font-size: 20px;
  font-weight: 500;
  letter-spacing: -0.01em;
  white-space: nowrap;
}

.count {
  font-size: 13px;
  color: var(--color-ink-faint);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.arrows {
  display: flex;
  gap: 6px;
  margin-left: auto;
}

.arrow {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
  color: var(--color-ink-muted);
  cursor: pointer;
}

.arrow:hover:not(:disabled) {
  color: var(--color-ink);
}

.arrow:disabled {
  opacity: 0.4;
  cursor: default;
}

.all {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: background-color 0.2s;
}

.arrows + .all {
  margin-left: 4px;
}

.head > .count + .all {
  margin-left: auto;
}

.all:hover {
  background: color-mix(in srgb, var(--color-gold) 30%, var(--color-gold-tint));
}

.error {
  margin: 0;
  font-size: 13px;
  color: var(--color-red);
}

.track-wrap {
  --fade-l: 0px;
  --fade-r: 0px;

  margin: 0 -6px;
  mask-image: linear-gradient(to right, transparent, #000 var(--fade-l), #000 calc(100% - var(--fade-r)), transparent);
}

.track-wrap.fade-start {
  --fade-l: 28px;
}

.track-wrap.fade-end {
  --fade-r: 28px;
}

.track {
  display: flex;
  gap: 14px;
  padding: 6px 6px 16px;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  scrollbar-width: none;
}

.track::-webkit-scrollbar {
  display: none;
}

.slot {
  flex: 0 0 236px;
  display: flex;
  scroll-snap-align: start;
}

.slot > * {
  flex: 1;
}

.slot.wide {
  flex-basis: min(100%, 460px);
}

.skeleton {
  flex: 0 0 236px;
  height: 268px;
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

.create {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 268px;
  padding: 20px;
  box-sizing: border-box;
  border: 1.5px dashed var(--color-line-heavy);
  border-radius: 24px;
  background: transparent;
  color: var(--color-ink);
  font: inherit;
  text-align: center;
  cursor: pointer;
  transition:
    border-color 0.2s,
    background-color 0.2s;
}

.create:hover {
  border-color: var(--color-gold);
  background: color-mix(in srgb, var(--color-gold-tint) 50%, transparent);
}

.plus {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: var(--color-gold);
  color: var(--color-paper);
}

.create b {
  font-size: 16px;
  font-weight: 500;
}

.create small {
  max-width: 260px;
  font-size: 13px;
  line-height: 1.4;
  color: var(--color-ink-faint);
}

@media (max-width: 767px), (hover: none) {
  .arrows {
    display: none;
  }

  .head > .count + .all,
  .arrows + .all {
    margin-left: auto;
  }
}

@media (max-width: 767px) {
  h2 {
    font-size: 18px;
  }

  .slot,
  .skeleton {
    flex-basis: 210px;
  }

  .all {
    padding: 0 12px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton {
    animation: none;
  }
}
</style>
