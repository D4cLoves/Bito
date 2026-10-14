<script setup lang="ts">
import { computed } from 'vue'
import { motion, useReducedMotion } from 'motion-v'
import { PhCoins, PhLightning } from '@phosphor-icons/vue'
import CardBack from './CardBack.vue'
import StakeChip from './StakeChip.vue'
import PickField from './PickField.vue'
import { MIN_STAKE, STAKES, TABLE_FEE, TOP_UP_TO, bestPayout } from '../../constants/economy'
import { useGameSound } from '../../composables/useGameSound'
import type { CardBackId } from '../../stores/settings'
import type { AnyDeck, AnyMode } from '../../types/game'
import type { PickOption } from '../../types/ui'

const stake = defineModel<number>('stake', { required: true })
const mode = defineModel<AnyMode>('mode', { required: true })
const deck = defineModel<AnyDeck>('deck', { required: true })

const props = defineProps<{ balance: number | null; busy: boolean; cardBack: CardBackId }>()
const emit = defineEmits<{ (e: 'play'): void; (e: 'top-up'): void }>()

const sound = useGameSound()
const reduced = useReducedMotion()

const affordable = (s: number) => props.balance === null || s <= props.balance
const broke = computed(() => props.balance !== null && props.balance < MIN_STAKE)

const stakeOptions = computed<PickOption<number>[]>(() =>
  STAKES.map((s) => ({
    value: s,
    label: `${s.toLocaleString('ru-RU')} фишек`,
    short: s.toLocaleString('ru-RU'),
    chip: s,
    disabled: !affordable(s),
    hint: affordable(s) ? `выигрыш до +${bestPayout(s, 2).toLocaleString('ru-RU')}` : 'не хватает фишек',
  })),
)

const modeOptions: PickOption<AnyMode>[] = [
  { value: 'any', label: 'Любой', hint: 'соперник найдётся быстрее' },
  { value: 'podkidnoy', label: 'Подкидной', suit: '♠' },
  { value: 'perevodnoy', label: 'Переводной', suit: '♥', red: true },
]

const deckOptions: PickOption<AnyDeck>[] = [
  { value: 0, label: 'Любая', hint: 'соперник найдётся быстрее' },
  { value: 24, label: '24 карты' },
  { value: 36, label: '36 карт' },
  { value: 52, label: '52 карты' },
]

const win = computed(() => bestPayout(stake.value, 2))

const FAN = 5
const fan = Array.from({ length: FAN }, (_, i) => {
  const offset = i - (FAN - 1) / 2
  return { rotate: offset * 9, y: offset * offset * 3, x: offset * 30 }
})

const spring = { type: 'spring', stiffness: 380, damping: 30 } as const

function onPlay() {
  if (props.busy) return
  if (broke.value) emit('top-up')
  else emit('play')
}
</script>

<template>
  <section class="quick" aria-labelledby="quick-title">
    <div class="hero">
      <div class="hero-bg" aria-hidden="true">
        <span class="lamp"></span>
        <span class="rim"></span>
      </div>

      <div class="controls">
        <span class="badge"><PhLightning :size="14" weight="fill" aria-hidden="true" />Быстрая игра</span>
        <h2 id="quick-title">Один на один</h2>
        <p class="lead">
          <template v-if="broke">Фишек не хватает даже на минимальную ставку — возьмите подачку и возвращайтесь за стол.</template>
          <template v-else>Сервер сам найдёт соперника с такими же настройками. «Любой» ускоряет поиск.</template>
        </p>

        <div class="fields">
          <PickField v-model="stake" class="field-stake" label="Ставка" :options="stakeOptions" @change="sound.playChip()" />
          <PickField v-model="mode" label="Режим" :options="modeOptions" @change="sound.playClick()" />
          <PickField v-model="deck" label="Колода" :options="deckOptions" align="end" @change="sound.playClick()" />
        </div>

        <div class="go-row">
          <button type="button" class="go" :class="{ broke }" :disabled="busy" @click="onPlay">
            <span class="shine" aria-hidden="true"></span>
            <span v-if="busy" class="spinner" aria-hidden="true"></span>
            <PhCoins v-else-if="broke" :size="22" weight="fill" aria-hidden="true" />
            <PhLightning v-else :size="22" weight="fill" aria-hidden="true" />
            {{ busy ? 'Ищем стол…' : broke ? `Получить ${TOP_UP_TO} фишек` : 'Играть' }}
          </button>
          <dl v-if="!broke" class="odds">
            <div>
              <dt>Выигрыш</dt>
              <dd class="plus">+{{ win.toLocaleString('ru-RU') }}</dd>
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
        </div>
      </div>

      <div class="art" aria-hidden="true">
        <div class="fan">
          <motion.div
            v-for="(c, i) in fan"
            :key="i"
            class="fan-card"
            :initial="{ opacity: 0, y: 40, rotate: 0 }"
            :animate="{ opacity: 1, x: c.x, y: c.y, rotate: c.rotate }"
            :transition="{ ...spring, delay: reduced ? 0 : i * 0.04 }"
          >
            <CardBack :variant="cardBack" />
          </motion.div>
        </div>
        <div class="stack">
          <StakeChip v-for="i in 4" :key="i" class="stack-chip" :value="stake" :size="58" :label="i === 4" :style="{ '--i': i }" />
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.quick {
  position: relative;
  z-index: 2;
  container-type: inline-size;
}

.hero {
  --on-felt: var(--color-on-felt);
  --on-felt-muted: var(--color-on-felt-muted);
  --on-felt-faint: var(--color-on-felt-faint);

  position: relative;
  isolation: isolate;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 250px;
  gap: 20px;
  padding: 28px 32px 30px;
  border-radius: 32px;
  color: var(--on-felt);
}

.hero-bg {
  position: absolute;
  inset: 0;
  z-index: -1;
  overflow: hidden;
  border-radius: inherit;
  background:
    radial-gradient(ellipse 80% 120% at 72% 0%, rgba(255, 255, 255, 0.12), transparent 60%),
    radial-gradient(ellipse at 50% 120%, var(--color-felt-deep), var(--color-felt) 70%);
  box-shadow:
    inset 0 0 0 1px var(--color-felt-line),
    inset 0 0 90px color-mix(in srgb, var(--color-felt-deep) 80%, transparent),
    0 30px 60px -36px rgba(0, 0, 0, 0.55);
}

.hero-bg::before {
  content: '';
  position: absolute;
  inset: 0;
  background-image: radial-gradient(rgba(255, 255, 255, 0.05) 1px, transparent 1px);
  background-size: 4px 4px;
}

.lamp {
  position: absolute;
  top: -45%;
  right: 4%;
  width: 62%;
  height: 130%;
  background: radial-gradient(ellipse at 50% 30%, rgba(255, 228, 170, 0.26), transparent 65%);
}

.rim {
  position: absolute;
  inset: 10px;
  border-radius: 24px;
  border: 1.5px dashed var(--color-felt-line);
}

.controls {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  align-self: flex-start;
  padding: 5px 12px 5px 9px;
  border-radius: 999px;
  background: var(--color-gold);
  color: #2b1f08;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.02em;
}

h2 {
  margin: 0;
  font-family: var(--font-card);
  font-size: clamp(38px, 4vw, 56px);
  font-style: italic;
  font-weight: 400;
  letter-spacing: -0.02em;
  line-height: 1.02;
}

.lead {
  max-width: 440px;
  margin: -4px 0 4px;
  font-size: 14px;
  line-height: 1.5;
  color: var(--on-felt-muted);
}

.fields {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.go-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px 24px;
  margin-top: 6px;
}

.go {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-width: 240px;
  height: 62px;
  padding: 0 34px;
  overflow: hidden;
  border: 0;
  border-radius: 999px;
  background: linear-gradient(180deg, #f7e2ad, #d6aa58);
  color: #2b1f08;
  font: inherit;
  font-size: 20px;
  font-weight: 500;
  letter-spacing: -0.01em;
  cursor: pointer;
  box-shadow:
    0 0 0 6px rgba(214, 170, 88, 0.18),
    0 20px 36px -16px rgba(0, 0, 0, 0.8);
  transition:
    transform 0.2s cubic-bezier(0.2, 0.8, 0.2, 1),
    box-shadow 0.2s;
}

.go:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow:
    0 0 0 8px rgba(214, 170, 88, 0.22),
    0 26px 40px -16px rgba(0, 0, 0, 0.85);
}

.go:active:not(:disabled) {
  transform: translateY(0);
}

.go:disabled {
  cursor: progress;
}

.go:focus-visible {
  outline: 3px solid var(--color-felt-target);
  outline-offset: 4px;
}

.go.broke {
  background: linear-gradient(180deg, #bfe3b0, #6f9a6a);
  color: #12301f;
  box-shadow:
    0 0 0 6px rgba(111, 154, 106, 0.2),
    0 20px 36px -16px rgba(0, 0, 0, 0.8);
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

.spinner {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  border: 2px solid rgba(43, 31, 8, 0.25);
  border-top-color: #2b1f08;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.odds {
  display: flex;
  gap: 20px;
  margin: 0;
}

.odds dt {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--on-felt-faint);
}

.odds dd {
  margin: 2px 0 0;
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

.art {
  position: relative;
  min-height: 230px;
}

.fan {
  position: absolute;
  inset: 0;
}

.fan-card {
  position: absolute;
  left: 50%;
  top: 22%;
  width: 92px;
  height: 132px;
  margin-left: -46px;
  transform-origin: 50% 120%;
}

.stack {
  position: absolute;
  left: 8%;
  bottom: 4%;
  width: 58px;
  height: 90px;
}

.stack-chip {
  position: absolute;
  left: 0;
  bottom: calc((var(--i) - 1) * 9px);
  z-index: var(--i);
}

@container (max-width: 780px) {
  .hero {
    grid-template-columns: minmax(0, 1fr);
  }

  .art {
    display: none;
  }
}

@container (max-width: 470px) {
  .hero {
    padding: 22px 18px 24px;
  }

  .fields {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  }

  .field-stake {
    grid-column: 1 / -1;
  }

  .fields :deep(.value .suit) {
    display: none;
  }

  .go {
    width: 100%;
    min-width: 0;
  }

  .odds {
    width: 100%;
    justify-content: space-between;
  }
}

@media (prefers-reduced-motion: reduce) {
  .shine {
    animation: none;
  }
}
</style>
