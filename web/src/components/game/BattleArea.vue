<script setup lang="ts">
import { computed } from 'vue'
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

const deckBacks = computed(() => Math.min(5, Math.ceil(props.cardsLeft / 6)))
const redTrump = computed(() => props.trumpSuit === 2 || props.trumpSuit === 3)
</script>

<template>
  <div class="battle">
    <div class="pile deck">
      <div class="deck-stack">
        <GameCard v-if="cardsLeft > 0" :card="trumpCard" size="sm" trump class="trump-card" />
        <GameCard
          v-for="i in Math.max(0, deckBacks - 1)"
          :key="i"
          face-down
          size="sm"
          class="deck-back"
          :style="{ transform: `translate(${i * 1.5}px, ${-i * 1.5}px)` }"
        />
        <div v-if="cardsLeft === 0" class="trump-only" :class="{ red: redTrump }">{{ getSuitSymbol(trumpSuit) }}</div>
      </div>
      <span class="label">{{ cardsLeft ? `${cardsLeft} в колоде` : 'колода пуста' }}</span>
    </div>

    <div class="center">
      <TransitionGroup name="pair" tag="div" class="pairs">
        <div
          v-for="pair in pairs"
          :key="cardKey(pair.attack)"
          class="pair"
          :class="{ target: targets.has(cardKey(pair.attack)) }"
          @click="$emit('pair', pair)"
        >
          <GameCard :card="pair.attack" size="md" :trump="pair.attack.suit === trumpSuit" />
          <Transition name="defend">
            <GameCard v-if="pair.defend" :card="pair.defend" size="md" class="defend-card" :trump="pair.defend.suit === trumpSuit" />
          </Transition>
        </div>
      </TransitionGroup>
      <p v-if="!pairs.length" class="hint">{{ hint }}</p>
    </div>

    <div class="pile discard">
      <div class="discard-stack" :class="{ empty: !discardSize }">
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
  grid-template-columns: 124px minmax(0, 1fr) 124px;
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
  min-width: 0;
  min-height: 180px;
}

.pairs {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 14px 22px;
}

.pair {
  position: relative;
  width: 112px;
  height: 160px;
  border-radius: 14px;
  transition: box-shadow 0.2s;
}

.pair.target {
  cursor: pointer;
  box-shadow: 0 0 0 3px var(--color-felt-target);
  animation: target 1.2s ease-in-out infinite;
}

@keyframes target {
  50% {
    box-shadow: 0 0 0 7px color-mix(in srgb, var(--color-felt-target) 30%, transparent);
  }
}

.defend-card {
  position: absolute;
  left: 22px;
  top: 22px;
  transform: rotate(9deg);
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

.pair-enter-active {
  transition:
    opacity 0.35s,
    transform 0.45s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.pair-enter-from {
  opacity: 0;
  transform: translateY(60px) scale(0.9);
}

.pair-leave-active {
  pointer-events: none;
  transition:
    opacity 0.3s,
    transform 0.3s;
}

.pair-leave-to {
  opacity: 0;
  transform: scale(0.85);
}

.defend-enter-active {
  transition:
    opacity 0.3s,
    transform 0.4s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.defend-enter-from {
  opacity: 0;
  transform: rotate(30deg) translate(30px, 60px);
}

@media (max-width: 1100px) {
  .pair {
    width: 90px;
    height: 128px;
  }

  .pair :deep(.game-card) {
    --w: 76px;
  }

  .defend-card {
    left: 15px;
    top: 15px;
  }
}

@media (max-width: 767px) {
  .battle {
    grid-template-columns: 1fr 1fr;
    grid-template-rows: auto 1fr;
    gap: 8px;
  }

  .center {
    grid-column: 1 / -1;
    grid-row: 2;
    min-height: 140px;
  }

  .pile {
    flex-direction: row;
    gap: 8px;
  }

  .discard {
    flex-direction: row-reverse;
    justify-content: flex-start;
  }

  .deck-stack {
    flex: none;
    width: 116px;
    height: 84px;
  }

  .discard-stack {
    flex: none;
    height: 84px;
  }

  .discard-stack.empty {
    width: 52px;
    height: 72px;
    margin: 0;
  }

  .pair {
    width: 72px;
    height: 102px;
  }

  .pair :deep(.game-card) {
    --w: 60px;
  }

  .defend-card {
    left: 12px;
    top: 12px;
  }

  .pairs {
    gap: 10px 14px;
  }
}
</style>
