<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { PhArrowRight, PhCaretLeft, PhCaretRight, PhPlus } from '@phosphor-icons/vue'
import TableCard from './TableCard.vue'
import TableTop from './TableTop.vue'
import { plural } from '../../composables/useLanding'
import { canAfford, sortTables, stakeOf, TABLE_PRESETS, type TablePreset } from '../../composables/tableList'
import { bestPayout } from '../../constants/economy'
import type { GameMode, TableInfo } from '../../types/game'

const MAX_CARDS = 12

type ModeFilter = 'any' | GameMode
type StakeFilter = 'any' | 100 | 250 | 500

const props = defineProps<{
  tables: TableInfo[]
  loaded: boolean
  error: string
  myId?: string
  me?: string
  now: number
  balance: number | null
  busy?: boolean
}>()

defineEmits<{
  (e: 'join', table: TableInfo): void
  (e: 'create'): void
  (e: 'preset', preset: TablePreset): void
  (e: 'all'): void
}>()

const affordable = ref(false)
const mode = ref<ModeFilter>('any')
const stake = ref<StakeFilter>('any')

const STAKE_CHIPS: { value: StakeFilter; label: string }[] = [
  { value: 100, label: '100' },
  { value: 250, label: '250' },
  { value: 500, label: '500+' },
]

const filtering = computed(() => affordable.value || mode.value !== 'any' || stake.value !== 'any')

function matches(t: TableInfo) {
  if (affordable.value && !canAfford(t, props.balance)) return false
  if (mode.value !== 'any' && t.mode !== mode.value) return false
  if (stake.value === 500) return stakeOf(t) >= 500
  if (stake.value !== 'any') return stakeOf(t) === stake.value
  return true
}

const shown = computed(() => sortTables(props.tables.filter(matches), 'full', props.balance).slice(0, MAX_CARDS))

function toggleMode(m: GameMode) {
  mode.value = mode.value === m ? 'any' : m
}

function toggleStake(s: StakeFilter) {
  stake.value = stake.value === s ? 'any' : s
}

function reset() {
  affordable.value = false
  mode.value = 'any'
  stake.value = 'any'
}

const presetAffordable = (p: TablePreset) => props.balance === null || p.stake <= props.balance
const presetMode = (p: TablePreset) => (p.mode === 'perevodnoy' ? 'Переводной' : 'Подкидной')
const presetWho = (p: TablePreset) => `${p.players} ${plural(p.players, ['игрок', 'игрока', 'игроков'])}`

const countLabel = computed(() => {
  const n = props.tables.length
  if (!n) return 'пока никого'
  const all = `${n} ${plural(n, ['стол', 'стола', 'столов'])}`
  return filtering.value ? `${shown.value.length} из ${all}` : all
})

const track = ref<HTMLElement | null>(null)
const atStart = ref(true)
const atEnd = ref(true)

function update() {
  const el = track.value
  if (!el) return
  atStart.value = el.scrollLeft <= 8
  atEnd.value = el.scrollLeft + el.clientWidth >= el.scrollWidth - 8
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
      <span class="count">{{ countLabel }}</span>

      <div v-if="tables.length" class="filters" role="group" aria-label="Фильтры столов">
        <button type="button" class="chip" :class="{ on: affordable }" :aria-pressed="affordable" @click="affordable = !affordable">
          По карману
        </button>
        <button type="button" class="chip" :class="{ on: mode === 'podkidnoy' }" :aria-pressed="mode === 'podkidnoy'" @click="toggleMode('podkidnoy')">
          Подкидной
        </button>
        <button type="button" class="chip" :class="{ on: mode === 'perevodnoy' }" :aria-pressed="mode === 'perevodnoy'" @click="toggleMode('perevodnoy')">
          Переводной
        </button>
        <button
          v-for="s in STAKE_CHIPS"
          :key="s.label"
          type="button"
          class="chip"
          :class="{ on: stake === s.value }"
          :aria-pressed="stake === s.value"
          @click="toggleStake(s.value)"
        >
          {{ s.label }}
        </button>
        <button v-if="filtering" type="button" class="reset" @click="reset">Сбросить</button>
      </div>

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
          <span v-for="i in 4" :key="i" class="slot skeleton" role="listitem" aria-hidden="true"></span>
        </template>

        <template v-else-if="shown.length">
          <div v-for="t in shown" :key="t.id" class="slot" role="listitem">
            <TableCard
              :table="t"
              :mine="t.createdBy === myId"
              :me="me"
              :can-afford="canAfford(t, balance)"
              :now="now"
              @join="$emit('join', t)"
            />
          </div>
        </template>

        <template v-else>
          <div v-for="p in TABLE_PRESETS" :key="`${p.players}-${p.stake}`" class="slot" role="listitem">
            <article class="preset" :class="{ poor: !presetAffordable(p) }">
              <TableTop :seats="p.players" :taken="0" :stake="p.stake" muted />
              <div class="info">
                <span class="title">
                  <span class="suit" aria-hidden="true">{{ p.mode === 'perevodnoy' ? '♥' : '♠' }}</span>
                  {{ presetWho(p) }} · ставка {{ p.stake.toLocaleString('ru-RU') }}
                </span>
                <span class="meta">{{ presetMode(p) }} · {{ p.deck }} карт</span>
                <span class="win">выигрыш до +{{ bestPayout(p.stake, p.players).toLocaleString('ru-RU') }}</span>
              </div>
              <button
                type="button"
                class="make"
                :disabled="busy || !presetAffordable(p)"
                :aria-label="`Создать стол: ${presetWho(p)}, ставка ${p.stake}, ${presetMode(p)}, ${p.deck} карт`"
                @click="$emit('preset', p)"
              >
                {{ presetAffordable(p) ? 'Создать' : 'Мало фишек' }}
              </button>
            </article>
          </div>
        </template>

        <div v-if="loaded" class="slot" role="listitem">
          <button type="button" class="create" @click="$emit('create')">
            <span class="plus" aria-hidden="true"><PhPlus :size="24" weight="bold" /></span>
            <b>Свой стол</b>
            <small>ставка, режим, колода и <span class="nowrap">3–6 игроков</span></small>
          </button>
        </div>
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
  align-items: baseline;
  gap: 12px;
  min-width: 0;
}

h2 {
  margin: 0;
  font-family: var(--font-card);
  font-size: 28px;
  font-style: italic;
  font-weight: 400;
  letter-spacing: -0.01em;
  white-space: nowrap;
}

.count {
  font-size: 13px;
  color: var(--color-ink-faint);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.filters {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  margin-left: 12px;
  overflow: hidden;
}

.chip,
.reset {
  flex: none;
  height: 28px;
  padding: 0 12px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    color 0.2s;
}

.chip:hover {
  color: var(--color-ink);
}

.chip.on {
  background: var(--color-ink);
  box-shadow: none;
  color: var(--color-paper);
}

.reset {
  background: none;
  box-shadow: none;
  color: var(--color-ink-faint);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.arrows {
  display: flex;
  align-self: center;
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
  align-self: center;
  gap: 6px;
  flex: none;
  height: 34px;
  margin-left: auto;
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
  gap: 16px;
  padding: 6px 6px 18px;
  overflow-x: auto;
  scroll-snap-type: x mandatory;
  scrollbar-width: none;
}

.track::-webkit-scrollbar {
  display: none;
}

.slot {
  flex: 0 0 max(220px, calc((100% - 64px) / 5));
  min-height: 320px;
  scroll-snap-align: start;
}

.skeleton {
  border-radius: 26px;
  background: linear-gradient(100deg, var(--color-paper-sunken) 30%, var(--color-paper-hover) 50%, var(--color-paper-sunken) 70%);
  background-size: 200% 100%;
  animation: shimmer 1.4s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

.preset {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 10px 10px 14px;
  box-sizing: border-box;
  border-radius: 26px;
  background: color-mix(in srgb, var(--color-paper-raised) 70%, transparent);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.preset.poor {
  opacity: 0.6;
}

.info {
  display: flex;
  flex-direction: column;
  gap: 3px;
  padding: 4px 8px 12px;
}

.title {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow: hidden;
  font-size: 15px;
  font-weight: 500;
  white-space: nowrap;
}

.suit {
  color: var(--color-gold);
}

.meta {
  color: var(--color-ink-faint);
  font-size: 12px;
}

.win {
  color: var(--color-green);
  font-size: 12px;
  font-variant-numeric: tabular-nums;
}

.make {
  height: 42px;
  margin: auto 2px 0;
  border: 0;
  border-radius: 999px;
  background: none;
  box-shadow: inset 0 0 0 1.5px var(--color-gold);
  color: var(--color-gold-ink);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.15s;
}

.make:hover:not(:disabled) {
  background: var(--color-gold-tint);
}

.make:active:not(:disabled) {
  transform: scale(0.98);
}

.make:disabled {
  box-shadow: none;
  color: var(--color-red);
  cursor: not-allowed;
}

.create {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  height: 100%;
  padding: 20px;
  box-sizing: border-box;
  border: 0;
  border-radius: 26px;
  background: none;
  box-shadow: inset 0 0 0 1.5px var(--color-line-heavy);
  color: var(--color-ink);
  font: inherit;
  text-align: center;
  cursor: pointer;
  transition:
    box-shadow 0.2s,
    background-color 0.2s;
}

.create:hover {
  background: color-mix(in srgb, var(--color-gold-tint) 50%, transparent);
  box-shadow: inset 0 0 0 1.5px var(--color-gold);
}

.plus {
  display: grid;
  place-items: center;
  width: 54px;
  height: 54px;
  border-radius: 50%;
  background: var(--color-gold);
  color: var(--color-on-gold);
}

.create b {
  font-size: 16px;
  font-weight: 500;
}

.create small {
  max-width: 180px;
  font-size: 13px;
  line-height: 1.4;
  color: var(--color-ink-faint);
}

.nowrap {
  white-space: nowrap;
}

@media (max-height: 920px) {
  .track {
    --table-h: 156px;
  }

  .slot {
    min-height: 290px;
  }
}

@media (max-width: 1180px) {
  .filters {
    display: none;
  }
}

@media (max-width: 767px), (hover: none) {
  .arrows {
    display: none;
  }

  .all {
    margin-left: auto;
  }
}

@media (max-width: 767px) {
  .head {
    flex-wrap: wrap;
    row-gap: 8px;
  }

  h2 {
    font-size: 22px;
  }

  .slot {
    flex-basis: 220px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .skeleton {
    animation: none;
  }
}
</style>
