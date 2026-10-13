<script setup lang="ts">
import { computed } from 'vue'
import { motion } from 'motion-v'
import { PhPaintBrush, PhGameController, PhUserCircle, PhLockKey, PhCheck, PhLockSimple } from '@phosphor-icons/vue'
import CardBack from '../client/CardBack.vue'
import ChipIcon from '../client/ChipIcon.vue'
import AccountForm from '../profile/AccountForm.vue'
import SecurityForm from '../profile/SecurityForm.vue'
import { useSettingsStore, type InvitePolicy } from '../../stores/settings'
import { CARD_BACKS as BACKS, FELTS, THEMES, type CardBackId, type CosmeticKind, type FeltId, type ThemeId } from '../../constants/cosmetics'
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

function unlocked(kind: CosmeticKind, id: string) {
  if (settings.isOwned(kind, id)) return true
  sound.playClick()
  emit('shop')
  return false
}

function setTheme(id: ThemeId) {
  if (settings.theme === id || !unlocked('theme', id)) return
  sound.playClick()
  settings.theme = id
}

function setFelt(id: FeltId) {
  if (settings.felt === id || !unlocked('felt', id)) return
  sound.playClick()
  settings.felt = id
}

function setCardBack(id: CardBackId) {
  if (settings.cardBack === id || !unlocked('back', id)) return
  sound.playClick()
  settings.cardBack = id
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
        <section class="block">
          <h3>Тема</h3>
          <div class="options themes">
            <button
              v-for="t in THEMES"
              :key="t.id"
              type="button"
              class="option theme"
              :class="{ on: settings.theme === t.id, locked: !settings.isOwned('theme', t.id) }"
              :aria-pressed="settings.theme === t.id"
              @click="setTheme(t.id)"
            >
              <span class="theme-preview" :data-theme="t.id" aria-hidden="true">
                <span class="tp-rail"></span>
                <span class="tp-main">
                  <span class="tp-felt"></span>
                  <span class="tp-row"><i></i><i></i><i></i></span>
                </span>
                <span class="tp-side"><i></i><i></i></span>
              </span>
              <span class="option-text">
                <b>{{ t.name }}</b>
                <small>{{ t.note }}</small>
              </span>
              <PhCheck v-if="settings.theme === t.id" class="check" :size="16" weight="bold" />
              <span v-if="!settings.isOwned('theme', t.id)" class="price"><PhLockSimple :size="12" weight="bold" /><ChipIcon :size="14" />{{ t.price.toLocaleString('ru-RU') }}</span>
            </button>
          </div>
        </section>

        <section class="block">
          <h3>Сукно стола</h3>
          <div class="options felts">
            <button
              v-for="f in FELTS"
              :key="f.id"
              type="button"
              class="option felt"
              :class="{ on: settings.felt === f.id, locked: !settings.isOwned('felt', f.id) }"
              :aria-pressed="settings.felt === f.id"
              @click="setFelt(f.id)"
            >
              <span class="felt-preview" :data-felt="f.id" aria-hidden="true"></span>
              <span class="option-text"><b>{{ f.name }}</b></span>
              <PhCheck v-if="settings.felt === f.id" class="check" :size="16" weight="bold" />
              <span v-if="!settings.isOwned('felt', f.id)" class="price"><PhLockSimple :size="12" weight="bold" /><ChipIcon :size="14" />{{ f.price.toLocaleString('ru-RU') }}</span>
            </button>
          </div>
        </section>

        <section class="block">
          <h3>Рубашка карт</h3>
          <div class="options backs">
            <button
              v-for="b in BACKS"
              :key="b.id"
              type="button"
              class="option back"
              :class="{ on: settings.cardBack === b.id, locked: !settings.isOwned('back', b.id) }"
              :aria-pressed="settings.cardBack === b.id"
              @click="setCardBack(b.id)"
            >
              <span class="back-preview"><CardBack :variant="b.id" /></span>
              <span class="option-text"><b>{{ b.name }}</b></span>
              <PhCheck v-if="settings.cardBack === b.id" class="check" :size="16" weight="bold" />
              <span v-if="!settings.isOwned('back', b.id)" class="price"><PhLockSimple :size="12" weight="bold" /><ChipIcon :size="14" />{{ b.price.toLocaleString('ru-RU') }}</span>
            </button>
          </div>
          <p class="hint">Закрытые предметы продаются в магазине за фишки — нажмите, чтобы перейти.</p>
        </section>
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
              <b>Общий чат в лобби</b>
              <small>Скрыть, если отвлекает</small>
            </div>
            <button
              type="button"
              class="switch"
              role="switch"
              :aria-checked="settings.showChat"
              aria-label="Общий чат в лобби"
              @click="settings.showChat = !settings.showChat"
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

.block h3 {
  margin: 0 0 12px;
  font-size: 16px;
  font-weight: 500;
}

.options {
  display: grid;
  gap: 12px;
}

.themes {
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
}

.felts {
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
}

.backs {
  grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
}

.option {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px;
  border: 0;
  border-radius: 18px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line);
  color: var(--color-ink);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: box-shadow 0.2s;
}

.option:hover {
  box-shadow: inset 0 0 0 1px var(--color-line-heavy);
}

.option.on {
  box-shadow: inset 0 0 0 2px var(--color-gold);
}

.option.locked > :not(.price) {
  opacity: 0.78;
  filter: saturate(0.85);
}

.option.locked:hover > :not(.price) {
  opacity: 0.85;
  filter: none;
}

.price {
  position: absolute;
  top: 14px;
  right: 14px;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 8px 3px 6px;
  border-radius: 999px;
  background: var(--color-paper-raised);
  box-shadow: 0 2px 8px -2px rgba(0, 0, 0, 0.25);
  color: var(--color-gold-ink);
  font-size: 12px;
  font-weight: 500;
  font-variant-numeric: tabular-nums;
}

.check {
  position: absolute;
  top: 14px;
  right: 14px;
  padding: 4px;
  border-radius: 50%;
  background: var(--color-gold);
  color: var(--color-paper);
}

.option-text {
  display: flex;
  flex-direction: column;
  padding: 0 4px 2px;
}

.option-text b {
  font-size: 14px;
  font-weight: 500;
}

.option-text small {
  font-size: 12px;
  color: var(--color-ink-faint);
}

.theme-preview {
  display: flex;
  gap: 4px;
  height: 84px;
  padding: 6px;
  box-sizing: border-box;
  border-radius: 12px;
  background: var(--color-paper);
  box-shadow: inset 0 0 0 1px var(--color-line);
}

.tp-rail {
  width: 12px;
  border-radius: 4px;
  background: var(--color-paper-sunken);
}

.tp-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1;
}

.tp-felt {
  flex: 1;
  border-radius: 6px;
  background: var(--color-gold-tint);
  box-shadow: inset 0 0 0 1px var(--color-gold);
}

.tp-row {
  display: flex;
  gap: 4px;
  height: 18px;
}

.tp-row i,
.tp-side i {
  flex: 1;
  border-radius: 4px;
  background: var(--color-paper-raised);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
}

.tp-side {
  display: flex;
  flex-direction: column;
  gap: 4px;
  width: 26%;
}

.felt-preview {
  aspect-ratio: 2 / 1;
  border-radius: 999px;
  background: radial-gradient(ellipse at 50% 30%, var(--color-felt), var(--color-felt-deep));
  box-shadow:
    inset 0 0 0 4px var(--color-felt-well),
    inset 0 0 0 5px var(--color-felt-line);
}

.back-preview {
  display: block;
  width: 62%;
  aspect-ratio: 5 / 7;
  margin: 4px auto 0;
}

.hint {
  margin: 12px 0 0;
  font-size: 13px;
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

  .themes {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  }

  .felts,
  .backs {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
