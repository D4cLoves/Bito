<script setup lang="ts">
import { computed } from 'vue'
import { motion, useReducedMotion } from 'motion-v'
import { PhLightning, PhPlus, PhLockSimple, PhLink } from '@phosphor-icons/vue'
import CardBack from './CardBack.vue'
import { STAKES, TABLE_FEE, bestPayout, formatChips } from '../../constants/economy'
import { useGameSound } from '../../composables/useGameSound'
import type { CardBackId } from '../../stores/settings'
import type { DeckSize, GameMode } from '../../types/game'

const HAND = 6
const DECKS: DeckSize[] = [24, 36, 52]
const SEATS = [2, 3, 4, 5, 6]
const MODES: { id: GameMode; name: string; suit: string }[] = [
  { id: 'podkidnoy', name: 'Подкидной', suit: '♠' },
  { id: 'perevodnoy', name: 'Переводной', suit: '♥' },
]

const mode = defineModel<GameMode>('mode', { required: true })
const deck = defineModel<DeckSize>('deck', { required: true })
const players = defineModel<number>('players', { required: true })
const stake = defineModel<number>('stake', { required: true })

const props = defineProps<{
  busy: 'find' | 'create' | null
  balance: number | null
  cardBack: CardBackId
}>()

const emit = defineEmits<{ (e: 'find'): void; (e: 'create'): void; (e: 'private'): void }>()

const sound = useGameSound()
const reduced = useReducedMotion()

const fits = (n: number, size: number = deck.value) => n * HAND <= size
const affordable = (s: number) => props.balance === null || s <= props.balance
const payout = computed(() => bestPayout(stake.value, players.value))
const broke = computed(() => props.balance !== null && !affordable(stake.value))

function pickStake(s: number) {
  if (!affordable(s) || stake.value === s) return
  sound.playClick()
  stake.value = s
}

function pickMode(m: GameMode) {
  if (mode.value === m) return
  sound.playClick()
  mode.value = m
}

function pickDeck(d: DeckSize) {
  if (deck.value === d) return
  sound.playClick()
  deck.value = d
  if (!fits(players.value, d)) players.value = Math.floor(d / HAND)
}

function pickPlayers(n: number) {
  if (!fits(n) || players.value === n) return
  sound.playCardDeal()
  players.value = n
}

const fan = Array.from({ length: HAND }, (_, i) => {
  const offset = i - (HAND - 1) / 2
  return { rotate: offset * 8, y: offset * offset * 2.5, x: offset * 30 }
})

const spring = { type: 'spring', stiffness: 380, damping: 30 } as const
</script>

<template>
  <section class="play" aria-labelledby="play-title">
    <div class="lamp" aria-hidden="true"></div>
    <div class="rim" aria-hidden="true"></div>

    <div class="controls">
      <p class="eyebrow">Новая партия</p>
      <h2 id="play-title">Сыграем?</h2>

      <div class="group">
        <span class="group-label">Ставка</span>
        <div class="stakes" role="radiogroup" aria-label="Ставка">
          <button
            v-for="s in STAKES"
            :key="s"
            type="button"
            role="radio"
            class="chip"
            :class="[`v${s}`, { on: stake === s, locked: !affordable(s) }]"
            :aria-checked="stake === s"
            :disabled="!affordable(s)"
            :title="affordable(s) ? `${s} фишек` : 'Не хватает фишек'"
            @click="pickStake(s)"
          >
            <span class="chip-face">
              <PhLockSimple v-if="!affordable(s)" :size="13" weight="bold" />
              <template v-else>{{ formatChips(s) }}</template>
            </span>
          </button>
        </div>
      </div>

      <div class="row">
        <div class="group">
          <span class="group-label">Режим</span>
          <div class="switch" role="radiogroup" aria-label="Режим">
            <button
              v-for="m in MODES"
              :key="m.id"
              type="button"
              role="radio"
              :aria-checked="mode === m.id"
              :class="{ on: mode === m.id }"
              @click="pickMode(m.id)"
            >
              <motion.span v-if="mode === m.id" layout-id="play-mode" class="pill" :transition="spring" />
              <span class="suit" :class="{ red: m.suit === '♥' }">{{ m.suit }}</span>
              <span class="txt">{{ m.name }}</span>
            </button>
          </div>
        </div>

        <div class="group">
          <span class="group-label">Колода</span>
          <div class="switch" role="radiogroup" aria-label="Колода">
            <button
              v-for="d in DECKS"
              :key="d"
              type="button"
              role="radio"
              :aria-checked="deck === d"
              :class="{ on: deck === d }"
              @click="pickDeck(d)"
            >
              <motion.span v-if="deck === d" layout-id="play-deck" class="pill" :transition="spring" />
              <span class="txt">{{ d }}</span>
            </button>
          </div>
        </div>

        <div class="group">
          <span class="group-label">Игроков</span>
          <div class="switch" role="radiogroup" aria-label="Число игроков">
            <button
              v-for="n in SEATS"
              :key="n"
              type="button"
              role="radio"
              :aria-checked="players === n"
              :class="{ on: players === n }"
              :disabled="!fits(n)"
              :title="fits(n) ? '' : `В колоде ${deck} карт не хватит на ${n} игроков`"
              @click="pickPlayers(n)"
            >
              <motion.span v-if="players === n" layout-id="play-seats" class="pill" :transition="spring" />
              <span class="txt">{{ n }}</span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <div class="action">
      <div class="hand" aria-hidden="true">
        <motion.div
          v-for="(c, i) in fan"
          :key="i"
          class="hand-card"
          :initial="{ opacity: 0, y: 40, rotate: 0 }"
          :animate="{ opacity: 1, x: c.x, y: c.y, rotate: c.rotate }"
          :transition="{ ...spring, delay: reduced ? 0 : i * 0.03 }"
        >
          <CardBack :variant="cardBack" />
        </motion.div>
      </div>

      <dl class="odds">
        <div>
          <dt>Выигрыш</dt>
          <dd class="plus">до +{{ payout.toLocaleString('ru-RU') }}</dd>
        </div>
        <div>
          <dt>Проигрыш</dt>
          <dd class="minus">−{{ stake.toLocaleString('ru-RU') }}</dd>
        </div>
        <div>
          <dt>Комиссия</dt>
          <dd>{{ Math.round(TABLE_FEE * 100) }}%</dd>
        </div>
      </dl>

      <motion.button
        type="button"
        class="go"
        :disabled="busy !== null || broke"
        :while-hover="busy ? {} : { scale: 1.03 }"
        :while-press="busy ? {} : { scale: 0.97 }"
        :transition="spring"
        @click="emit('find')"
      >
        <span class="shine" aria-hidden="true"></span>
        <span v-if="busy === 'find'" class="spinner" aria-hidden="true"></span>
        <PhLightning v-else :size="22" weight="fill" aria-hidden="true" />
        {{ busy === 'find' ? 'Ищем стол…' : 'Играть' }}
      </motion.button>

      <div class="secondary">
        <button type="button" class="ghost" :disabled="busy !== null || broke" @click="emit('create')">
          <span v-if="busy === 'create'" class="spinner small" aria-hidden="true"></span>
          <PhPlus v-else :size="16" weight="bold" aria-hidden="true" />
          Свой стол
        </button>
        <button type="button" class="ghost" :disabled="busy !== null" @click="emit('private')">
          <PhLink :size="16" weight="bold" aria-hidden="true" />
          По ссылке
          <span class="soon">скоро</span>
        </button>
      </div>
    </div>
  </section>
</template>

<style scoped>
.play {
  --on-felt: var(--color-on-felt);
  --on-felt-muted: var(--color-on-felt-muted);
  --on-felt-faint: var(--color-on-felt-faint);

  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(280px, 1fr);
  gap: 28px;
  padding: 30px 34px;
  overflow: hidden;
  border-radius: 32px;
  background:
    radial-gradient(ellipse 80% 120% at 70% 0%, rgba(255, 255, 255, 0.1), transparent 60%),
    radial-gradient(ellipse at 50% 120%, var(--color-felt-deep), var(--color-felt) 70%);
  color: var(--on-felt);
  box-shadow:
    inset 0 0 0 1px var(--color-felt-line),
    inset 0 0 80px color-mix(in srgb, var(--color-felt-deep) 80%, transparent),
    0 30px 60px -36px rgba(0, 0, 0, 0.5);
}

.play::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image: radial-gradient(rgba(255, 255, 255, 0.05) 1px, transparent 1px);
  background-size: 4px 4px;
  pointer-events: none;
}

.lamp {
  position: absolute;
  top: -40%;
  right: 8%;
  width: 60%;
  height: 120%;
  background: radial-gradient(ellipse at 50% 30%, rgba(255, 228, 170, 0.22), transparent 65%);
  pointer-events: none;
}

.rim {
  position: absolute;
  inset: 10px;
  border-radius: 24px;
  border: 1.5px dashed var(--color-felt-line);
  pointer-events: none;
}

.controls,
.action {
  position: relative;
  min-width: 0;
}

.controls {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.eyebrow {
  margin: 0 0 -14px;
  font-size: 12px;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: var(--on-felt-faint);
}

h2 {
  margin: 0;
  font-family: var(--font-card);
  font-size: clamp(40px, 4.2vw, 60px);
  font-style: italic;
  font-weight: 400;
  letter-spacing: -0.02em;
  line-height: 1.05;
}

.group {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}

.group-label {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--on-felt-faint);
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px 18px;
}

.stakes {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.chip {
  --chip: #e9e4d8;
  --chip-ink: #2b2824;

  position: relative;
  display: grid;
  place-items: center;
  width: 58px;
  height: 58px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background:
    repeating-conic-gradient(from 0deg, rgba(255, 255, 255, 0.85) 0 10deg, transparent 10deg 45deg),
    var(--chip);
  box-shadow:
    0 4px 0 rgba(0, 0, 0, 0.35),
    0 10px 18px -8px rgba(0, 0, 0, 0.6);
  cursor: pointer;
  transition:
    transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1),
    box-shadow 0.25s;
}

.chip-face {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--chip);
  box-shadow: inset 0 0 0 1.5px rgba(255, 255, 255, 0.55);
  color: var(--chip-ink);
  font-size: 13px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.chip:hover:not(:disabled) {
  transform: translateY(-3px);
}

.chip.on {
  transform: translateY(-6px);
  box-shadow:
    0 0 0 3px var(--on-felt),
    0 4px 0 rgba(0, 0, 0, 0.35),
    0 16px 24px -8px rgba(0, 0, 0, 0.7);
}

.chip.locked {
  cursor: not-allowed;
  filter: grayscale(1) brightness(0.55);
}

.v250 {
  --chip: #b6463c;
  --chip-ink: #fff;
}

.v500 {
  --chip: #2f7a55;
  --chip-ink: #fff;
}

.v1000 {
  --chip: #23232a;
  --chip-ink: #f3eee4;
}

.v2500 {
  --chip: #6a4a9a;
  --chip-ink: #fff;
}

.v5000 {
  --chip: #d4ab5c;
  --chip-ink: #3a2a0c;
}

.switch {
  display: flex;
  padding: 4px;
  border-radius: 999px;
  background: var(--color-felt-well);
  box-shadow: inset 0 0 0 1px var(--color-felt-line);
}

.switch button {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-width: 40px;
  height: 40px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--on-felt-muted);
  font: inherit;
  font-size: 15px;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: color 0.2s;
}

.switch button:hover:not(:disabled):not(.on) {
  color: var(--on-felt);
}

.switch button:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.switch button.on {
  color: var(--color-on-felt-invert);
}

.pill {
  position: absolute;
  inset: 0;
  border-radius: 999px;
  background: var(--on-felt);
}

.txt,
.suit {
  position: relative;
}

.suit {
  font-family: var(--font-card);
  font-size: 17px;
  line-height: 1;
}

.suit.red {
  color: var(--color-felt-minus);
}

.on .suit.red {
  color: var(--color-felt-pill-red);
}

.action {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
}

.hand {
  position: relative;
  width: 100%;
  height: 168px;
  margin-bottom: 8px;
}

.hand-card {
  position: absolute;
  left: 50%;
  bottom: 30px;
  width: 84px;
  height: 120px;
  margin-left: -42px;
  transform-origin: 50% 120%;
}

.odds {
  display: grid;
  grid-template-columns: repeat(3, auto);
  gap: 22px;
  margin: 12px 0 0;
}

.odds div {
  text-align: center;
}

.odds dt {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--on-felt-faint);
}

.odds dd {
  margin: 3px 0 0;
  font-size: 17px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.plus {
  color: var(--color-felt-plus);
}

.minus {
  color: var(--color-felt-minus);
}

.go {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  width: min(320px, 100%);
  height: 64px;
  margin-top: 18px;
  overflow: hidden;
  border: 0;
  border-radius: 999px;
  background: linear-gradient(180deg, #f7e2ad, #d6aa58);
  color: #2b1f08;
  font: inherit;
  font-size: 21px;
  font-weight: 500;
  letter-spacing: -0.01em;
  cursor: pointer;
  box-shadow:
    0 0 0 6px rgba(214, 170, 88, 0.16),
    0 20px 36px -16px rgba(0, 0, 0, 0.8);
}

.go:disabled {
  cursor: not-allowed;
  filter: saturate(0.4) brightness(0.8);
}

.shine {
  position: absolute;
  inset: 0;
  background: linear-gradient(110deg, transparent 30%, rgba(255, 255, 255, 0.45) 48%, transparent 66%);
  transform: translateX(-120%);
  animation: shine 3.6s ease-in-out infinite 1.2s;
}

@keyframes shine {
  0%,
  55% {
    transform: translateX(-120%);
  }
  100% {
    transform: translateX(120%);
  }
}

.secondary {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-top: 12px;
}

.ghost {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--color-felt-well);
  box-shadow: inset 0 0 0 1px var(--color-felt-line);
  color: var(--on-felt);
  font: inherit;
  font-size: 14px;
  white-space: nowrap;
  cursor: pointer;
  transition: background-color 0.2s;
}

.ghost:hover:not(:disabled) {
  box-shadow: inset 0 0 0 1px var(--color-on-felt-faint);
}

.ghost:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.soon {
  padding: 2px 7px;
  border-radius: 999px;
  background: var(--color-felt-well);
  font-size: 11px;
  color: var(--on-felt-muted);
}

.spinner {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid rgba(43, 31, 8, 0.25);
  border-top-color: #2b1f08;
  animation: spin 0.7s linear infinite;
}

.spinner.small {
  width: 14px;
  height: 14px;
  border-color: var(--color-on-felt-faint);
  border-top-color: var(--on-felt);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1100px) {
  .play {
    grid-template-columns: minmax(0, 1fr);
  }

  .hand {
    height: 120px;
  }

  .hand-card {
    width: 70px;
    height: 100px;
    margin-left: -35px;
  }
}

@media (max-width: 560px) {
  .play {
    padding: 22px 18px;
    border-radius: 24px;
  }

  .chip {
    width: 42px;
    height: 42px;
  }

  .chip-face {
    width: 30px;
    height: 30px;
    font-size: 11px;
  }

  .switch button {
    height: 36px;
    padding: 0 11px;
    font-size: 14px;
  }

  .odds {
    gap: 14px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .shine {
    animation: none;
  }
}
</style>
