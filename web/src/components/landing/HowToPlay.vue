<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { createTimeline, onScroll, utils } from 'animejs'
import type { Timeline } from 'animejs'
import PaperCard from './PaperCard.vue'
import { requestJoin } from '../../composables/useLanding'

// Закреплённая сцена: прокрутка проигрывает один раунд дурака.
// Для каждой карты задана поза на каждом из 8 состояний стола,
// таймлайн строится из разницы соседних состояний.

type Suit = '♠' | '♣' | '♥' | '♦'
interface Pose {
  x: number
  y: number
  r: number
  up: boolean
  z: number
}
interface DemoCard {
  id: string
  rank: string
  suit: Suit
}

const STEPS = [
  {
    title: 'Everyone gets six',
    text: 'Each player is dealt six cards. What’s left becomes the deck.',
  },
  {
    title: 'One suit rules',
    text: 'The bottom card of the deck sets the trump. A trump beats any card of another suit.',
  },
  {
    title: 'Lead the attack',
    text: 'The attacker opens the bout with any card from their hand.',
  },
  {
    title: 'Beat it',
    text: 'The defender covers it with a higher card of the same suit — or any trump. Can’t? Take the whole table.',
  },
  {
    title: 'Pile on',
    text: 'Attackers may toss in cards of a rank already on the table. Every one has to be beaten too.',
  },
  {
    title: 'Бито!',
    text: 'All attacks beaten — the cards go to the discard pile, “bito”. That’s where our name comes from.',
  },
  {
    title: 'Don’t be the durak',
    text: 'Everyone draws back to six and play moves on. When the deck runs dry, the last player holding cards is the durak.',
  },
]
const STEP_MS = 1000
const MOVE_MS = 700

const W = 720
const H = 480
const DECK = { x: 96, y: 240 }
const SLOT_1 = { x: 300, y: 236 }
const SLOT_2 = { x: 430, y: 236 }
const DISCARD = { x: 628, y: 240 }

const you: DemoCard[] = [
  { id: 'y1', rank: '7', suit: '♠' },
  { id: 'y2', rank: '7', suit: '♦' },
  { id: 'y3', rank: '10', suit: '♣' },
  { id: 'y4', rank: 'K', suit: '♣' },
  { id: 'y5', rank: 'J', suit: '♦' },
  { id: 'y6', rank: 'A', suit: '♠' },
  { id: 'y7', rank: 'Q', suit: '♠' },
  { id: 'y8', rank: '9', suit: '♦' },
]
const opp: DemoCard[] = [
  { id: 'o1', rank: '8', suit: '♣' },
  { id: 'o2', rank: '10', suit: '♠' },
  { id: 'o3', rank: 'Q', suit: '♦' },
  { id: 'o4', rank: '9', suit: '♠' },
  { id: 'o5', rank: '6', suit: '♥' },
  { id: 'o6', rank: 'K', suit: '♦' },
  { id: 'o7', rank: 'J', suit: '♣' },
  { id: 'o8', rank: '8', suit: '♠' },
]
const trump: DemoCard = { id: 't', rank: '8', suit: '♥' }
const cards = [trump, ...you, ...opp]

// Веер руки: края ниже и развёрнуты наружу. dir = 1 — снизу (вы), -1 — сверху.
function fan(ids: string[], dir: 1 | -1, up: boolean): Record<string, Pose> {
  const out: Record<string, Pose> = {}
  const cy = dir === 1 ? 418 : 62
  ids.forEach((id, i) => {
    const o = i - (ids.length - 1) / 2
    out[id] = { x: 360 + o * 50, y: cy + dir * o * o * 3, r: dir * o * 5, up, z: 10 + i }
  })
  return out
}

function buildStates(): Record<string, Pose>[] {
  const deckPose = (i: number): Pose => ({ x: DECK.x, y: DECK.y - i * 0.6, r: 0, up: false, z: 6 + i })
  const s0: Record<string, Pose> = { t: { x: DECK.x, y: DECK.y, r: 0, up: false, z: 1 } }
  ;[...you, ...opp].forEach((c, i) => (s0[c.id] = deckPose(i)))

  const s1 = {
    ...s0,
    ...fan(['y1', 'y2', 'y3', 'y4', 'y5', 'y6'], 1, true),
    ...fan(['o1', 'o2', 'o3', 'o4', 'o5', 'o6'], -1, false),
  }
  const s2 = { ...s1, t: { x: DECK.x + 40, y: DECK.y, r: 90, up: true, z: 1 } }
  const s3 = {
    ...s2,
    ...fan(['y2', 'y3', 'y4', 'y5', 'y6'], 1, true),
    y1: { x: SLOT_1.x, y: SLOT_1.y, r: -4, up: true, z: 30 },
  }
  const s4 = {
    ...s3,
    ...fan(['o1', 'o3', 'o4', 'o5', 'o6'], -1, false),
    o2: { x: SLOT_1.x + 16, y: SLOT_1.y + 20, r: 9, up: true, z: 31 },
  }
  const s5 = {
    ...s4,
    ...fan(['y3', 'y4', 'y5', 'y6'], 1, true),
    ...fan(['o1', 'o3', 'o4', 'o6'], -1, false),
    y2: { x: SLOT_2.x, y: SLOT_2.y, r: 3, up: true, z: 32 },
    o5: { x: SLOT_2.x + 16, y: SLOT_2.y + 20, r: 11, up: true, z: 33 },
  }
  const s6 = {
    ...s5,
    y1: { x: DISCARD.x - 4, y: DISCARD.y + 2, r: -14, up: false, z: 40 },
    o2: { x: DISCARD.x + 3, y: DISCARD.y - 3, r: 7, up: false, z: 41 },
    y2: { x: DISCARD.x - 2, y: DISCARD.y + 4, r: 19, up: false, z: 42 },
    o5: { x: DISCARD.x + 2, y: DISCARD.y - 1, r: -5, up: false, z: 43 },
  }
  const s7 = {
    ...s6,
    ...fan(['y3', 'y4', 'y5', 'y6', 'y7', 'y8'], 1, true),
    ...fan(['o1', 'o3', 'o4', 'o6', 'o7', 'o8'], -1, false),
  }
  return [s0, s1, s2, s3, s4, s5, s6, s7]
}

const states = buildStates()

const sectionEl = ref<HTMLElement | null>(null)
const tableWrap = ref<HTMLElement | null>(null)
const tableEl = ref<HTMLElement | null>(null)
const step = ref(0)
const tableScale = ref(1)
const current = computed(() => STEPS[step.value])

let timeline: Timeline | null = null
let resizeObserver: ResizeObserver | null = null

function poseProps(p: Pose) {
  return { x: p.x - 40, y: p.y - 58, rotate: p.r, rotateY: p.up ? 0 : 180, zIndex: p.z }
}

function buildTimeline() {
  const root = tableEl.value!
  const el = (id: string) => root.querySelector<HTMLElement>(`[data-card="${id}"]`)!

  cards.forEach((c) => utils.set(el(c.id), poseProps(states[0][c.id])))
  utils.set(root.querySelector('.bito-word')!, { opacity: 0, scale: 0.8 })

  timeline = createTimeline({
    autoplay: onScroll({
      target: sectionEl.value!,
      enter: 'top top',
      leave: 'bottom bottom',
      sync: 0.15,
    }),
    defaults: { duration: MOVE_MS, ease: 'inOut(3)' },
    onUpdate: (tl) => {
      step.value = Math.min(STEPS.length - 1, Math.floor(tl.currentTime / STEP_MS))
    },
  })

  for (let s = 0; s < states.length - 1; s++) {
    const from = states[s]
    const to = states[s + 1]
    let order = 0
    for (const c of cards) {
      const a = from[c.id]
      const b = to[c.id]
      if (a.x === b.x && a.y === b.y && a.r === b.r && a.up === b.up && a.z === b.z) continue
      // при раздаче карты летят по очереди, в остальных шагах почти одновременно
      const delay = s === 0 ? order * 45 : order * 30
      timeline.add(el(c.id), { ...poseProps(b), duration: s === 0 ? 520 : MOVE_MS }, s * STEP_MS + delay)
      order++
    }
  }

  const word = root.querySelector('.bito-word')!
  timeline
    .add(word, { opacity: 1, scale: 1, duration: 500, ease: 'outBack(1.6)' }, 5 * STEP_MS + 450)
    .add(word, { opacity: 0, y: -18, duration: 400 }, 6 * STEP_MS + 100)
    // держим последний кадр до конца секции
    .add({ hold: 0 }, { hold: 1, duration: 1 }, states.length * STEP_MS - 1)
}

function fitTable() {
  const w = tableWrap.value?.clientWidth ?? W
  const h = tableWrap.value?.clientHeight ?? H
  tableScale.value = Math.min(w / W, h / H, 1.25)
}

function goToStep(i: number) {
  const section = sectionEl.value
  if (!section) return
  const top = section.getBoundingClientRect().top + window.scrollY
  const travel = section.offsetHeight - window.innerHeight
  window.scrollTo({ top: top + ((i + 0.55) / STEPS.length) * travel, behavior: 'smooth' })
}

onMounted(() => {
  fitTable()
  resizeObserver = new ResizeObserver(fitTable)
  if (tableWrap.value) resizeObserver.observe(tableWrap.value)
  buildTimeline()
})

onUnmounted(() => {
  resizeObserver?.disconnect()
  timeline?.revert()
})
</script>

<template>
  <section id="how-to-play" ref="sectionEl" class="htp" :style="{ height: `calc(100vh + ${STEPS.length * 75}vh)` }">
    <div class="htp-pin">
      <div class="htp-copy">
        <p class="eyebrow">How to Play</p>
        <div class="step-head">
          <span class="step-num">{{ String(step + 1).padStart(2, '0') }}</span>
          <span class="step-total">/ {{ String(STEPS.length).padStart(2, '0') }}</span>
        </div>
        <Transition name="caption" mode="out-in">
          <div :key="step" class="caption">
            <h2 class="caption-title">{{ current.title }}</h2>
            <p class="caption-text">{{ current.text }}</p>
          </div>
        </Transition>

        <ol class="rail" aria-label="Steps">
          <li v-for="(s, i) in STEPS" :key="s.title">
            <button type="button" class="rail-btn" :class="{ active: i === step, done: i < step }" @click="goToStep(i)">
              <span class="rail-dot"></span>
              <span class="rail-label">{{ s.title }}</span>
            </button>
          </li>
        </ol>

        <button type="button" class="pill-btn play" @click="requestJoin">Take a seat</button>
      </div>

      <div ref="tableWrap" class="htp-stage">
        <div
          ref="tableEl"
          class="table"
          :style="{ width: `${W}px`, height: `${H}px`, transform: `translate(-50%, -50%) scale(${tableScale})` }"
        >
          <div class="felt"></div>
          <span class="zone zone--deck">Deck</span>
          <span class="zone zone--bito">Bito</span>
          <span class="zone zone--you">You</span>
          <span class="zone zone--opp">Opponent</span>
          <div class="slot" :style="{ left: `${SLOT_1.x - 46}px`, top: `${SLOT_1.y - 64}px` }"></div>
          <div class="slot" :style="{ left: `${SLOT_2.x - 46}px`, top: `${SLOT_2.y - 64}px` }"></div>
          <div class="slot slot--bito" :style="{ left: `${DISCARD.x - 46}px`, top: `${DISCARD.y - 64}px` }"></div>

          <!-- Неподвижная часть колоды -->
          <div
            v-for="i in 4"
            :key="`deck-${i}`"
            class="demo-card"
            :style="{ transform: `translate(${DECK.x - 40}px, ${DECK.y - 58 + 3 - i}px) rotateY(180deg)`, zIndex: 2 + i }"
          >
            <PaperCard />
          </div>

          <div v-for="c in cards" :key="c.id" :data-card="c.id" class="demo-card">
            <PaperCard :rank="c.rank" :suit="c.suit" />
          </div>

          <div class="bito-word" :style="{ left: `${DISCARD.x}px`, top: `${DISCARD.y - 104}px` }">Бито!</div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.htp {
  position: relative;
}

.htp-pin {
  position: sticky;
  top: 0;
  height: 100vh;
  height: 100svh;
  box-sizing: border-box;
  max-width: 1280px;
  margin: 0 auto;
  padding: 96px 48px 40px;
  display: grid;
  grid-template-columns: 380px 1fr;
  gap: 48px;
  align-items: center;
}

.eyebrow {
  margin: 0 0 18px;
  font-size: 13px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #8a857a;
}

.step-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 8px;
  font-variant-numeric: tabular-nums;
}

.step-num {
  font-size: 22px;
  color: #1b1916;
}

.step-total {
  font-size: 15px;
  color: #9a958a;
}

.caption {
  min-height: 210px;
}

.caption-title {
  margin: 0 0 16px;
  font-size: 52px;
  font-weight: 400;
  line-height: 1.02;
  letter-spacing: -0.03em;
  color: #36322c;
}

.caption-text {
  margin: 0;
  max-width: 340px;
  font-size: 18px;
  line-height: 27px;
  letter-spacing: -0.01em;
  color: #57534b;
}

.caption-enter-active,
.caption-leave-active {
  transition:
    opacity 0.32s ease,
    transform 0.32s ease;
}

.caption-enter-from {
  opacity: 0;
  transform: translateY(14px);
}

.caption-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

.rail {
  list-style: none;
  margin: 28px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.rail-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 5px 0;
  border: 0;
  background: none;
  font: inherit;
  font-size: 14px;
  color: #a39e92;
  cursor: pointer;
  transition: color 0.25s;
}

.rail-btn:hover,
.rail-btn.done {
  color: #6b675e;
}

.rail-btn.active {
  color: #1b1916;
}

.rail-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.55;
  transition:
    transform 0.25s,
    opacity 0.25s;
}

.rail-btn.active .rail-dot {
  opacity: 1;
  transform: scale(1.5);
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
  margin-top: 28px;
  height: 46px;
  padding: 0 26px;
}

.htp-stage {
  position: relative;
  height: 100%;
  min-height: 0;
}

.table {
  position: absolute;
  left: 50%;
  top: 50%;
  transform-origin: center;
  perspective: 1400px;
}

.felt {
  position: absolute;
  inset: 0;
  border-radius: 140px;
  background:
    radial-gradient(ellipse at 50% 45%, rgba(255, 255, 255, 0.65), rgba(255, 255, 255, 0) 70%),
    rgba(236, 233, 225, 0.55);
  box-shadow:
    0 0 0 1.5px #dcd7cc,
    0 0 0 10px rgba(220, 215, 204, 0.35),
    0 40px 80px -40px rgba(80, 72, 56, 0.25);
}

.zone {
  position: absolute;
  font-size: 11px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: #a39e92;
}

.zone--deck {
  left: 72px;
  top: 312px;
}

.zone--bito {
  left: 611px;
  top: 312px;
}

.zone--you {
  left: 50%;
  bottom: -30px;
  transform: translateX(-50%);
}

.zone--opp {
  left: 50%;
  top: -30px;
  transform: translateX(-50%);
}

.slot {
  position: absolute;
  width: 92px;
  height: 128px;
  box-sizing: border-box;
  border: 1.5px dashed rgba(170, 160, 140, 0.45);
  border-radius: 11px;
}

.slot--bito {
  border-style: solid;
  border-color: rgba(170, 160, 140, 0.3);
  background: rgba(220, 214, 202, 0.25);
}

.demo-card {
  position: absolute;
  left: 0;
  top: 0;
  width: 80px;
  height: 116px;
  transform-style: preserve-3d;
  will-change: transform;
}

.bito-word {
  position: absolute;
  z-index: 60;
  transform-origin: center;
  translate: -50% 0;
  font-family: Georgia, 'Times New Roman', serif;
  font-size: 34px;
  font-style: italic;
  color: #36322c;
  white-space: nowrap;
}

@media (max-width: 900px) {
  .htp-pin {
    grid-template-columns: 1fr;
    grid-template-rows: auto 1fr;
    gap: 8px;
    padding: 84px 20px 20px;
  }

  .caption {
    min-height: 150px;
  }

  .caption-title {
    font-size: 34px;
    margin-bottom: 10px;
  }

  .caption-text {
    font-size: 16px;
    line-height: 23px;
  }

  .rail,
  .play {
    display: none;
  }
}
</style>
