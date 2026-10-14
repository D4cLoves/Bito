<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { motion } from 'motion-v'
import { PhLockSimple, PhX } from '@phosphor-icons/vue'
import StakeChip from './StakeChip.vue'
import { STAKES, TABLE_FEE, bestPayout } from '../../constants/economy'
import { fitsDeck } from '../../composables/useLobbyPrefs'
import { useGameSound } from '../../composables/useGameSound'
import type { DeckSize, GameMode } from '../../types/game'

const stake = defineModel<number>('stake', { required: true })
const mode = defineModel<GameMode>('mode', { required: true })
const deck = defineModel<DeckSize>('deck', { required: true })
const players = defineModel<number>('players', { required: true })

const props = defineProps<{ balance: number | null; busy: boolean }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'create'): void }>()

const DECKS: DeckSize[] = [24, 36, 52]
const SEATS = [3, 4, 5, 6]
const MODES: { id: GameMode; label: string; suit: string }[] = [
  { id: 'podkidnoy', label: 'Подкидной', suit: '♠' },
  { id: 'perevodnoy', label: 'Переводной', suit: '♥' },
]

const sound = useGameSound()
const body = ref<HTMLElement | null>(null)

const affordable = (s: number) => props.balance === null || s <= props.balance
const payout = computed(() => bestPayout(stake.value, players.value))
const canCreate = computed(() => affordable(stake.value) && fitsDeck(players.value, deck.value))

watch(deck, (d) => {
  if (!fitsDeck(players.value, d)) players.value = Math.floor(d / 6)
})

function pickStake(s: number) {
  if (!affordable(s) || stake.value === s) return
  sound.playChip()
  stake.value = s
}

function pickMode(m: GameMode) {
  if (mode.value === m) return
  sound.playClick()
  mode.value = m
}

function pickDeck(d: DeckSize) {
  if (deck.value === d) return
  sound.playClick()
  deck.value = d
}

function pickPlayers(n: number) {
  if (!fitsDeck(n, deck.value) || players.value === n) return
  sound.playClick()
  players.value = n
}

function submit() {
  if (props.busy || !canCreate.value) return
  emit('create')
}

function onKey(e: KeyboardEvent) {
  if (e.key !== 'Escape') return
  e.stopPropagation()
  emit('close')
}

onMounted(async () => {
  window.addEventListener('keydown', onKey, true)
  await nextTick()
  body.value?.querySelector<HTMLButtonElement>('.chip-btn.on')?.focus()
})

onUnmounted(() => window.removeEventListener('keydown', onKey, true))
</script>

<template>
  <motion.div
    class="modal"
    :initial="{ opacity: 0 }"
    :animate="{ opacity: 1 }"
    :exit="{ opacity: 0 }"
    :transition="{ duration: 0.18 }"
    @click.self="emit('close')"
  >
    <motion.section
      class="dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="create-title"
      :initial="{ opacity: 0, y: 16, scale: 0.97 }"
      :animate="{ opacity: 1, y: 0, scale: 1 }"
      :exit="{ opacity: 0, y: 10, scale: 0.98 }"
      :transition="{ type: 'spring', stiffness: 420, damping: 34 }"
    >
      <header class="head">
        <h2 id="create-title">Новый стол</h2>
        <button type="button" class="close" aria-label="Закрыть" @click="emit('close')">
          <PhX :size="20" />
        </button>
      </header>
      <p class="sub">Компания на 3–6 игроков. Стол появится в списке, соперники подсядут, и партия начнётся сама, когда соберутся все.</p>

      <div ref="body" class="body">
        <div class="group">
          <span id="c-stake" class="label">Ставка</span>
          <div class="chips" role="radiogroup" aria-labelledby="c-stake">
            <button
              v-for="s in STAKES"
              :key="s"
              type="button"
              role="radio"
              class="chip-btn"
              :class="{ on: stake === s, locked: !affordable(s) }"
              :aria-checked="stake === s"
              :aria-label="affordable(s) ? `${s} фишек` : `${s} фишек — не хватает`"
              :title="affordable(s) ? `${s.toLocaleString('ru-RU')} фишек` : 'Не хватает фишек'"
              @click="pickStake(s)"
            >
              <StakeChip :value="s" :size="52" />
              <PhLockSimple v-if="!affordable(s)" class="lock" :size="16" weight="bold" aria-hidden="true" />
            </button>
          </div>
        </div>

        <div class="row">
          <div class="group">
            <span id="c-mode" class="label">Режим</span>
            <div class="seg" role="radiogroup" aria-labelledby="c-mode">
              <button
                v-for="m in MODES"
                :key="m.id"
                type="button"
                role="radio"
                :aria-checked="mode === m.id"
                :class="{ on: mode === m.id }"
                @click="pickMode(m.id)"
              >
                <span class="suit" :class="{ red: m.id === 'perevodnoy' }" aria-hidden="true">{{ m.suit }}</span>
                {{ m.label }}
              </button>
            </div>
          </div>

          <div class="group">
            <span id="c-deck" class="label">Колода</span>
            <div class="seg" role="radiogroup" aria-labelledby="c-deck">
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
        </div>

        <div class="group">
          <span id="c-seats" class="label">Игроков</span>
          <div class="seg" role="radiogroup" aria-labelledby="c-seats">
            <button
              v-for="n in SEATS"
              :key="n"
              type="button"
              role="radio"
              :aria-checked="players === n"
              :class="{ on: players === n }"
              :disabled="!fitsDeck(n, deck)"
              :title="fitsDeck(n, deck) ? '' : `В колоде ${deck} карт не хватит на ${n} игроков`"
              @click="pickPlayers(n)"
            >
              {{ n }}
            </button>
          </div>
        </div>
      </div>

      <dl class="odds">
        <div>
          <dt>Выигрыш</dt>
          <dd class="plus">до +{{ payout.toLocaleString('ru-RU') }}</dd>
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

      <footer class="actions">
        <button type="button" class="btn ghost" @click="emit('close')">Отмена</button>
        <button type="button" class="btn primary" :disabled="busy || !canCreate" @click="submit">
          <span v-if="busy" class="spinner" aria-hidden="true"></span>
          {{ busy ? 'Создаём…' : 'Создать и сесть' }}
        </button>
      </footer>
    </motion.section>
  </motion.div>
</template>

<style scoped>
.modal {
  position: fixed;
  inset: 0;
  z-index: 120;
  display: grid;
  place-items: center;
  padding: 16px;
  background: var(--color-scrim);
}

.dialog {
  display: flex;
  flex-direction: column;
  gap: 18px;
  width: min(540px, 100%);
  max-height: calc(100dvh - 32px);
  padding: 24px 26px 22px;
  box-sizing: border-box;
  overflow-y: auto;
  border-radius: 28px;
  background: var(--color-paper-raised);
  box-shadow: var(--shadow-panel);
  color: var(--color-ink);
}

.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

h2 {
  margin: 0;
  font-family: var(--font-card);
  font-size: 30px;
  font-style: italic;
  font-weight: 400;
  letter-spacing: -0.02em;
}

.close {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  cursor: pointer;
}

.close:hover {
  color: var(--color-ink);
}

.sub {
  margin: -8px 0 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-ink-faint);
}

.body {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.row {
  display: flex;
  flex-wrap: wrap;
  gap: 18px 24px;
}

.group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.label {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-ink-faint);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  padding-top: 6px;
}

.chip-btn {
  position: relative;
  display: grid;
  place-items: center;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: none;
  cursor: pointer;
  transition: transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.chip-btn:hover:not(.locked) {
  transform: translateY(-3px);
}

.chip-btn.on {
  transform: translateY(-6px);
  box-shadow: 0 0 0 3px var(--color-paper-raised), 0 0 0 5px var(--color-gold);
}

.chip-btn:focus-visible {
  outline: 2px solid var(--color-gold);
  outline-offset: 4px;
}

.chip-btn.locked {
  cursor: not-allowed;
  filter: grayscale(1) brightness(0.8);
  opacity: 0.6;
}

.lock {
  position: absolute;
  color: #fff;
  filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.6));
}

.seg {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.seg button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-width: 46px;
  height: 42px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 15px;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition:
    background-color 0.2s,
    color 0.2s;
}

.seg button:hover:not(:disabled):not(.on) {
  color: var(--color-ink);
}

.seg button.on {
  background: var(--color-ink-soft);
  box-shadow: none;
  color: var(--color-paper);
}

.seg button:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.suit {
  font-family: var(--font-card);
  font-size: 17px;
  line-height: 1;
}

.suit.red {
  color: var(--color-red);
}

.on .suit.red {
  color: var(--color-red-soft);
}

.odds {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
  margin: 0;
  padding: 14px 16px;
  border-radius: 18px;
  background: var(--color-paper-sunken);
}

.odds dt {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-ink-faint);
}

.odds dd {
  margin: 3px 0 0;
  font-size: 17px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.plus {
  color: var(--color-green);
}

.minus {
  color: var(--color-red);
}

.actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 46px;
  padding: 0 22px;
  border: 0;
  border-radius: 999px;
  font: inherit;
  font-size: 15px;
  font-weight: 500;
  cursor: pointer;
}

.btn.ghost {
  background: var(--color-paper-hover);
  color: var(--color-ink);
}

.btn.primary {
  background: linear-gradient(180deg, #f7e2ad, #d6aa58);
  color: #2b1f08;
  box-shadow: 0 12px 24px -14px rgba(0, 0, 0, 0.6);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.spinner {
  width: 16px;
  height: 16px;
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

@media (max-width: 560px) {
  .dialog {
    padding: 20px 18px 18px;
    border-radius: 24px;
  }

  .chips {
    gap: 8px;
  }

  .actions {
    flex-direction: column-reverse;
  }

  .btn {
    width: 100%;
  }
}
</style>
