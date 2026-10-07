<script setup lang="ts">
import { ref } from 'vue'
import { animate } from 'animejs'
import { useReveal } from '../../composables/useReveal'

const ITEMS = [
  {
    q: 'What is Durak?',
    a: 'The most popular card game across Russia and much of Eastern Europe. You don’t try to win the most — you try not to be the last one left holding cards. That player is the durak, the fool.',
  },
  {
    q: 'Is there real money involved?',
    a: 'No. Chips are play money. Every new account starts with 2,500 of them, and there’s no way to cash in or cash out.',
  },
  {
    q: 'Which rules do you play?',
    a: 'Throw-in (podkidnoy) and Transfer (perevodnoy), with 24, 36 or 52-card decks and two to six players at a table.',
  },
  {
    q: 'What do I need to sign up?',
    a: 'A username, an email and a password. We send a 6-digit code to your email to make sure it’s really yours.',
  },
  {
    q: 'What if my internet drops mid-game?',
    a: 'Your seat is held for 30 seconds. Reconnect in time and you pick up exactly where you left off, with your hand intact.',
  },
]

const rootEl = ref<HTMLElement | null>(null)
const open = ref<number | null>(0)

useReveal(rootEl)

function toggle(i: number) {
  open.value = open.value === i ? null : i
}

// Раскрытие ответа: анимируем высоту от текущей к целевой
function onEnter(el: Element, done: () => void) {
  const node = el as HTMLElement
  const h = node.scrollHeight
  animate(node, {
    height: [0, h],
    opacity: [0, 1],
    duration: 420,
    ease: 'out(3)',
    onComplete: () => {
      node.style.height = ''
      done()
    },
  })
}
function onLeave(el: Element, done: () => void) {
  const node = el as HTMLElement
  animate(node, { height: [node.offsetHeight, 0], opacity: 0, duration: 300, ease: 'inOut(2)', onComplete: done })
}
</script>

<template>
  <section id="faq" ref="rootEl" class="faq">
    <div class="head">
      <p class="eyebrow" data-reveal>FAQ</p>
      <h2 class="title" data-reveal>Before you sit down.</h2>
    </div>

    <ul class="list">
      <li v-for="(item, i) in ITEMS" :key="item.q" class="item" :class="{ open: open === i }" data-reveal>
        <button type="button" class="q" :aria-expanded="open === i" @click="toggle(i)">
          <span>{{ item.q }}</span>
          <span class="plus" aria-hidden="true"></span>
        </button>
        <Transition :css="false" @enter="onEnter" @leave="onLeave">
          <div v-if="open === i" class="a-wrap">
            <p class="a">{{ item.a }}</p>
          </div>
        </Transition>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.faq {
  max-width: 1280px;
  margin: 0 auto;
  padding: 120px 48px;
  display: grid;
  grid-template-columns: 440px 1fr;
  gap: 56px;
  align-items: start;
}

.eyebrow {
  margin: 0 0 18px;
  font-size: 13px;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: #8a857a;
}

.title {
  margin: 0;
  font-size: 56px;
  font-weight: 400;
  line-height: 1;
  letter-spacing: -0.03em;
  color: #36322c;
}

.list {
  list-style: none;
  margin: 0;
  padding: 0;
  border-top: 1.5px solid #e1dcd1;
}

.item {
  border-bottom: 1.5px solid #e1dcd1;
}

.q {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  padding: 24px 0;
  border: 0;
  background: none;
  font: inherit;
  font-size: 20px;
  letter-spacing: -0.015em;
  text-align: left;
  color: #1b1916;
  cursor: pointer;
}

.plus {
  position: relative;
  flex: none;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  box-shadow: 0 0 0 1.5px #dcd7cc;
  transition:
    transform 0.35s cubic-bezier(0.3, 0.7, 0.2, 1),
    background-color 0.25s;
}

.plus::before,
.plus::after {
  content: '';
  position: absolute;
  left: 50%;
  top: 50%;
  width: 11px;
  height: 1.5px;
  margin: -0.75px 0 0 -5.5px;
  background: #36322c;
}

.plus::after {
  transform: rotate(90deg);
}

.item.open .plus {
  transform: rotate(45deg);
  background: #e4ded2;
}

.a-wrap {
  overflow: hidden;
}

.a {
  margin: 0;
  padding: 0 60px 26px 0;
  font-size: 16.5px;
  line-height: 25px;
  color: #57534b;
}

@media (max-width: 1000px) {
  .faq {
    grid-template-columns: 1fr;
    padding: 90px 20px;
    gap: 28px;
  }

  .title {
    font-size: 40px;
  }

  .q {
    font-size: 17px;
  }

  .a {
    padding-right: 0;
  }
}
</style>
