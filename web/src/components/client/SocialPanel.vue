<script setup lang="ts">
import FriendsList from './FriendsList.vue'
import ChatBox from './ChatBox.vue'
import type { Friend, FriendRequest, ChatEntry } from '../../types/social'

defineProps<{
  friends: Friend[]
  requests: FriendRequest[]
  messages: ChatEntry[]
  myId?: string
  showChat: boolean
}>()

defineEmits<{
  (e: 'add-friend'): void
  (e: 'invite', friend: Friend): void
  (e: 'accept', req: FriendRequest): void
  (e: 'decline', req: FriendRequest): void
  (e: 'send', text: string): void
}>()
</script>

<template>
  <aside class="social" :class="{ 'no-chat': !showChat }" aria-label="Друзья и чат">
    <FriendsList
      class="block friends"
      :friends="friends"
      :requests="requests"
      @add="$emit('add-friend')"
      @invite="(f) => $emit('invite', f)"
      @accept="(r) => $emit('accept', r)"
      @decline="(r) => $emit('decline', r)"
    />
    <ChatBox v-if="showChat" class="block chat" :messages="messages" :my-id="myId" @send="(t) => $emit('send', t)" />
  </aside>
</template>

<style scoped>
.social {
  display: flex;
  flex-direction: column;
  gap: 14px;
  height: 100%;
  min-height: 0;
  padding: 16px;
  box-sizing: border-box;
}

.block {
  padding: 14px 12px;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.friends {
  flex: 0 1 auto;
  max-height: 44%;
}

.no-chat .friends {
  flex: 1;
  max-height: none;
}

.chat {
  flex: 1;
}
</style>
