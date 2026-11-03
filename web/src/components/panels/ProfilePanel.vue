<script setup lang="ts">
import { PhPencilSimple, PhLockKey } from '@phosphor-icons/vue'
import PlayerCard from '../profile/PlayerCard.vue'
import StatTiles from '../profile/StatTiles.vue'
import GameHistory from './GameHistory.vue'
import AvatarGlyph from '../client/AvatarGlyph.vue'
import { memberSince, rankTitle } from '../../composables/usePlayerRank'
import type { Profile } from '../../types/user'
import type { GameHistoryEntry } from '../../types/social'

defineProps<{ profile: Profile | null; error: string; history: GameHistoryEntry[] }>()
defineEmits<{ (e: 'settings', section: string): void; (e: 'retry'): void }>()
</script>

<template>
  <div class="profile">
    <p v-if="error && !profile" class="load-error" role="alert">
      {{ error }}
      <button type="button" @click="$emit('retry')">Повторить</button>
    </p>

    <div class="top">
      <PlayerCard :profile="profile" class="player-card" />

      <div class="side">
        <div class="who">
          <span class="mini" aria-hidden="true"><AvatarGlyph v-if="profile" :name="profile.username" self /></span>
          <div class="who-text">
            <span class="name">{{ profile?.username ?? '…' }}</span>
            <span class="meta">{{ profile?.email }}<template v-if="profile"> · в Bito с {{ memberSince(profile.createdAt) }}</template></span>
            <span v-if="profile" class="mini-rank">{{ rankTitle(profile.stats.totalGames) }}</span>
          </div>
        </div>

        <StatTiles :profile="profile" />

        <div class="links">
          <button type="button" class="link" @click="$emit('settings', 'account')">
            <PhPencilSimple :size="16" aria-hidden="true" />
            Сменить имя
          </button>
          <button type="button" class="link" @click="$emit('settings', 'security')">
            <PhLockKey :size="16" aria-hidden="true" />
            Пароль и безопасность
          </button>
        </div>
      </div>
    </div>

    <GameHistory :games="history" />
  </div>
</template>

<style scoped>
.profile {
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
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.who-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.mini,
.mini-rank {
  display: none;
}

.name {
  font-size: 26px;
  font-weight: 500;
  letter-spacing: -0.03em;
}

.meta {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  color: var(--color-ink-faint);
}

.links {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
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

@media (max-width: 760px) {
  .profile {
    gap: 22px;
  }

  .top {
    grid-template-columns: 1fr;
  }

  .top .player-card {
    display: none;
  }

  .mini {
    display: grid;
    place-items: center;
    flex: none;
    width: 64px;
    height: 64px;
    border-radius: 50%;
    background: var(--color-gold);
    box-shadow: 0 0 0 3px var(--color-paper-raised), 0 0 0 4.5px var(--color-gold);
    color: var(--color-on-gold);
    font-family: var(--font-card);
    font-size: 26px;
  }

  .mini :deep(svg) {
    width: 32px;
    height: 32px;
  }

  .mini-rank {
    display: inline-block;
    align-self: flex-start;
    margin-top: 4px;
    padding: 2px 10px;
    border-radius: 999px;
    background: var(--color-gold-tint);
    color: var(--color-gold-ink);
    font-size: 12px;
    font-weight: 500;
  }

  .name {
    font-size: 22px;
  }

  .meta {
    font-size: 13px;
  }

  .link {
    flex: 1 0 auto;
    justify-content: center;
    padding: 0 14px;
    white-space: nowrap;
  }
}
</style>
