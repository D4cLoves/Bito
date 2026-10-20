<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { PhUserPlus, PhUserMinus, PhPaperPlaneTilt, PhCheck, PhX, PhMagnifyingGlass } from '@phosphor-icons/vue'
import type { Friend, FriendRequest, FriendStatus } from '../../types/social'
import { avatarHue, avatarInitial } from '../../composables/avatar'

const props = defineProps<{
  friends: Friend[]
  requests: FriendRequest[]
  outgoing: FriendRequest[]
  add: (username: string) => Promise<string>
}>()
defineEmits<{
  (e: 'invite', friend: Friend): void
  (e: 'accept', req: FriendRequest): void
  (e: 'decline', req: FriendRequest): void
  (e: 'cancel', req: FriendRequest): void
  (e: 'remove', friend: Friend): void
}>()

const adding = defineModel<boolean>('adding', { default: false })
const newName = ref('')
const addBusy = ref(false)
const addResult = ref<{ ok: boolean; text: string } | null>(null)
const addInput = ref<HTMLInputElement | null>(null)

watch(
  adding,
  async (open) => {
    addResult.value = null
    if (!open) return
    await nextTick()
    addInput.value?.focus()
  },
  { immediate: true },
)

async function submitAdd() {
  const name = newName.value.trim()
  if (!name || addBusy.value) return
  addBusy.value = true
  try {
    addResult.value = { ok: true, text: await props.add(name) }
    newName.value = ''
  } catch (err) {
    addResult.value = { ok: false, text: err instanceof Error ? err.message : 'Не получилось' }
  } finally {
    addBusy.value = false
  }
}

type Filter = 'online' | 'all'

const ORDER: Record<FriendStatus, number> = { playing: 0, online: 1, offline: 2 }

const query = ref('')
const filter = ref<Filter>('all')

const onlineCount = computed(() => props.friends.filter((f) => f.status !== 'offline').length)

const shown = computed(() => {
  const q = query.value.trim().toLowerCase()
  return props.friends
    .filter((f) => filter.value === 'all' || f.status !== 'offline')
    .filter((f) => !q || f.username.toLowerCase().includes(q))
    .sort((a, b) => ORDER[a.status] - ORDER[b.status] || a.username.localeCompare(b.username, 'ru'))
})

const emptyText = computed(() => {
  if (!props.friends.length) return 'Пока никого. Добавьте друга по имени.'
  if (query.value.trim()) return 'Никого не нашли'
  return 'Сейчас никого нет в сети'
})

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
  <section class="friends" aria-label="Друзья">
    <div class="tools">
      <label class="search">
        <PhMagnifyingGlass :size="17" aria-hidden="true" />
        <input v-model="query" type="search" placeholder="Найти среди друзей" aria-label="Найти среди друзей" />
      </label>

      <div class="filters" role="radiogroup" aria-label="Показать">
        <button
          type="button"
          role="radio"
          :aria-checked="filter === 'online'"
          :class="{ on: filter === 'online' }"
          @click="filter = 'online'"
        >
          В сети <span>{{ onlineCount }}</span>
        </button>
        <button
          type="button"
          role="radio"
          :aria-checked="filter === 'all'"
          :class="{ on: filter === 'all' }"
          @click="filter = 'all'"
        >
          Все <span>{{ friends.length }}</span>
        </button>
      </div>
    </div>

    <section v-if="requests.length" class="group" aria-labelledby="requests-title">
      <h3 id="requests-title">Заявки <span>{{ requests.length }}</span></h3>
      <ul class="rows requests">
        <li v-for="r in requests" :key="r.id" class="row">
          <span class="avatar request" :style="{ '--h': avatarHue(r.username) }">{{ avatarInitial(r.username) }}</span>
          <span class="who">
            <b>{{ r.username }}</b>
            <small>хочет дружить</small>
          </span>
          <button type="button" class="mini ok" :aria-label="`Принять ${r.username}`" title="Принять" @click="$emit('accept', r)">
            <PhCheck :size="15" weight="bold" />
          </button>
          <button type="button" class="mini" :aria-label="`Отклонить ${r.username}`" title="Отклонить" @click="$emit('decline', r)">
            <PhX :size="15" weight="bold" />
          </button>
        </li>
      </ul>
    </section>

    <section class="group" aria-labelledby="list-title">
      <h3 id="list-title">{{ filter === 'online' ? 'В сети' : 'Все друзья' }}</h3>
      <ul v-if="shown.length" class="rows">
        <li v-for="f in shown" :key="f.id" class="row friend" :class="f.status">
          <span class="avatar" :style="{ '--h': avatarHue(f.username) }">
            {{ avatarInitial(f.username) }}
            <i v-if="f.status !== 'offline'" class="dot" aria-hidden="true"></i>
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
          <button
            type="button"
            class="mini invite"
            :aria-label="`Удалить ${f.username} из друзей`"
            title="Удалить из друзей"
            @click="$emit('remove', f)"
          >
            <PhUserMinus :size="15" />
          </button>
        </li>
      </ul>
      <p v-else class="empty">{{ emptyText }}</p>
    </section>

    <section v-if="outgoing.length" class="group" aria-labelledby="outgoing-title">
      <h3 id="outgoing-title">Отправленные <span>{{ outgoing.length }}</span></h3>
      <ul class="rows">
        <li v-for="r in outgoing" :key="r.id" class="row friend offline">
          <span class="avatar" :style="{ '--h': avatarHue(r.username) }">{{ avatarInitial(r.username) }}</span>
          <span class="who">
            <b>{{ r.username }}</b>
            <small>ждёт ответа</small>
          </span>
          <button type="button" class="mini" :aria-label="`Отменить заявку для ${r.username}`" title="Отменить" @click="$emit('cancel', r)">
            <PhX :size="15" weight="bold" />
          </button>
        </li>
      </ul>
    </section>

    <form v-if="adding" class="add-form" @submit.prevent="submitAdd" @keydown.esc="adding = false">
      <label class="search">
        <PhUserPlus :size="17" aria-hidden="true" />
        <input
          ref="addInput"
          v-model="newName"
          type="text"
          maxlength="64"
          autocomplete="off"
          placeholder="Имя игрока"
          aria-label="Имя игрока"
        />
      </label>
      <div class="add-actions">
        <button type="button" class="ghost" @click="adding = false">Отмена</button>
        <button type="submit" class="send" :disabled="!newName.trim() || addBusy">
          {{ addBusy ? 'Отправляем…' : 'Отправить заявку' }}
        </button>
      </div>
      <p v-if="addResult" class="add-result" :class="{ bad: !addResult.ok }" role="status">{{ addResult.text }}</p>
    </form>
    <button v-else type="button" class="add" @click="adding = true">
      <PhUserPlus :size="18" aria-hidden="true" />
      Добавить друга
    </button>
  </section>
</template>

<style scoped>
.friends {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-height: 0;
}

.tools {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.search {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 42px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink-faint);
  transition: box-shadow 0.2s;
}

.search:focus-within {
  box-shadow: inset 0 0 0 1.5px var(--color-gold);
}

.search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: none;
  color: var(--color-ink);
  font: inherit;
  font-size: 14px;
}

.search input::placeholder {
  color: var(--color-ink-ghost);
}

.filters {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.filters button {
  height: 32px;
  border: 0;
  border-radius: 999px;
  background: none;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
  transition: background-color 0.2s, color 0.2s;
}

.filters button span {
  margin-left: 2px;
  color: var(--color-ink-ghost);
  font-variant-numeric: tabular-nums;
}

.filters button:hover {
  color: var(--color-ink);
}

.filters button.on {
  background: var(--color-paper-raised);
  color: var(--color-ink);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.12), inset 0 0 0 1px var(--color-line);
}

.filters button.on span {
  color: var(--color-ink-faint);
}

.group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

h3 {
  margin: 0;
  padding: 0 8px;
  color: var(--color-ink-faint);
  font-size: 12px;
  font-weight: 500;
  letter-spacing: 0.06em;
  text-transform: uppercase;
}

h3 span {
  margin-left: 4px;
  color: var(--color-gold-ink);
}

.rows {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.requests {
  padding: 4px;
  border-radius: 18px;
  background: var(--color-gold-tint);
}

.row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
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
  width: 38px;
  height: 38px;
  border-radius: 50%;
  background: hsl(var(--h) 46% 50%);
  color: #fff;
  font-size: 15px;
  font-weight: 500;
}

.avatar.request {
  box-shadow:
    0 0 0 2px var(--color-gold-tint),
    0 0 0 3.5px var(--color-gold);
}

.dot {
  position: absolute;
  right: -1px;
  bottom: -1px;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: var(--color-green);
  box-shadow: 0 0 0 2.5px var(--color-paper);
}

.playing .dot {
  background: var(--color-red);
}

.friend:hover .dot {
  box-shadow: 0 0 0 2.5px var(--color-paper-hover);
}

.offline .avatar {
  filter: grayscale(0.85);
  opacity: 0.45;
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

.offline .who b {
  color: var(--color-ink-muted);
}

.who small {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.online .who small {
  color: var(--color-green);
}

.playing .who small {
  color: var(--color-red);
}

.mini {
  display: grid;
  place-items: center;
  flex: none;
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: 50%;
  background: var(--color-paper-raised);
  color: var(--color-ink-muted);
  cursor: pointer;
  transition: color 0.2s, transform 0.2s;
}

.mini:hover {
  color: var(--color-ink);
  transform: scale(1.06);
}

.mini.ok {
  background: var(--color-green);
  color: #fff;
}

.invite {
  background: var(--color-paper-sunken);
  opacity: 0;
  transition: opacity 0.2s, color 0.2s;
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

.empty {
  margin: 0;
  padding: 20px 8px;
  color: var(--color-ink-faint);
  font-size: 13px;
  text-align: center;
}

.add {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 44px;
  border: 1.5px dashed var(--color-line-heavy);
  border-radius: 999px;
  background: none;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
  transition: color 0.2s, border-color 0.2s;
}

.add:hover {
  border-color: var(--color-gold);
  color: var(--color-gold-ink);
}

.add-form {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.add-actions {
  display: flex;
  gap: 8px;
}

.add-actions button {
  flex: 1;
  height: 40px;
  border: 0;
  border-radius: 999px;
  font: inherit;
  font-size: 14px;
  cursor: pointer;
  transition: opacity 0.2s, background-color 0.2s;
}

.ghost {
  background: var(--color-paper-sunken);
  color: var(--color-ink-muted);
}

.send {
  background: var(--color-gold);
  color: var(--color-ink);
}

.send:disabled {
  opacity: 0.5;
  cursor: default;
}

.add-result {
  margin: 0;
  padding: 0 8px;
  color: var(--color-green);
  font-size: 13px;
}

.add-result.bad {
  color: var(--color-red);
}
</style>
