<script setup lang="ts">
import { computed } from 'vue'
import { Plus, Zap } from 'lucide-vue-next'
import { plural } from '../../composables/useLanding'
import { useGameSound } from '../../composables/useGameSound'
import type { DeckSize, GameMode } from '../../types/game'

const HAND = 6
const DECKS: DeckSize[] = [24, 36, 52]
const PLAYERS = [2, 3, 4, 5, 6]
const MODES: { id: GameMode; name: string; text: string }[] = [
  { id: 'podkidnoy', name: 'Подкидной', text: 'Подкидывают все, кроме защитника.' },
  { id: 'perevodnoy', name: 'Переводной', text: 'Атаку можно перевести картой того же достоинства.' },
]

const mode = defineModel<GameMode>('mode', { required: true })
const deck = defineModel<DeckSize>('deck', { required: true })
const players = defineModel<number>('players', { required: true })

defineProps<{ busy: 'find' | 'create' | null; openCount: number }>()
defineEmits<{ (e: 'find'): void; (e: 'create'): void }>()

const sound = useGameSound()

const fits = (n: number, size: number = deck.value) => n * HAND <= size
const left = computed(() => deck.value - players.value * HAND)
const modeText = computed(() => MODES.find((m) => m.id === mode.value)!.text)

const seats = computed(() =>
  Array.from({ length: players.value }, (_, i) => {
    const a = Math.PI / 2 + (i * 2 * Math.PI) / players.value
    return { x: 50 + 40 * Math.cos(a), y: 50 + 32 * Math.sin(a), you: i === 0 }
  }),
)

function pickMode(m: GameMode) {
  sound.playClick()
  mode.value = m
}

function pickDeck(d: DeckSize) {
  sound.playClick()
  deck.value = d
  if (!fits(players.value, d)) players.value = Math.floor(d / HAND)
}

function pickPlayers(n: number) {
  if (!fits(n)) return
  sound.playClick()
  players.value = n
}
</script>

<template>
  <section class="quick" aria-labelledby="quick-title">
    <div class="quick-head">
      <div>
        <h2 id="quick-title">Быстрая игра</h2>
        <p class="sub">Выберите правила — подсадим за подходящий стол.</p>
      </div>
      <span v-if="openCount" class="match-pill">
        <i aria-hidden="true"></i>{{ openCount }} {{ plural(openCount, ['стол ждёт', 'стола ждут', 'столов ждут']) }}
      </span>
    </div>

    <div class="body">
      <div class="controls">
        <div class="field">
          <span class="label">Режим</span>
          <div class="segmented" role="radiogroup" aria-label="Режим">
            <span class="thumb" :class="{ right: mode === 'perevodnoy' }" aria-hidden="true"></span>
            <button
              v-for="m in MODES"
              :key="m.id"
              type="button"
              role="radio"
              :aria-checked="mode === m.id"
              :class="{ on: mode === m.id }"
              @click="pickMode(m.id)"
            >
              {{ m.name }}
            </button>
          </div>
          <Transition name="hint" mode="out-in">
            <p :key="mode" class="hint">{{ modeText }}</p>
          </Transition>
        </div>

        <div class="row-fields">
          <div class="field">
            <span class="label">Колода</span>
            <div class="chips" role="radiogroup" aria-label="Колода">
              <button
                v-for="d in DECKS"
                :key="d"
                type="button"
                role="radio"
                :aria-checked="deck === d"
                :class="{ on: deck === d }"
                @click="pickDeck(d)"
              >
                {{ d }}
              </button>
            </div>
          </div>

          <div class="field">
            <span class="label">Игроков</span>
            <div class="chips" role="radiogroup" aria-label="Игроков">
              <button
                v-for="n in PLAYERS"
                :key="n"
                type="button"
                role="radio"
                :aria-checked="players === n"
                :class="{ on: players === n }"
                :disabled="!fits(n)"
                :title="fits(n) ? '' : `В колоде ${deck} карт не хватит на ${n} игроков`"
                @click="pickPlayers(n)"
              >
                {{ n }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="preview" aria-hidden="true">
        <div class="felt">
          <TransitionGroup name="seat">
            <span
              v-for="(s, i) in seats"
              :key="`${players}-${i}`"
              class="seat"
              :class="{ you: s.you }"
              :style="{ left: `${s.x}%`, top: `${s.y}%`, transitionDelay: `${i * 40}ms` }"
            >
              {{ s.you ? 'вы' : '' }}
            </span>
          </TransitionGroup>
          <span class="deck-mark">
            <b>{{ left }}</b>
            <small>{{ plural(left, ['карта', 'карты', 'карт']) }} в колоде</small>
          </span>
        </div>
      </div>
    </div>

    <div class="actions">
      <button type="button" class="primary" :disabled="busy !== null" @click="$emit('find')">
        <span v-if="busy === 'find'" class="spinner" aria-hidden="true"></span>
        <Zap v-else :size="18" aria-hidden="true" />
        {{ busy === 'find' ? 'Ищем стол…' : 'Найти игру' }}
      </button>
      <button type="button" class="secondary" :disabled="busy !== null" @click="$emit('create')">
        <span v-if="busy === 'create'" class="spinner dark" aria-hidden="true"></span>
        <Plus v-else :size="18" aria-hidden="true" />
        Свой стол
      </button>
    </div>
  </section>
</template>

<style scoped>
.quick {
  padding: 28px;
  border-radius: 28px;
  background: var(--color-paper-raised);
  box-shadow:
    0 0 0 1px rgba(214, 208, 196, 0.9),
    0 24px 48px -32px rgba(70, 62, 48, 0.35);
}

.quick-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

h2 {
  margin: 0;
  font-size: 26px;
  font-weight: 500;
  letter-spacing: -0.03em;
}

.sub {
  margin: 6px 0 0;
  font-size: 15px;
  color: var(--color-ink-muted);
}

.match-pill {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex: none;
  padding: 7px 12px;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  font-size: 13px;
  color: var(--color-ink-muted);
}

.match-pill i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-green);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  50% {
    opacity: 0.35;
  }
}

.body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 220px;
  gap: 24px;
  margin-top: 22px;
}

.field + .field,
.row-fields {
  margin-top: 20px;
}

.row-fields {
  display: grid;
  grid-template-columns: minmax(0, 3fr) minmax(0, 5fr);
  gap: 16px;
}

.row-fields .field {
  margin-top: 0;
}

.label {
  display: block;
  margin-bottom: 10px;
  font-size: 13px;
  color: var(--color-ink-faint);
}

.segmented {
  position: relative;
  display: grid;
  grid-template-columns: 1fr 1fr;
  padding: 4px;
  border-radius: 999px;
  background: var(--color-paper-sunken);
}

.thumb {
  position: absolute;
  top: 4px;
  bottom: 4px;
  left: 4px;
  width: calc(50% - 4px);
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: 0 2px 8px -4px rgba(70, 62, 48, 0.4);
  transition: transform 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.thumb.right {
  transform: translateX(100%);
}

.segmented button,
.chips button {
  position: relative;
  height: 42px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 15px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    color 0.2s,
    transform 0.15s;
}

.segmented button.on {
  color: var(--color-ink);
}

.hint {
  margin: 10px 4px 0;
  font-size: 14px;
  line-height: 1.45;
  color: var(--color-ink-subtle);
}

.chips {
  display: flex;
  gap: 6px;
}

.chips button {
  flex: 1;
  min-width: 40px;
  background: var(--color-paper-sunken);
  font-variant-numeric: tabular-nums;
}

.chips button:hover:not(:disabled):not(.on) {
  background: var(--color-paper-hover);
  color: var(--color-ink);
}

.chips button:active:not(:disabled) {
  transform: scale(0.94);
}

.chips button.on {
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

.chips button:disabled {
  opacity: 0.35;
  cursor: not-allowed;
  text-decoration: line-through;
}

.preview {
  display: flex;
  align-items: center;
}

.felt {
  position: relative;
  width: 100%;
  aspect-ratio: 1.25;
  border-radius: 50%;
  background:
    radial-gradient(ellipse at 50% 40%, rgba(255, 253, 247, 0.9), rgba(239, 235, 226, 0.4) 70%),
    var(--color-paper-sunken);
  box-shadow: 0 0 0 6px var(--color-line-strong) inset;
}

.seat {
  position: absolute;
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  translate: -50% -50%;
  border-radius: 50%;
  background: var(--color-paper-raised);
  box-shadow: 0 0 0 1.5px var(--color-line-heavy);
  font-size: 10px;
  color: var(--color-paper);
  transition:
    left 0.45s cubic-bezier(0.2, 0.8, 0.2, 1),
    top 0.45s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.seat.you {
  background: var(--color-ink-soft);
  box-shadow: none;
}

.deck-mark {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.deck-mark b {
  font-size: 26px;
  font-weight: 500;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}

.deck-mark small {
  font-size: 11px;
  color: var(--color-ink-faint);
}

.actions {
  display: flex;
  gap: 10px;
  margin-top: 24px;
}

.primary,
.secondary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  height: 54px;
  border: 0;
  border-radius: 999px;
  font: inherit;
  font-size: 16px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.2s;
}

.primary {
  flex: 1;
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

.primary:hover:not(:disabled) {
  background: var(--color-ink);
}

.secondary {
  padding: 0 24px;
  background: var(--color-paper-sunken);
  color: var(--color-ink);
}

.secondary:hover:not(:disabled) {
  background: var(--color-paper-hover);
}

.primary:active:not(:disabled),
.secondary:active:not(:disabled) {
  transform: scale(0.98);
}

.primary:disabled,
.secondary:disabled {
  opacity: 0.7;
  cursor: wait;
}

.spinner {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid rgba(245, 244, 239, 0.35);
  border-top-color: var(--color-paper);
  animation: spin 0.7s linear infinite;
}


.spinner.dark {
  border-color: rgba(27, 25, 22, 0.2);
  border-top-color: var(--color-ink);
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.hint-enter-active,
.hint-leave-active {
  transition:
    opacity 0.2s,
    transform 0.2s;
}

.hint-enter-from {
  opacity: 0;
  transform: translateY(4px);
}

.hint-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.seat-enter-active {
  transition:
    opacity 0.3s,
    scale 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.seat-enter-from {
  opacity: 0;
  scale: 0.4;
}

.seat-leave-active {
  display: none;
}

@media (max-width: 720px) {
  .quick {
    padding: 22px 18px;
  }

  .body {
    grid-template-columns: 1fr;
  }

  .preview {
    display: none;
  }

  .row-fields {
    grid-template-columns: 1fr;
  }

  .quick-head {
    flex-direction: column;
  }
}
</style>
