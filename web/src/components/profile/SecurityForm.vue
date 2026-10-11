<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { motion, AnimatePresence } from 'motion-v'
import { PhKey, PhEye, PhEyeSlash, PhShieldWarning, PhTrash } from '@phosphor-icons/vue'
import { changePassword, deleteAccount } from '../../api/users'
import { ApiError } from '../../api/http'
import { useAuthStore } from '../../stores/auth'
import { useProfileStore } from '../../stores/profile'
import { useToast } from '../../composables/useToast'
import { useGameSound } from '../../composables/useGameSound'
import { useShake } from '../../composables/useShake'

const MIN = 8
const MAX_BYTES = 72

const router = useRouter()
const auth = useAuthStore()
const profileStore = useProfileStore()
const toast = useToast()
const sound = useGameSound()
const shake = useShake()

const passwordEl = ref<HTMLElement | null>(null)
const dangerEl = ref<HTMLElement | null>(null)

const oldPassword = ref('')
const newPassword = ref('')
const repeatPassword = ref('')
const visible = ref(false)
const error = ref('')
const busy = ref(false)

const dangerOpen = ref(false)
const confirmName = ref('')
const deleting = ref(false)

const username = computed(() => profileStore.profile?.username ?? auth.user?.username ?? '')
const filled = computed(() => !!oldPassword.value && !!newPassword.value && !!repeatPassword.value)
const mismatch = computed(() => !!repeatPassword.value && repeatPassword.value !== newPassword.value)

const strength = computed(() => {
  const p = newPassword.value
  if (!p) return 0
  let s = 0
  if (p.length >= MIN) s++
  if (p.length >= 12) s++
  if (/[A-ZА-Я]/.test(p) && /[a-zа-я]/.test(p)) s++
  if (/\d/.test(p) && /[^A-Za-zА-Яа-я0-9]/.test(p)) s++
  return Math.max(s, 1)
})
const strengthLabel = computed(() => ['', 'Слабый', 'Средний', 'Хороший', 'Надёжный'][strength.value])

async function save() {
  error.value = ''
  if (newPassword.value.length < MIN) error.value = `Новый пароль — минимум ${MIN} символов.`
  else if (new TextEncoder().encode(newPassword.value).length > MAX_BYTES) error.value = 'Новый пароль слишком длинный.'
  else if (newPassword.value === oldPassword.value) error.value = 'Новый пароль совпадает с текущим.'
  else if (mismatch.value) error.value = 'Пароли не совпадают.'
  if (error.value) return shake(passwordEl.value)

  busy.value = true
  try {
    await changePassword({ oldPassword: oldPassword.value, newPassword: newPassword.value })
    oldPassword.value = ''
    newPassword.value = ''
    repeatPassword.value = ''
    sound.playSuccess()
    toast.show('Пароль изменён', 'success')
  } catch (err) {
    error.value =
      err instanceof ApiError && err.status === 403
        ? 'Текущий пароль указан неверно.'
        : err instanceof ApiError && err.status === 400
          ? 'Проверьте новый пароль: от 8 символов и не такой же, как текущий.'
          : 'Не удалось сменить пароль. Попробуйте ещё раз.'
    shake(passwordEl.value)
  } finally {
    busy.value = false
  }
}

async function remove() {
  if (confirmName.value.trim() !== username.value) return shake(dangerEl.value)
  deleting.value = true
  try {
    await deleteAccount()
    auth.logout()
    profileStore.reset()
    toast.show('Аккаунт удалён')
    router.push('/')
  } catch {
    toast.show('Не удалось удалить аккаунт', 'error')
    shake(dangerEl.value)
  } finally {
    deleting.value = false
  }
}
</script>

<template>
  <div class="security">
    <form ref="passwordEl" class="card" novalidate @submit.prevent="save">
      <div class="card-head">
        <PhKey :size="20" weight="duotone" aria-hidden="true" />
        <div>
          <h3>Пароль</h3>
          <p>После смены войдите новым паролем на других устройствах.</p>
        </div>
        <button type="button" class="eye" :aria-label="visible ? 'Скрыть пароли' : 'Показать пароли'" @click="visible = !visible">
          <PhEyeSlash v-if="visible" :size="18" />
          <PhEye v-else :size="18" />
        </button>
      </div>

      <label class="field">
        <span class="label">Текущий пароль</span>
        <input v-model="oldPassword" :type="visible ? 'text' : 'password'" autocomplete="current-password" @input="error = ''" />
      </label>

      <label class="field">
        <span class="label">Новый пароль</span>
        <input v-model="newPassword" :type="visible ? 'text' : 'password'" autocomplete="new-password" @input="error = ''" />
      </label>

      <div class="strength" :data-score="strength" aria-live="polite">
        <span v-for="i in 4" :key="i">
          <motion.i :initial="false" :animate="{ scaleX: i <= strength ? 1 : 0 }" :transition="{ type: 'spring', stiffness: 300, damping: 30 }" />
        </span>
        <em>{{ strengthLabel }}</em>
      </div>

      <label class="field">
        <span class="label">Повторите новый пароль</span>
        <input
          v-model="repeatPassword"
          :type="visible ? 'text' : 'password'"
          autocomplete="new-password"
          :aria-invalid="mismatch"
          @input="error = ''"
        />
      </label>

      <AnimatePresence>
        <motion.p
          v-if="mismatch && !error"
          key="warn"
          class="hint-warn"
          :initial="{ opacity: 0, y: -4 }"
          :animate="{ opacity: 1, y: 0 }"
          :exit="{ opacity: 0 }"
        >
          Пароли пока не совпадают.
        </motion.p>
        <motion.p
          v-if="error"
          key="err"
          class="error"
          role="alert"
          :initial="{ opacity: 0, y: -6 }"
          :animate="{ opacity: 1, y: 0 }"
          :exit="{ opacity: 0, y: -6 }"
        >
          {{ error }}
        </motion.p>
      </AnimatePresence>

      <div class="actions">
        <motion.button type="submit" class="primary" :disabled="busy || !filled" :while-press="{ scale: 0.97 }">
          <span v-if="busy" class="spinner" aria-hidden="true"></span>
          {{ busy ? 'Меняем…' : 'Сменить пароль' }}
        </motion.button>
      </div>
    </form>

    <section ref="dangerEl" class="card danger">
      <div class="card-head">
        <PhShieldWarning :size="20" weight="duotone" aria-hidden="true" />
        <div>
          <h3>Удаление аккаунта</h3>
          <p>Рейтинг, статистика и фишки пропадут. Восстановить аккаунт будет нельзя.</p>
        </div>
      </div>

      <AnimatePresence mode="wait">
        <motion.div
          v-if="dangerOpen"
          key="confirm"
          :initial="{ opacity: 0, height: 0 }"
          :animate="{ opacity: 1, height: 'auto' }"
          :exit="{ opacity: 0, height: 0 }"
          style="overflow: hidden"
        >
          <label class="field">
            <span class="label">Введите «{{ username }}», чтобы подтвердить</span>
            <input v-model="confirmName" type="text" autocomplete="off" />
          </label>
          <div class="actions">
            <button type="button" class="ghost" @click="dangerOpen = false">Отмена</button>
            <button type="button" class="primary danger-btn" :disabled="deleting || confirmName.trim() !== username" @click="remove">
              <span v-if="deleting" class="spinner" aria-hidden="true"></span>
              <PhTrash v-else :size="17" weight="bold" aria-hidden="true" />
              Удалить навсегда
            </button>
          </div>
        </motion.div>
        <div v-else key="open" class="actions">
          <button type="button" class="ghost danger-ghost" @click="dangerOpen = true">Удалить аккаунт</button>
        </div>
      </AnimatePresence>
    </section>
  </div>
</template>

<style scoped src="./form.css"></style>
