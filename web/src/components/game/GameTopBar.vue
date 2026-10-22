<script setup lang="ts">
import {
  PhArrowLeft,
  PhChatsCircle,
  PhEye,
  PhFlag,
  PhLink,
  PhSpeakerHigh,
  PhSpeakerSlash,
} from '@phosphor-icons/vue'
import ChipIcon from '../client/ChipIcon.vue'
import type { SocketStatus } from '../../composables/useGameSocket'

defineProps<{
  modeLabel: string
  isTransfer: boolean
  deck: number | null
  stake: number
  status: SocketStatus
  soundOn: boolean
  chatOpen: boolean
  unread: number
  canSurrender: boolean
  waiting: boolean
  copied: boolean
  watchers?: number
  watching?: boolean
}>()

defineEmits<{
  (e: 'back'): void
  (e: 'sound'): void
  (e: 'chat'): void
  (e: 'surrender'): void
  (e: 'copy'): void
}>()

const STATUS_TEXT: Record<SocketStatus, string> = {
  connecting: 'Подключаемся…',
  open: 'На связи',
  reconnecting: 'Переподключаемся…',
  closed: 'Нет связи',
}
</script>

<template>
  <header class="bar">
    <button type="button" class="back" @click="$emit('back')">
      <PhArrowLeft :size="18" weight="bold" aria-hidden="true" />
      <span class="back-text">Лобби</span>
    </button>

    <div class="meta">
      <span class="mode">
        <span class="suit" :class="{ red: isTransfer }" aria-hidden="true">{{ isTransfer ? '♥' : '♠' }}</span>
        {{ modeLabel }}
      </span>
      <span v-if="deck" class="pill">{{ deck }} карт</span>
      <span class="pill gold"><ChipIcon :size="16" />{{ stake.toLocaleString('ru-RU') }}</span>
      <span v-if="watching" class="pill watch"><PhEye :size="15" weight="bold" aria-hidden="true" />Вы смотрите</span>
      <span v-if="watchers" class="pill" :title="`Смотрят: ${watchers}`" :aria-label="`Смотрят: ${watchers}`">
        <PhEye :size="15" aria-hidden="true" />{{ watchers }}
      </span>
      <span class="conn" :class="status"><i aria-hidden="true"></i>{{ STATUS_TEXT[status] }}</span>
    </div>

    <div class="tools">
      <button v-if="waiting" type="button" class="tool wide" @click="$emit('copy')">
        <PhLink :size="18" aria-hidden="true" />
        <span class="tool-text">{{ copied ? 'Скопировано' : 'Позвать по ссылке' }}</span>
      </button>
      <button
        type="button"
        class="tool"
        :aria-label="soundOn ? 'Выключить звук' : 'Включить звук'"
        :title="soundOn ? 'Выключить звук' : 'Включить звук'"
        @click="$emit('sound')"
      >
        <PhSpeakerHigh v-if="soundOn" :size="19" />
        <PhSpeakerSlash v-else :size="19" />
      </button>
      <button v-if="canSurrender" type="button" class="tool" aria-label="Сдаться" title="Сдаться" @click="$emit('surrender')">
        <PhFlag :size="19" />
      </button>
      <button
        type="button"
        class="tool"
        :class="{ on: chatOpen }"
        :aria-pressed="chatOpen"
        aria-label="Чат стола"
        title="Чат стола"
        @click="$emit('chat')"
      >
        <PhChatsCircle :size="20" :weight="chatOpen ? 'fill' : 'regular'" />
        <span v-if="unread" class="badge">{{ unread > 9 ? '9+' : unread }}</span>
      </button>
    </div>
  </header>
</template>

<style scoped>
.bar {
  display: flex;
  align-items: center;
  gap: 14px;
  height: 64px;
  padding: 0 18px;
  box-sizing: border-box;
}

.back {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  padding: 0 16px 0 12px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
  color: var(--color-ink);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.back:hover {
  background: var(--color-paper-hover);
}

.meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
  overflow: hidden;
}

.mode {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: var(--font-card);
  font-size: 22px;
  font-style: italic;
  white-space: nowrap;
}

.suit {
  font-style: normal;
}

.suit.red {
  color: var(--color-red);
}

.pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--color-paper-hover);
  font-size: 13px;
  color: var(--color-ink-muted);
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.pill.gold {
  padding-left: 6px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.pill.watch {
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.conn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: 4px;
  font-size: 13px;
  color: var(--color-ink-faint);
  white-space: nowrap;
}

.conn i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-green);
}

.conn.connecting i,
.conn.reconnecting i {
  background: var(--color-gold);
  animation: blink 1s ease-in-out infinite;
}

.conn.closed i {
  background: var(--color-red);
}

.conn.closed {
  color: var(--color-red);
}

@keyframes blink {
  50% {
    opacity: 0.3;
  }
}

.tools {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tool {
  position: relative;
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
  color: var(--color-ink-muted);
  font: inherit;
  cursor: pointer;
}

.tool:hover {
  color: var(--color-ink);
}

.tool.on {
  background: var(--color-gold-tint);
  box-shadow: inset 0 0 0 1px var(--color-gold);
  color: var(--color-gold-ink);
}

.tool.wide {
  display: inline-flex;
  gap: 8px;
  width: auto;
  padding: 0 16px;
  border-radius: 999px;
  font-size: 14px;
}

.badge {
  position: absolute;
  top: -3px;
  right: -3px;
  display: grid;
  place-items: center;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  box-sizing: border-box;
  border-radius: 999px;
  background: var(--color-red);
  color: #fff;
  font-size: 11px;
}

@media (max-width: 767px) {
  .bar {
    gap: 8px;
    height: 56px;
    padding: 0 10px;
  }

  .back {
    padding: 0 12px;
  }

  .back-text,
  .tool-text,
  .pill,
  .conn.open {
    display: none;
  }

  .mode {
    font-size: 17px;
  }

  .tool.wide {
    width: 42px;
    padding: 0;
    border-radius: 50%;
  }
}
</style>
