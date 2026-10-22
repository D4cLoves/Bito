<script setup lang="ts">
import { computed } from 'vue'
import { motion, AnimatePresence } from 'motion-v'
import AvatarGlyph from '../client/AvatarGlyph.vue'
import GameCard from './GameCard.vue'
import AnimatedEmoji from './AnimatedEmoji.vue'
import { emojiOf } from '../../constants/emoji'
import { plural } from '../../composables/useLanding'
import type { ChatEntry } from '../../types/social'

const RING = 2 * Math.PI * 27

const props = withDefaults(
  defineProps<{
    name: string | null
    handCount?: number
    role?: 'attacker' | 'defender' | null
    out?: boolean
    offline?: boolean
    turnFraction?: number | null
    bubble?: ChatEntry | null
    me?: boolean
    userId?: string
    bubbleSide?: 'up' | 'down'
  }>(),
  { handCount: 0, role: null, out: false, offline: false, turnFraction: null, bubble: null, me: false, bubbleSide: 'down' },
)

const backs = computed(() => Math.min(props.handCount, 8))
const bubbleEmoji = computed(() => (props.bubble ? emojiOf(props.bubble.text) : null))
const urgent = computed(() => props.turnFraction !== null && props.turnFraction < 0.25)
</script>

<template>
  <div class="seat" :class="{ empty: !name, out, offline, me, [role ?? '']: !!role }">
    <div v-if="name && !me && backs" class="backs" aria-hidden="true">
      <GameCard
        v-for="i in backs"
        :key="i"
        face-down
        size="xs"
        :style="{ transform: `rotate(${(i - (backs + 1) / 2) * 8}deg) translateY(${Math.abs(i - (backs + 1) / 2) * 1.5}px)` }"
      />
    </div>

    <div class="avatar-wrap">
      <svg v-if="turnFraction !== null" class="timer-ring" :class="{ urgent }" viewBox="0 0 60 60" aria-hidden="true">
        <circle cx="30" cy="30" r="27" class="ring-track" />
        <circle cx="30" cy="30" r="27" class="ring-bar" :stroke-dasharray="RING" :stroke-dashoffset="RING * (1 - turnFraction)" />
      </svg>
      <span class="avatar"><AvatarGlyph v-if="name" :name="name" :user-id="userId" :self="me" /></span>
      <span v-if="offline" class="dot warn" title="Нет связи" aria-hidden="true"></span>
    </div>

    <div class="info">
      <span class="name">{{ name ?? 'Свободно' }}</span>
      <span v-if="!name" class="sub">ждём игрока…</span>
      <span v-else-if="out" class="sub">вышел</span>
      <span v-else-if="offline" class="sub warn">нет связи</span>
      <span v-else-if="role" class="tag" :class="role">{{ role === 'defender' ? 'отбивается' : 'ходит' }}</span>
      <span v-else-if="!me" class="sub">{{ handCount }} {{ plural(handCount, ['карта', 'карты', 'карт']) }}</span>
    </div>

    <AnimatePresence>
      <motion.div
        v-if="bubble"
        :key="bubble.id"
        class="bubble"
        :class="[bubbleSide, { emoji: !!bubbleEmoji }]"
        role="status"
        :initial="{ opacity: 0, scale: 0.6, y: bubbleSide === 'up' ? 10 : -10 }"
        :animate="{ opacity: 1, scale: 1, y: 0 }"
        :exit="{ opacity: 0, scale: 0.8 }"
        :transition="{ type: 'spring', stiffness: 420, damping: 26 }"
      >
        <AnimatedEmoji v-if="bubbleEmoji" :emoji="bubbleEmoji" :size="54" />
        <template v-else>{{ bubble.text }}</template>
      </motion.div>
    </AnimatePresence>
  </div>
</template>

<style scoped>
.seat {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  width: 128px;
  text-align: center;
}

.backs {
  display: flex;
  justify-content: center;
  height: 40px;
  margin-bottom: -10px;
}

.backs :deep(.game-card) {
  margin: 0 -9px;
  transform-origin: 50% 140%;
}

.avatar-wrap {
  position: relative;
  display: grid;
  place-items: center;
  width: 60px;
  height: 60px;
}

.timer-ring {
  position: absolute;
  inset: 0;
  transform: rotate(-90deg);
}

.ring-track,
.ring-bar {
  fill: none;
  stroke-width: 3.5;
}

.ring-track {
  stroke: var(--color-line-strong);
}

.ring-bar {
  stroke: var(--color-gold);
  stroke-linecap: round;
  transition: stroke-dashoffset 0.25s linear;
}

.timer-ring.urgent .ring-bar {
  stroke: var(--color-red);
}

.avatar {
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--color-ink-soft);
  box-shadow:
    0 0 0 3px var(--color-paper-raised),
    0 10px 20px -10px rgba(0, 0, 0, 0.6);
  color: var(--color-paper);
  font-family: var(--font-card);
  font-size: 20px;
}

.me .avatar {
  background: var(--color-gold);
}

.empty .avatar {
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1.5px var(--color-line-heavy);
  border: 0;
}

.out .avatar {
  opacity: 0.45;
}

.dot {
  position: absolute;
  right: 4px;
  bottom: 4px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  box-shadow: 0 0 0 2.5px var(--color-paper-raised);
}

.dot.warn {
  background: var(--color-red);
}

.info {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  margin-top: 4px;
  padding: 5px 10px 6px;
  border-radius: 14px;
  background: var(--color-glass);
  box-shadow: inset 0 0 0 1px var(--color-line);
  backdrop-filter: blur(8px);
  max-width: 100%;
  box-sizing: border-box;
}

.name {
  max-width: 108px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 500;
  color: var(--color-ink);
}

.empty .name {
  color: var(--color-ink-faint);
  font-weight: 400;
}

.sub {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.sub.warn {
  color: var(--color-red);
}

.tag {
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 11px;
}

.tag.attacker {
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.tag.defender {
  background: var(--color-red-tint);
  color: var(--color-red-strong);
}

.bubble {
  position: absolute;
  z-index: 20;
  left: 50%;
  max-width: 200px;
  width: max-content;
  padding: 8px 12px;
  translate: -50% 0;
  border-radius: 16px;
  background: var(--color-paper-raised);
  box-shadow:
    inset 0 0 0 1px var(--color-line-strong),
    0 14px 30px -14px rgba(0, 0, 0, 0.5);
  color: var(--color-ink);
  font-size: 14px;
  line-height: 1.35;
  overflow-wrap: anywhere;
  pointer-events: none;
}

.bubble.emoji {
  padding: 4px;
  border-radius: 20px;
}

.bubble.down {
  top: calc(100% + 8px);
}

.bubble.up {
  bottom: calc(100% + 8px);
}

.bubble::after {
  content: '';
  position: absolute;
  left: 50%;
  width: 12px;
  height: 12px;
  translate: -50% 0;
  rotate: 45deg;
  background: inherit;
  box-shadow: inherit;
  clip-path: polygon(0 0, 100% 0, 0 100%);
}

.bubble.down::after {
  top: -6px;
}

.bubble.up::after {
  bottom: -6px;
  rotate: 225deg;
}
</style>
