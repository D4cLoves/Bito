<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { PhSignOut } from '@phosphor-icons/vue'
import { useAuthStore } from '../../stores/auth'
import { useProfileStore } from '../../stores/profile'

const router = useRouter()
const auth = useAuthStore()
const profileStore = useProfileStore()

const balance = computed(() => profileStore.profile?.wallet.balance)

onMounted(() => {
  if (auth.isAuthenticated) profileStore.load()
})

const initial = computed(() => (auth.user?.username?.[0] ?? '?').toUpperCase())

function logout() {
  auth.logout()
  profileStore.reset()
  router.push('/')
}
</script>

<template>
  <header class="app-header">
    <RouterLink to="/lobby" class="brand" aria-label="Bito — лобби">
      <svg viewBox="0 0 40 38" aria-hidden="true">
        <rect x="17.5" y="4.5" width="18" height="26" rx="3.2" transform="rotate(12 26.5 17.5)" fill="none" stroke="currentColor" stroke-width="2" />
        <rect x="4" y="5.5" width="18" height="26" rx="3.2" transform="rotate(-14 13 18.5)" fill="#f5f4ef" stroke="currentColor" stroke-width="2" />
        <path
          transform="rotate(-14 13 18.5)"
          d="M13 13.2c-1.9 2.3-4.1 3.9-4.1 6 0 1.4 1 2.3 2.2 2.3.8 0 1.4-.4 1.6-.9-.1 1-.5 1.9-1.2 2.6h3c-.7-.7-1.1-1.6-1.2-2.6.2.5.8.9 1.6.9 1.2 0 2.2-.9 2.2-2.3 0-2.1-2.2-3.7-4.1-6z"
          fill="currentColor"
        />
      </svg>
      <span>Bito</span>
    </RouterLink>

    <div class="middle">
      <slot />
    </div>

    <div class="me">
      <RouterLink to="/profile" class="me-btn" title="Профиль">
        <span class="avatar" aria-hidden="true">{{ initial }}</span>
        <span class="name">{{ auth.user?.username ?? 'Игрок' }}</span>
        <span v-if="balance !== undefined" class="elo">{{ balance.toLocaleString('ru-RU') }}</span>
      </RouterLink>
      <button type="button" class="icon-btn" title="Выйти" aria-label="Выйти" @click="logout">
        <PhSignOut :size="18" weight="bold" />
      </button>
    </div>
  </header>
</template>

<style scoped>
.app-header {
  position: sticky;
  z-index: 50;
  top: 14px;
  width: min(1120px, calc(100% - 24px));
  height: 60px;
  margin: 14px auto 0;
  box-sizing: border-box;
  padding: 0 8px 0 18px;
  display: flex;
  align-items: center;
  gap: 12px;
  border-radius: 999px;
  background: rgba(247, 246, 241, 0.78);
  box-shadow:
    0 0 0 1px rgba(214, 208, 196, 0.9),
    0 18px 40px -22px rgba(70, 62, 48, 0.35);
  backdrop-filter: blur(18px) saturate(1.1);
  -webkit-backdrop-filter: blur(18px) saturate(1.1);
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 21px;
  letter-spacing: -0.01em;
  color: var(--color-ink);
  text-decoration: none;
}

.brand svg {
  width: 30px;
  height: 29px;
  color: var(--color-ink);
}

.middle {
  flex: 1;
  min-width: 0;
  display: flex;
  justify-content: center;
}

.me {
  display: flex;
  align-items: center;
  gap: 10px;
}

.me-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 44px;
  padding: 0 14px 0 5px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  font: inherit;
  color: inherit;
  text-decoration: none;
  cursor: pointer;
  transition: background-color 0.2s;
}

.me-btn.router-link-active {
  background: var(--color-paper-hover);
}

.elo {
  padding: 3px 9px;
  border-radius: 999px;
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.me-btn:hover {
  background: var(--color-paper-hover);
}

.avatar {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font-size: 14px;
}

.name {
  font-size: 15px;
  color: var(--color-ink);
  max-width: 160px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.icon-btn {
  display: grid;
  place-items: center;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-hover);
  color: var(--color-ink);
  cursor: pointer;
  transition: background-color 0.2s;
}

.icon-btn:hover {
  background: var(--color-line-strong);
}

@media (max-width: 640px) {
  .name {
    display: none;
  }
}
</style>
