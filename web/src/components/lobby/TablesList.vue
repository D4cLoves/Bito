<script setup lang="ts">
import { computed, ref } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
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

defineEmits<{ (e: 'join', table: TableInfo): void; (e: 'refresh'): void }>()

const FILTERS: { id: Filter; label: string }[] = [
  { id: 'all', label: 'Все' },
  { id: 'podkidnoy', label: 'Подкидной' },
  { id: 'perevodnoy', label: 'Переводной' },
]

const filter = ref<Filter>('all')

const visible = computed(() => (filter.value === 'all' ? props.tables : props.tables.filter((t) => t.mode === filter.value)))
const countFor = (f: Filter) => (f === 'all' ? props.tables.length : props.tables.filter((t) => t.mode === f).length)

const modeName = (m: GameMode) => (m === 'perevodnoy' ? 'Переводной' : 'Подкидной')

function ago(iso: string) {
  const s = Math.max(0, Math.round((props.now - new Date(iso).getTime()) / 1000))
  if (s < 10) return 'только что'
  if (s < 60) return `${s} сек назад`
  const m = Math.round(s / 60)
  return `${m} ${plural(m, ['минуту', 'минуты', 'минут'])} назад`
}
</script>

<template>
  <section class="tables" aria-labelledby="tables-title">
    <div class="head">
      <h2 id="tables-title">Открытые столы</h2>
      <span class="live"><i aria-hidden="true"></i>обновляется сам</span>
      <button
        type="button"
        class="icon-btn"
        :class="{ spin: refreshing }"
        aria-label="Обновить список"
        title="Обновить список"
        @click="$emit('refresh')"
      >
        <RefreshCw :size="17" />
      </button>
    </div>

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
        {{ f.label }}
        <span class="count">{{ countFor(f.id) }}</span>
      </button>
    </div>

    <p v-if="error" class="error" role="alert">{{ error }}</p>

    <ul v-if="!loaded" class="list" aria-busy="true">
      <li v-for="i in 3" :key="i" class="row skeleton"></li>
    </ul>

    <TransitionGroup v-else-if="visible.length" tag="ul" name="row" class="list">
      <li v-for="(t, i) in visible" :key="t.id" class="row" :style="{ '--i': i }">
        <div class="row-cards" aria-hidden="true">
          <span class="rc rc1"><PaperCard rank="6" suit="♠" /></span>
          <span class="rc rc2">
            <PaperCard :rank="t.mode === 'perevodnoy' ? '6' : 'A'" :suit="t.mode === 'perevodnoy' ? '♥' : '♦'" />
          </span>
        </div>

        <div class="row-main">
          <div class="row-title">
            {{ modeName(t.mode) }}
            <span class="tag">{{ t.deck }} карт</span>
            <span v-if="t.createdBy === myId" class="tag mine">ваш стол</span>
          </div>
          <div class="row-meta">
            <span class="seats" :aria-label="`Занято ${t.players} из ${t.maxPlayers}`">
              <i v-for="s in t.maxPlayers" :key="s" :class="{ taken: s <= t.players }"></i>
            </span>
            <span>{{ t.players }}/{{ t.maxPlayers }}</span>
            <span aria-hidden="true">·</span>
            <span>{{ ago(t.createdAt) }}</span>
          </div>
        </div>

        <button type="button" class="join" @click="$emit('join', t)">
          {{ t.createdBy === myId ? 'Вернуться' : 'Сесть' }}
        </button>
      </li>
    </TransitionGroup>

    <div v-else class="empty">
      <div class="empty-cards" aria-hidden="true">
        <span class="ec ec1"><PaperCard rank="K" suit="♣" /></span>
        <span class="ec ec2"><PaperCard rank="Q" suit="♥" /></span>
      </div>
      <p class="empty-title">{{ tables.length ? 'В этом режиме пока пусто' : 'Пока никто не ждёт' }}</p>
      <p class="empty-text">Создайте стол выше — он появится здесь, и соперники смогут подсесть.</p>
    </div>
  </section>
</template>

<style scoped>
.tables {
  padding: 28px;
  border-radius: 28px;
  background: var(--color-paper-raised);
  box-shadow:
    0 0 0 1px rgba(214, 208, 196, 0.9),
    0 24px 48px -32px rgba(70, 62, 48, 0.35);
}

.head {
  display: flex;
  align-items: center;
  gap: 12px;
}

h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 500;
  letter-spacing: -0.02em;
}

.live {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-left: auto;
  font-size: 13px;
  color: var(--color-ink-faint);
}

.live i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-green);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  50% {
    opacity: 0.35;
  }
}

.icon-btn {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  color: var(--color-ink);
  cursor: pointer;
  transition: background-color 0.2s;
}

.icon-btn:hover {
  background: var(--color-paper-hover);
}

.icon-btn.spin svg {
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.filters {
  display: flex;
  gap: 6px;
  margin-top: 18px;
  overflow-x: auto;
}

.filters button {
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
  transition:
    background-color 0.2s,
    color 0.2s;
}

.filters button:hover:not(.on) {
  background: var(--color-paper-sunken);
  color: var(--color-ink);
}

.filters button.on {
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

.count {
  min-width: 20px;
  padding: 1px 6px;
  box-sizing: border-box;
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

.error {
  margin: 16px 0 0;
  padding: 10px 14px;
  border-radius: 14px;
  background: var(--color-red-tint);
  color: var(--color-red-strong);
  font-size: 14px;
}

.list {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 18px 0 0;
  padding: 0;
  list-style: none;
}

.row {
  display: flex;
  align-items: center;
  gap: 16px;
  min-height: 76px;
  padding: 12px 12px 12px 16px;
  box-sizing: border-box;
  border-radius: 20px;
  background: var(--color-paper);
  box-shadow: 0 0 0 1px rgba(214, 208, 196, 0.8);
  transition:
    box-shadow 0.25s,
    transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.row:not(.skeleton):hover {
  transform: translateY(-2px);
  box-shadow:
    0 0 0 1px var(--color-line-stronger),
    0 16px 30px -22px rgba(70, 62, 48, 0.5);
}

.row:not(.skeleton):hover .rc1 {
  transform: rotate(-16deg) translateX(-2px);
}

.row:not(.skeleton):hover .rc2 {
  transform: rotate(14deg) translateX(3px);
}

.skeleton {
  background: linear-gradient(90deg, var(--color-paper-sunken) 0%, var(--color-paper-raised) 50%, var(--color-paper-sunken) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

.row-cards {
  position: relative;
  flex: none;
  width: 52px;
  height: 52px;
}

.rc {
  position: absolute;
  top: 2px;
  width: 32px;
  height: 46px;
  transition: transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.rc :deep(.pip) {
  font-size: 18px;
}

.rc :deep(.index) {
  display: none;
}

.rc1 {
  left: 0;
  transform: rotate(-10deg);
}

.rc2 {
  left: 16px;
  transform: rotate(8deg);
}

.row-main {
  flex: 1;
  min-width: 0;
}

.row-title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 17px;
  letter-spacing: -0.01em;
}

.tag {
  padding: 3px 9px;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  font-size: 12px;
  color: var(--color-ink-muted);
}

.tag.mine {
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.row-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 6px;
  font-size: 13px;
  color: var(--color-ink-faint);
  font-variant-numeric: tabular-nums;
}

.seats {
  display: flex;
  gap: 4px;
}

.seats i {
  width: 9px;
  height: 9px;
  border-radius: 50%;
  box-shadow: inset 0 0 0 1.5px var(--color-line-heavy);
}

.seats i.taken {
  background: var(--color-ink-soft);
  box-shadow: none;
}

.join {
  flex: none;
  height: 44px;
  padding: 0 22px;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font: inherit;
  font-size: 15px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.2s;
}

.join:hover {
  background: var(--color-ink);
}

.join:active {
  transform: scale(0.97);
}

.empty {
  padding: 40px 16px 24px;
  text-align: center;
}

.empty-cards {
  position: relative;
  width: 120px;
  height: 100px;
  margin: 0 auto 18px;
}

.ec {
  position: absolute;
  top: 0;
  width: 62px;
  height: 88px;
  animation: float 4s ease-in-out infinite;
}

.ec1 {
  left: 10px;
  rotate: -12deg;
}

.ec2 {
  left: 46px;
  rotate: 9deg;
  animation-delay: -2s;
}

@keyframes float {
  50% {
    translate: 0 -6px;
  }
}

.empty-title {
  margin: 0;
  font-size: 18px;
}

.empty-text {
  max-width: 320px;
  margin: 8px auto 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-ink-subtle);
}

.row-enter-active {
  transition:
    opacity 0.4s,
    transform 0.45s cubic-bezier(0.2, 0.8, 0.2, 1);
  transition-delay: calc(var(--i, 0) * 40ms);
}

.row-leave-active {
  position: absolute;
  left: 0;
  right: 0;
  transition:
    opacity 0.25s,
    transform 0.25s;
}

.row-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.row-leave-to {
  opacity: 0;
  transform: scale(0.97);
}

.row-move {
  transition: transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}

@media (max-width: 560px) {
  .tables {
    padding: 22px 16px;
  }

  .live {
    display: none;
  }

  .head .icon-btn {
    margin-left: auto;
  }

  .row-cards {
    display: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ec,
  .live i {
    animation: none;
  }
}
</style>
