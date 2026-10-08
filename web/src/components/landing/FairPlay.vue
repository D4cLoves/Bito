<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { animate, createTimeline, stagger } from 'animejs'
import type { JSAnimation, Timeline } from 'animejs'
import { useReveal } from '../../composables/useReveal'

// Только то, что действительно делает сервер (internal/game, internal/delivery/ws).
const TILES = [
  {
    id: 'shuffle',
    title: 'Тасует crypto/rand',
    text: 'Каждую колоду сервер тасует криптостойким алгоритмом Фишера — Йетса. Угадать раскладку нельзя.',
  },
  {
    id: 'hidden',
    title: 'Ваши карты видите только вы',
    text: 'Сервер присылает каждому только его карты. Соперники знают, сколько их у вас, но не какие.',
  },
  {
    id: 'timer',
    title: '20 секунд на ход',
    text: 'Таймер не даёт партии зависнуть. Не успели — сервер сам сделает безопасный ход.',
  },
  {
    id: 'reconnect',
    title: '30 секунд, чтобы вернуться',
    text: 'Пропала связь? Место ждёт вас полминуты, и только потом партия засчитывается как сдача.',
  },
]

const rootEl = ref<HTMLElement | null>(null)
const seconds = ref(20)
const running: Array<JSAnimation | Timeline> = []

useReveal(rootEl)

onMounted(() => {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  const root = rootEl.value!
  const q = (s: string) => root.querySelectorAll<HTMLElement>(s)

  // Тасовка: колода делится пополам и «врезается» обратно
  const shuffle = createTimeline({ loop: true, loopDelay: 700, defaults: { duration: 420, ease: 'inOut(3)' } })
  shuffle
    .add(q('.sh-card:nth-child(odd)'), { x: -30, rotate: -8, delay: stagger(40) })
    .add(q('.sh-card:nth-child(even)'), { x: 30, rotate: 8, delay: stagger(40) }, '<<')
    .add(q('.sh-card'), { x: 0, rotate: 0, y: [{ to: -6 }, { to: 0 }], delay: stagger(55, { from: 'center' }) }, '+=150')
  running.push(shuffle)

  // Скрытая рука: чужая карта пытается приоткрыться и захлопывается
  const peek = createTimeline({ loop: true, loopDelay: 900, defaults: { ease: 'inOut(3)' } })
  peek
    .add(q('.hd-peek'), { rotateY: [180, 140], duration: 500 })
    .add(q('.hd-lock'), { opacity: [0, 1], scale: [0.6, 1], duration: 300, ease: 'outBack(2)' }, '-=150')
    .add(q('.hd-peek'), { rotateY: 180, duration: 400 }, '+=250')
    .add(q('.hd-lock'), { opacity: 0, duration: 300 }, '<<')
  running.push(peek)

  // Таймер хода: кольцо и цифры за 4 секунды «прокручивают» 20 секунд
  const counter = { s: 20 }
  running.push(
    animate(q('.tm-ring'), { strokeDashoffset: [0, 151], duration: 4000, ease: 'linear', loop: true }),
    animate(counter, {
      s: [20, 0],
      duration: 4000,
      ease: 'linear',
      loop: true,
      onUpdate: () => {
        seconds.value = Math.ceil(counter.s)
      },
    }),
  )

  // Переподключение: дуги сигнала загораются по очереди
  running.push(
    animate(q('.rc-arc'), {
      opacity: [0.15, 1],
      duration: 500,
      delay: stagger(220),
      loop: true,
      loopDelay: 600,
      alternate: true,
      ease: 'inOut(2)',
    }),
  )
})

onUnmounted(() => running.forEach((a) => a.revert()))
</script>

<template>
  <section id="fair-play" ref="rootEl" class="fair">
    <div class="head">
      <p class="eyebrow" data-reveal>Честная игра</p>
      <h2 class="title" data-reveal>Никаких подтасовок.</h2>
      <p class="lead" data-reveal>Правила живут на сервере, а не в вашем браузере. Вот что это значит за столом.</p>
    </div>

    <div class="grid">
      <article v-for="t in TILES" :key="t.id" class="tile" data-reveal>
        <div class="art" aria-hidden="true">
          <!-- Тасовка -->
          <div v-if="t.id === 'shuffle'" class="sh">
            <span v-for="i in 6" :key="i" class="sh-card"></span>
          </div>

          <!-- Скрытая рука -->
          <div v-else-if="t.id === 'hidden'" class="hd">
            <span class="hd-card hd-back" style="transform: translateX(-46px) rotate(-10deg)"></span>
            <span class="hd-card hd-peek">
              <span class="hd-face">♣</span>
              <span class="hd-cover"></span>
            </span>
            <span class="hd-card hd-front" style="transform: translateX(46px) rotate(10deg)">A<i>♠</i></span>
            <svg class="hd-lock" viewBox="0 0 24 24">
              <rect x="5" y="10.5" width="14" height="10" rx="2.5" />
              <path d="M8.5 10.5V8a3.5 3.5 0 0 1 7 0v2.5" />
            </svg>
          </div>

          <!-- Таймер -->
          <div v-else-if="t.id === 'timer'" class="tm">
            <svg viewBox="0 0 60 60">
              <circle cx="30" cy="30" r="24" class="tm-track" />
              <circle cx="30" cy="30" r="24" class="tm-ring" />
            </svg>
            <span class="tm-num">{{ seconds }}</span>
          </div>

          <!-- Переподключение -->
          <div v-else class="rc">
            <svg viewBox="0 0 48 40">
              <path class="rc-arc" d="M4 15a29 29 0 0 1 40 0" />
              <path class="rc-arc" d="M10.5 21.5a20 20 0 0 1 27 0" />
              <path class="rc-arc" d="M17 28a11 11 0 0 1 14 0" />
              <circle class="rc-arc" cx="24" cy="34" r="2.6" />
            </svg>
            <span class="rc-time">0:30</span>
          </div>
        </div>
        <h3 class="tile-title">{{ t.title }}</h3>
        <p class="tile-text">{{ t.text }}</p>
      </article>
    </div>
  </section>
</template>

<style scoped>
.fair {
  max-width: 1280px;
  margin: 0 auto;
  padding: 120px 48px;
}

.head {
  max-width: 620px;
  margin-bottom: 48px;
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
  margin: 0;
  font-size: 18px;
  line-height: 27px;
  color: #57534b;
}

.grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 18px;
}

.tile {
  padding: 22px 22px 26px;
  border-radius: 22px;
  background: rgba(250, 249, 245, 0.6);
  box-shadow:
    0 0 0 1.5px #e1dcd1,
    0 24px 50px -30px rgba(80, 72, 56, 0.25);
}

.art {
  position: relative;
  height: 150px;
  margin-bottom: 22px;
  border-radius: 14px;
  background: radial-gradient(ellipse at 50% 40%, #fbfaf6, #ece8de);
  display: grid;
  place-items: center;
  overflow: hidden;
}

.tile-title {
  margin: 0 0 8px;
  font-size: 19px;
  font-weight: 500;
  letter-spacing: -0.015em;
  color: #1b1916;
}

.tile-text {
  margin: 0;
  font-size: 15px;
  line-height: 22px;
  color: #6b675e;
}

/* Общая рубашка мини-карт */
.sh-card,
.hd-back,
.hd-cover {
  background:
    repeating-linear-gradient(45deg, transparent 0 4px, rgba(160, 148, 124, 0.2) 4px 5px),
    #e7e1d4;
}

.sh {
  position: relative;
  width: 52px;
  height: 74px;
}

.sh-card {
  position: absolute;
  inset: 0;
  border-radius: 6px;
  box-shadow: 0 0 0 1px rgba(160, 150, 132, 0.5);
}

.sh-card:nth-child(1) { translate: 0 5px; }
.sh-card:nth-child(2) { translate: 0 4px; }
.sh-card:nth-child(3) { translate: 0 3px; }
.sh-card:nth-child(4) { translate: 0 2px; }
.sh-card:nth-child(5) { translate: 0 1px; }

.hd {
  position: relative;
  width: 52px;
  height: 74px;
  perspective: 600px;
}

.hd-card {
  position: absolute;
  inset: 0;
  border-radius: 6px;
  box-shadow: 0 0 0 1px rgba(160, 150, 132, 0.5);
}

.hd-front {
  background: #fbfaf6;
  padding: 6px 7px;
  box-sizing: border-box;
  font-family: Georgia, serif;
  font-size: 14px;
  line-height: 1;
  color: #2b2824;
  display: flex;
  flex-direction: column;
}

.hd-front i {
  font-style: normal;
  font-size: 12px;
}

.hd-peek {
  z-index: 2;
  transform: rotateY(180deg);
  transform-style: preserve-3d;
  box-shadow: none;
}

.hd-face,
.hd-cover {
  position: absolute;
  inset: 0;
  border-radius: 6px;
  backface-visibility: hidden;
  box-shadow: 0 0 0 1px rgba(160, 150, 132, 0.5);
}

.hd-face {
  display: grid;
  place-items: center;
  background: #fbfaf6;
  font-size: 22px;
  color: #2b2824;
}

.hd-cover {
  transform: rotateY(180deg);
}

.hd-lock {
  position: absolute;
  z-index: 3;
  left: 50%;
  top: -22px;
  width: 22px;
  height: 22px;
  margin-left: -11px;
  opacity: 0;
  fill: none;
  stroke: #a1473f;
  stroke-width: 1.8;
  stroke-linecap: round;
}

.tm {
  position: relative;
  width: 84px;
  height: 84px;
}

.tm svg {
  width: 100%;
  height: 100%;
  transform: rotate(-90deg);
}

.tm-track,
.tm-ring {
  fill: none;
  stroke-width: 4;
}

.tm-track {
  stroke: #e1dcd1;
}

.tm-ring {
  stroke: #36322c;
  stroke-linecap: round;
  stroke-dasharray: 151;
}

.tm-num {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  font-size: 24px;
  color: #1b1916;
  font-variant-numeric: tabular-nums;
}

.rc {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
}

.rc svg {
  width: 64px;
  height: 54px;
  fill: none;
  stroke: #36322c;
  stroke-width: 3;
  stroke-linecap: round;
}

.rc circle {
  fill: #36322c;
  stroke: none;
}

.rc-time {
  font-size: 14px;
  color: #6b675e;
  font-variant-numeric: tabular-nums;
}

@media (max-width: 1100px) {
  .grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 640px) {
  .fair {
    padding: 90px 20px;
  }

  .title {
    font-size: 40px;
  }

  .grid {
    grid-template-columns: 1fr;
  }
}
</style>
