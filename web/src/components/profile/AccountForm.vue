<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { motion, AnimatePresence } from 'motion-v'
import { PhEnvelopeSimple, PhCalendarBlank, PhPencilSimple } from '@phosphor-icons/vue'
import { updateUsername } from '../../api/users'
import { ApiError } from '../../api/http'
import { useAuthStore } from '../../stores/auth'
import { useProfileStore } from '../../stores/profile'
import { useToast } from '../../composables/useToast'
import { useGameSound } from '../../composables/useGameSound'
import { useShake } from '../../composables/useShake'
import { memberSince } from '../../composables/usePlayerRank'
import type { Profile } from '../../types/user'

const MIN = 3
const MAX = 32

const props = defineProps<{ profile: Profile | null }>()

const auth = useAuthStore()
const profileStore = useProfileStore()
const toast = useToast()
const sound = useGameSound()
const shake = useShake()

const formEl = ref<HTMLElement | null>(null)
const username = ref('')
const error = ref('')
const busy = ref(false)

const current = computed(() => props.profile?.username ?? auth.user?.username ?? '')
const length = computed(() => [...username.value.trim()].length)
const changed = computed(() => username.value.trim() !== current.value)
const valid = computed(() => length.value >= MIN && length.value <= MAX)

watch(current, (v) => (username.value = v), { immediate: true })

async function save() {
  error.value = ''
  if (!valid.value) {
    error.value = `Имя должно быть от ${MIN} до ${MAX} символов.`
    return shake(formEl.value)
  }
  if (!changed.value) return

  busy.value = true
  try {
    const user = await updateUsername(username.value.trim())
    auth.setUser(user)
    profileStore.setUsername(user.username)
    sound.playSuccess()
    toast.show('Имя обновлено', 'success')
  } catch (err) {
    error.value =
      err instanceof ApiError && err.status === 409
        ? 'Это имя уже занято. Попробуйте другое.'
        : err instanceof ApiError && err.status === 400
          ? `Имя должно быть от ${MIN} до ${MAX} символов.`
          : 'Не удалось сохранить имя. Попробуйте ещё раз.'
    shake(formEl.value)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="account">
    <dl class="facts">
      <div>
        <dt><PhEnvelopeSimple :size="18" weight="duotone" aria-hidden="true" />Почта</dt>
        <dd>{{ profile?.email ?? auth.user?.email }}</dd>
      </div>
      <div>
        <dt><PhCalendarBlank :size="18" weight="duotone" aria-hidden="true" />С нами</dt>
        <dd>с {{ memberSince(profile?.createdAt) }}</dd>
      </div>
    </dl>

    <form ref="formEl" class="card" novalidate @submit.prevent="save">
      <div class="card-head">
        <PhPencilSimple :size="20" weight="duotone" aria-hidden="true" />
        <div>
          <h3>Имя в игре</h3>
          <p>Так вас видят соперники за столом и в рейтинге.</p>
        </div>
      </div>

      <label class="field">
        <span class="sr-only">Имя в игре</span>
        <input v-model="username" type="text" autocomplete="nickname" maxlength="64" :aria-invalid="!!error" @input="error = ''" />
        <span class="counter" :class="{ bad: length > MAX || (length > 0 && length < MIN) }">{{ length }}/{{ MAX }}</span>
      </label>

      <AnimatePresence>
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
        <button v-if="changed" type="button" class="ghost" @click="username = current">Отменить</button>
        <motion.button
          type="submit"
          class="primary"
          :disabled="busy || !changed"
          :while-press="{ scale: 0.97 }"
        >
          <span v-if="busy" class="spinner" aria-hidden="true"></span>
          {{ busy ? 'Сохраняем…' : 'Сохранить' }}
        </motion.button>
      </div>
    </form>
  </div>
</template>

<style scoped src="./form.css"></style>
