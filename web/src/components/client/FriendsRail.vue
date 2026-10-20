<script setup lang="ts">
import { computed } from 'vue'
import { PhUsersThree, PhUserPlus, PhPlus } from '@phosphor-icons/vue'
import type { Friend, FriendRequest, FriendStatus } from '../../types/social'
import { avatarTone, avatarInitial } from '../../composables/avatar'

const props = defineProps<{ username: string; friends: Friend[]; requests: FriendRequest[] }>()
defineEmits<{ (e: 'profile'): void; (e: 'friends'): void; (e: 'add'): void; (e: 'open', id: string): void }>()

const ORDER: Record<FriendStatus, number> = { playing: 0, online: 1, offline: 2 }
const STATUS: Record<FriendStatus, string> = { online: 'в лобби', playing: 'за столом', offline: 'не в сети' }

const sorted = computed(() => [...props.friends].sort((a, b) => ORDER[a.status] - ORDER[b.status]))
const online = computed(() => props.friends.filter((f) => f.status !== 'offline').length)
</script>

<template>
  <aside class="friends-rail" aria-label="Друзья">
    <button type="button" class="me" :title="`${username} — профиль`" :aria-label="`Профиль: ${username}`" @click="$emit('profile')">
      <span class="avatar" :style="{ '--tone': avatarTone(username) }">{{ avatarInitial(username) }}</span>
    </button>

    <button type="button" class="friends-btn" title="Друзья" :aria-label="`Друзья, ${online} в сети`" @click="$emit('friends')">
      <PhUsersThree :size="22" aria-hidden="true" />
      <span v-if="online" class="count" aria-hidden="true">{{ online }}</span>
    </button>

    <ul class="list">
      <li v-for="r in requests" :key="r.id">
        <button
          type="button"
          class="friend request"
          :title="`${r.username} хочет дружить`"
          :aria-label="`${r.username} хочет дружить`"
          @click="$emit('friends')"
        >
          <span class="avatar" :style="{ '--tone': avatarTone(r.username) }">
            {{ avatarInitial(r.username) }}
            <i class="plus" aria-hidden="true"><PhPlus :size="10" weight="bold" /></i>
          </span>
        </button>
      </li>
      <li v-if="requests.length && sorted.length" class="sep" aria-hidden="true"></li>
      <li v-for="f in sorted" :key="f.id">
        <button
          type="button"
          class="friend"
          :class="f.status"
          :title="`${f.username} — ${STATUS[f.status]}`"
          :aria-label="`${f.username}, ${STATUS[f.status]}`"
          @click="$emit('open', f.id)"
        >
          <span class="avatar" :style="{ '--tone': avatarTone(f.username) }">
            {{ avatarInitial(f.username) }}
            <i v-if="f.status === 'online'" class="dot" aria-hidden="true"></i>
          </span>
          <span v-if="f.status === 'playing'" class="tag" aria-hidden="true">в игре</span>
        </button>
      </li>
    </ul>

    <button type="button" class="add" title="Добавить друга" aria-label="Добавить друга" @click="$emit('add')">
      <PhUserPlus :size="20" aria-hidden="true" />
    </button>
  </aside>
</template>

<style scoped>
.friends-rail {
  display: flex;
  flex-direction: column;
  align-items: center;
  height: calc(100% - 28px);
  margin: 14px;
  padding: 16px 0 14px;
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
  background: color-mix(in srgb, var(--color-gold) var(--tone, 52%), #000);
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
  flex: none;
  width: 44px;
  height: 44px;
  margin: 18px 0 8px;
  border-radius: 14px;
  color: var(--color-ink-faint);
  transition: color 0.2s, background-color 0.2s;
}

.friends-btn::before {
  content: '';
  position: absolute;
  top: -10px;
  left: 50%;
  width: 28px;
  height: 1px;
  margin-left: -14px;
  background: var(--color-line-heavy);
}

.friends-btn:hover {
  color: var(--color-ink);
  background: var(--color-line);
}

.count {
  position: absolute;
  top: 2px;
  right: 0;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  box-sizing: border-box;
  border-radius: 999px;
  background: var(--color-green);
  color: var(--color-on-green);
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
  font-variant-numeric: tabular-nums;
  box-shadow: 0 0 0 2px var(--color-paper-raised);
}

.list {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 14px;
  flex: 1;
  width: 100%;
  min-height: 0;
  margin: 0;
  padding: 8px 0 12px;
  box-sizing: border-box;
  overflow-y: auto;
  list-style: none;
  scrollbar-width: none;
  mask-image: linear-gradient(to bottom, transparent, #000 8px, #000 calc(100% - 16px), transparent);
}

.list::-webkit-scrollbar {
  display: none;
}

.list > li {
  flex: none;
}

.list > .sep {
  width: 20px;
  height: 1px;
  background: var(--color-line-heavy);
}

.friend {
  position: relative;
  display: block;
}

.dot {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: var(--color-green);
  box-shadow: 0 0 0 2.5px var(--color-paper-raised);
}

.friend.playing {
  margin-bottom: 6px;
}

.playing .avatar {
  box-shadow:
    0 0 0 2px var(--color-paper-raised),
    0 0 0 4px var(--color-red);
}

.tag {
  position: absolute;
  left: 50%;
  bottom: -7px;
  transform: translateX(-50%);
  padding: 0 6px;
  border-radius: 999px;
  background: var(--color-red);
  color: var(--color-on-red);
  font-size: 9px;
  font-weight: 600;
  line-height: 15px;
  letter-spacing: 0.02em;
  white-space: nowrap;
  box-shadow: 0 0 0 2px var(--color-paper-raised);
  pointer-events: none;
}

.offline .avatar {
  filter: grayscale(0.85);
  opacity: 0.45;
}

.offline:hover .avatar {
  opacity: 0.75;
}

.request .avatar {
  box-shadow:
    0 0 0 2px var(--color-paper-raised),
    0 0 0 4px var(--color-gold);
}

.plus {
  position: absolute;
  right: -3px;
  bottom: -3px;
  display: grid;
  place-items: center;
  width: 17px;
  height: 17px;
  border-radius: 50%;
  background: var(--color-gold);
  color: var(--color-paper);
  box-shadow: 0 0 0 2px var(--color-paper-raised);
}

.add {
  display: grid;
  place-items: center;
  flex: none;
  width: 44px;
  height: 44px;
  margin-top: 6px;
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
