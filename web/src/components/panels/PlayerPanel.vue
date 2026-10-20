<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { PhUserPlus, PhUserMinus, PhCheck, PhX, PhPaperPlaneTilt, PhUserCircle } from '@phosphor-icons/vue'
import PlayerCard from '../profile/PlayerCard.vue'
import StatTiles from '../profile/StatTiles.vue'
import GameHistory from './GameHistory.vue'
import { getPlayer } from '../../api/friends'
import { memberSince } from '../../composables/usePlayerRank'
import type { Profile } from '../../types/user'
import type { FriendStatus, PlayerProfile } from '../../types/social'

const props = defineProps<{ id: string | null; revision: number }>()
defineEmits<{
  (e: 'add', username: string): void
  (e: 'accept', player: PlayerProfile): void
  (e: 'decline', player: PlayerProfile): void
  (e: 'cancel', player: PlayerProfile): void
  (e: 'remove', player: PlayerProfile): void
  (e: 'invite', player: PlayerProfile): void
  (e: 'own'): void
}>()

const STATUS: Record<FriendStatus, string> = { online: 'в лобби', playing: 'за столом', offline: 'не в сети' }

const player = ref<PlayerProfile | null>(null)
const error = ref('')

let seq = 0
async function load() {
  const id = props.id
  const mine = ++seq
  if (!id) {
    player.value = null
    error.value = 'Игрок не выбран'
    return
  }
  try {
    const p = await getPlayer(id)
    if (mine !== seq) return
    player.value = p
    error.value = ''
  } catch (err) {
    if (mine !== seq) return
    error.value = err instanceof Error ? err.message : 'Не удалось загрузить профиль'
  }
}

watch(
  () => props.id,
  () => {
    player.value = null
    load()
  },
  { immediate: true },
)
watch(() => props.revision, load)

const asProfile = computed<Profile | null>(() => {
  const p = player.value
  if (!p) return null
  return {
    id: p.id,
    username: p.username,
    email: '',
    createdAt: p.createdAt,
    stats: { elo: 0, ...p.stats },
    wallet: { balance: 0 },
  }
})
</script>

<template>
  <div class="player">
    <p v-if="error && !player" class="load-error" role="alert">
      {{ error }}
      <button v-if="id" type="button" @click="load">Повторить</button>
    </p>

    <div class="top">
      <PlayerCard :profile="asProfile" class="player-card" />

      <div class="side">
        <div class="who">
          <span class="name">{{ player?.username ?? '…' }}</span>
          <span v-if="player" class="meta">
            <i v-if="player.status" class="status" :class="player.status">{{ STATUS[player.status] }}</i>
            в Bito с {{ memberSince(player.createdAt) }}
          </span>
        </div>

        <StatTiles :profile="asProfile" hide-wallet />

        <div v-if="player" class="links">
          <template v-if="player.relation === 'friend'">
            <button v-if="player.status === 'online'" type="button" class="link primary" @click="$emit('invite', player)">
              <PhPaperPlaneTilt :size="16" aria-hidden="true" />
              Позвать за стол
            </button>
            <button type="button" class="link" @click="$emit('remove', player)">
              <PhUserMinus :size="16" aria-hidden="true" />
              Удалить из друзей
            </button>
          </template>

          <template v-else-if="player.relation === 'incoming'">
            <span class="note">Хочет добавить вас в друзья</span>
            <button type="button" class="link primary" @click="$emit('accept', player)">
              <PhCheck :size="16" weight="bold" aria-hidden="true" />
              Принять
            </button>
            <button type="button" class="link" @click="$emit('decline', player)">
              <PhX :size="16" weight="bold" aria-hidden="true" />
              Отклонить
            </button>
          </template>

          <template v-else-if="player.relation === 'outgoing'">
            <span class="note">Заявка отправлена, ждём ответа</span>
            <button type="button" class="link" @click="$emit('cancel', player)">
              <PhX :size="16" weight="bold" aria-hidden="true" />
              Отменить заявку
            </button>
          </template>

          <button v-else-if="player.relation === 'none'" type="button" class="link primary" @click="$emit('add', player.username)">
            <PhUserPlus :size="16" aria-hidden="true" />
            Добавить в друзья
          </button>

          <button v-else type="button" class="link" @click="$emit('own')">
            <PhUserCircle :size="16" aria-hidden="true" />
            Это вы — открыть мой профиль
          </button>
        </div>
      </div>
    </div>

    <GameHistory v-if="player" :games="player.games" hide-payout empty-text="Пока ни одной сыгранной партии." />
  </div>
</template>

<style scoped>
.player {
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.load-error {
  margin: 0;
  padding: 12px 16px;
  border-radius: 14px;
  background: var(--color-red-tint);
  color: var(--color-red-strong);
}

.load-error button {
  margin-left: 10px;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  text-decoration: underline;
  cursor: pointer;
}

.top {
  display: grid;
  grid-template-columns: 250px minmax(0, 1fr);
  gap: 28px;
  align-items: start;
}

.player-card :deep(.card) {
  width: 100%;
}

.side {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.who {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.name {
  font-size: 26px;
  font-weight: 500;
  letter-spacing: -0.03em;
}

.meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-ink-faint);
}

.status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-style: normal;
}

.status::before {
  content: '';
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-ink-ghost);
}

.status.online {
  color: var(--color-green);
}

.status.online::before {
  background: var(--color-green);
}

.status.playing {
  color: var(--color-red);
}

.status.playing::before {
  background: var(--color-red);
}

.links {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.note {
  width: 100%;
  font-size: 14px;
  color: var(--color-ink-muted);
}

.link {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-hover);
  color: var(--color-ink);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.link:hover {
  background: var(--color-line-stronger);
}

.link.primary {
  background: var(--color-gold);
}

.link.primary:hover {
  filter: brightness(0.96);
}

@media (max-width: 760px) {
  .top {
    grid-template-columns: 1fr;
  }

  .player-card {
    max-width: 240px;
    margin: 0 auto;
  }
}
</style>
