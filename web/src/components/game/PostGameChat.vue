<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { PhPaperPlaneRight } from '@phosphor-icons/vue'
import AnimatedEmoji from './AnimatedEmoji.vue'
import { EMOJI, emojiOf } from '../../constants/emoji'
import type { ChatEntry } from '../../types/social'

const PHRASES = ['gg', 'Ещё партию?', 'Повезло тебе', 'Хорошо сыграли', 'Реванш!']
const REACTIONS = ['😂', '👏', '🤝', '😎', '😭', '🤡']
const SHOWN = 4

const props = defineProps<{ messages: ChatEntry[]; myId: string }>()
const emit = defineEmits<{ (e: 'send', text: string): void }>()

const draft = ref('')
const listEl = ref<HTMLElement | null>(null)
const reactions = EMOJI.filter((e) => REACTIONS.includes(e.char))
const recent = computed(() => props.messages.filter((m) => !m.system).slice(-SHOWN))

function send(text: string) {
  const value = text.trim()
  if (!value) return
  emit('send', value)
  draft.value = ''
}

watch(
  () => recent.value.length && recent.value[recent.value.length - 1].id,
  async () => {
    await nextTick()
    if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight
  },
)
</script>

<template>
  <section class="post-chat" aria-label="Чат после партии">
    <ol v-if="recent.length" ref="listEl" class="feed">
      <li v-for="m in recent" :key="m.id" class="line" :class="{ mine: m.userId === myId }">
        <b>{{ m.userId === myId ? 'вы' : m.username }}</b>
        <AnimatedEmoji v-if="emojiOf(m.text)" :emoji="emojiOf(m.text)!" :size="26" />
        <span v-else>{{ m.text }}</span>
      </li>
    </ol>

    <div class="quick">
      <button
        v-for="e in reactions"
        :key="e.code"
        type="button"
        class="react"
        :title="e.label"
        :aria-label="e.label"
        @click="send(e.char)"
      >
        <AnimatedEmoji :emoji="e" :size="26" :animated="false" />
      </button>
    </div>

    <div class="phrases">
      <button v-for="p in PHRASES" :key="p" type="button" class="phrase" @click="send(p)">{{ p }}</button>
    </div>

    <form class="compose" @submit.prevent="send(draft)">
      <input v-model="draft" type="text" maxlength="200" placeholder="Пара слов сопернику…" aria-label="Сообщение сопернику" />
      <button type="submit" :disabled="!draft.trim()" aria-label="Отправить">
        <PhPaperPlaneRight :size="16" weight="fill" />
      </button>
    </form>
  </section>
</template>

<style scoped>
.post-chat {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--color-line);
  text-align: left;
}

.feed {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 110px;
  margin: 0;
  padding: 0;
  overflow-y: auto;
  list-style: none;
}

.line {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  line-height: 1.35;
  overflow-wrap: anywhere;
}

.line b {
  flex: none;
  font-size: 12px;
  font-weight: 500;
  color: var(--color-gold-ink);
}

.line.mine b {
  color: var(--color-ink-faint);
}

.quick {
  display: flex;
  justify-content: space-between;
  gap: 4px;
}

.react {
  display: grid;
  place-items: center;
  flex: 1;
  height: 40px;
  border: 0;
  border-radius: 12px;
  background: var(--color-paper-sunken);
  cursor: pointer;
  transition:
    transform 0.2s cubic-bezier(0.22, 1, 0.36, 1),
    background-color 0.2s;
}

.react:hover {
  background: var(--color-paper-hover);
  transform: translateY(-2px) scale(1.08);
}

.phrases {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  scrollbar-width: none;
}

.phrases::-webkit-scrollbar {
  display: none;
}

.phrase {
  flex: none;
  height: 32px;
  padding: 0 12px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 13px;
  white-space: nowrap;
  cursor: pointer;
}

.phrase:hover {
  color: var(--color-ink);
  box-shadow: inset 0 0 0 1px var(--color-line-heavy);
}

.compose {
  display: flex;
  gap: 6px;
}

.compose input {
  flex: 1;
  min-width: 0;
  height: 40px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink);
  font: inherit;
  font-size: 14px;
  outline: none;
}

.compose input:focus {
  box-shadow: inset 0 0 0 1.5px var(--color-ink-faint);
}

.compose button {
  display: grid;
  place-items: center;
  flex: none;
  width: 40px;
  height: 40px;
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
