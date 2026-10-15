<script setup lang="ts">
import { computed, ref } from 'vue'
import { motion } from 'motion-v'
import { PhPaintBrush, PhGameController, PhUserCircle, PhLockKey, PhStorefront } from '@phosphor-icons/vue'
import CosmeticTile from '../cosmetics/CosmeticTile.vue'
import CosmeticFilter from '../cosmetics/CosmeticFilter.vue'
import AccountForm from '../profile/AccountForm.vue'
import SecurityForm from '../profile/SecurityForm.vue'
import { useSettingsStore, type InvitePolicy } from '../../stores/settings'
import { ALL_ITEMS, type CatalogEntry, type CosmeticFilterValue } from '../../constants/cosmetics'
import { useGameSound } from '../../composables/useGameSound'
import type { Profile } from '../../types/user'

type Section = 'appearance' | 'game' | 'account' | 'security'

const props = defineProps<{ section: string | null; profile: Profile | null }>()
const emit = defineEmits<{ (e: 'section', id: Section): void; (e: 'shop'): void }>()

const settings = useSettingsStore()
const sound = useGameSound()

const SECTIONS: { id: Section; label: string; icon: typeof PhPaintBrush }[] = [
  { id: 'appearance', label: 'Внешний вид', icon: PhPaintBrush },
  { id: 'game', label: 'Игра', icon: PhGameController },
  { id: 'account', label: 'Аккаунт', icon: PhUserCircle },
  { id: 'security', label: 'Безопасность', icon: PhLockKey },
]

const INVITES: { id: InvitePolicy; label: string }[] = [
  { id: 'all', label: 'От всех' },
  { id: 'friends', label: 'От друзей' },
  { id: 'nobody', label: 'Ни от кого' },
]

const current = computed<Section>(() => SECTIONS.find((s) => s.id === props.section)?.id ?? 'appearance')

const filter = ref<CosmeticFilterValue>('all')
const owned = computed(() => ALL_ITEMS.filter((i) => settings.isOwned(i.kind, i.id)))
const ownedVisible = computed(() => (filter.value === 'all' ? owned.value : owned.value.filter((i) => i.kind === filter.value)))
const ownedCounts = computed(() => ({
  all: owned.value.length,
  back: owned.value.filter((i) => i.kind === 'back').length,
  felt: owned.value.filter((i) => i.kind === 'felt').length,
  theme: owned.value.filter((i) => i.kind === 'theme').length,
}))
const lockedLeft = computed(
  () => ALL_ITEMS.filter((i) => (filter.value === 'all' || i.kind === filter.value) && !settings.isOwned(i.kind, i.id)).length,
)

function pick(item: CatalogEntry) {
  if (settings.isActive(item.kind, item.id)) return
  if (settings.select(item.kind, item.id)) sound.playClick()
}

function toggleSound() {
  settings.soundOn = !settings.soundOn
  if (settings.soundOn) sound.playClick()
}

const spring = { type: 'spring', stiffness: 380, damping: 32 } as const
</script>

<template>
  <div class="settings">
    <nav class="tabs" aria-label="Разделы настроек">
      <button
        v-for="s in SECTIONS"
        :key="s.id"
        type="button"
        :class="{ on: current === s.id }"
        :aria-current="current === s.id ? 'page' : undefined"
        @click="emit('section', s.id)"
      >
        <motion.span v-if="current === s.id" layout-id="settings-tab" class="pill" :transition="spring" />
        <component :is="s.icon" :size="18" :weight="current === s.id ? 'fill' : 'regular'" class="tab-icon" aria-hidden="true" />
        <span class="tab-label">{{ s.label }}</span>
      </button>
    </nav>

    <div class="content">
      <template v-if="current === 'appearance'">
        <div class="wardrobe-head">
          <CosmeticFilter v-model="filter" :counts="ownedCounts" layout-id="settings-filter" />
        </div>
        <ul class="wardrobe">
          <li v-for="item in ownedVisible" :key="`${item.kind}:${item.id}`">
            <CosmeticTile
              :kind="item.kind"
              :id="item.id"
              :name="item.name"
              :active="settings.isActive(item.kind, item.id)"
              clickable
              @pick="pick(item)"
            />
          </li>
          <li v-if="lockedLeft">
            <button type="button" class="more" @click="emit('shop')">
              <PhStorefront :size="28" />
              <b>Ещё {{ lockedLeft }} в магазине</b>
              <small>рубашки, сукно и темы за фишки</small>
            </button>
          </li>
        </ul>
      </template>

      <template v-else-if="current === 'game'">
        <section class="block list">
          <div class="setting">
            <div class="setting-text">
              <b>Звуки</b>
              <small>Раздача, ходы, таймер и кнопки</small>
            </div>
            <button type="button" class="switch" role="switch" :aria-checked="settings.soundOn" aria-label="Звуки" @click="toggleSound">
              <span class="knob"></span>
            </button>
          </div>

          <div class="setting">
            <div class="setting-text">
              <b>Музыка за столом</b>
              <small>Лаунж-джаз Kevin MacLeod, incompetech.com, CC BY 4.0</small>
            </div>
            <button
              type="button"
              class="switch"
              role="switch"
              :aria-checked="settings.music"
              aria-label="Музыка за столом"
              @click="settings.music = !settings.music"
            >
              <span class="knob"></span>
            </button>
          </div>

          <div class="setting">
            <div class="setting-text">
              <b>Приглашения за стол</b>
              <small>Кто может позвать вас в партию</small>
            </div>
            <div class="seg" role="radiogroup" aria-label="Приглашения за стол">
              <button
                v-for="o in INVITES"
                :key="o.id"
                type="button"
                role="radio"
                :aria-checked="settings.invites === o.id"
                :class="{ on: settings.invites === o.id }"
                @click="settings.invites = o.id"
              >
                {{ o.label }}
              </button>
            </div>
          </div>
        </section>
      </template>

      <AccountForm v-else-if="current === 'account'" :profile="profile" />
      <SecurityForm v-else />
    </div>
  </div>
</template>

<style scoped>
.settings {
  display: grid;
  grid-template-columns: 200px minmax(0, 1fr);
  gap: 28px;
  align-items: start;
}

.tabs {
  position: sticky;
  top: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.tabs button {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  height: 46px;
  padding: 0 14px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 15px;
  text-align: left;
  cursor: pointer;
}

.tabs button:hover,
.tabs button.on {
  color: var(--color-ink);
}

.pill {
  position: absolute;
  inset: 0;
  border-radius: 14px;
  background: var(--color-paper-hover);
}

.tab-icon,
.tab-label {
  position: relative;
}

.tabs button.on .tab-icon {
  color: var(--color-gold);
}

.content {
  display: flex;
  flex-direction: column;
  gap: 26px;
  min-width: 0;
}

.wardrobe-head {
  margin-bottom: 16px;
}

.wardrobe {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 14px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.more {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 100%;
  height: 100%;
  min-height: 200px;
  padding: 16px;
  box-sizing: border-box;
  border: 1.5px dashed var(--color-line-heavy);
  border-radius: 22px;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  text-align: center;
  cursor: pointer;
  transition:
    border-color 0.2s,
    color 0.2s,
    background-color 0.2s;
}

.more:hover {
  border-color: var(--color-gold);
  background: var(--color-gold-tint);
  color: var(--color-gold-ink);
}

.more b {
  font-size: 15px;
  font-weight: 500;
}

.more small {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.list {
  display: flex;
  flex-direction: column;
  padding: 6px 18px;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.setting {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px 20px;
  padding: 16px 0;
}

.setting + .setting {
  border-top: 1px solid var(--color-line);
}

.setting-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.setting-text b {
  font-size: 15px;
  font-weight: 500;
}

.setting-text small {
  font-size: 13px;
  color: var(--color-ink-faint);
}

.switch {
  position: relative;
  flex: none;
  width: 50px;
  height: 30px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: var(--color-line-stronger);
  cursor: pointer;
  transition: background-color 0.2s;
}

.switch[aria-checked='true'] {
  background: var(--color-gold);
}

.knob {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
  transition: transform 0.25s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.switch[aria-checked='true'] .knob {
  transform: translateX(20px);
}

.seg {
  display: flex;
  padding: 4px;
  border-radius: 999px;
  background: var(--color-paper-sunken);
}

.seg button {
  height: 34px;
  padding: 0 12px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--color-ink-muted);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.seg button.on {
  background: var(--color-ink-soft);
  color: var(--color-paper);
}

@media (max-width: 860px) {
  .settings {
    grid-template-columns: 1fr;
    gap: 18px;
  }

  .tabs {
    position: static;
    flex-direction: row;
    overflow-x: auto;
  }

  .tabs button {
    flex: none;
  }

  .wardrobe {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }
}
</style>
