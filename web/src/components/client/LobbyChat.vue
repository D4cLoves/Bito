<script setup lang="ts">
import ChatBox from './ChatBox.vue'
import type { ChatEntry } from '../../types/social'

defineProps<{ messages: ChatEntry[]; myId?: string; online: number }>()
defineEmits<{ (e: 'send', text: string): void }>()
</script>

<template>
  <section class="lobby-chat">
    <ChatBox class="box" title="Чат сервера" placeholder="Написать всем…" :messages="messages" :my-id="myId" @send="(t) => $emit('send', t)">
      <template #head>
        <span class="online"><i aria-hidden="true"></i>{{ online }} онлайн</span>
      </template>
    </ChatBox>
  </section>
</template>

<style scoped>
.lobby-chat {
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 16px 14px 14px;
  border-radius: 26px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.box {
  flex: 1;
  min-height: 0;
}

.online {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
  font-size: 12px;
  color: var(--color-ink-faint);
  font-variant-numeric: tabular-nums;
}

.online i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--color-green);
}
</style>
