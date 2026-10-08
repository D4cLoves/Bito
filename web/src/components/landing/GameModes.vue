<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { animate } from 'animejs'
import type { JSAnimation } from 'animejs'
import { plural, requestJoin } from '../../composables/useLanding'
import { useReveal } from '../../composables/useReveal'

// Конструктор стола: те же параметры, что принимает сервер — режим, колода, 2–6 игроков.
type Mode = 'podkidnoy' | 'perevodnoy'

const MODES: { id: Mode; name: string; text: string }[] = [
  {
    id: 'podkidnoy',
    name: 'Подкидной',
    text: 'Классика. Все, кроме защитника, могут подкидывать карты того достоинства, что уже есть на столе.',
  },
  {
    id: 'perevodnoy',
    name: 'Переводной',
    text: 'Защитник может перевести атаку картой того же достоинства — и отбиваться придётся следующему.',
  },
]
const DECKS = [
  { size: 24, note: 'С девяток. Короткие и острые партии.' },
  { size: 36, note: 'С шестёрок. Так играет большинство.' },
  { size: 52, note: 'Полная колода. Партии дольше, запоминать больше.' },
]
const cardsWord = (n: number) => plural(n, ['карта', 'карты', 'карт'])
const playersWord = (n: number) => plural(n, ['игрок', 'игрока', 'игроков'])
const PLAYERS = [2, 3, 4, 5, 6]
const HAND = 6

const mode = ref<Mode>('podkidnoy')
const deck = ref(36)
const players = ref(2)
const hint = ref('')

const deckLeft = computed(() => deck.value - players.value * HAND)
const shownLeft = ref(deckLeft.value)
const modeInfo = computed(() => MODES.find((m) => m.id === mode.value)!)
const deckInfo = computed(() => DECKS.find((d) => d.size === deck.value)!)
const fits = (n: number, size = deck.value) => n * HAND <= size

// Места по эллипсу; «вы» всегда снизу, дальше по часовой стрелке.
const RX = 190
const RY = 118
const seats = computed(() =>
  Array.from({ length: players.value }, (_, i) => {
    const a = Math.PI / 2 + (i * 2 * Math.PI) / players.value
    return { id: i, x: 250 + RX * Math.cos(a), y: 160 + RY * Math.sin(a), you: i === 0 }
  }),
)
// Атакует «вы» (0), защищается следующий (1), при переводе атака уходит к 2-му.
const defender = computed(() => seats.value[1])
const nextDefender = computed(() => seats.value[2 % players.value])

const deckLayers = computed(() => Math.max(1, Math.ceil(deckLeft.value / 4)))

function pickDeck(size: number) {
  deck.value = size
  if (!fits(players.value, size)) {
    const max = Math.floor(size / HAND)
    hint.value = `На ${size} ${cardsWord(size)} — не больше ${max} ${playersWord(max)}.`
    players.value = max
  } else {
    hint.value = ''
  }
}

function pickPlayers(n: number) {
  if (!fits(n)) return
  hint.value = ''
  players.value = n
}

const rootEl = ref<HTMLElement | null>(null)

function seatEnter(el: Element, done: () => void) {
  animate(el, { scale: [0, 1], opacity: [0, 1], duration: 520, ease: 'outBack(1.8)', onComplete: done })
}
function seatLeave(el: Element, done: () => void) {
  animate(el, { scale: 0, opacity: 0, duration: 260, ease: 'in(2)', onComplete: done })
}

watch(deckLeft, (to) => {
  const counter = { v: shownLeft.value }
  animate(counter, {
    v: to,
    duration: 600,
    ease: 'out(3)',
    onUpdate: () => {
      shownLeft.value = Math.round(counter.v)
    },
  })
})

useReveal(rootEl)

// «Бегущий» пунктир стрелок. Линии пересоздаются при смене режима и числа игроков,
// поэтому анимацию перезапускаем после каждого изменения.
let flow: JSAnimation | null = null
function runFlow() {
  flow?.revert()
  const lines = rootEl.value?.querySelectorAll('.flow')
  if (!lines?.length) return
  flow = animate(lines, { strokeDashoffset: [0, -24], duration: 900, ease: 'linear', loop: true })
}

onMounted(runFlow)
watch([mode, players], () => nextTick(runFlow))
onUnmounted(() => flow?.revert())
</script>

<template>
  <section id="modes" ref="rootEl" class="modes">
    <div class="modes-copy">
      <p class="eyebrow" data-reveal>Режимы</p>
      <h2 class="title" data-reveal>Соберите свой стол.</h2>
      <p class="lead" data-reveal>Два классических варианта правил, три колоды, от двух до шести игроков. Попробуйте сочетания.</p>

      <div class="control" data-reveal>
        <span class="control-label">Режим</span>
        <div class="segmented">
          <button
            v-for="m in MODES"
            :key="m.id"
            type="button"
            class="seg"
            :class="{ on: mode === m.id }"
            @click="mode = m.id"
          >
            {{ m.name }}
          </button>
        </div>
        <Transition name="swap" mode="out-in">
          <p :key="mode" class="control-note">{{ modeInfo.text }}</p>
        </Transition>
      </div>

      <div class="control" data-reveal>
        <span class="control-label">Колода</span>
        <div class="segmented">
          <button
            v-for="d in DECKS"
            :key="d.size"
            type="button"
            class="seg"
            :class="{ on: deck === d.size }"
            @click="pickDeck(d.size)"
          >
            {{ d.size }} {{ cardsWord(d.size) }}
          </button>
        </div>
        <Transition name="swap" mode="out-in">
          <p :key="deck" class="control-note">{{ deckInfo.note }}</p>
        </Transition>
      </div>

      <div class="control" data-reveal>
        <span class="control-label">Игроки</span>
        <div class="segmented">
          <button
            v-for="n in PLAYERS"
            :key="n"
            type="button"
            class="seg seg--num"
            :class="{ on: players === n }"
            :disabled="!fits(n)"
            :title="fits(n) ? '' : `Не хватит карт на ${n} ${playersWord(n)}`"
            @click="pickPlayers(n)"
          >
            {{ n }}
          </button>
        </div>
        <Transition name="swap" mode="out-in">
          <p :key="hint || 'ok'" class="control-note" :class="{ warn: hint }">
            {{ hint || `По шесть карт каждому — раздаём ${players * HAND}.` }}
          </p>
        </Transition>
      </div>

      <button type="button" class="pill-btn play" data-reveal @click="requestJoin">Играть за таким столом</button>
    </div>

    <div class="modes-visual" data-reveal>
      <svg class="oval" viewBox="0 0 500 320" aria-hidden="true">
        <defs>
          <marker id="arrow" viewBox="0 0 10 10" refX="7" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
            <path d="M0 0L10 5L0 10z" fill="#8f8a7d" />
          </marker>
        </defs>
        <ellipse cx="250" cy="160" rx="232" ry="146" class="oval-rim" />
        <ellipse cx="250" cy="160" rx="214" ry="130" class="oval-felt" />

        <!-- Подкидной: все, кроме защитника, подкидывают в центр -->
        <g v-if="mode === 'podkidnoy'">
          <line
            v-for="s in seats.filter((s) => s.id !== 1)"
            :key="`a-${s.id}-${players}`"
            class="flow"
            :x1="s.x + (250 - s.x) * 0.3"
            :y1="s.y + (160 - s.y) * 0.3"
            :x2="250 + (s.x - 250) * 0.28"
            :y2="160 + (s.y - 160) * 0.28"
            marker-end="url(#arrow)"
          />
        </g>
        <!-- Переводной: защитник переводит атаку на следующего -->
        <g v-else>
          <path
            :key="`t-${players}`"
            class="flow flow--transfer"
            :d="`M ${defender.x} ${defender.y} Q 250 160 ${nextDefender.x} ${nextDefender.y}`"
            marker-end="url(#arrow)"
          />
        </g>
      </svg>

      <!-- Колода в центре: высота стопки = остаток после раздачи -->
      <div class="deck">
        <span class="trump"></span>
        <span
          v-for="i in deckLayers"
          :key="i"
          class="deck-card"
          :style="{ transform: `translate(${-i * 0.4}px, ${-i * 1.1}px)` }"
        ></span>
        <span class="deck-count" :style="{ transform: `translate(${-deckLayers * 0.4}px, ${-deckLayers * 1.1}px)` }">
          {{ shownLeft }}
        </span>
      </div>

      <TransitionGroup :css="false" @enter="seatEnter" @leave="seatLeave">
        <div
          v-for="s in seats"
          :key="s.id"
          class="seat"
          :class="{ you: s.you, defending: s.id === 1 }"
          :style="{ left: `${(s.x / 500) * 100}%`, top: `${(s.y / 320) * 100}%` }"
        >
          <span class="mini-hand">
            <i v-for="k in 3" :key="k" :style="{ transform: `rotate(${(k - 2) * 12}deg)` }"></i>
          </span>
          <span class="seat-name">{{ s.you ? 'Вы' : s.id === 1 ? 'Защита' : `Игрок ${s.id + 1}` }}</span>
        </div>
      </TransitionGroup>

      <p class="summary">
        {{ modeInfo.name }} · {{ deck }} {{ cardsWord(deck) }} · {{ players }} {{ playersWord(players) }} · в колоде {{ deckLeft }}
      </p>
    </div>
  </section>
</template>

<style scoped>
.modes {
  max-width: 1280px;
  margin: 0 auto;
  padding: 140px 48px 120px;
  display: grid;
  grid-template-columns: 440px 1fr;
  gap: 56px;
  align-items: center;
}

.eyebrow {
  margin: 0 0 18px;
  font-size: 13px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #8a857a;
}

.title {
  margin: 0 0 16px;
  font-size: 56px;
  font-weight: 400;
  line-height: 1;
  letter-spacing: -0.03em;
  color: #36322c;
}

.lead {
  margin: 0 0 36px;
  max-width: 380px;
  font-size: 18px;
  line-height: 27px;
  color: #57534b;
}

.control {
  margin-bottom: 26px;
}

.control-label {
  display: block;
  margin-bottom: 10px;
  font-size: 12px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #8a857a;
}

.segmented {
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  border: 1.5px solid #dcd7cc;
  border-radius: 999px;
  background: rgba(250, 249, 245, 0.6);
}

.seg {
  height: 36px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  font: inherit;
  font-size: 15px;
  color: #57534b;
  cursor: pointer;
  transition:
    background-color 0.25s,
    color 0.25s;
}

.seg:hover:not(:disabled) {
  color: #1b1916;
}

.seg.on {
  background: #d7d3c8;
  color: #1b1916;
}

.seg:disabled {
  color: #c4bfb3;
  cursor: not-allowed;
}

.seg--num {
  width: 40px;
  padding: 0;
}

.control-note {
  min-height: 42px;
  margin: 10px 0 0;
  max-width: 400px;
  font-size: 14.5px;
  line-height: 21px;
  color: #6b675e;
}

.control-note.warn {
  color: #a2412f;
}

.swap-enter-active,
.swap-leave-active {
  transition:
    opacity 0.2s,
    transform 0.2s;
}

.swap-enter-from {
  opacity: 0;
  transform: translateY(6px);
}

.swap-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.pill-btn {
  border: 0;
  border-radius: 999px;
  background: #d7d3c8;
  color: #1b1916;
  font: inherit;
  font-size: 15.5px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.2s;
}

.pill-btn:hover {
  background: #cdc8bb;
}

.pill-btn:active {
  transform: scale(0.985);
}

.play {
  margin-top: 8px;
  height: 46px;
  padding: 0 26px;
}

/* Визуализация стола */
.modes-visual {
  position: relative;
  aspect-ratio: 500 / 320;
}

.oval {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  overflow: visible;
}

.oval-rim {
  fill: rgba(236, 233, 225, 0.6);
  stroke: #dcd7cc;
  stroke-width: 1.5;
}

.oval-felt {
  fill: rgba(250, 249, 245, 0.7);
  stroke: rgba(170, 160, 140, 0.35);
  stroke-dasharray: 3 5;
}

.flow {
  fill: none;
  stroke: #8f8a7d;
  stroke-width: 1.6;
  stroke-dasharray: 5 7;
  stroke-linecap: round;
}

.flow--transfer {
  stroke: #a1473f;
}

.deck {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 46px;
  height: 66px;
  margin: -33px 0 0 -23px;
}

.deck-card,
.trump {
  position: absolute;
  inset: 0;
  border-radius: 6px;
  box-shadow: 0 0 0 1px rgba(160, 150, 132, 0.45);
  transition: transform 0.4s;
}

.deck-card {
  background:
    repeating-linear-gradient(45deg, transparent 0 4px, rgba(160, 148, 124, 0.2) 4px 5px),
    #e7e1d4;
}

.trump {
  background: #fbfaf6;
  transform: translateX(22px) rotate(90deg);
}

.trump::after {
  content: '♥';
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%) rotate(-90deg);
  font-size: 18px;
  color: #a1473f;
}

/* Остаток колоды — прямо на верхней карте стопки */
.deck-count {
  position: absolute;
  z-index: 50;
  isolation: isolate;
  inset: 0;
  display: grid;
  place-items: center;
  font-size: 15px;
  font-weight: 500;
  color: #4a453d;
  font-variant-numeric: tabular-nums;
  transition: transform 0.4s;
}

.deck-count::before {
  content: '';
  position: absolute;
  width: 30px;
  height: 22px;
  border-radius: 999px;
  background: rgba(251, 250, 246, 0.92);
  box-shadow: 0 0 0 1px rgba(160, 150, 132, 0.4);
  z-index: -1;
}

.seat {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  translate: -50% -50%;
  transition:
    left 0.55s cubic-bezier(0.3, 0.7, 0.2, 1),
    top 0.55s cubic-bezier(0.3, 0.7, 0.2, 1);
}

.mini-hand {
  position: relative;
  width: 44px;
  height: 40px;
}

.mini-hand i {
  position: absolute;
  left: 13px;
  top: 2px;
  width: 22px;
  height: 32px;
  border-radius: 4px;
  transform-origin: 50% 100%;
  background:
    repeating-linear-gradient(45deg, transparent 0 3px, rgba(160, 148, 124, 0.22) 3px 4px),
    #e7e1d4;
  box-shadow: 0 0 0 1px rgba(160, 150, 132, 0.5);
}

.seat.you .mini-hand i {
  background: #fbfaf6;
}

.seat-name {
  padding: 3px 10px;
  border-radius: 999px;
  background: rgba(250, 249, 245, 0.9);
  box-shadow: 0 0 0 1px #e1dcd1;
  font-size: 12.5px;
  color: #57534b;
  white-space: nowrap;
}

.seat.you .seat-name {
  background: #36322c;
  box-shadow: none;
  color: #f5f4ef;
}

.seat.defending .seat-name {
  color: #a1473f;
  box-shadow: 0 0 0 1px rgba(161, 71, 63, 0.35);
}

.summary {
  position: absolute;
  left: 50%;
  bottom: -54px;
  transform: translateX(-50%);
  margin: 0;
  font-size: 14px;
  color: #8a857a;
  white-space: nowrap;
}

@media (max-width: 1000px) {
  .modes {
    grid-template-columns: 1fr;
    padding: 100px 20px 110px;
    gap: 40px;
  }

  .title {
    font-size: 40px;
  }

  .seg {
    padding: 0 12px;
    font-size: 14px;
  }

  .seg--num {
    width: 36px;
    padding: 0;
  }
}
</style>
