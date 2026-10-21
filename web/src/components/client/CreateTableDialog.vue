<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { motion } from 'motion-v'
import { useMediaQuery } from '@vueuse/core'
import { PhCheck, PhGlobeSimple, PhLockSimple, PhX } from '@phosphor-icons/vue'
import AvatarGlyph from './AvatarGlyph.vue'
import StakeChip from './StakeChip.vue'
import { STAKES, TABLE_FEE, bestPayout } from '../../constants/economy'
import { fitsDeck } from '../../composables/useLobbyPrefs'
import { useGameSound } from '../../composables/useGameSound'
import { avatarTone } from '../../composables/avatar'
import type { DeckSize, GameMode } from '../../types/game'
import type { Friend } from '../../types/social'

const stake = defineModel<number>('stake', { required: true })
const mode = defineModel<GameMode>('mode', { required: true })
const deck = defineModel<DeckSize>('deck', { required: true })
const players = defineModel<number>('players', { required: true })
const isPrivate = defineModel<boolean>('private', { default: false })
const invited = defineModel<string[]>('invited', { default: () => [] })

const props = defineProps<{ balance: number | null; busy: boolean; friends?: Friend[] }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'create'): void }>()

const DECKS: DeckSize[] = [24, 36, 52]
const withFriends = computed(() => isPrivate.value || invited.value.length > 0)
const SEATS = computed(() => (withFriends.value ? [2, 3, 4, 5, 6] : [3, 4, 5, 6]))
const compact = useMediaQuery('(max-width: 560px)')
const lobbyFriends = computed(() => (props.friends ?? []).filter((f) => f.status === 'online'))
const roomForFriends = computed(() => players.value - 1)
const MODES: { id: GameMode; label: string; suit: string }[] = [
  { id: 'podkidnoy', label: 'Подкидной', suit: '♠' },
  { id: 'perevodnoy', label: 'Переводной', suit: '♥' },
]

const sound = useGameSound()
const body = ref<HTMLElement | null>(null)

const affordable = (s: number) => props.balance === null || s <= props.balance
const payout = computed(() => bestPayout(stake.value, players.value))
const canCreate = computed(
  () => affordable(stake.value) && fitsDeck(players.value, deck.value) && invited.value.length <= roomForFriends.value,
)

watch(withFriends, (on) => {
  if (!on && players.value < 3) players.value = 3
})

function toggleFriend(id: string) {
  sound.playClick()
  if (invited.value.includes(id)) {
    invited.value = invited.value.filter((f) => f !== id)
    return
  }
  if (invited.value.length >= 5) return
  invited.value = [...invited.value, id]
  if (players.value < invited.value.length + 1) players.value = Math.min(6, invited.value.length + 1)
}

function pickAccess(priv: boolean) {
  if (isPrivate.value === priv) return
  sound.playClick()
  isPrivate.value = priv
}

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
        <h2 id="create-title">{{ invited.length ? 'Стол для друзей' : 'Новый стол' }}</h2>
        <button type="button" class="close" aria-label="Закрыть" @click="emit('close')">
          <PhX :size="20" />
        </button>
      </header>
      <div ref="body" class="body">
        <p v-if="isPrivate" class="sub">Приватный стол не видно в списке. Сесть смогут только те, кого вы позовёте, или у кого есть ссылка.</p>
        <p v-else-if="invited.length" class="sub">Позванным держим места 45 секунд. Остальные места займут игроки из списка столов.</p>
        <p v-else class="sub">Компания на 3–6 игроков. Стол появится в списке, соперники подсядут, и партия начнётся сама, когда соберутся все.</p>

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
              <StakeChip :value="s" :size="compact ? 42 : 52" />
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

        <div class="group">
          <span id="c-access" class="label">Кто может сесть</span>
          <div class="seg" role="radiogroup" aria-labelledby="c-access">
            <button type="button" role="radio" :aria-checked="!isPrivate" :class="{ on: !isPrivate }" @click="pickAccess(false)">
              <PhGlobeSimple :size="16" aria-hidden="true" />
              Все
            </button>
            <button type="button" role="radio" :aria-checked="isPrivate" :class="{ on: isPrivate }" @click="pickAccess(true)">
              <PhLockSimple :size="16" aria-hidden="true" />
              По приглашению
            </button>
          </div>
        </div>

        <div v-if="friends" class="group">
          <span id="c-friends" class="label">Позвать друзей · {{ invited.length }} из {{ roomForFriends }}</span>
          <div v-if="lobbyFriends.length" class="friends" role="group" aria-labelledby="c-friends">
            <button
              v-for="f in lobbyFriends"
              :key="f.id"
              type="button"
              class="friend"
              :class="{ on: invited.includes(f.id) }"
              :aria-pressed="invited.includes(f.id)"
              :disabled="!invited.includes(f.id) && invited.length >= roomForFriends"
              @click="toggleFriend(f.id)"
            >
              <span class="avatar" :style="{ '--tone': avatarTone(f.username) }">
                <AvatarGlyph :name="f.username" :user-id="f.id" />
                <i v-if="invited.includes(f.id)" class="tick" aria-hidden="true"><PhCheck :size="10" weight="bold" /></i>
              </span>
              {{ f.username }}
            </button>
          </div>
          <p v-else class="none">Сейчас никого из друзей нет в лобби. Позвать можно будет и со страницы стола или по ссылке.</p>
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
          {{ busy ? (invited.length ? 'Зовём…' : 'Создаём…') : invited.length ? 'Создать и позвать' : 'Создать и сесть' }}
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
  gap: 16px;
  width: min(540px, 100%);
  max-height: calc(100dvh - 32px);
  padding: 24px 26px 22px;
  box-sizing: border-box;
  overflow: hidden;
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
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-ink-faint);
}

.body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 18px;
  min-height: 0;
  margin: -4px -26px 0;
  padding: 4px 26px 4px;
  overflow-y: auto;
  overscroll-behavior: contain;
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

.friends {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.friend {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 14px 0 5px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
  transition: background-color 0.2s, color 0.2s;
}

.friend.on {
  background: var(--color-gold-tint);
  box-shadow: inset 0 0 0 1.5px var(--color-gold);
  color: var(--color-ink);
}

.friend:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.friend .avatar {
  position: relative;
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-gold) var(--tone, 52%), #000);
  color: #fff;
  font-size: 13px;
  font-weight: 500;
}

.friend .tick {
  position: absolute;
  right: -3px;
  bottom: -3px;
  display: grid;
  place-items: center;
  width: 15px;
  height: 15px;
  border-radius: 50%;
  background: var(--color-green);
  color: var(--color-on-green);
  box-shadow: 0 0 0 2px var(--color-paper-raised);
}

.none {
  margin: 0;
  font-size: 13px;
  color: var(--color-ink-faint);
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
  background: var(--gradient-gold);
  color: var(--color-on-gold);
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
  border: 2px solid color-mix(in srgb, var(--color-on-gold) 25%, transparent);
  border-top-color: var(--color-on-gold);
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 560px) {
  .dialog {
    gap: 14px;
    padding: 18px 18px 16px;
    border-radius: 24px;
  }

  .body {
    gap: 16px;
    margin: -4px -18px 0;
    padding: 4px 18px 4px;
  }

  .chips {
    gap: 6px;
  }

  .odds {
    gap: 18px;
    padding: 12px 14px;
  }

  .btn.primary {
    flex: 1;
  }
}
</style>
