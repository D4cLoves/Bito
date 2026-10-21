<script setup lang="ts">
import AvatarGlyph from './AvatarGlyph.vue'
import { onMounted, onUnmounted, ref, watch } from 'vue'
import { PhX } from '@phosphor-icons/vue'
import { avatarTone } from '../../composables/avatar'
import type { TableInvite } from '../../types/game'

const INVITE_MS = 45_000

const props = defineProps<{ invites: TableInvite[] }>()
const emit = defineEmits<{
  (e: 'accept', invite: TableInvite): void
  (e: 'decline', invite: TableInvite): void
  (e: 'expire', invite: TableInvite): void
}>()

const now = ref(Date.now())
let timer: number | null = null

const leftMs = (inv: TableInvite) => Math.max(new Date(inv.expiresAt).getTime() - now.value, 0)
const leftSec = (inv: TableInvite) => Math.ceil(leftMs(inv) / 1000)
const share = (inv: TableInvite) => Math.min(leftMs(inv) / INVITE_MS, 1)

function details(inv: TableInvite) {
  const mode = inv.mode === 'perevodnoy' ? 'Переводной' : 'Подкидной'
  const seats = inv.maxPlayers === 2 ? 'один на один' : `${inv.maxPlayers} игроков`
  return `${mode} · ${inv.deck} карт · ${seats} · ставка ${inv.stake.toLocaleString('ru-RU')}`
}

watch(now, () => {
  for (const inv of props.invites) {
    if (leftMs(inv) === 0) emit('expire', inv)
  }
})

onMounted(() => {
  timer = window.setInterval(() => (now.value = Date.now()), 250)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="invites" aria-live="polite">
    <TransitionGroup name="invite">
      <article v-for="inv in invites" :key="inv.tableId" class="invite" :aria-label="`${inv.fromName} зовёт за стол`">
        <span class="avatar" :style="{ '--tone': avatarTone(inv.fromName) }" aria-hidden="true"><AvatarGlyph :name="inv.fromName" :user-id="inv.fromId" /></span>
        <div class="text">
          <b>{{ inv.fromName }} зовёт за стол</b>
          <small>{{ details(inv) }}</small>
        </div>
        <button type="button" class="close" :aria-label="`Отказаться от приглашения ${inv.fromName}`" @click="emit('decline', inv)">
          <PhX :size="14" weight="bold" />
        </button>
        <div class="actions">
          <button type="button" class="sit" @click="emit('accept', inv)">Сесть · {{ leftSec(inv) }} с</button>
          <button type="button" class="no" @click="emit('decline', inv)">Нет</button>
        </div>
        <span class="time" :style="{ transform: `scaleX(${share(inv)})` }" aria-hidden="true"></span>
      </article>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.invites {
  position: fixed;
  z-index: 350;
  right: 24px;
  bottom: 24px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: min(340px, calc(100vw - 32px));
  pointer-events: none;
}

.invite {
  position: relative;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 12px;
  align-items: center;
  padding: 14px 14px 16px;
  overflow: hidden;
  border-radius: 20px;
  background: var(--color-paper-raised);
  box-shadow: var(--shadow-panel);
  pointer-events: auto;
}

.avatar {
  display: grid;
  place-items: center;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: color-mix(in srgb, var(--color-gold) var(--tone, 52%), #000);
  color: #fff;
  font-size: 16px;
  font-weight: 500;
}

.text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.text b {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 500;
}

.text small {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.close {
  display: grid;
  place-items: center;
  align-self: start;
  width: 26px;
  height: 26px;
  border: 0;
  border-radius: 50%;
  background: none;
  color: var(--color-ink-faint);
  cursor: pointer;
}

.close:hover {
  background: var(--color-paper-hover);
  color: var(--color-ink);
}

.actions {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 8px;
}

.actions button {
  height: 38px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.sit {
  background: var(--gradient-gold);
  color: var(--color-on-gold);
  font-variant-numeric: tabular-nums;
}

.sit:hover {
  filter: brightness(1.04);
}

.no {
  background: var(--color-paper-sunken);
  color: var(--color-ink-muted);
}

.no:hover {
  color: var(--color-ink);
}

.time {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 3px;
  background: var(--color-gold);
  transform-origin: left;
  transition: transform 0.25s linear;
}

.invite-enter-active,
.invite-leave-active {
  transition: opacity 0.25s, transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.invite-enter-from,
.invite-leave-to {
  opacity: 0;
  transform: translateY(12px) scale(0.97);
}

@media (prefers-reduced-motion: reduce) {
  .time,
  .invite-enter-active,
  .invite-leave-active {
    transition: none;
  }
}
</style>
