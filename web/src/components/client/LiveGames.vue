<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { PhEye } from '@phosphor-icons/vue'
import ChipIcon from './ChipIcon.vue'
import { listLiveTables } from '../../api/tables'
import { plural } from '../../composables/useLanding'
import type { TableInfo } from '../../types/game'

const props = defineProps<{ now: number }>()
defineEmits<{ (e: 'watch', table: TableInfo): void }>()

const games = ref<TableInfo[]>([])
const loaded = ref(false)
const failed = ref(false)
let timer: number | null = null

async function load() {
  try {
    games.value = await listLiveTables()
    failed.value = false
  } catch {
    failed.value = true
  } finally {
    loaded.value = true
  }
}

function since(t: TableInfo) {
  const min = Math.max(0, Math.floor((props.now - new Date(t.createdAt).getTime()) / 60000))
  return min < 1 ? 'только что' : `${min} мин`
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 5000)
})

onUnmounted(() => {
  if (timer) window.clearInterval(timer)
})
</script>

<template>
  <div class="live">
    <p class="lead">Партии, которые идут прямо сейчас. Зритель видит стол и бой, но не карты на руках, и не может ходить.</p>

    <div v-if="!loaded" class="list" aria-busy="true">
      <span v-for="i in 3" :key="i" class="skeleton"></span>
    </div>

    <ul v-else-if="games.length" class="list">
      <li v-for="g in games" :key="g.id" class="game">
        <span class="suit" :class="{ red: g.mode === 'perevodnoy' }" aria-hidden="true">{{ g.mode === 'perevodnoy' ? '♥' : '♠' }}</span>
        <div class="info">
          <span class="title">
            {{ g.mode === 'perevodnoy' ? 'Переводной' : 'Подкидной' }} · {{ g.players }} {{ plural(g.players, ['игрок', 'игрока', 'игроков']) }}
          </span>
          <span class="sub">{{ g.deck }} карт · идёт {{ since(g) }}</span>
        </div>
        <span class="stake"><ChipIcon :size="16" />{{ (g.stake ?? 0).toLocaleString('ru-RU') }}</span>
        <span class="eyes" :class="{ none: !g.watchers }" :aria-label="`Смотрят: ${g.watchers ?? 0}`">
          <PhEye :size="15" aria-hidden="true" />{{ g.watchers ?? 0 }}
        </span>
        <button type="button" class="watch" @click="$emit('watch', g)">Смотреть</button>
      </li>
    </ul>

    <div v-else class="empty">
      <PhEye :size="28" aria-hidden="true" />
      <p>{{ failed ? 'Не удалось загрузить партии. Попробуем ещё раз через пару секунд.' : 'Сейчас никто не играет. Загляните чуть позже.' }}</p>
    </div>
  </div>
</template>

<style scoped>
.live {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.lead {
  margin: 0;
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-ink-muted);
}

.list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.game {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 12px 12px 16px;
  border-radius: 18px;
  background: var(--color-paper-sunken);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.suit {
  display: grid;
  place-items: center;
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: var(--color-paper-raised);
  font-family: var(--font-card);
  font-size: 22px;
  color: var(--color-ink);
}

.suit.red {
  color: var(--color-red);
}

.info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.title {
  font-size: 15px;
  color: var(--color-ink);
}

.sub {
  font-size: 13px;
  color: var(--color-ink-faint);
}

.stake,
.eyes {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 10px 0 6px;
  border-radius: 999px;
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.stake {
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.eyes {
  padding-left: 9px;
  background: var(--color-paper-hover);
  color: var(--color-ink-muted);
}

.eyes.none {
  color: var(--color-ink-faint);
}

.watch {
  height: 38px;
  padding: 0 18px;
  border: 0;
  border-radius: 999px;
  background: var(--gradient-gold);
  color: var(--color-on-gold);
  font: inherit;
  font-size: 14px;
  cursor: pointer;
}

.watch:focus-visible {
  outline: 2px solid var(--color-gold);
  outline-offset: 2px;
}

.empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 40px 16px;
  text-align: center;
  color: var(--color-ink-faint);
}

.empty p {
  margin: 0;
  font-size: 14px;
}

.skeleton {
  height: 66px;
  border-radius: 18px;
  background: linear-gradient(100deg, var(--color-paper-sunken) 30%, var(--color-paper-hover) 50%, var(--color-paper-sunken) 70%);
  background-size: 200% 100%;
  animation: shimmer 1.4s linear infinite;
}

@keyframes shimmer {
  to {
    background-position: -200% 0;
  }
}

@media (max-width: 560px) {
  .game {
    flex-wrap: wrap;
  }

  .info {
    flex-basis: calc(100% - 54px);
  }

  .stake {
    margin-left: 54px;
  }

  .watch {
    margin-left: auto;
  }
}
</style>
