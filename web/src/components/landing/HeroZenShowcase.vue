<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { animate, createTimeline, splitText, stagger, utils } from 'animejs'
import type { JSAnimation, Timeline } from 'animejs'
import { useAuthStore } from '../../stores/auth'
import { useGameSound } from '../../composables/useGameSound'
import { LANDING_SECTIONS, onJoinRequest, scrollToSection } from '../../composables/useLanding'
import JoinCard from './JoinCard.vue'

// Сцена свёрстана в координатах макета 1376×768 и целиком масштабируется под окно.
// На узких экранах (compact) элементы уходят в обычный поток.
const STAGE_W = 1376
const STAGE_H = 768
const COMPACT_BELOW = 900

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const sound = useGameSound()

const scale = ref(1)
const compact = ref(false)
const initialMode = route.query.auth === 'login' ? 'login' : 'join'

const heroEl = ref<HTMLElement | null>(null)
const joinCard = ref<InstanceType<typeof JoinCard> | null>(null)
const secondsText = ref('20')

function fit() {
  const w = window.innerWidth
  const h = window.innerHeight
  compact.value = w < COMPACT_BELOW || w / h < 1.15
  scale.value = Math.min(w / STAGE_W, h / STAGE_H)
}

// --------------------------------------------------------------------------
// Параллакс: дама ближе к зрителю, поэтому смещается сильнее туза
// --------------------------------------------------------------------------
const pointerX = ref(0)
const pointerY = ref(0)

function handleMouseMove(e: MouseEvent) {
  pointerX.value = e.clientX / window.innerWidth - 0.5
  pointerY.value = e.clientY / window.innerHeight - 0.5
}

function handleMouseLeave() {
  pointerX.value = 0
  pointerY.value = 0
}

const aceParallax = computed(() => ({
  transform: `translate3d(${pointerX.value * 10}px, ${pointerY.value * 8}px, 0)`,
}))
const queenParallax = computed(() => ({
  transform: `translate3d(${pointerX.value * 14}px, ${pointerY.value * 11}px, 0) rotate(${pointerX.value * 0.8}deg)`,
}))

// Регистрация и так открыта в карточке справа, поэтому в шапке — вход
function handleHeaderButton() {
  sound.playClick()
  if (authStore.isAuthenticated) {
    router.push('/lobby')
    return
  }
  joinCard.value?.openLogin()
}

// --------------------------------------------------------------------------
// Анимации: интро, парение карт, дрейф облаков, счётчики
// --------------------------------------------------------------------------
const running: Array<JSAnimation | Timeline> = []

function q<T extends Element = HTMLElement>(sel: string) {
  return heroEl.value!.querySelector<T>(sel)!
}
function qa(sel: string) {
  return heroEl.value!.querySelectorAll<HTMLElement>(sel)
}

function countUp(target: { value: string }, to: number, format: (v: number) => string, delay: number) {
  const counter = { v: 0 }
  target.value = format(0)
  running.push(
    animate(counter, {
      v: to,
      duration: 1500,
      delay,
      ease: 'out(3)',
      onUpdate: () => {
        target.value = format(counter.v)
      },
    }),
  )
}

function startAmbient() {
  running.push(
    // Карты парят в одной фазе: под дамой у туза нет настоящих пикселей,
    // поэтому взаимный сдвиг держим в пределах 3–4px
    animate(q('.card-float--ace'), {
      y: [0, -10],
      rotate: [0, -0.6],
      duration: 3600,
      ease: 'inOutSine',
      loop: true,
      alternate: true,
    }),
    animate(q('.card-float--queen'), {
      y: [0, -12.5],
      rotate: [0, 0.8],
      duration: 3600,
      ease: 'inOutSine',
      loop: true,
      alternate: true,
    }),
    animate(q('.cards-shadow'), {
      scaleX: [1, 0.9],
      opacity: [1, 0.7],
      duration: 3500,
      ease: 'inOutSine',
      loop: true,
      alternate: true,
    }),
    animate(q('.mist--near'), { x: [0, 26], duration: 24000, ease: 'inOutSine', loop: true, alternate: true }),
    animate(q('.mist--far'), { x: [0, -18], duration: 30000, ease: 'inOutSine', loop: true, alternate: true }),
    animate(q('.mist--sun'), {
      scale: [1, 1.06],
      opacity: [0.85, 1],
      duration: 7000,
      ease: 'inOutSine',
      loop: true,
      alternate: true,
    }),
  )
}

function playIntro() {
  const header = qa('.header .logo, .header .nav a, .header .header-btn')
  const title = splitText(q('.title'), { words: { wrap: 'clip' } })
  // Строки anime.js делит асинхронно, а слова — сразу, поэтому подзаголовок режем по словам
  const subtitle = splitText(q('.subtitle'), { words: { wrap: 'clip' } })
  const join = q('.join')
  const joinItems = join.querySelectorAll<HTMLElement>('[data-anim]')

  utils.set(q('.frame'), { opacity: 0, scale: 0.985 })
  utils.set(header, { opacity: 0, y: -14 })
  utils.set(title.words, { y: '105%' })
  utils.set(subtitle.words, { y: '105%' })
  utils.set(q('.rating'), { opacity: 0, y: 16 })
  utils.set(q('.card-float--ace'), { opacity: 0, y: 90, rotate: -12 })
  utils.set(q('.card-float--queen'), { opacity: 0, y: 120, rotate: 16 })
  utils.set(q('.cards-shadow'), { opacity: 0, scaleX: 0.4 })
  utils.set(join, { opacity: 0, y: 34 })
  utils.set(joinItems, { opacity: 0, y: 14 })

  const tl = createTimeline({ defaults: { ease: 'out(4)', duration: 900 } })
  tl.add(q('.frame'), { opacity: 1, scale: 1, duration: 1200 }, 100)
    .add(header, { opacity: 1, y: 0, delay: stagger(60) }, 250)
    .add(title.words, { y: '0%', duration: 1100, delay: stagger(90) }, 350)
    .add(subtitle.words, { y: '0%', duration: 800, delay: stagger(22) }, 900)
    .add(q('.rating'), { opacity: 1, y: 0, duration: 800 }, 1150)
    .add(q('.cards-shadow'), { opacity: 1, scaleX: 1, duration: 1300 }, 650)
    .add(q('.card-float--ace'), { opacity: 1, y: 0, rotate: 0, duration: 1400, ease: 'outBack(1.15)' }, 600)
    .add(q('.card-float--queen'), { opacity: 1, y: 0, rotate: 0, duration: 1400, ease: 'outBack(1.15)' }, 760)
    .add(join, { opacity: 1, y: 0, duration: 1000 }, 850)
    .add(joinItems, { opacity: 1, y: 0, duration: 700, delay: stagger(70) }, 1000)
    .call(() => {
      title.revert()
      subtitle.revert()
      // Снимаем инлайн-трансформы, чтобы снова работали :hover/:active из CSS
      for (const el of [q('.frame'), q('.rating'), join, ...header, ...joinItems]) {
        el.style.removeProperty('transform')
        el.style.removeProperty('opacity')
      }
      startAmbient()
    }, 2300)

  running.push(tl)
  countUp(secondsText, 20, (v) => String(Math.round(v)), 1250)
  joinCard.value?.countChips(1100, false)
}

let offJoin: (() => void) | null = null

onMounted(async () => {
  fit()
  window.addEventListener('resize', fit)
  offJoin = onJoinRequest(() => joinCard.value?.openJoin())
  await nextTick()
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  playIntro()
})

onUnmounted(() => {
  window.removeEventListener('resize', fit)
  offJoin?.()
  running.forEach((a) => a.revert())
})
</script>

<template>
  <div ref="heroEl" class="hero" :class="{ compact }" @mousemove="handleMouseMove" @mouseleave="handleMouseLeave">
    <div class="stage" :style="compact ? undefined : { transform: `translate(-50%, -50%) scale(${scale})` }">
      <!-- Туман и облака: слои под рамкой и за её пределами -->
      <svg class="mist mist--base" viewBox="-688 -384 2752 1536" preserveAspectRatio="xMidYMid slice" aria-hidden="true">
        <defs>
          <!-- Солнечное гало за пиком облачной гряды -->
          <radialGradient id="sun" cx="1000" cy="200" r="300" gradientUnits="userSpaceOnUse">
            <stop offset="0" stop-color="#f6ebd3" stop-opacity="1" />
            <stop offset="0.4" stop-color="#f6eedc" stop-opacity="0.75" />
            <stop offset="0.75" stop-color="#f5f1e6" stop-opacity="0.3" />
            <stop offset="1" stop-color="#f5f4ef" stop-opacity="0" />
          </radialGradient>

          <!-- Клубы облаков: размытые круги с рваными краями -->
          <filter id="puff" x="-688" y="-384" width="2752" height="1536" filterUnits="userSpaceOnUse">
            <feTurbulence type="fractalNoise" baseFrequency="0.009" numOctaves="4" seed="4" result="n" />
            <feDisplacementMap in="SourceGraphic" in2="n" scale="70" xChannelSelector="R" yChannelSelector="G" />
            <feGaussianBlur stdDeviation="11" />
          </filter>
          <filter id="puff-soft" x="-688" y="-384" width="2752" height="1536" filterUnits="userSpaceOnUse">
            <feTurbulence type="fractalNoise" baseFrequency="0.006" numOctaves="3" seed="9" result="n" />
            <feDisplacementMap in="SourceGraphic" in2="n" scale="90" xChannelSelector="R" yChannelSelector="G" />
            <feGaussianBlur stdDeviation="34" />
          </filter>
        </defs>
        <rect x="-688" y="-384" width="2752" height="1536" fill="#f5f4ef" />
      </svg>

      <svg class="mist mist--sun" viewBox="-688 -384 2752 1536" preserveAspectRatio="xMidYMid slice" aria-hidden="true">
        <rect x="-688" y="-384" width="2752" height="1536" fill="url(#sun)" />
      </svg>

      <svg class="mist mist--far" viewBox="-688 -384 2752 1536" preserveAspectRatio="xMidYMid slice" aria-hidden="true">
        <!-- Дальний слой: мягкая дымка под грядой -->
        <g filter="url(#puff-soft)" fill="#e4e4df" fill-opacity="0.6">
          <ellipse cx="420" cy="560" rx="240" ry="60" />
          <ellipse cx="800" cy="450" rx="220" ry="110" />
          <ellipse cx="1100" cy="330" rx="180" ry="80" />
          <ellipse cx="950" cy="790" rx="600" ry="110" />
        </g>

        <!-- Гряда: от подзаголовка к пику за формой -->
        <g filter="url(#puff)" fill="#dfdfda" fill-opacity="0.75">
          <circle cx="330" cy="545" r="50" />
          <circle cx="430" cy="505" r="55" />
          <circle cx="530" cy="520" r="55" />
          <circle cx="640" cy="475" r="65" />
          <circle cx="760" cy="405" r="75" />
          <circle cx="870" cy="330" r="70" />
          <circle cx="960" cy="290" r="55" />
          <circle cx="1050" cy="258" r="50" />
          <circle cx="1130" cy="265" r="52" />
          <circle cx="1210" cy="305" r="50" />
        </g>

        <!-- Масса под пиком, просвечивает сквозь форму -->
        <g filter="url(#puff-soft)" fill="#d5d6d0" fill-opacity="0.7">
          <ellipse cx="1130" cy="470" rx="170" ry="120" />
        </g>
      </svg>

      <svg class="mist mist--near" viewBox="-688 -384 2752 1536" preserveAspectRatio="xMidYMid slice" aria-hidden="true">
        <!-- Правая тёмная масса -->
        <g filter="url(#puff)" fill="#cbccc6" fill-opacity="0.9">
          <circle cx="1360" cy="440" r="60" />
          <circle cx="1330" cy="520" r="65" />
          <circle cx="1440" cy="500" r="100" />
        </g>

        <!-- Нижняя плотная полоса -->
        <g filter="url(#puff)" fill="#cacbc5" fill-opacity="0.9">
          <circle cx="600" cy="815" r="80" />
          <circle cx="740" cy="790" r="75" />
          <circle cx="880" cy="770" r="80" />
          <circle cx="1010" cy="750" r="70" />
          <circle cx="1130" cy="760" r="80" />
          <circle cx="1260" cy="735" r="85" />
          <circle cx="1390" cy="715" r="95" />
          <circle cx="950" cy="880" r="150" />
        </g>

        <!-- Подсветка верхушек облаков -->
        <g filter="url(#puff)" fill="#f3f2ed" fill-opacity="0.75">
          <circle cx="1060" cy="225" r="32" />
          <circle cx="950" cy="262" r="30" />
          <circle cx="760" cy="350" r="40" />
          <circle cx="1330" cy="390" r="35" />
          <circle cx="1230" cy="650" r="35" />
          <circle cx="930" cy="670" r="30" />
          <circle cx="500" cy="480" r="28" />
        </g>
      </svg>

      <!-- Рамка -->
      <div class="frame" aria-hidden="true"></div>

      <!-- Шапка -->
      <header class="header">
        <router-link to="/" class="logo" aria-label="Bito">
          <svg class="logo-icon" viewBox="0 0 40 38" aria-hidden="true">
            <rect x="17.5" y="4.5" width="18" height="26" rx="3.2" transform="rotate(12 26.5 17.5)" fill="none" stroke="currentColor" stroke-width="2" />
            <rect x="4" y="5.5" width="18" height="26" rx="3.2" transform="rotate(-14 13 18.5)" fill="#f5f4ef" stroke="currentColor" stroke-width="2" />
            <path
              transform="rotate(-14 13 18.5)"
              d="M13 13.2c-1.9 2.3-4.1 3.9-4.1 6 0 1.4 1 2.3 2.2 2.3.8 0 1.4-.4 1.6-.9-.1 1-.5 1.9-1.2 2.6h3c-.7-.7-1.1-1.6-1.2-2.6.2.5.8.9 1.6.9 1.2 0 2.2-.9 2.2-2.3 0-2.1-2.2-3.7-4.1-6z"
              fill="currentColor"
            />
          </svg>
          <span class="logo-text">Bito</span>
        </router-link>

        <nav class="nav">
          <a v-for="s in LANDING_SECTIONS" :key="s.id" :href="`#${s.id}`" @click.prevent="scrollToSection(s.id)">
            {{ s.label }}
          </a>
        </nav>

        <button type="button" class="pill-btn header-btn" @click="handleHeaderButton">
          {{ authStore.isAuthenticated ? 'В лобби' : 'Войти' }}
        </button>
      </header>

      <!-- Заголовок и подзаголовок -->
      <div class="copy">
        <h1 class="title">Расчёт.<br />Карты.<br />Вживую.</h1>
        <p class="subtitle">Карты раздаёт случай, а&nbsp;партию выигрывает голова.</p>
        <!-- Реальные параметры сервера: таймер хода 20 с, за столом 2–6 игроков -->
        <div class="rating">
          <svg class="star" viewBox="0 0 24 24" aria-hidden="true">
            <circle cx="12" cy="13.5" r="8" fill="none" stroke="currentColor" stroke-width="2" />
            <path d="M12 9v4.8l3 1.8M9.5 2.5h5" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
          </svg>
          <span class="rating-score num"><span class="num-ghost">20&nbsp;с</span><span class="num-live">{{ secondsText }}&nbsp;с</span></span>
          <span class="rating-dot"></span>
          <span class="rating-text">на ход · 2–6 игроков<span class="rating-extra"> за столом</span></span>
        </div>
      </div>

      <!-- Карты: туз и дама отдельными слоями -->
      <div class="cards" role="img" aria-label="Туз пик и дама червей">
        <div class="cards-shadow"></div>
        <div class="card-layer" :style="aceParallax">
          <div class="card-float card-float--ace">
            <img src="/cards/card-ace.png" alt="" draggable="false" />
          </div>
        </div>
        <div class="card-layer" :style="queenParallax">
          <div class="card-float card-float--queen">
            <img src="/cards/card-queen.png" alt="" draggable="false" />
          </div>
        </div>
      </div>

      <!-- Вход и регистрация -->
      <JoinCard ref="joinCard" class="join" :initial-mode="initialMode" />
    </div>
  </div>
</template>

<style scoped>
.hero {
  --ink: #36322c;
  --ink-strong: #1b1916;
  --pill: #d7d3c8;
  --pill-hover: #cdc8bb;
  position: relative;
  width: 100%;
  height: 100vh;
  height: 100dvh;
  overflow: hidden;
  /* clip, а не hidden: не даём браузеру прокручивать сцену при фокусе полей */
  overflow: clip;
  background: #f5f4ef;
  font-family: 'Inter', -apple-system, 'Helvetica Neue', Arial, sans-serif;
  color: var(--ink-strong);
}

/* ---------------------------- Сцена 1376×768 ---------------------------- */
.stage {
  position: absolute;
  left: 50%;
  top: 50%;
  width: 1376px;
  height: 768px;
  transform-origin: center;
}

.mist {
  position: absolute;
  left: -688px;
  top: -384px;
  width: 2752px;
  height: 1536px;
  pointer-events: none;
  will-change: transform, opacity;
}

.mist--sun {
  transform-origin: 1688px 584px;
}

.frame {
  position: absolute;
  left: 37px;
  top: 11px;
  width: 1302px;
  height: 719px;
  border: 8px solid #d3d0c8;
  border-radius: 54px;
  pointer-events: none;
}

/* Шапка */
.header {
  position: absolute;
  left: 102px;
  top: 68px;
  width: 1176px;
  height: 45px;
  display: flex;
  align-items: center;
}

.logo {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #26231e;
  text-decoration: none;
}

.logo-icon {
  width: 41px;
  height: 39px;
}

.logo-text {
  font-size: 27.5px;
  letter-spacing: -0.008em;
  color: var(--ink-strong);
  line-height: 1;
  transform: translate(-1.5px, 2.3px);
}

.nav {
  display: flex;
  gap: 30px;
  margin-left: auto;
  margin-right: 32px;
  font-size: 16.6px;
  letter-spacing: -0.03em;
  transform: translateY(0.9px);
}

.nav a {
  color: var(--ink-strong);
  text-decoration: none;
  transition: opacity 0.2s;
}

.nav a:hover {
  opacity: 0.6;
}

.pill-btn {
  border: 0;
  border-radius: 999px;
  background: var(--pill);
  color: var(--ink-strong);
  font: inherit;
  font-size: 15.5px;
  letter-spacing: -0.006em;
  cursor: pointer;
  transition: background-color 0.2s, transform 0.2s;
}

.pill-btn:hover {
  background: var(--pill-hover);
}

.pill-btn:active {
  transform: scale(0.985);
}

.header-btn {
  width: 153px;
  height: 45px;
  padding: 2px 0 0;
}

/* Текст слева */
.copy {
  position: absolute;
  left: 100.5px;
  top: 187px;
}

.title {
  margin: 0;
  font-size: 97.5px;
  font-weight: 400;
  line-height: 97px;
  letter-spacing: -0.03em;
  color: var(--ink);
  white-space: nowrap;
}

.subtitle {
  width: 360px;
  margin: 39.3px 0 0 2.5px;
  font-size: 21.3px;
  line-height: 30px;
  letter-spacing: -0.012em;
  color: #47433c;
}

.rating {
  display: inline-flex;
  align-items: center;
  box-sizing: border-box;
  height: 45px;
  margin: 48.7px 0 0 3.5px;
  padding: 0 15px 0 17px;
  border: 1.5px solid #dcdbd5;
  border-radius: 999px;
  background: rgba(250, 250, 247, 0.45);
  color: var(--ink-strong);
  white-space: nowrap;
}

.star {
  width: 24px;
  height: 24px;
  color: #1d1a15;
}

.rating-score {
  margin-left: 8px;
  font-size: 22px;
  letter-spacing: -0.01em;
  transform: translateY(1.5px);
}

.rating-dot {
  width: 5px;
  height: 5px;
  margin: 0 10px 0 11px;
  border-radius: 50%;
  background: #1d1a15;
}

.rating-text {
  font-size: 17.5px;
  letter-spacing: -0.008em;
  transform: translateY(1.5px);
}

/* Карты */
/* Счётчики: невидимое итоговое значение держит ширину, живое число поверх */
.num {
  position: relative;
  display: inline-block;
}

.num-ghost {
  visibility: hidden;
}

.num-live {
  position: absolute;
  top: 0;
  right: 0;
}

/* Карты: бокс слоёв совпадает с вырезкой из макета */
.cards {
  position: absolute;
  left: 509px;
  top: 245px;
  width: 397px;
  height: 393px;
  pointer-events: none;
}

.card-layer {
  position: absolute;
  inset: 0;
  transition: transform 0.6s cubic-bezier(0.2, 0.7, 0.2, 1);
}

.card-float {
  position: absolute;
  inset: 0;
  will-change: transform;
}

.card-float img {
  display: block;
  width: 100%;
  height: 100%;
  user-select: none;
}

/* Точки вращения — центры карт */
.card-float--ace {
  transform-origin: 32% 67%;
}

.card-float--queen {
  transform-origin: 69% 38%;
}

.cards-shadow {
  position: absolute;
  left: 44px;
  top: 396px;
  width: 260px;
  height: 26px;
  border-radius: 50%;
  background: radial-gradient(closest-side, rgba(95, 92, 84, 0.42), rgba(95, 92, 84, 0));
  filter: blur(5px);
}

/* Форма: визуальный стиль внутри JoinCard, здесь только место на сцене */
.join {
  position: absolute;
  left: 941px;
  top: 251px;
  width: 336px;
}

/* ------------------------- Компактный режим ------------------------- */
.hero.compact {
  height: auto;
  min-height: 100dvh;
}

.compact .stage {
  position: relative;
  left: auto;
  top: auto;
  width: auto;
  height: auto;
  min-height: 100dvh;
  padding: 12px;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
}

/* Туман остаётся внутри первого экрана, чтобы не просвечивать под секциями ниже */
.compact .mist {
  left: 50%;
  top: 0;
  margin-left: -1376px;
}

.compact .frame {
  left: 12px;
  top: 12px;
  width: auto;
  height: auto;
  right: 12px;
  bottom: 12px;
  border-width: 6px;
  border-radius: 36px;
}

.compact .header,
.compact .copy,
.compact .cards,
.compact .join {
  position: relative;
  left: auto;
  top: auto;
}

.compact .header {
  width: auto;
  margin: 28px 28px 0;
}

.compact .nav {
  display: none;
}

.compact .header-btn {
  margin-left: auto;
  width: auto;
  padding: 0 20px;
  height: 42px;
  font-size: 15px;
}

.compact .copy {
  margin: 44px 28px 0;
}

.compact .title {
  font-size: clamp(44px, 12vw, 80px);
  line-height: 1;
  white-space: normal;
}

.compact .subtitle {
  width: auto;
  max-width: 360px;
  margin-top: 22px;
  font-size: 18px;
  line-height: 26px;
}

.compact .rating {
  margin: 24px 0 0;
  height: 42px;
  padding: 0 14px;
}

.compact .star {
  width: 20px;
  height: 20px;
}

.compact .rating-score {
  margin-left: 6px;
  font-size: 19px;
}

.compact .rating-dot {
  margin: 0 8px;
}

.compact .rating-text {
  font-size: 15px;
}

/* на телефоне хвост «за столом» не помещается в плашку */
@media (max-width: 420px) {
  .compact .rating-extra {
    display: none;
  }
}

.compact .cards {
  width: min(320px, 78vw);
  height: auto;
  aspect-ratio: 397 / 393;
  margin: 28px auto 0;
}

.compact .cards-shadow {
  left: 10%;
  top: 100%;
  width: 67%;
}

.compact .join {
  width: auto;
  max-width: 400px;
  margin: 40px 28px 36px;
  align-self: stretch;
}

@media (min-width: 480px) {
  .compact .join {
    align-self: center;
    width: calc(100% - 56px);
  }
}
</style>
