<script setup lang="ts">
import { computed } from 'vue'
import { PhUsersThree, PhUserPlus } from '@phosphor-icons/vue'
import type { Friend, FriendRequest, FriendStatus } from '../../types/social'

const props = defineProps<{ username: string; friends: Friend[]; requests: FriendRequest[] }>()
defineEmits<{ (e: 'profile'): void; (e: 'social'): void; (e: 'add'): void }>()

const ORDER: Record<FriendStatus, number> = { playing: 0, online: 1, offline: 2 }
const STATUS: Record<FriendStatus, string> = { online: 'в лобби', playing: 'за столом', offline: 'не в сети' }

const sorted = computed(() => [...props.friends].sort((a, b) => ORDER[a.status] - ORDER[b.status]))
const online = computed(() => props.friends.filter((f) => f.status !== 'offline').length)

const initial = (name: string) => (name[0] ?? '?').toUpperCase()

function hue(name: string) {
  let h = 0
  for (const ch of name) h = (h * 31 + (ch.codePointAt(0) ?? 0)) % 360
  return h
}
</script>

<template>
  <aside class="friends-rail" aria-label="Друзья">
    <button type="button" class="me" :title="`${username} — профиль`" :aria-label="`Профиль: ${username}`" @click="$emit('profile')">
      <span class="avatar" :style="{ '--h': hue(username) }">{{ initial(username) }}</span>
    </button>

    <button type="button" class="friends-btn" title="Друзья и чат" aria-label="Друзья и чат" @click="$emit('social')">
      <PhUsersThree :size="22" aria-hidden="true" />
      <span class="count" :aria-label="`${online} в сети`">{{ online }}</span>
    </button>

    <ul class="list">
      <li v-for="r in requests" :key="r.id">
        <button
          type="button"
          class="friend request"
          :title="`${r.username} хочет дружить`"
          :aria-label="`${r.username} хочет дружить`"
          @click="$emit('social')"
        >
          <span class="avatar" :style="{ '--h': hue(r.username) }">
            {{ initial(r.username) }}
            <i class="plus" aria-hidden="true">+</i>
          </span>
        </button>
      </li>
      <li v-for="f in sorted" :key="f.id">
        <button
          type="button"
          class="friend"
          :class="f.status"
          :title="`${f.username} — ${STATUS[f.status]}`"
          :aria-label="`${f.username}, ${STATUS[f.status]}`"
          @click="$emit('social')"
        >
          <span class="avatar" :style="{ '--h': hue(f.username) }">
            {{ initial(f.username) }}
            <i class="dot" aria-hidden="true"></i>
          </span>
          <span v-if="f.status === 'playing'" class="tag" aria-hidden="true">в игре</span>
        </button>
      </li>
    </ul>

    <button type="button" class="add" title="Добавить друга" aria-label="Добавить друга" @click="$emit('add')">
      <PhUserPlus :size="20" />
    </button>
  </aside>
</template>

<style scoped>
.friends-rail {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  height: calc(100% - 28px);
  margin: 14px;
  padding: 14px 0;
  box-sizing: border-box;
  border-radius: 30px;
  background: var(--color-paper-raised);
  box-shadow:
    inset 0 0 0 1px var(--color-line),
    0 24px 48px -36px rgba(0, 0, 0, 0.5);
}

button {
  border: 0;
  padding: 0;
  background: none;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.avatar {
  position: relative;
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: hsl(var(--h) 46% 50%);
  color: #fff;
  font-size: 16px;
  font-weight: 500;
  transition: transform 0.2s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.me .avatar {
  width: 48px;
  height: 48px;
  box-shadow:
    0 0 0 3px var(--color-paper-raised),
    0 0 0 5px var(--color-gold);
}

.me:hover .avatar,
.friend:hover .avatar {
  transform: scale(1.07);
}

.friends-btn {
  position: relative;
  display: grid;
  place-items: center;
  width: 48px;
  height: 40px;
  margin-top: 6px;
  border-top: 1px solid var(--color-line);
  padding-top: 10px;
  color: var(--color-ink-faint);
}

.friends-btn:hover {
  color: var(--color-ink);
}

.count {
  position: absolute;
  top: 6px;
  right: 2px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  box-sizing: border-box;
  border-radius: 999px;
  background: var(--color-green);
  color: #fff;
  font-size: 10px;
  line-height: 16px;
  font-variant-numeric: tabular-nums;
}

.list {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  flex: 1;
  min-height: 0;
  margin: 0;
  padding: 4px 0;
  overflow-y: auto;
  list-style: none;
  scrollbar-width: none;
}

.list::-webkit-scrollbar {
  display: none;
}

.friend {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
}

.dot {
  position: absolute;
  right: -1px;
  bottom: -1px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-ink-ghost);
  box-shadow: 0 0 0 3px var(--color-paper-raised);
}

.online .dot {
  background: var(--color-green);
}

.playing .dot {
  background: var(--color-red);
}

.offline .avatar {
  filter: grayscale(0.85);
  opacity: 0.5;
}

.tag {
  margin-top: -7px;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--color-red);
  color: #fff;
  font-size: 9px;
  line-height: 14px;
  white-space: nowrap;
  box-shadow: 0 0 0 2px var(--color-paper-raised);
}

.request .avatar {
  box-shadow:
    0 0 0 2px var(--color-paper-raised),
    0 0 0 4px var(--color-gold);
}

.plus {
  position: absolute;
  right: -4px;
  bottom: -4px;
  display: grid;
  place-items: center;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: var(--color-gold);
  color: var(--color-paper);
  font-size: 14px;
  font-style: normal;
  line-height: 1;
  box-shadow: 0 0 0 2px var(--color-paper-raised);
}

.add {
  display: grid;
  place-items: center;
  flex: none;
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 1.5px dashed var(--color-line-heavy);
  color: var(--color-ink-faint);
  transition: color 0.2s, border-color 0.2s;
}

.add:hover {
  border-color: var(--color-gold);
  color: var(--color-gold-ink);
}
</style>
