<script setup lang="ts">
import { computed, ref } from 'vue'
import { motion, useMotionValue, useSpring, useTransform, useReducedMotion } from 'motion-v'
import { PhLightning, PhPlus } from '@phosphor-icons/vue'
import PaperCard from '../landing/PaperCard.vue'
import { plural } from '../../composables/useLanding'
import { useGameSound } from '../../composables/useGameSound'
import type { DeckSize, GameMode } from '../../types/game'

const HAND = 6
const MAX_SEATS = 6
const DECKS: DeckSize[] = [24, 36, 52]
const MODES: { id: GameMode; name: string; suit: string; text: string }[] = [
  { id: 'podkidnoy', name: 'Подкидной', suit: '♠', text: 'подкидывают все, кроме защитника' },
  { id: 'perevodnoy', name: 'Переводной', suit: '♥', text: 'атаку можно перевести дальше' },
]
const TABLE_SIZE = ['', '', 'один на один', 'на троих', 'на четверых', 'на пятерых', 'на шестерых']

const mode = defineModel<GameMode>('mode', { required: true })
const deck = defineModel<DeckSize>('deck', { required: true })
const players = defineModel<number>('players', { required: true })

defineProps<{ busy: 'find' | 'create' | null; username: string }>()
defineEmits<{ (e: 'find'): void; (e: 'create'): void }>()

const sound = useGameSound()
const reduced = useReducedMotion()

const stageEl = ref<HTMLElement | null>(null)
const px = useMotionValue(0)
const py = useMotionValue(0)
const sx = useSpring(px, { stiffness: 90, damping: 18 })
const sy = useSpring(py, { stiffness: 90, damping: 18 })
const aceX = useTransform(sx, [-0.5, 0.5], [-26, 26])
const aceY = useTransform(sy, [-0.5, 0.5], [-18, 18])
const aceRotate = useTransform(sx, [-0.5, 0.5], [-20, -8])
const queenX = useTransform(sx, [-0.5, 0.5], [34, -34])
const queenY = useTransform(sy, [-0.5, 0.5], [22, -22])
const queenRotate = useTransform(sx, [-0.5, 0.5], [8, 20])

function onPointer(e: PointerEvent) {
  if (reduced.value || !stageEl.value) return
  const r = stageEl.value.getBoundingClientRect()
  px.set((e.clientX - r.left) / r.width - 0.5)
  py.set((e.clientY - r.top) / r.height - 0.5)
}

function onLeave() {
  px.set(0)
  py.set(0)
}

const fits = (n: number, size: number = deck.value) => n * HAND <= size
const left = computed(() => deck.value - players.value * HAND)
const modeInfo = computed(() => MODES.find((m) => m.id === mode.value)!)

const seats = computed(() =>
  Array.from({ length: MAX_SEATS }, (_, i) => {
    const a = Math.PI / 2 + (i * 2 * Math.PI) / MAX_SEATS
    return {
      i,
      x: 50 + 47 * Math.cos(a),
      y: 50 + 44 * Math.sin(a),
      taken: i < players.value,
      allowed: i === 0 || fits(Math.max(2, i + 1)),
    }
  }),
)

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

function pickSeat(i: number) {
  const n = Math.max(2, i + 1)
  if (i === 0 || !fits(n) || players.value === n) return
  sound.playCardDeal()
  players.value = n
}

const spring = { type: 'spring', stiffness: 380, damping: 30 } as const
</script>

<template>
  <section ref="stageEl" class="stage" aria-labelledby="stage-title" @pointermove="onPointer" @pointerleave="onLeave">
    <div class="lamp" aria-hidden="true"></div>

    <div class="felt">
      <div class="felt-inner" aria-hidden="true"></div>

      <button
        v-for="s in seats"
        :key="s.i"
        type="button"
        class="seat"
        :class="{ taken: s.taken, you: s.i === 0, locked: !s.allowed }"
        :style="{ left: `${s.x}%`, top: `${s.y}%` }"
        :disabled="s.i === 0 || !s.allowed"
        :aria-label="s.i === 0 ? 'Ваше место' : `Стол на ${s.i + 1} игроков`"
        :title="s.allowed ? '' : `В колоде ${deck} карт не хватит`"
        @click="pickSeat(s.i)"
      >
        <motion.span
          class="seat-dot"
          :animate="{ scale: s.taken ? 1 : 0.62 }"
          :transition="spring"
        >
          {{ s.i === 0 ? username.slice(0, 1).toUpperCase() : '' }}
        </motion.span>
      </button>

      <motion.div
        class="float ace"
        :style="{ x: aceX, y: aceY, rotate: aceRotate }"
        aria-hidden="true"
      >
        <motion.div
          class="float-inner"
          :animate="reduced ? {} : { y: [0, -12, 0] }"
          :transition="{ duration: 5, repeat: Infinity, ease: 'easeInOut' }"
        >
          <PaperCard rank="A" suit="♠" />
        </motion.div>
      </motion.div>

      <motion.div
        class="float queen"
        :style="{ x: queenX, y: queenY, rotate: queenRotate }"
        aria-hidden="true"
      >
        <motion.div
          class="float-inner"
          :animate="reduced ? {} : { y: [0, -14, 0] }"
          :transition="{ duration: 6, repeat: Infinity, ease: 'easeInOut', delay: 0.8 }"
        >
          <PaperCard rank="Q" :suit="mode === 'perevodnoy' ? '♥' : '♦'" />
        </motion.div>
      </motion.div>

      <div class="center">
        <p class="eyebrow rise" style="--d: 1">Новая партия</p>
        <h1 id="stage-title" class="rise" style="--d: 2">Сыграем?</h1>

        <div class="rules rise" style="--d: 3">
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
              <motion.span v-if="mode === m.id" layout-id="mode-pill" class="pill" :transition="spring" />
              <span class="suit" :class="{ red: m.suit === '♥' }">{{ m.suit }}</span>
              <span class="txt">{{ m.name }}</span>
            </button>
          </div>

          <div class="switch small" role="radiogroup" aria-label="Колода">
            <button
              v-for="d in DECKS"
              :key="d"
              type="button"
              role="radio"
              :aria-checked="deck === d"
              :class="{ on: deck === d }"
              @click="pickDeck(d)"
            >
              <motion.span v-if="deck === d" layout-id="deck-pill" class="pill" :transition="spring" />
              <span class="txt">{{ d }} карт</span>
            </button>
          </div>
        </div>

        <p class="summary rise" style="--d: 4">
          {{ modeInfo.name }}, {{ TABLE_SIZE[players] }} — {{ modeInfo.text }}.
          <span class="muted">В колоде останется {{ left }} {{ plural(left, ['карта', 'карты', 'карт']) }}.</span>
        </p>

        <div class="cta rise" style="--d: 5">
        <motion.button
          type="button"
          class="play"
          :disabled="busy !== null"
          :while-hover="busy ? {} : { scale: 1.04 }"
          :while-press="busy ? {} : { scale: 0.97 }"
          :transition="spring"
          @click="$emit('find')"
        >
          <span class="shine" aria-hidden="true"></span>
          <span v-if="busy === 'find'" class="spinner" aria-hidden="true"></span>
          <PhLightning v-else :size="22" weight="fill" aria-hidden="true" />
          {{ busy === 'find' ? 'Ищем стол…' : 'Играть' }}
        </motion.button>

        <button type="button" class="create" :disabled="busy !== null" @click="$emit('create')">
          <span v-if="busy === 'create'" class="spinner dark" aria-hidden="true"></span>
          <PhPlus v-else :size="16" weight="bold" aria-hidden="true" />
          или собрать свой стол
        </button>
        </div>

        <p class="seat-hint">Нажмите на место у стола, чтобы позвать больше игроков</p>
      </div>
    </div>
  </section>
</template>

<style scoped>
.stage {
  position: relative;
  padding: 36px 0 30px;
}

.lamp {
  position: absolute;
  inset: -60px 10% auto;
  height: 420px;
  background: radial-gradient(ellipse 50% 60% at 50% 30%, rgba(246, 232, 200, 0.85), rgba(246, 232, 200, 0) 70%);
  pointer-events: none;
}

.felt {
  position: relative;
  max-width: 1040px;
  margin: 0 auto;
  aspect-ratio: 1.85;
  min-height: 520px;
  border-radius: 50%;
  background:
    radial-gradient(ellipse at 50% 35%, rgba(255, 253, 247, 0.95), rgba(239, 235, 226, 0.5) 65%),
    var(--color-paper-sunken);
  box-shadow:
    0 0 0 12px var(--color-line-strong) inset,
    0 0 0 13px rgba(160, 150, 132, 0.35) inset,
    0 50px 90px -50px rgba(70, 62, 48, 0.55);
}

.felt-inner {
  position: absolute;
  inset: 34px;
  border-radius: 50%;
  border: 1.5px dashed rgba(160, 150, 132, 0.35);
  pointer-events: none;
}

.seat {
  position: absolute;
  z-index: 3;
  display: grid;
  place-items: center;
  width: 54px;
  height: 54px;
  padding: 0;
  translate: -50% -50%;
  border: 0;
  border-radius: 50%;
  background: transparent;
  cursor: pointer;
}

.seat:disabled {
  cursor: default;
}

.seat.locked {
  opacity: 0.35;
  cursor: not-allowed;
}

.seat-dot {
  display: grid;
  place-items: center;
  width: 46px;
  height: 46px;
  border-radius: 50%;
  background: var(--color-paper-raised);
  box-shadow:
    0 0 0 2px var(--color-line-heavy),
    0 8px 18px -10px rgba(70, 62, 48, 0.5);
  color: var(--color-paper);
  font-family: var(--font-card);
  font-size: 19px;
  transition:
    background-color 0.25s,
    box-shadow 0.25s;
}

.seat:not(:disabled):hover .seat-dot {
  box-shadow:
    0 0 0 2px var(--color-ink-soft),
    0 10px 22px -10px rgba(70, 62, 48, 0.6);
}

.seat.taken .seat-dot {
  background: var(--color-ink-soft);
  box-shadow:
    0 0 0 3px var(--color-paper-raised),
    0 10px 22px -10px rgba(54, 50, 44, 0.7);
}

.seat.you .seat-dot {
  background: var(--color-ink);
}

.float {
  position: absolute;
  z-index: 1;
  width: 132px;
  height: 188px;
  pointer-events: none;
}

.float-inner {
  width: 100%;
  height: 100%;
  filter: drop-shadow(0 26px 30px rgba(70, 62, 48, 0.28));
}

.ace {
  left: 9%;
  top: 22%;
}

.queen {
  right: 9%;
  top: 18%;
}

.center {
  position: relative;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 0 18%;
  box-sizing: border-box;
  text-align: center;
}

.eyebrow {
  margin: 0;
  font-size: 13px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: var(--color-ink-faint);
}

h1 {
  margin: 6px 0 0;
  font-family: var(--font-card);
  font-size: clamp(48px, 7vw, 84px);
  font-style: italic;
  font-weight: 400;
  letter-spacing: -0.02em;
  line-height: 1;
}

.rules {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
  margin-top: 26px;
}

.switch {
  display: flex;
  padding: 5px;
  border-radius: 999px;
  background: rgba(255, 253, 247, 0.75);
  box-shadow:
    0 0 0 1px rgba(214, 208, 196, 0.9),
    0 10px 24px -18px rgba(70, 62, 48, 0.6);
}

.switch button {
  position: relative;
  display: flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 18px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 15px;
  cursor: pointer;
  transition: color 0.2s;
}

.switch.small button {
  padding: 0 14px;
  font-variant-numeric: tabular-nums;
}

.switch button:hover:not(.on) {
  color: var(--color-ink);
}

.switch button.on {
  color: var(--color-paper);
}

.pill {
  position: absolute;
  inset: 0;
  z-index: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
}

.txt,
.suit {
  position: relative;
  z-index: 1;
}

.suit {
  font-family: var(--font-card);
  font-size: 18px;
  line-height: 1;
}

.suit.red {
  color: var(--color-red);
}

.switch button.on .suit.red {
  color: var(--color-red-soft);
}

.summary {
  max-width: 440px;
  margin: 16px 0 0;
  font-size: 15px;
  line-height: 1.5;
  color: var(--color-ink-muted);
}

.muted {
  color: var(--color-ink-faint);
}

.cta {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.play {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 12px;
  height: 68px;
  margin-top: 24px;
  padding: 0 46px;
  overflow: hidden;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink);
  color: var(--color-paper);
  font: inherit;
  font-size: 21px;
  letter-spacing: -0.01em;
  cursor: pointer;
  box-shadow:
    0 0 0 6px rgba(27, 25, 22, 0.08),
    0 24px 40px -18px rgba(27, 25, 22, 0.7);
}

.play:disabled {
  cursor: wait;
}

.shine {
  position: absolute;
  inset: 0;
  background: linear-gradient(110deg, transparent 30%, rgba(255, 255, 255, 0.18) 48%, transparent 66%);
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

.create {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
  padding: 8px 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 15px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    color 0.2s;
}

.create:hover:not(:disabled) {
  background: rgba(255, 253, 247, 0.8);
  color: var(--color-ink);
}

.seat-hint {
  margin: 10px 0 0;
  font-size: 12px;
  color: var(--color-ink-faint);
}

.spinner {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid rgba(245, 244, 239, 0.35);
  border-top-color: var(--color-paper);
  animation: spin 0.7s linear infinite;
}

.spinner.dark {
  width: 14px;
  height: 14px;
  border-color: rgba(27, 25, 22, 0.2);
  border-top-color: var(--color-ink);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 900px) {
  .felt {
    aspect-ratio: auto;
    min-height: 0;
    padding: 70px 0 80px;
    border-radius: 48px;
  }

  .felt-inner {
    inset: 22px;
    border-radius: 34px;
  }

  .center {
    padding: 0 24px;
  }

  .float {
    width: 84px;
    height: 120px;
  }

  .ace {
    left: -10px;
    top: 10px;
  }

  .queen {
    right: -10px;
    top: 4px;
  }

  .seat {
    width: 44px;
    height: 44px;
  }

  .seat-dot {
    width: 36px;
    height: 36px;
    font-size: 15px;
  }
}

@media (max-width: 520px) {
  .play {
    height: 60px;
    padding: 0 36px;
    font-size: 19px;
  }

  .switch button {
    height: 40px;
    padding: 0 12px;
    font-size: 14px;
  }

  .float {
    opacity: 0.55;
  }
}

@media (prefers-reduced-motion: reduce) {
  .shine {
    animation: none;
  }
}
</style>
