<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { motion, AnimatePresence } from 'motion-v'
import { PhArrowLeft, PhUserCircle, PhLockKey } from '@phosphor-icons/vue'
import AppHeader from '../../components/app/AppHeader.vue'
import ToastStack from '../../components/app/ToastStack.vue'
import PlayerCard from '../../components/profile/PlayerCard.vue'
import StatTiles from '../../components/profile/StatTiles.vue'
import AccountForm from '../../components/profile/AccountForm.vue'
import SecurityForm from '../../components/profile/SecurityForm.vue'
import { useProfileStore } from '../../stores/profile'

type Tab = 'account' | 'security'

const TABS: { id: Tab; label: string; icon: typeof PhUserCircle }[] = [
  { id: 'account', label: 'Аккаунт', icon: PhUserCircle },
  { id: 'security', label: 'Безопасность', icon: PhLockKey },
]

const route = useRoute()
const router = useRouter()
const profileStore = useProfileStore()

const tab = computed<Tab>(() => (route.query.tab === 'security' ? 'security' : 'account'))
const profile = computed(() => profileStore.profile)

function selectTab(t: Tab) {
  router.replace({ query: { ...route.query, tab: t === 'account' ? undefined : t } })
}

onMounted(() => profileStore.load(true))

const spring = { type: 'spring', stiffness: 380, damping: 32 } as const
</script>

<template>
  <div class="profile-page">
    <AppHeader />

    <main class="wrap">
      <RouterLink to="/lobby" class="back">
        <PhArrowLeft :size="18" weight="bold" aria-hidden="true" />
        В лобби
      </RouterLink>

      <p v-if="profileStore.error && !profile" class="load-error" role="alert">
        {{ profileStore.error }}
        <button type="button" @click="profileStore.load(true)">Повторить</button>
      </p>

      <div class="layout">
        <aside class="side">
          <PlayerCard :profile="profile" />
        </aside>

        <section class="content">
          <h1 class="rise" style="--d: 1">Ваш профиль</h1>

          <StatTiles :profile="profile" />

          <div class="tabs" role="tablist" aria-label="Разделы профиля">
            <button
              v-for="t in TABS"
              :key="t.id"
              type="button"
              role="tab"
              :aria-selected="tab === t.id"
              :class="{ on: tab === t.id }"
              @click="selectTab(t.id)"
            >
              <component :is="t.icon" :size="18" :weight="tab === t.id ? 'fill' : 'regular'" aria-hidden="true" />
              {{ t.label }}
              <motion.span v-if="tab === t.id" layout-id="tab-underline" class="underline" :transition="spring" />
            </button>
          </div>

          <AnimatePresence mode="wait" :initial="false">
            <motion.div
              :key="tab"
              role="tabpanel"
              :initial="{ opacity: 0, x: tab === 'security' ? 24 : -24 }"
              :animate="{ opacity: 1, x: 0 }"
              :exit="{ opacity: 0, x: tab === 'security' ? -24 : 24 }"
              :transition="{ duration: 0.22 }"
            >
              <AccountForm v-if="tab === 'account'" :profile="profile" />
              <SecurityForm v-else />
            </motion.div>
          </AnimatePresence>
        </section>
      </div>
    </main>

    <ToastStack />
  </div>
</template>

<style scoped>
.profile-page {
  min-height: 100vh;
  background:
    radial-gradient(ellipse 60% 45% at 20% 0%, rgba(246, 232, 200, 0.7), rgba(246, 232, 200, 0) 70%),
    var(--color-paper);
  color: var(--color-ink);
  overflow-x: hidden;
}

.wrap {
  max-width: 1120px;
  margin: 0 auto;
  padding: 28px 24px 80px;
}

.back {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 16px 0 12px;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: 0 0 0 1px rgba(214, 208, 196, 0.9);
  color: var(--color-ink);
  font-size: 14px;
  text-decoration: none;
  transition: background-color 0.2s;
}

.back:hover {
  background: var(--color-paper-sunken);
}

.load-error {
  margin: 18px 0 0;
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

.layout {
  display: grid;
  grid-template-columns: 340px minmax(0, 1fr);
  gap: 44px;
  align-items: start;
  margin-top: 26px;
}

.side {
  position: sticky;
  top: 100px;
}

.content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-width: 0;
}

h1 {
  margin: 0;
  font-size: clamp(32px, 4vw, 46px);
  font-weight: 500;
  letter-spacing: -0.035em;
}

.tabs {
  display: flex;
  gap: 6px;
  margin-top: 10px;
  border-bottom: 1px solid var(--color-line);
}

.tabs button {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 48px;
  padding: 0 14px;
  border: 0;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 15px;
  cursor: pointer;
  transition: color 0.2s;
}

.tabs button:hover,
.tabs button.on {
  color: var(--color-ink);
}

.underline {
  position: absolute;
  left: 8px;
  right: 8px;
  bottom: -1px;
  height: 2px;
  border-radius: 2px;
  background: var(--color-ink);
}

@media (max-width: 900px) {
  .layout {
    grid-template-columns: 1fr;
    gap: 24px;
  }

  .side {
    position: static;
  }
}

@media (max-width: 520px) {
  .wrap {
    padding: 20px 14px 64px;
  }
}
</style>
