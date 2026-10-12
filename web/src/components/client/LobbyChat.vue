<script setup lang="ts">
import { nextTick, onMounted, ref, watch } from 'vue'
import { PhPaperPlaneRight } from '@phosphor-icons/vue'
import type { LobbyMessage } from '../../types/social'

const MAX_LENGTH = 200

const props = defineProps<{ messages: LobbyMessage[]; myId?: string }>()
const emit = defineEmits<{ (e: 'send', text: string): void }>()

const draft = ref('')
const listEl = ref<HTMLElement | null>(null)

function time(iso: string) {
  return new Date(iso).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

function send() {
  const text = draft.value.trim()
  if (!text) return
  emit('send', text.slice(0, MAX_LENGTH))
  draft.value = ''
}

async function scrollDown() {
  await nextTick()
  if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight
}

onMounted(scrollDown)
watch(() => props.messages.length, scrollDown)
</script>

<template>
  <section class="chat" aria-labelledby="chat-title">
    <div class="head">
      <h2 id="chat-title">Общий чат</h2>
    </div>

    <ol ref="listEl" class="messages" aria-live="polite">
      <li v-for="m in messages" :key="m.id" class="msg" :class="{ system: m.system, mine: m.userId === myId }">
        <template v-if="m.system">{{ m.text }}</template>
        <template v-else>
          <span class="meta">
            <b>{{ m.username }}</b>
            <time :datetime="m.at">{{ time(m.at) }}</time>
          </span>
          <span class="text">{{ m.text }}</span>
        </template>
      </li>
    </ol>

    <form class="compose" @submit.prevent="send">
      <input v-model="draft" type="text" :maxlength="MAX_LENGTH" placeholder="Написать всем…" aria-label="Сообщение в общий чат" />
      <button type="submit" :disabled="!draft.trim()" aria-label="Отправить">
        <PhPaperPlaneRight :size="18" weight="fill" />
      </button>
    </form>
  </section>
</template>

<style scoped>
.chat {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.head {
  padding: 0 4px 10px;
}

h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
}

.messages {
  display: flex;
  flex-direction: column;
  gap: 10px;
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 4px 4px 12px;
  overflow-y: auto;
  list-style: none;
}

.msg {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-width: 92%;
}

.msg.mine {
  align-self: flex-end;
  align-items: flex-end;
}

.meta {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 12px;
}

.meta b {
  font-weight: 500;
  color: var(--color-gold-ink);
}

.mine .meta b {
  color: var(--color-ink-muted);
}

.meta time {
  color: var(--color-ink-ghost);
  font-variant-numeric: tabular-nums;
}

.text {
  padding: 8px 12px;
  border-radius: 4px 16px 16px 16px;
  background: var(--color-paper-hover);
  font-size: 14px;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.mine .text {
  border-radius: 16px 4px 16px 16px;
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

.system {
  align-self: center;
  max-width: 100%;
  padding: 6px 12px;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  font-size: 12px;
  color: var(--color-ink-faint);
  text-align: center;
}

.compose {
  display: flex;
  gap: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--color-line);
}

input {
  flex: 1;
  min-width: 0;
  height: 44px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink);
  font: inherit;
  font-size: 14px;
  outline: none;
}

input:focus {
  box-shadow: inset 0 0 0 1.5px var(--color-ink-faint);
}

input::placeholder {
  color: var(--color-ink-ghost);
}

.compose button {
  display: grid;
  place-items: center;
  flex: none;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 50%;
  background: var(--color-gold);
  color: var(--color-paper);
  cursor: pointer;
}

.compose button:disabled {
  background: var(--color-paper-hover);
  color: var(--color-ink-ghost);
  cursor: default;
}
</style>
