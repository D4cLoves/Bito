<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { animate } from 'animejs'
import { Eye, EyeOff, KeyRound, Mail, PencilLine, ShieldAlert, X, CalendarDays } from 'lucide-vue-next'
import { changePassword, deleteAccount, updateUsername } from '../../api/users'
import { ApiError } from '../../api/http'
import { useAuthStore } from '../../stores/auth'
import { useToast } from '../../composables/useToast'
import { useGameSound } from '../../composables/useGameSound'
import type { Profile } from '../../types/user'

const MIN_NAME = 3
const MAX_NAME = 32
const MIN_PASSWORD = 8
const MAX_PASSWORD_BYTES = 72

const props = defineProps<{ open: boolean; profile: Profile | null }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'renamed', username: string): void }>()

const router = useRouter()
const auth = useAuthStore()
const toast = useToast()
const sound = useGameSound()

const panelEl = ref<HTMLElement | null>(null)
const nameSection = ref<HTMLElement | null>(null)
const passwordSection = ref<HTMLElement | null>(null)
const dangerSection = ref<HTMLElement | null>(null)

const username = ref('')
const nameError = ref('')
const nameBusy = ref(false)

const oldPassword = ref('')
const newPassword = ref('')
const repeatPassword = ref('')
const showPasswords = ref(false)
const passwordError = ref('')
const passwordBusy = ref(false)

const dangerOpen = ref(false)
const confirmName = ref('')
const deleteBusy = ref(false)

const current = computed(() => props.profile?.username ?? auth.user?.username ?? '')
const initial = computed(() => (current.value[0] ?? '?').toUpperCase())
const nameLength = computed(() => [...username.value.trim()].length)
const nameChanged = computed(() => username.value.trim() !== current.value)
const nameValid = computed(() => nameLength.value >= MIN_NAME && nameLength.value <= MAX_NAME)

const since = computed(() =>
  props.profile?.createdAt
    ? new Date(props.profile.createdAt).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long', year: 'numeric' })
    : '',
)

const strength = computed(() => {
  const p = newPassword.value
  if (!p) return 0
  let s = 0
  if (p.length >= MIN_PASSWORD) s++
  if (p.length >= 12) s++
  if (/[A-ZА-Я]/.test(p) && /[a-zа-я]/.test(p)) s++
  if (/\d/.test(p) && /[^A-Za-zА-Яа-я0-9]/.test(p)) s++
  return Math.max(s, 1)
})
const strengthLabel = computed(() => ['', 'Слабый', 'Средний', 'Хороший', 'Надёжный'][strength.value])
const passwordFilled = computed(() => !!oldPassword.value && !!newPassword.value && !!repeatPassword.value)
const repeatMismatch = computed(() => !!repeatPassword.value && repeatPassword.value !== newPassword.value)

watch(
  () => props.open,
  async (open) => {
    if (open) {
      reset()
      document.addEventListener('keydown', onKey)
      document.documentElement.style.overflow = 'hidden'
      await nextTick()
      panelEl.value?.querySelector<HTMLElement>('.close')?.focus()
    } else {
      document.removeEventListener('keydown', onKey)
      document.documentElement.style.overflow = ''
    }
  },
)

onUnmounted(() => {
  document.removeEventListener('keydown', onKey)
  document.documentElement.style.overflow = ''
})

function reset() {
  username.value = current.value
  nameError.value = ''
  oldPassword.value = ''
  newPassword.value = ''
  repeatPassword.value = ''
  passwordError.value = ''
  showPasswords.value = false
  dangerOpen.value = false
  confirmName.value = ''
}

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
}

function shake(el: HTMLElement | null) {
  sound.playError()
  if (!el || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
  animate(el, {
    x: [{ to: -8 }, { to: 7 }, { to: -5 }, { to: 3 }, { to: 0 }],
    duration: 420,
    ease: 'inOut(2)',
  })
}

async function saveName() {
  nameError.value = ''
  if (!nameValid.value) {
    nameError.value = `Имя должно быть от ${MIN_NAME} до ${MAX_NAME} символов.`
    return shake(nameSection.value)
  }
  if (!nameChanged.value) return

  nameBusy.value = true
  try {
    const user = await updateUsername(username.value.trim())
    auth.setUser(user)
    emit('renamed', user.username)
    username.value = user.username
    sound.playSuccess()
    toast.show('Имя обновлено', 'success')
  } catch (err) {
    nameError.value =
      err instanceof ApiError && err.status === 409
        ? 'Это имя уже занято. Попробуйте другое.'
        : err instanceof ApiError && err.status === 400
          ? `Имя должно быть от ${MIN_NAME} до ${MAX_NAME} символов.`
          : 'Не удалось сохранить имя. Попробуйте ещё раз.'
    shake(nameSection.value)
  } finally {
    nameBusy.value = false
  }
}

async function savePassword() {
  passwordError.value = ''
  if (newPassword.value.length < MIN_PASSWORD) passwordError.value = `Новый пароль — минимум ${MIN_PASSWORD} символов.`
  else if (new TextEncoder().encode(newPassword.value).length > MAX_PASSWORD_BYTES) passwordError.value = 'Новый пароль слишком длинный.'
  else if (newPassword.value === oldPassword.value) passwordError.value = 'Новый пароль совпадает со старым.'
  else if (repeatMismatch.value) passwordError.value = 'Пароли не совпадают.'
  if (passwordError.value) return shake(passwordSection.value)

  passwordBusy.value = true
  try {
    await changePassword({ oldPassword: oldPassword.value, newPassword: newPassword.value })
    oldPassword.value = ''
    newPassword.value = ''
    repeatPassword.value = ''
    sound.playSuccess()
    toast.show('Пароль изменён', 'success')
  } catch (err) {
    passwordError.value =
      err instanceof ApiError && err.status === 403
        ? 'Текущий пароль указан неверно.'
        : err instanceof ApiError && err.status === 400
          ? 'Проверьте новый пароль: от 8 символов и не такой же, как старый.'
          : 'Не удалось сменить пароль. Попробуйте ещё раз.'
    shake(passwordSection.value)
  } finally {
    passwordBusy.value = false
  }
}

async function removeAccount() {
  if (confirmName.value.trim() !== current.value) return shake(dangerSection.value)
  deleteBusy.value = true
  try {
    await deleteAccount()
    auth.logout()
    toast.show('Аккаунт удалён')
    router.push('/')
  } catch {
    toast.show('Не удалось удалить аккаунт', 'error')
    shake(dangerSection.value)
  } finally {
    deleteBusy.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div v-if="open" class="overlay" @click.self="emit('close')">
        <aside ref="panelEl" class="panel" role="dialog" aria-modal="true" aria-labelledby="drawer-title">
          <header class="top">
            <span class="avatar" aria-hidden="true">{{ initial }}</span>
            <div class="who">
              <h2 id="drawer-title">{{ current }}</h2>
              <p>Настройки профиля</p>
            </div>
            <button type="button" class="close" aria-label="Закрыть" @click="emit('close')">
              <X :size="20" />
            </button>
          </header>

          <div class="scroll">
            <dl class="facts" data-stagger>
              <div>
                <dt><Mail :size="16" aria-hidden="true" />Почта</dt>
                <dd>{{ profile?.email ?? auth.user?.email }}</dd>
              </div>
              <div v-if="since">
                <dt><CalendarDays :size="16" aria-hidden="true" />С нами</dt>
                <dd>с {{ since }}</dd>
              </div>
            </dl>

            <form ref="nameSection" class="card" data-stagger novalidate @submit.prevent="saveName">
              <div class="card-head">
                <PencilLine :size="18" aria-hidden="true" />
                <h3>Имя в игре</h3>
              </div>
              <label class="field">
                <span class="sr-only">Имя в игре</span>
                <input
                  v-model="username"
                  type="text"
                  autocomplete="nickname"
                  maxlength="64"
                  :aria-invalid="!!nameError"
                  @input="nameError = ''"
                />
                <span class="counter" :class="{ over: nameLength > MAX_NAME || (nameLength > 0 && nameLength < MIN_NAME) }">
                  {{ nameLength }}/{{ MAX_NAME }}
                </span>
              </label>
              <p class="help">Так вас видят соперники за столом.</p>
              <Transition name="msg">
                <p v-if="nameError" class="error" role="alert">{{ nameError }}</p>
              </Transition>
              <button type="submit" class="primary" :disabled="nameBusy || !nameChanged">
                <span v-if="nameBusy" class="spinner" aria-hidden="true"></span>
                {{ nameBusy ? 'Сохраняем…' : 'Сохранить имя' }}
              </button>
            </form>

            <form ref="passwordSection" class="card" data-stagger novalidate @submit.prevent="savePassword">
              <div class="card-head">
                <KeyRound :size="18" aria-hidden="true" />
                <h3>Пароль</h3>
                <button
                  type="button"
                  class="eye"
                  :aria-label="showPasswords ? 'Скрыть пароли' : 'Показать пароли'"
                  @click="showPasswords = !showPasswords"
                >
                  <EyeOff v-if="showPasswords" :size="17" />
                  <Eye v-else :size="17" />
                </button>
              </div>

              <label class="field stacked">
                <span class="label">Текущий пароль</span>
                <input
                  v-model="oldPassword"
                  :type="showPasswords ? 'text' : 'password'"
                  autocomplete="current-password"
                  @input="passwordError = ''"
                />
              </label>

              <label class="field stacked">
                <span class="label">Новый пароль</span>
                <input
                  v-model="newPassword"
                  :type="showPasswords ? 'text' : 'password'"
                  autocomplete="new-password"
                  @input="passwordError = ''"
                />
              </label>

              <div class="strength" :data-score="strength" aria-live="polite">
                <span v-for="i in 4" :key="i" :class="{ on: i <= strength }"></span>
                <em>{{ strengthLabel }}</em>
              </div>

              <label class="field stacked">
                <span class="label">Повторите новый пароль</span>
                <input
                  v-model="repeatPassword"
                  :type="showPasswords ? 'text' : 'password'"
                  autocomplete="new-password"
                  :aria-invalid="repeatMismatch"
                  @input="passwordError = ''"
                />
              </label>
              <Transition name="msg">
                <p v-if="repeatMismatch && !passwordError" class="hint-warn">Пароли пока не совпадают.</p>
              </Transition>

              <Transition name="msg">
                <p v-if="passwordError" class="error" role="alert">{{ passwordError }}</p>
              </Transition>
              <button type="submit" class="primary" :disabled="passwordBusy || !passwordFilled">
                <span v-if="passwordBusy" class="spinner" aria-hidden="true"></span>
                {{ passwordBusy ? 'Меняем…' : 'Сменить пароль' }}
              </button>
            </form>

            <section ref="dangerSection" class="card danger" data-stagger>
              <div class="card-head">
                <ShieldAlert :size="18" aria-hidden="true" />
                <h3>Удаление аккаунта</h3>
              </div>
              <p class="help">Статистика и фишки пропадут, восстановить аккаунт будет нельзя.</p>

              <Transition name="expand">
                <div v-if="dangerOpen" class="confirm">
                  <label class="field stacked">
                    <span class="label">Введите «{{ current }}», чтобы подтвердить</span>
                    <input v-model="confirmName" type="text" autocomplete="off" />
                  </label>
                  <div class="row">
                    <button type="button" class="ghost" @click="dangerOpen = false">Отмена</button>
                    <button
                      type="button"
                      class="primary danger-btn"
                      :disabled="deleteBusy || confirmName.trim() !== current"
                      @click="removeAccount"
                    >
                      <span v-if="deleteBusy" class="spinner" aria-hidden="true"></span>
                      Удалить навсегда
                    </button>
                  </div>
                </div>
              </Transition>
              <button v-if="!dangerOpen" type="button" class="ghost danger-ghost" @click="dangerOpen = true">
                Удалить аккаунт
              </button>
            </section>
          </div>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.overlay {
  position: fixed;
  z-index: 300;
  inset: 0;
  display: flex;
  justify-content: flex-end;
  background: rgba(40, 36, 30, 0.32);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
  font-family: var(--font-sans);
  color: var(--color-ink);
}

.panel {
  display: flex;
  flex-direction: column;
  width: min(460px, 100%);
  height: 100%;
  box-sizing: border-box;
  background: var(--color-paper);
  box-shadow: -30px 0 60px -30px rgba(40, 36, 30, 0.5);
}

.top {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 22px 22px 18px;
  border-bottom: 1px solid var(--color-line);
}

.avatar {
  display: grid;
  place-items: center;
  flex: none;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font-family: var(--font-card);
  font-size: 22px;
}

.who {
  flex: 1;
  min-width: 0;
}

.who h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 500;
  letter-spacing: -0.02em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.who p {
  margin: 3px 0 0;
  font-size: 13px;
  color: var(--color-ink-faint);
}

.close {
  display: grid;
  place-items: center;
  flex: none;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  color: var(--color-ink);
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.close:hover {
  background: var(--color-paper-hover);
  transform: rotate(90deg);
}

.scroll {
  flex: 1;
  overflow-y: auto;
  padding: 18px 22px 32px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.facts {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 14px 16px;
  border-radius: 18px;
  background: var(--color-paper-sunken);
}

.facts div {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

dt {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--color-ink-muted);
}

dd {
  margin: 0;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.card {
  padding: 18px;
  border-radius: 22px;
  background: var(--color-paper-raised);
  box-shadow: 0 0 0 1px rgba(214, 208, 196, 0.9);
}

.card-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  color: var(--color-ink-muted);
}

h3 {
  flex: 1;
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: var(--color-ink);
}

.eye {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--color-ink-muted);
  cursor: pointer;
}

.eye:hover {
  background: var(--color-paper-sunken);
  color: var(--color-ink);
}

.field {
  position: relative;
  display: block;
}

.field.stacked + .field.stacked,
.strength + .field.stacked {
  margin-top: 12px;
}

.label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  color: var(--color-ink-faint);
}

input {
  width: 100%;
  height: 48px;
  padding: 0 16px;
  box-sizing: border-box;
  border: 0;
  border-radius: 14px;
  background: var(--color-paper);
  box-shadow: inset 0 0 0 1px var(--color-line-strong);
  color: var(--color-ink);
  font: inherit;
  font-size: 16px;
  outline: none;
  transition: box-shadow 0.2s;
}

.field:not(.stacked) input {
  padding-right: 68px;
}

input:hover {
  box-shadow: inset 0 0 0 1px var(--color-line-heavy);
}

input:focus {
  box-shadow:
    inset 0 0 0 1.5px var(--color-ink-soft),
    0 0 0 4px rgba(54, 50, 44, 0.08);
}

input[aria-invalid='true'] {
  box-shadow: inset 0 0 0 1.5px var(--color-red);
}

.counter {
  position: absolute;
  right: 14px;
  top: 50%;
  translate: 0 -50%;
  font-size: 12px;
  color: var(--color-ink-faint);
  font-variant-numeric: tabular-nums;
}

.counter.over {
  color: var(--color-red);
}

.help {
  margin: 8px 2px 0;
  font-size: 13px;
  line-height: 1.45;
  color: var(--color-ink-subtle);
}

.strength {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
}

.strength span {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: var(--color-paper-sunken);
  transition: background-color 0.3s;
}

.strength[data-score='1'] span.on {
  background: var(--color-red);
}

.strength[data-score='2'] span.on {
  background: var(--color-gold);
}

.strength[data-score='3'] span.on,
.strength[data-score='4'] span.on {
  background: var(--color-green);
}

.strength em {
  min-width: 72px;
  margin-left: 6px;
  font-size: 12px;
  font-style: normal;
  color: var(--color-ink-faint);
  text-align: right;
}

.hint-warn {
  margin: 8px 2px 0;
  font-size: 13px;
  color: var(--color-gold-ink);
}

.error {
  margin: 12px 0 0;
  padding: 10px 14px;
  border-radius: 14px;
  background: var(--color-red-tint);
  color: var(--color-red-strong);
  font-size: 14px;
  line-height: 1.4;
}

.primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  height: 48px;
  margin-top: 14px;
  border: 0;
  border-radius: 999px;
  background: var(--color-ink-soft);
  color: var(--color-paper);
  font: inherit;
  font-size: 15px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    opacity 0.2s,
    transform 0.2s;
}

.primary:hover:not(:disabled) {
  background: var(--color-ink);
}

.primary:active:not(:disabled) {
  transform: scale(0.98);
}

.primary:disabled {
  opacity: 0.4;
  cursor: default;
}

.ghost {
  height: 44px;
  padding: 0 18px;
  border: 0;
  border-radius: 999px;
  background: var(--color-paper-sunken);
  color: var(--color-ink);
  font: inherit;
  font-size: 15px;
  cursor: pointer;
}

.ghost:hover {
  background: var(--color-paper-hover);
}

.danger {
  box-shadow: 0 0 0 1px rgba(161, 71, 63, 0.25);
}

.danger .card-head {
  color: var(--color-red);
}

.danger-ghost {
  margin-top: 14px;
  background: var(--color-red-tint);
  color: var(--color-red-strong);
}

.danger-ghost:hover {
  background: var(--color-red-tint-strong);
}

.confirm {
  margin-top: 14px;
  overflow: hidden;
}

.confirm .row {
  display: flex;
  gap: 8px;
  margin-top: 12px;
}

.confirm .primary {
  margin-top: 0;
  flex: 1;
}

.danger-btn {
  background: var(--color-red);
}

.danger-btn:hover:not(:disabled) {
  background: var(--color-red-strong);
}

.spinner {
  width: 15px;
  height: 15px;
  border-radius: 50%;
  border: 2px solid rgba(245, 244, 239, 0.35);
  border-top-color: var(--color-paper);
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  white-space: nowrap;
}

.drawer-enter-active,
.drawer-leave-active {
  transition: opacity 0.3s;
}

.drawer-enter-active .panel,
.drawer-leave-active .panel {
  transition: transform 0.45s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.drawer-enter-from,
.drawer-leave-to {
  opacity: 0;
}

.drawer-enter-from .panel,
.drawer-leave-to .panel {
  transform: translateX(40px);
}

.drawer-enter-active [data-stagger] {
  animation: rise 0.5s cubic-bezier(0.2, 0.8, 0.2, 1) both;
}

.drawer-enter-active [data-stagger]:nth-child(2) {
  animation-delay: 60ms;
}

.drawer-enter-active [data-stagger]:nth-child(3) {
  animation-delay: 120ms;
}

.drawer-enter-active [data-stagger]:nth-child(4) {
  animation-delay: 180ms;
}

@keyframes rise {
  from {
    opacity: 0;
    transform: translateY(14px);
  }
}

.msg-enter-active,
.msg-leave-active {
  transition:
    opacity 0.2s,
    transform 0.2s;
}

.msg-enter-from,
.msg-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.expand-enter-active,
.expand-leave-active {
  transition:
    opacity 0.25s,
    max-height 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
  max-height: 220px;
}

.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  max-height: 0;
}

@media (max-width: 520px) {
  .panel {
    width: 100%;
  }

  .top,
  .scroll {
    padding-left: 16px;
    padding-right: 16px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .drawer-enter-active .panel,
  .drawer-leave-active .panel,
  .drawer-enter-active [data-stagger] {
    transition: none;
    animation: none;
  }
}
</style>
