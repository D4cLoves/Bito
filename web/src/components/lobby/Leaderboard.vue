<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { Trophy } from 'lucide-vue-next'
import { getLeaderboard } from '../../api/users'
import { isNotImplemented } from '../../api/http'
import type { LeaderboardEntry } from '../../types/user'

defineProps<{ myId?: string }>()

type State = 'loading' | 'ready' | 'soon' | 'error'

const state = ref<State>('loading')
const entries = ref<LeaderboardEntry[]>([])

async function load() {
  state.value = 'loading'
  try {
    entries.value = await getLeaderboard(8)
    state.value = 'ready'
  } catch (err) {
    state.value = isNotImplemented(err) ? 'soon' : 'error'
  }
}

onMounted(load)
</script>

<template>
  <section class="board" aria-labelledby="board-title">
    <div class="head">
      <Trophy :size="18" aria-hidden="true" />
      <h2 id="board-title">Топ игроков</h2>
    </div>

    <ol v-if="state === 'loading'" class="list" aria-busy="true">
      <li v-for="i in 5" :key="i" class="sk"></li>
    </ol>

    <div v-else-if="state === 'soon'" class="note">
      <div class="podium" aria-hidden="true">
        <span class="p p2"></span>
        <span class="p p1"></span>
        <span class="p p3"></span>
      </div>
      <p class="note-title">Рейтинг скоро откроется</p>
      <p class="note-text">Здесь появятся лучшие игроки по ELO.</p>
    </div>

    <div v-else-if="state === 'error'" class="note">
      <p class="note-text">Не удалось загрузить рейтинг.</p>
      <button type="button" class="ghost" @click="load">Повторить</button>
    </div>

    <p v-else-if="!entries.length" class="note note-text">Пока никто не сыграл ни одной партии.</p>

    <ol v-else class="list">
      <li v-for="(e, i) in entries" :key="e.id" class="entry" :class="{ me: e.id === myId }" :style="{ '--i': i }">
        <span class="place" :class="`top${i + 1}`">{{ i + 1 }}</span>
        <span class="who">{{ e.username }}</span>
        <span class="elo">{{ e.elo }}</span>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.board {
  padding: 24px;
  border-radius: 28px;
  background: var(--color-paper-raised);
  box-shadow:
    0 0 0 1px rgba(214, 208, 196, 0.9),
    0 24px 48px -32px rgba(70, 62, 48, 0.35);
}

.head {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--color-gold-ink);
}

h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  letter-spacing: -0.02em;
  color: var(--color-ink);
}

.list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin: 16px 0 0;
  padding: 0;
  list-style: none;
}

.entry {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 9px 10px;
  border-radius: 14px;
  animation: rise 0.45s cubic-bezier(0.2, 0.8, 0.2, 1) both;
  animation-delay: calc(var(--i) * 45ms);
}

.entry.me {
  background: var(--color-gold-tint);
}

.place {
  display: grid;
  place-items: center;
  flex: none;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--color-paper-sunken);
  font-size: 12px;
  color: var(--color-ink-muted);
  font-variant-numeric: tabular-nums;
}

.place.top1 {
  background: var(--color-gold);
  color: var(--color-paper-raised);
}

.place.top2,
.place.top3 {
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

.who {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
}

.elo {
  font-size: 14px;
  color: var(--color-ink-muted);
  font-variant-numeric: tabular-nums;
}

.sk {
  height: 40px;
  border-radius: 14px;
  background: linear-gradient(90deg, var(--color-paper-sunken) 0%, var(--color-paper-raised) 50%, var(--color-paper-sunken) 100%);
  background-size: 200% 100%;
  animation: shimmer 1.2s linear infinite;
}

.note {
  padding: 18px 4px 6px;
  text-align: center;
}

.podium {
  display: flex;
  align-items: flex-end;
  justify-content: center;
  gap: 6px;
  height: 64px;
  margin-bottom: 14px;
}

.p {
  width: 30px;
  border-radius: 8px 8px 4px 4px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
  transform-origin: bottom;
  animation: grow 0.7s cubic-bezier(0.2, 0.8, 0.2, 1) both;
}

.p1 {
  height: 64px;
  background: var(--color-gold-tint);
  animation-delay: 0.1s;
}

.p2 {
  height: 44px;
}

.p3 {
  height: 32px;
  animation-delay: 0.2s;
}

.note-title {
  margin: 0;
  font-size: 15px;
}

.note-text {
  margin: 6px 0 0;
  font-size: 14px;
  line-height: 1.45;
  color: var(--color-ink-subtle);
}

.ghost {
  height: 38px;
  margin-top: 12px;
  padding: 0 16px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  color: var(--color-ink);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(6px);
  }
}

@keyframes grow {
  from {
    transform: scaleY(0);
  }
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .entry,
  .p {
    animation: none;
  }
}
</style>
