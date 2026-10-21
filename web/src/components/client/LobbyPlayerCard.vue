<script setup lang="ts">
import { computed } from 'vue'
import { PhArrowCounterClockwise, PhFire } from '@phosphor-icons/vue'
import { useCountUp } from '../../composables/useCountUp'
import { rankTitle, winRate } from '../../composables/usePlayerRank'
import { avatarTone } from '../../composables/avatar'
import { useSettingsStore } from '../../stores/settings'
import AvatarGlyph from './AvatarGlyph.vue'
import CardBack from './CardBack.vue'
import type { Profile } from '../../types/user'
import type { GameHistoryEntry, GameOutcome } from '../../types/social'

const SLOTS = 6

const settings = useSettingsStore()
const tilt = (i: number) => ({ '--i': i - (SLOTS - 1) / 2 })

const props = defineProps<{
  username: string
  profile: Profile | null
  history: GameHistoryEntry[]
  loaded: boolean
  now: number
}>()

defineEmits<{ (e: 'profile'): void; (e: 'again', entry: GameHistoryEntry): void }>()

const stats = computed(() => props.profile?.stats)
const rate = computed(() => (stats.value ? winRate(stats.value.wins, stats.value.losses) : null))
const rank = computed(() => rankTitle(stats.value?.totalGames ?? 0))

function outcomeOf(g: GameHistoryEntry): GameOutcome {
  if (g.outcome) return g.outcome
  return g.payout > 0 ? 'win' : g.payout < 0 ? 'lose' : 'draw'
}

const streak = computed(() => {
  let n = 0
  for (const g of props.history) {
    if (outcomeOf(g) !== 'win') break
    n++
  }
  return n
})

const best = computed(() => props.history.reduce((m, g) => Math.max(m, g.payout), 0))

const rateShown = useCountUp(computed(() => rate.value ?? 0), 1100)
const gamesShown = useCountUp(computed(() => stats.value?.totalGames))

const recent = computed(() => props.history.slice(0, SLOTS))
const empty = computed(() => Math.max(0, SLOTS - recent.value.length))
const last = computed(() => props.history[0] ?? null)

const isDuel = (g: GameHistoryEntry) => g.players === 2
const isFirst = (g: GameHistoryEntry) => outcomeOf(g) === 'win' && (isDuel(g) || g.place === 1)

function titleOf(g: GameHistoryEntry) {
  const o = outcomeOf(g)
  if (o === 'draw') return 'Ничья'
  if (o === 'lose') return isDuel(g) ? 'Поражение' : 'Проигрыш'
  return isDuel(g) || g.place === 1 ? 'Победа' : `${g.place} место`
}

function metaOf(g: GameHistoryEntry) {
  const mode = g.mode === 'perevodnoy' ? 'Переводной' : 'Подкидной'
  const who = isDuel(g) ? '1 на 1' : `${g.players} игроков`
  return `${mode} · ${g.deck} карт · ${who} · ${ago(g.finishedAt)}`
}

function placeOf(g: GameHistoryEntry) {
  return outcomeOf(g) === 'win' && !isDuel(g) ? String(g.place) : ''
}

const suitOf = (g: GameHistoryEntry) => (g.mode === 'perevodnoy' ? '♥' : '♠')

function shortPayout(g: GameHistoryEntry) {
  const abs = Math.abs(g.payout)
  const sign = g.payout > 0 ? '+' : g.payout < 0 ? '−' : ''
  if (abs < 1000) return `${sign}${abs}`
  return `${sign}${(abs / 1000).toFixed(1).replace('.0', '').replace('.', ',')}к`
}

function payoutOf(g: GameHistoryEntry) {
  if (g.payout > 0) return `+${g.payout.toLocaleString('ru-RU')}`
  if (g.payout < 0) return `−${Math.abs(g.payout).toLocaleString('ru-RU')}`
  return '0'
}

function ago(iso: string) {
  const min = Math.max(0, Math.round((props.now - new Date(iso).getTime()) / 60000))
  if (min < 1) return 'только что'
  if (min < 60) return `${min} мин назад`
  const h = Math.round(min / 60)
  if (h < 24) return `${h} ч назад`
  return new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
}
</script>

<template>
  <section class="player" aria-label="Карточка игрока">
    <header class="who">
      <button type="button" class="me" :aria-label="`Профиль: ${username}`" @click="$emit('profile')">
        <span class="avatar" :style="{ '--tone': avatarTone(username) }"><AvatarGlyph :name="username" self /></span>
        <i class="suit" aria-hidden="true">♠</i>
      </button>
      <button type="button" class="id" @click="$emit('profile')">
        <span class="label">Игрок</span>
        <span class="name">{{ username }}</span>
        <span class="rank">{{ rank }}</span>
      </button>
      <div class="rate">
        <b>{{ rate === null ? '—' : `${rateShown}%` }}</b>
        <span>побед</span>
      </div>
    </header>

    <p class="line">
      <span><b>{{ stats ? gamesShown : '—' }}</b> партий</span>
      <span :class="{ hot: streak >= 2 }">
        <PhFire v-if="streak >= 2" :size="14" weight="fill" aria-hidden="true" />
        <b>{{ streak }}</b> подряд
      </span>
      <span v-if="best > 0"><b>+{{ best.toLocaleString('ru-RU') }}</b> лучший</span>
    </p>

    <ol v-if="loaded" class="minis" aria-label="Последние партии">
      <li
        v-for="(g, i) in recent"
        :key="g.id"
        class="mini"
        :style="tilt(i)"
        :class="[outcomeOf(g), { first: isFirst(g) }]"
        :title="`${titleOf(g)} ${payoutOf(g)} · ${metaOf(g)}`"
        :aria-label="`${titleOf(g)}, ${payoutOf(g)}, ${metaOf(g)}`"
      >
        <template v-if="outcomeOf(g) === 'lose'">
          <CardBack class="back" :variant="settings.cardBack" />
          <span class="sum tag" aria-hidden="true">{{ shortPayout(g) }}</span>
        </template>
        <template v-else>
          <span class="index" :class="{ red: g.mode === 'perevodnoy' }" aria-hidden="true">
            <b v-if="placeOf(g)">{{ placeOf(g) }}</b>
            <i>{{ suitOf(g) }}</i>
          </span>
          <span class="pip" :class="{ red: g.mode === 'perevodnoy' }" aria-hidden="true">{{ suitOf(g) }}</span>
          <span class="sum" aria-hidden="true">{{ shortPayout(g) }}</span>
        </template>
      </li>
      <li v-for="i in empty" :key="`e${i}`" class="mini slot" :style="tilt(recent.length + i - 1)" aria-hidden="true"></li>
    </ol>
    <div v-else class="minis-skeleton" aria-busy="true"></div>

    <div v-if="last" class="last">
      <span class="last-what">
        <small>Последняя · {{ metaOf(last) }}</small>
        <b>{{ titleOf(last) }}</b>
      </span>
      <span class="last-sum" :class="outcomeOf(last)">{{ payoutOf(last) }}</span>
      <button type="button" class="again" @click="$emit('again', last)">
        <PhArrowCounterClockwise :size="15" weight="bold" aria-hidden="true" />
        Ещё партия
      </button>
    </div>
    <p v-else-if="loaded" class="last hint">Сыграйте первую партию — карты начнут ложиться сюда.</p>
  </section>
</template>

<style scoped>
.player {
  --on-felt: var(--color-on-felt);
  --on-felt-muted: var(--color-on-felt-muted);
  --on-felt-faint: var(--color-on-felt-faint);

  position: relative;
  isolation: isolate;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  padding: 24px 24px 22px;
  box-sizing: border-box;
  overflow: hidden;
  border-radius: 32px;
  background:
    radial-gradient(ellipse 80% 120% at 72% 0%, rgba(255, 255, 255, 0.12), transparent 60%),
    radial-gradient(ellipse at 50% 120%, var(--color-felt-deep), var(--color-felt) 70%);
  box-shadow:
    inset 0 0 0 1px var(--color-felt-line),
    inset 0 0 90px color-mix(in srgb, var(--color-felt-deep) 80%, transparent),
    0 30px 60px -36px rgba(0, 0, 0, 0.55);
  color: var(--on-felt);
}

.player::before {
  content: '';
  position: absolute;
  inset: 0;
  z-index: -1;
  background-image: radial-gradient(rgba(255, 255, 255, 0.05) 1px, transparent 1px);
  background-size: 4px 4px;
  pointer-events: none;
}

button {
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.who {
  display: flex;
  align-items: center;
  gap: 16px;
  min-width: 0;
}

.me {
  position: relative;
  flex: none;
  border-radius: 50%;
}

.avatar {
  display: grid;
  place-items: center;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-gold) var(--tone, 52%), #000);
  box-shadow:
    0 0 0 3px color-mix(in srgb, var(--color-felt-deep) 70%, transparent),
    0 0 0 5px var(--color-gold),
    0 14px 24px -12px rgba(0, 0, 0, 0.6);
  color: #fff;
  font-family: var(--font-card);
  font-size: 30px;
  font-style: italic;
  transition: transform 0.2s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.me:hover .avatar {
  transform: scale(1.05);
}

.suit {
  position: absolute;
  right: -4px;
  bottom: -2px;
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--color-gold);
  color: var(--color-on-gold);
  font-size: 13px;
  font-style: normal;
  box-shadow: 0 0 0 2px var(--color-felt);
}

.id {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 4px;
  min-width: 0;
  text-align: left;
}

.label {
  color: var(--on-felt-faint);
  font-size: 11px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
}

.name {
  max-width: 100%;
  overflow: hidden;
  font-family: var(--font-card);
  font-size: 30px;
  font-style: italic;
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rank {
  padding: 3px 10px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--color-gold) 22%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-gold) 45%, transparent);
  color: var(--color-felt-target);
  font-size: 12px;
}

.rate {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  margin-left: auto;
}

.rate b {
  color: var(--color-felt-target);
  font-family: var(--font-card);
  font-size: 38px;
  font-style: italic;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.rate span {
  color: var(--on-felt-faint);
  font-size: 12px;
}

.line {
  display: flex;
  flex-wrap: wrap;
  gap: 4px 18px;
  margin: 0;
  color: var(--on-felt-muted);
  font-size: 13px;
}

.line span {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
}

.line b {
  color: var(--on-felt);
  font-family: var(--font-card);
  font-size: 17px;
  font-style: italic;
  font-weight: 400;
  font-variant-numeric: tabular-nums;
}

.line .hot,
.line .hot b {
  color: var(--color-gold);
}

.minis {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.mini {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  padding-bottom: 6px;
  aspect-ratio: 5 / 7;
  box-sizing: border-box;
  border: 1px solid color-mix(in srgb, var(--color-ink) 12%, transparent);
  border-radius: 9px;
  background: var(--color-paper-raised);
  box-shadow: 0 10px 16px -10px rgba(0, 0, 0, 0.6);
  color: var(--color-ink);
  backface-visibility: hidden;
  will-change: transform;
  transform: translateY(calc(var(--i, 0) * var(--i, 0) * 1.3px)) rotate(calc(var(--i, 0) * 2.4deg));
  transform-origin: 50% 120%;
  transition: transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.mini:not(.slot):hover {
  z-index: 1;
  transform: translateY(-6px) rotate(0deg) scale(1.06);
}

.mini .index {
  position: absolute;
  top: 5px;
  left: 6px;
  display: flex;
  flex-direction: column;
  align-items: center;
  font-family: var(--font-card);
  line-height: 1;
}

.mini .index b {
  font-size: 12px;
  font-weight: 400;
}

.mini .index i {
  margin-top: 1px;
  font-size: 9px;
  font-style: normal;
}

.mini .index i:only-child {
  margin-top: 0;
  font-size: 12px;
}

.mini .index.red i {
  color: var(--color-red);
}

.mini .pip {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -64%);
  font-family: var(--font-card);
  font-size: 24px;
  line-height: 1;
  color: var(--color-ink);
}

.mini .pip.red {
  color: var(--color-red);
}

.mini.first .pip {
  color: var(--color-gold);
}

.mini .sum {
  position: relative;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: -0.02em;
  font-variant-numeric: tabular-nums;
}

.mini.win .sum {
  color: var(--color-green);
}

.mini.first {
  border-color: color-mix(in srgb, var(--color-gold) 60%, transparent);
}

.mini.first .index b {
  color: var(--color-gold-ink);
}

.mini.draw .sum,
.mini.draw .index b {
  color: var(--color-ink-faint);
}

.mini.lose {
  border: 0;
  background: none;
  box-shadow: none;
}

.mini .back {
  position: absolute;
  inset: 0;
  border-radius: 9px;
}

.mini .tag {
  position: relative;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--color-red);
  color: var(--color-on-red);
  font-size: 10px;
  box-shadow: 0 2px 6px -2px rgba(0, 0, 0, 0.5);
}

.mini.slot {
  border: 1.5px dashed var(--color-felt-line);
  background: none;
  box-shadow: none;
}

.minis-skeleton {
  height: 90px;
  border-radius: 12px;
  background: var(--color-felt-well);
}

.last {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: auto 0 0;
  padding: 10px 10px 10px 16px;
  border-radius: 20px;
  background: var(--color-felt-well);
  box-shadow: inset 0 0 0 1px var(--color-felt-line);
}

.last-what {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.last-what small {
  overflow: hidden;
  color: var(--on-felt-faint);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.last-what b {
  font-size: 15px;
  font-weight: 500;
}

.last-sum {
  flex: none;
  margin-left: auto;
  font-family: var(--font-card);
  font-size: 24px;
  font-style: italic;
  font-variant-numeric: tabular-nums;
}

.last-sum.win {
  color: var(--color-felt-plus);
}

.last-sum.lose {
  color: var(--color-felt-minus);
}

.last-sum.draw {
  color: var(--on-felt-muted);
}

.again {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
  height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  background: var(--on-felt);
  color: var(--color-on-felt-invert);
  font-size: 14px;
  font-weight: 500;
  transition: transform 0.2s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.again:hover {
  transform: translateY(-1px);
}

.hint {
  color: var(--on-felt-muted);
  font-size: 13px;
  line-height: 1.4;
}

@media (prefers-reduced-motion: reduce) {
  .mini,
  .avatar,
  .again {
    transition: none;
  }
}
</style>
