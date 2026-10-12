<script setup lang="ts">
import { computed } from 'vue'
import { PhUserPlus, PhPaperPlaneTilt, PhCheck, PhX } from '@phosphor-icons/vue'
import type { Friend, FriendRequest } from '../../types/social'

const props = defineProps<{ friends: Friend[]; requests: FriendRequest[] }>()
defineEmits<{
  (e: 'add'): void
  (e: 'invite', friend: Friend): void
  (e: 'accept', req: FriendRequest): void
  (e: 'decline', req: FriendRequest): void
}>()

const ORDER = { online: 0, playing: 1, offline: 2 } as const
const sorted = computed(() => [...props.friends].sort((a, b) => ORDER[a.status] - ORDER[b.status]))
const onlineCount = computed(() => props.friends.filter((f) => f.status !== 'offline').length)

function statusText(f: Friend) {
  if (f.status === 'online') return 'в лобби'
  if (f.status === 'playing') return 'за столом'
  if (!f.lastSeen) return 'не в сети'
  const m = Math.round((Date.now() - new Date(f.lastSeen).getTime()) / 60000)
  if (m < 60) return `был ${m} мин назад`
  const h = Math.round(m / 60)
  return h < 24 ? `был ${h} ч назад` : 'был давно'
}
</script>

<template>
  <section class="friends" aria-labelledby="friends-title">
    <div class="head">
      <h2 id="friends-title">Друзья</h2>
      <span class="count">{{ onlineCount }} в сети</span>
      <button type="button" class="add" aria-label="Добавить друга" title="Добавить друга" @click="$emit('add')">
        <PhUserPlus :size="18" />
      </button>
    </div>

    <ul v-if="requests.length" class="requests">
      <li v-for="r in requests" :key="r.id" class="request">
        <span class="avatar">{{ r.username[0].toUpperCase() }}</span>
        <span class="who">
          <b>{{ r.username }}</b>
          <small>хочет дружить</small>
        </span>
        <button type="button" class="mini ok" aria-label="Принять" @click="$emit('accept', r)"><PhCheck :size="15" weight="bold" /></button>
        <button type="button" class="mini" aria-label="Отклонить" @click="$emit('decline', r)"><PhX :size="15" weight="bold" /></button>
      </li>
    </ul>

    <ul class="list">
      <li v-for="f in sorted" :key="f.id" class="friend" :class="f.status">
        <span class="avatar">
          {{ f.username[0].toUpperCase() }}
          <i class="dot" aria-hidden="true"></i>
        </span>
        <span class="who">
          <b>{{ f.username }}</b>
          <small>{{ statusText(f) }}</small>
        </span>
        <button
          v-if="f.status === 'online'"
          type="button"
          class="mini invite"
          :aria-label="`Позвать ${f.username} за стол`"
          title="Позвать за стол"
          @click="$emit('invite', f)"
        >
          <PhPaperPlaneTilt :size="15" />
        </button>
      </li>
    </ul>
  </section>
</template>

<style scoped>
.friends {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 4px 10px;
}

h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
}

.count {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.add {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  margin-left: auto;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  cursor: pointer;
}

.add:hover {
  color: var(--color-ink);
}

.requests,
.list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.requests {
  margin-bottom: 8px;
  padding: 6px;
  border-radius: 16px;
  background: var(--color-gold-tint);
}

.list {
  overflow-y: auto;
}

.request,
.friend {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 8px;
  border-radius: 14px;
}

.friend:hover {
  background: var(--color-paper-hover);
}

.avatar {
  position: relative;
  display: grid;
  place-items: center;
  flex: none;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  font-size: 14px;
}

.dot {
  position: absolute;
  right: -1px;
  bottom: -1px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-ink-ghost);
  box-shadow: 0 0 0 2.5px var(--color-paper-raised);
}

.online .dot {
  background: var(--color-green);
}

.playing .dot {
  background: var(--color-gold);
}

.offline {
  opacity: 0.6;
}

.who {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
}

.who b {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 500;
}

.who small {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.playing .who small {
  color: var(--color-gold-ink);
}

.mini {
  display: grid;
  place-items: center;
  flex: none;
  width: 30px;
  height: 30px;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
  cursor: pointer;
}

.mini:hover {
  color: var(--color-ink);
}

.mini.ok {
  background: var(--color-green);
  color: #fff;
}

.invite {
  opacity: 0;
  transition: opacity 0.2s;
}

.friend:hover .invite,
.invite:focus-visible {
  opacity: 1;
}

@media (hover: none) {
  .invite {
    opacity: 1;
  }
}
</style>
