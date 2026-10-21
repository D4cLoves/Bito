<script setup lang="ts">
import { computed, ref } from 'vue'
import { useElementSize, useMediaQuery } from '@vueuse/core'
import GameCard from './GameCard.vue'
import { cardKey, getSuitSymbol, type Card, type CardPair } from '../../types/game'

const props = defineProps<{
  pairs: CardPair[]
  trumpCard: Card
  trumpSuit: number
  cardsLeft: number
  discardSize: number
  targets: Set<string>
  hint: string
}>()

defineEmits<{ (e: 'pair', pair: CardPair): void }>()

const PAIR_W = 1.24
const PAIR_H = 1.66
const GAP_X = 18
const GAP_Y = 12
const MIN_CARD = 40
const MAX_CARD = 92
const PILE_W = 124
const PILE_H = 124
const MIN_CENTER = 170

const phone = useMediaQuery('(max-width: 767px)')
const battleEl = ref<HTMLElement | null>(null)
const centerEl = ref<HTMLElement | null>(null)
const { width: battleW, height: battleH } = useElementSize(battleEl)
const { width: centerW, height: centerH } = useElementSize(centerEl)

const pileScale = computed(() => {
  if (phone.value || !battleW.value || !battleH.value) return 1
  const byWidth = (battleW.value - MIN_CENTER - 24) / (2 * PILE_W)
  const byHeight = battleH.value / PILE_H
  return Math.max(0.55, Math.min(1, byWidth, byHeight))
})

const layout = computed(() => {
  const n = Math.max(props.pairs.length, 1)
  if (!centerW.value || !centerH.value) return { card: MIN_CARD, cols: n }
  let best = { card: MIN_CARD, cols: n }
  for (let rows = 1; rows <= n; rows++) {
    const cols = Math.ceil(n / rows)
    const byWidth = (centerW.value - (cols - 1) * GAP_X) / (cols * PAIR_W)
    const byHeight = (centerH.value - (rows - 1) * GAP_Y) / (rows * PAIR_H)
    const card = Math.min(byWidth, byHeight, MAX_CARD)
    if (card > best.card) best = { card, cols }
  }
  return { card: Math.floor(best.card), cols: best.cols }
})

const pairsStyle = computed(() => {
  const { card, cols } = layout.value
  return {
    '--card': `${card}px`,
    maxWidth: `${Math.ceil(cols * card * PAIR_W + (cols - 1) * GAP_X) + 1}px`,
  }
})

const deckBacks = computed(() => (props.cardsLeft > 1 ? Math.min(5, Math.ceil((props.cardsLeft - 1) / 6)) : 0))
const redTrump = computed(() => props.trumpSuit === 2 || props.trumpSuit === 3)

function hash(key: string) {
  let h = 7
  for (const ch of key) h = (h * 31 + ch.charCodeAt(0)) % 9973
  return h
}

function toss(key: string, defend = false) {
  const h = hash(key)
  const rotate = defend ? 4 + (h % 13) : (h % 19) - 9
  const x = (h % 17) - 8
  const y = (Math.floor(h / 17) % 13) - 6
  return `translate(${x}px, ${y}px) rotate(${rotate}deg)`
}
</script>

<template>
  <div ref="battleEl" class="battle" :style="{ '--pile': pileScale }">
    <div class="pile deck">
      <div class="deck-stack" data-fly-deck>
        <GameCard v-if="cardsLeft > 0" :card="trumpCard" size="sm" class="trump-card" />
        <GameCard
          v-for="i in deckBacks"
          :key="i"
          face-down
          size="sm"
          class="deck-back"
          :style="{ transform: `translate(${(i - 1) * 1.5}px, ${-(i - 1) * 1.5}px)` }"
        />
        <div v-if="cardsLeft === 0" class="trump-only" :class="{ red: redTrump }">{{ getSuitSymbol(trumpSuit) }}</div>
      </div>
      <span class="label">{{ cardsLeft ? `${cardsLeft} в колоде` : 'колода пуста' }}</span>
    </div>

    <div ref="centerEl" class="center">
      <TransitionGroup tag="div" class="pairs" name="pair" data-fly-zone="table" :style="pairsStyle">
        <div
          v-for="pair in pairs"
          :key="cardKey(pair.attack)"
          class="pair"
          :class="{ target: targets.has(cardKey(pair.attack)) }"
          @click="$emit('pair', pair)"
        >
          <GameCard :card="pair.attack" size="md" :data-fly="cardKey(pair.attack)" :style="{ '--toss': toss(cardKey(pair.attack)) }" />
          <GameCard
            v-if="pair.defend"
            :card="pair.defend"
            size="md"
            class="defend-card"
            :data-fly="cardKey(pair.defend)"
            :style="{ '--toss': toss(cardKey(pair.defend), true) }"
          />
        </div>
      </TransitionGroup>
      <p v-if="!pairs.length" class="hint">{{ hint }}</p>
    </div>

    <div class="pile discard">
      <div class="discard-stack" :class="{ empty: !discardSize }" data-fly-discard>
        <GameCard v-if="discardSize" face-down size="sm" style="transform: rotate(14deg)" />
        <GameCard v-if="discardSize > 6" face-down size="sm" class="discard-2" />
      </div>
      <span class="label">бито: {{ discardSize }}</span>
    </div>
  </div>
</template>

<style scoped>
.battle {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  grid-template-rows: minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  width: 100%;
  height: 100%;
}

.pile {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  width: 124px;
  zoom: var(--pile, 1);
}

.deck-stack,
.discard-stack {
  position: relative;
  width: 116px;
  height: 92px;
}

.trump-card {
  position: absolute;
  left: 44px;
  top: 4px;
  transform: rotate(90deg);
}

.deck-back {
  position: absolute;
  left: 4px;
  top: 6px;
}

.trump-only {
  display: grid;
  place-items: center;
  width: 64px;
  height: 64px;
  margin: 14px auto 0;
  border-radius: 50%;
  border: 1.5px dashed var(--color-felt-line);
  font-family: var(--font-card);
  font-size: 34px;
  color: var(--color-on-felt);
}

.trump-only.red {
  color: var(--color-felt-minus);
}

.discard-stack {
  display: grid;
  place-items: center;
}

.discard-stack.empty {
  border-radius: 12px;
  border: 1.5px dashed var(--color-felt-line);
  width: 64px;
  margin: 0 16px;
}

.discard-stack :deep(.game-card) {
  position: absolute;
}

.discard-2 {
  transform: rotate(-8deg) translate(4px, 2px);
}

.label {
  font-size: 12px;
  color: var(--color-on-felt-faint);
  white-space: nowrap;
}

.center {
  display: grid;
  place-items: center;
  align-self: stretch;
  min-width: 0;
  min-height: 0;
}

.pairs {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px 18px;
}

.pair {
  position: relative;
  width: calc(var(--card) * 1.24);
  height: calc(var(--card) * 1.66);
  border-radius: 14px;
  transition: box-shadow 0.2s;
}

.pair :deep(.game-card) {
  --w: var(--card);
}

.pair.target {
  cursor: pointer;
}

.pair > :deep(.game-card:first-child) {
  transform: var(--toss);
  transition: transform 0.4s cubic-bezier(0.22, 1, 0.36, 1);
}

.pair.target > :deep(.game-card:first-child) {
  transform: translateY(-6px) var(--toss);
}

.pair.target > :deep(.game-card:first-child .face--front) {
  animation: glow 2.2s ease-in-out infinite;
}

.pair.target:hover > :deep(.game-card:first-child) {
  transform: translateY(-10px) var(--toss);
}

@keyframes glow {
  0%,
  100% {
    box-shadow:
      0 0 0 1px color-mix(in srgb, var(--color-felt-target) 45%, transparent),
      0 14px 28px -12px color-mix(in srgb, var(--color-felt-target) 55%, transparent);
  }
  50% {
    box-shadow:
      0 0 0 1.5px color-mix(in srgb, var(--color-felt-target) 75%, transparent),
      0 18px 34px -12px color-mix(in srgb, var(--color-felt-target) 80%, transparent);
  }
}

.pair-move {
  transition: transform 0.45s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.defend-card {
  position: absolute;
  left: calc(var(--card) * 0.24);
  top: calc(var(--card) * 0.24);
  transform: var(--toss);
}

.hint {
  margin: 0;
  padding: 10px 18px;
  border-radius: 999px;
  background: var(--color-felt-well);
  color: var(--color-on-felt-muted);
  font-size: 15px;
  text-align: center;
}

@media (max-width: 767px) {
  .battle {
    grid-template-columns: auto auto;
    grid-template-rows: auto minmax(0, 1fr);
    justify-content: space-between;
    gap: 8px;
  }

  .center {
    grid-column: 1 / -1;
    grid-row: 2;
  }

  .pile {
    gap: 6px;
    width: auto;
  }

  .deck-stack {
    width: 116px;
    height: 84px;
  }

  .discard-stack {
    width: 72px;
    height: 84px;
  }

  .discard-stack.empty {
    width: 52px;
    height: 72px;
    margin: 6px 10px;
  }
}
</style>
