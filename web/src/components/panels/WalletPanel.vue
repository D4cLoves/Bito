<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  PhArrowCounterClockwise,
  PhCardsThree,
  PhGift,
  PhHandCoins,
  PhShoppingBag,
  PhTrophy,
} from '@phosphor-icons/vue'
import ChipIcon from '../client/ChipIcon.vue'
import { getTransactions } from '../../api/wallet'
import type { TransactionKind, WalletTransaction } from '../../types/user'

const props = defineProps<{ balance: number | null }>()

const KINDS: Record<TransactionKind, { label: string; icon: typeof PhGift }> = {
  signup_bonus: { label: 'Бонус за регистрацию', icon: PhGift },
  opening: { label: 'Стартовый баланс', icon: PhGift },
  stake_lock: { label: 'Ставка за столом', icon: PhCardsThree },
  stake_refund: { label: 'Возврат ставки', icon: PhArrowCounterClockwise },
  win_payout: { label: 'Выплата за партию', icon: PhTrophy },
  top_up: { label: 'Пополнение', icon: PhHandCoins },
  purchase: { label: 'Покупка в магазине', icon: PhShoppingBag },
}

const state = ref<'loading' | 'ready' | 'error'>('loading')
const ops = ref<WalletTransaction[]>([])

async function load() {
  state.value = 'loading'
  try {
    ops.value = await getTransactions(100)
    state.value = 'ready'
  } catch {
    state.value = 'error'
  }
}

onMounted(load)

const shownBalance = computed(() => props.balance ?? ops.value[0]?.balanceAfter ?? 0)

const sumOf = (kinds: TransactionKind[]) =>
  ops.value.filter((o) => kinds.includes(o.kind)).reduce((s, o) => s + Math.abs(o.amount), 0)

const totals = computed(() => [
  { label: 'Выплаты', value: sumOf(['win_payout', 'stake_refund']), sign: '+' },
  { label: 'Ставки', value: sumOf(['stake_lock']), sign: '−' },
  { label: 'Покупки', value: sumOf(['purchase']), sign: '−' },
])

const series = computed(() => {
  if (!ops.value.length) return []
  const chrono = [...ops.value].reverse()
  const first = chrono[0]
  return [
    { balance: first.balanceAfter - first.amount, op: null as WalletTransaction | null },
    ...chrono.map((op) => ({ balance: op.balanceAfter, op })),
  ]
})

const W = 800
const H = 200
const PAD = 14

const range = computed(() => {
  const values = series.value.map((p) => p.balance)
  const min = Math.min(...values)
  const max = Math.max(...values)
  return max === min ? { min: min - 1, max: max + 1 } : { min, max }
})

const xOf = (i: number) => (series.value.length < 2 ? W / 2 : (i / (series.value.length - 1)) * W)
const yOf = (v: number) => PAD + (1 - (v - range.value.min) / (range.value.max - range.value.min)) * (H - PAD * 2)

const line = computed(() => series.value.map((p, i) => `${i ? 'L' : 'M'}${xOf(i).toFixed(1)},${yOf(p.balance).toFixed(1)}`).join(' '))
const area = computed(() => (series.value.length ? `${line.value} L${W},${H} L0,${H} Z` : ''))

const hover = ref<number | null>(null)
const point = computed(() => (hover.value === null ? null : series.value[hover.value]))

function track(e: PointerEvent) {
  const box = (e.currentTarget as HTMLElement).getBoundingClientRect()
  const ratio = Math.min(1, Math.max(0, (e.clientX - box.left) / box.width))
  hover.value = Math.round(ratio * (series.value.length - 1))
}

function when(iso: string) {
  const d = new Date(iso)
  const today = new Date()
  const time = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  if (d.toDateString() === today.toDateString()) return `сегодня, ${time}`
  const yesterday = new Date(today)
  yesterday.setDate(today.getDate() - 1)
  if (d.toDateString() === yesterday.toDateString()) return `вчера, ${time}`
  return `${d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })}, ${time}`
}

const fmt = (n: number) => n.toLocaleString('ru-RU')
const signed = (n: number) => (n === 0 ? '0' : `${n > 0 ? '+' : '−'}${fmt(Math.abs(n))}`)
</script>

<template>
  <div class="wallet">
    <div class="head">
      <div class="balance">
        <small>Баланс</small>
        <b><ChipIcon :size="26" />{{ fmt(shownBalance) }}</b>
      </div>
      <ul v-if="state === 'ready' && ops.length" class="totals">
        <li v-for="t in totals" :key="t.label">
          <small>{{ t.label }}</small>
          <b :class="t.sign === '+' ? 'plus' : 'minus'">{{ t.value ? `${t.sign}${fmt(t.value)}` : '0' }}</b>
        </li>
      </ul>
    </div>

    <p v-if="state === 'loading'" class="note">Загружаем журнал…</p>
    <p v-else-if="state === 'error'" class="note">
      Не удалось загрузить журнал.
      <button type="button" class="retry" @click="load">Повторить</button>
    </p>
    <p v-else-if="!ops.length" class="note">Операций пока нет. Сыграйте первую партию.</p>

    <template v-else>
      <section class="chart" aria-label="График баланса">
        <div class="plot" @pointermove="track" @pointerleave="hover = null">
          <svg :viewBox="`0 0 ${W} ${H}`" preserveAspectRatio="none" aria-hidden="true">
            <defs>
              <linearGradient id="wallet-fill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stop-color="var(--color-gold)" stop-opacity="0.28" />
                <stop offset="100%" stop-color="var(--color-gold)" stop-opacity="0" />
              </linearGradient>
            </defs>
            <path :d="area" fill="url(#wallet-fill)" />
            <path :d="line" fill="none" stroke="var(--color-gold)" stroke-width="2.5" stroke-linejoin="round" vector-effect="non-scaling-stroke" />
          </svg>
          <template v-if="point && hover !== null">
            <span class="guide" :style="{ left: `${(xOf(hover) / W) * 100}%` }" />
            <span class="dot" :style="{ left: `${(xOf(hover) / W) * 100}%`, top: `${(yOf(point.balance) / H) * 100}%` }" />
            <span class="tip" :class="{ flip: xOf(hover) > W * 0.7 }" :style="{ left: `${(xOf(hover) / W) * 100}%` }">
              <b>{{ fmt(point.balance) }}</b>
              <small v-if="point.op">{{ KINDS[point.op.kind].label }} · {{ signed(point.op.amount) }}</small>
              <small v-else>начало журнала</small>
            </span>
          </template>
        </div>
        <div class="axis">
          <span>мин. {{ fmt(range.min) }}</span>
          <span>последние {{ ops.length }} операций</span>
          <span>макс. {{ fmt(range.max) }}</span>
        </div>
      </section>

      <section aria-labelledby="ops-title">
        <h3 id="ops-title">Операции</h3>
        <ol class="list">
          <li v-for="op in ops" :key="op.id" class="row">
            <span class="icon" :class="op.amount >= 0 ? 'plus' : 'minus'" aria-hidden="true">
              <component :is="KINDS[op.kind].icon" :size="20" weight="duotone" />
            </span>
            <span class="what">
              <b>{{ KINDS[op.kind].label }}</b>
              <small>{{ when(op.createdAt) }} · баланс {{ fmt(op.balanceAfter) }}</small>
            </span>
            <span class="amount" :class="op.amount >= 0 ? 'plus' : 'minus'">{{ signed(op.amount) }}</span>
          </li>
        </ol>
      </section>
    </template>
  </div>
</template>

<style scoped>
.wallet {
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
}

.balance small,
.totals small {
  display: block;
  font-size: 13px;
  color: var(--color-ink-faint);
}

.balance b {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 40px;
  font-weight: 500;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.totals {
  display: flex;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.totals li {
  min-width: 104px;
  padding: 10px 14px;
  border-radius: 16px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.totals b {
  font-size: 17px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.plus {
  color: var(--color-green);
}

.minus {
  color: var(--color-red);
}

.note {
  margin: 0;
  padding: 28px;
  border-radius: 22px;
  background: var(--color-paper-raised);
  text-align: center;
  color: var(--color-ink-muted);
}

.retry {
  margin-left: 6px;
  border: 0;
  background: none;
  color: var(--color-gold-ink);
  font: inherit;
  text-decoration: underline;
  cursor: pointer;
}

.chart {
  padding: 16px 16px 10px;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.plot {
  position: relative;
  height: 180px;
  touch-action: pan-y;
  cursor: crosshair;
}

.plot svg {
  display: block;
  width: 100%;
  height: 100%;
}

.guide {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: var(--color-line-strong);
  pointer-events: none;
}

.dot {
  position: absolute;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-gold);
  box-shadow: 0 0 0 3px var(--color-paper-raised);
  transform: translate(-50%, -50%);
  pointer-events: none;
}

.tip {
  position: absolute;
  top: 0;
  display: flex;
  flex-direction: column;
  padding: 6px 10px;
  border-radius: 12px;
  background: var(--color-paper);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
  white-space: nowrap;
  transform: translateX(10px);
  pointer-events: none;
}

.tip.flip {
  transform: translateX(calc(-100% - 10px));
}

.tip b {
  font-size: 15px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.tip small {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.axis {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-ink-faint);
  font-variant-numeric: tabular-nums;
}

h3 {
  margin: 0 0 10px;
  font-size: 18px;
  font-weight: 500;
  letter-spacing: -0.01em;
}

.list {
  display: flex;
  flex-direction: column;
  margin: 0;
  padding: 6px;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  list-style: none;
}

.row {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 10px 12px;
}

.row + .row {
  border-top: 1px solid var(--color-line);
}

.icon {
  display: grid;
  place-items: center;
  flex: none;
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: var(--color-paper-hover);
}

.what {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.what b {
  font-size: 15px;
  font-weight: 500;
}

.what small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  color: var(--color-ink-faint);
}

.amount {
  font-size: 16px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 640px) {
  .totals {
    width: 100%;
  }

  .totals li {
    flex: 1;
    min-width: 0;
  }

  .balance b {
    font-size: 32px;
  }
}
</style>
