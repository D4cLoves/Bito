<script setup lang="ts">
import { computed, nextTick, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { animate, stagger, utils } from 'animejs'
import { useAuthStore } from '../../stores/auth'
import { useGameSound } from '../../composables/useGameSound'
import { plural } from '../../composables/useLanding'

// Вход и регистрация прямо в карточке на первом экране.
// join → password → code → welcome; отдельная ветка login; signed — уже вошёл.
type Step = 'join' | 'password' | 'code' | 'welcome' | 'login' | 'signed'

const props = defineProps<{ initialMode?: 'login' | 'join' }>()

const router = useRouter()
const authStore = useAuthStore()
const sound = useGameSound()

const MIN_PASSWORD = 8
const MAX_PASSWORD = 72
const RESEND_SECONDS = 60
const START_CHIPS = 2500

const step = ref<Step>(authStore.isAuthenticated ? 'signed' : props.initialMode === 'login' ? 'login' : 'join')
const direction = ref<1 | -1>(1)

const username = ref('')
const email = ref('')
const password = ref('')
const showPassword = ref(false)
const otp = ref<string[]>(['', '', '', '', '', ''])
const error = ref('')
const busy = ref(false)
const resendLeft = ref(0)
const chipsShown = ref(formatChips(START_CHIPS))

const cardEl = ref<HTMLElement | null>(null)
const bodyEl = ref<HTMLElement | null>(null)
const errorEl = ref<HTMLElement | null>(null)
const otpEls = ref<HTMLInputElement[]>([])

let resendTimer: number | null = null

function formatChips(n: number) {
  return `+${n.toLocaleString('ru-RU')} ${plural(n, ['фишка', 'фишки', 'фишек'])}`
}
let fromHeight = 0

const displayName = computed(() => authStore.user?.username || username.value.trim() || 'игрок')

const passwordScore = computed(() => {
  const p = password.value
  let s = 0
  if (p.length >= MIN_PASSWORD) s++
  if (p.length >= 12) s++
  if (/[A-Z]/.test(p) && /[a-z]/.test(p)) s++
  if (/\d/.test(p) && /[^A-Za-z0-9]/.test(p)) s++
  return p.length ? Math.max(s, 1) : 0
})

// --------------------------------------------------------------------------
// Переходы между шагами
// --------------------------------------------------------------------------
function go(next: Step, dir: 1 | -1 = 1) {
  error.value = ''
  direction.value = dir
  step.value = next
}

function onLeave(el: Element, done: () => void) {
  const body = bodyEl.value!
  fromHeight = body.offsetHeight
  body.style.height = `${fromHeight}px`
  animate(el.querySelectorAll('[data-anim]'), {
    opacity: 0,
    x: -18 * direction.value,
    duration: 180,
    delay: stagger(18),
    ease: 'in(2)',
    onComplete: done,
  })
}

function onEnter(el: Element, done: () => void) {
  const body = bodyEl.value!
  const items = el.querySelectorAll('[data-anim]')
  utils.set(items, { opacity: 0, x: 22 * direction.value })
  body.style.height = 'auto'
  const toHeight = body.offsetHeight
  body.style.height = `${fromHeight || toHeight}px`

  animate(body, {
    height: toHeight,
    duration: 420,
    ease: 'out(4)',
    onComplete: () => {
      body.style.height = ''
    },
  })
  animate(items, {
    opacity: 1,
    x: 0,
    duration: 520,
    delay: stagger(45, { start: 80 }),
    ease: 'out(4)',
    onComplete: () => {
      done()
      focusFirst(el)
    },
  })
}

function focusFirst(el: Element) {
  if (window.matchMedia('(hover: none)').matches) return
  if (step.value === 'code') {
    otpEls.value[0]?.focus()
    return
  }
  const input = el.querySelector<HTMLInputElement>('input')
  if (input && !input.value) input.focus()
}

// --------------------------------------------------------------------------
// Ошибки: тряска карточки и плавное появление текста
// --------------------------------------------------------------------------
async function fail(message: string) {
  error.value = message
  sound.playError()
  await nextTick()
  if (cardEl.value) {
    animate(cardEl.value, {
      x: [{ to: -9 }, { to: 8 }, { to: -6 }, { to: 4 }, { to: -2 }, { to: 0 }],
      duration: 460,
      ease: 'inOut(2)',
    })
  }
  if (errorEl.value) {
    animate(errorEl.value, { opacity: [0, 1], y: [-4, 0], duration: 300, ease: 'out(3)' })
  }
}

function humanize(raw: string | null | undefined): string {
  const msg = (raw || '').toLowerCase()
  if (msg.includes('invalid credentials')) return 'Неверная почта или пароль.'
  if (msg.includes('already exists')) return 'Эта почта или ник уже заняты.'
  if (msg.includes('неверный') || msg.includes('invalid verification')) return 'Код не подходит. Попробуйте ещё раз.'
  if (msg.includes('истек') || msg.includes('expired')) return 'Срок действия кода истёк. Запросите новый.'
  if (msg.includes('связаться') || msg.includes('недоступен') || msg.includes('502')) {
    return 'Сервер не отвечает. Попробуйте чуть позже.'
  }
  return raw || 'Что-то пошло не так. Попробуйте ещё раз.'
}

const emailOk = (v: string) => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(v.trim())

// --------------------------------------------------------------------------
// Шаги регистрации
// --------------------------------------------------------------------------
function submitJoin() {
  sound.playClick()
  const name = username.value.trim()
  const nameLength = [...name].length
  if (nameLength < 3 || nameLength > 32) return fail('Ник — от 3 до 32 символов.')
  if (!emailOk(email.value)) return fail('Проверьте адрес почты.')
  go('password')
}

async function submitPassword() {
  sound.playClick()
  if (password.value.length < MIN_PASSWORD) return fail(`Минимум ${MIN_PASSWORD} символов.`)
  if (new TextEncoder().encode(password.value).length > MAX_PASSWORD) return fail('Слишком длинный пароль.')
  await requestCode(true)
}

async function requestCode(advance: boolean) {
  busy.value = true
  const ok = await authStore.requestCode({
    email: email.value.trim(),
    username: username.value.trim(),
    password: password.value,
  })
  busy.value = false
  if (!ok) return fail(humanize(authStore.error))

  sound.playCardDeal()
  otp.value = ['', '', '', '', '', '']
  startResendTimer()
  if (advance) go('code')
  else {
    error.value = ''
    otpEls.value[0]?.focus()
  }
}

function startResendTimer() {
  resendLeft.value = RESEND_SECONDS
  if (resendTimer) clearInterval(resendTimer)
  resendTimer = window.setInterval(() => {
    resendLeft.value--
    if (resendLeft.value <= 0 && resendTimer) {
      clearInterval(resendTimer)
      resendTimer = null
    }
  }, 1000)
}

const resendLabel = computed(() => {
  const s = resendLeft.value
  return `0:${String(s).padStart(2, '0')}`
})

// --------------------------------------------------------------------------
// Поле кода из 6 цифр
// --------------------------------------------------------------------------
function popDigit(index: number) {
  const el = otpEls.value[index]
  if (el) animate(el, { scale: [1.14, 1], duration: 380, ease: 'out(3)' })
}

function onOtpInput(index: number, e: Event) {
  const target = e.target as HTMLInputElement
  const digits = target.value.replace(/\D/g, '')
  if (!digits) {
    otp.value[index] = ''
    return
  }
  // Автозаполнение с клавиатуры телефона может прислать весь код сразу.
  digits.split('').slice(0, 6 - index).forEach((d, i) => {
    otp.value[index + i] = d
    popDigit(index + i)
  })
  const next = Math.min(index + digits.length, 5)
  otpEls.value[next]?.focus()
  if (otp.value.every(Boolean)) submitCode()
}

function onOtpKeydown(index: number, e: KeyboardEvent) {
  if (e.key === 'Backspace' && !otp.value[index] && index > 0) {
    otp.value[index - 1] = ''
    otpEls.value[index - 1]?.focus()
  } else if (e.key === 'ArrowLeft' && index > 0) {
    otpEls.value[index - 1]?.focus()
  } else if (e.key === 'ArrowRight' && index < 5) {
    otpEls.value[index + 1]?.focus()
  }
}

function onOtpPaste(e: ClipboardEvent) {
  const digits = (e.clipboardData?.getData('text') || '').replace(/\D/g, '').slice(0, 6)
  if (!digits) return
  e.preventDefault()
  digits.split('').forEach((d, i) => {
    otp.value[i] = d
    popDigit(i)
  })
  otpEls.value[Math.min(digits.length, 5)]?.focus()
  if (digits.length === 6) submitCode()
}

async function submitCode() {
  if (busy.value) return
  const code = otp.value.join('')
  if (code.length !== 6) return fail('Введите все 6 цифр.')

  busy.value = true
  const ok = await authStore.verifyAndRegister({ email: email.value.trim(), code })
  busy.value = false

  if (!ok) {
    otp.value = ['', '', '', '', '', '']
    animate(otpEls.value, { y: [{ to: -5 }, { to: 0 }], duration: 360, delay: stagger(30), ease: 'out(3)' })
    otpEls.value[0]?.focus()
    return fail(humanize(authStore.error))
  }

  sound.playSuccess()
  go('welcome')
  await nextTick()
  countChips()
}

function countChips(delay = 380, withSound = true) {
  const counter = { value: 0 }
  animate(counter, {
    value: START_CHIPS,
    duration: 1600,
    delay,
    ease: 'out(3)',
    onUpdate: () => {
      chipsShown.value = formatChips(Math.round(counter.value))
    },
    onComplete: () => {
      if (withSound) sound.playChipsWin()
    },
  })
  chipsShown.value = formatChips(0)
}

// --------------------------------------------------------------------------
// Вход
// --------------------------------------------------------------------------
async function submitLogin() {
  sound.playClick()
  if (!emailOk(email.value)) return fail('Проверьте адрес почты.')
  if (!password.value) return fail('Введите пароль.')

  busy.value = true
  const ok = await authStore.login({ email: email.value.trim(), password: password.value })
  busy.value = false
  if (!ok) return fail(humanize(authStore.error))

  sound.playSuccess()
  router.push('/lobby')
}

function switchTo(mode: 'login' | 'join') {
  sound.playClick()
  password.value = ''
  go(mode, mode === 'login' ? 1 : -1)
}

function openLobby() {
  sound.playClick()
  router.push('/lobby')
}

function logout() {
  sound.playClick()
  authStore.logout()
  password.value = ''
  go('join', -1)
}

onUnmounted(() => {
  if (resendTimer) clearInterval(resendTimer)
})

// Вызов с любой кнопки «играть» или «войти» на странице: открыть нужный шаг и поставить фокус.
function open(mode: 'login' | 'join') {
  if (authStore.isAuthenticated) return
  if (step.value !== mode) {
    switchTo(mode)
    return
  }
  // ждём, пока страница доедет наверх, иначе фокус дёрнет прокрутку
  window.setTimeout(() => cardEl.value?.querySelector<HTMLInputElement>('input')?.focus({ preventScroll: true }), 450)
}

const openJoin = () => open('join')
const openLogin = () => open('login')

defineExpose({ switchTo, countChips, openJoin, openLogin })
</script>

<template>
  <div ref="cardEl" class="join-card">
    <div ref="bodyEl" class="join-body">
      <Transition :css="false" mode="out-in" @leave="onLeave" @enter="onEnter">
        <!-- Шаг 1: ник и почта (вид из макета) -->
        <form v-if="step === 'join'" key="join" class="step" novalidate @submit.prevent="submitJoin">
          <h2 class="title" data-anim>Сесть за стол</h2>
          <input v-model="username" data-anim type="text" placeholder="Ник" class="field" autocomplete="username" maxlength="32" />
          <input v-model="email" data-anim type="email" placeholder="Почта" class="field" autocomplete="email" />
          <div class="row" data-anim>
            <span class="chip">{{ chipsShown }}</span>
            <button type="button" class="link" @click="switchTo('login')">Войти</button>
          </div>
          <p v-if="error" ref="errorEl" class="error">{{ error }}</p>
          <button type="submit" class="pill-btn submit" data-anim>Продолжить</button>
        </form>

        <!-- Шаг 2: пароль -->
        <form v-else-if="step === 'password'" key="password" class="step" novalidate @submit.prevent="submitPassword">
          <div class="title-row" data-anim>
            <h2 class="title">Нужен пароль</h2>
            <button type="button" class="back" aria-label="Назад" @click="go('join', -1)">
              <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M12.5 4.5 7 10l5.5 5.5" /></svg>
            </button>
          </div>
          <p class="note" data-anim>Почти готово, {{ displayName }}. С ним вы будете входить.</p>
          <input v-model="username" type="text" autocomplete="username" class="visually-hidden" tabindex="-1" aria-hidden="true" />
          <div class="field-wrap" data-anim>
            <input
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="Пароль"
              class="field field--icon"
              autocomplete="new-password"
            />
            <button type="button" class="eye" :aria-label="showPassword ? 'Скрыть пароль' : 'Показать пароль'" @click="showPassword = !showPassword">
              <svg v-if="!showPassword" viewBox="0 0 24 24" aria-hidden="true"><path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12z" /><circle cx="12" cy="12" r="3" /></svg>
              <svg v-else viewBox="0 0 24 24" aria-hidden="true"><path d="M3 3l18 18M10.6 5.1A10.7 10.7 0 0 1 12 5c6.4 0 10 7 10 7a17 17 0 0 1-3.2 4M6.6 6.6C3.7 8.4 2 12 2 12s3.6 7 10 7c1.7 0 3.2-.5 4.5-1.2M9.9 9.9a3 3 0 0 0 4.2 4.2" /></svg>
            </button>
          </div>
          <div class="strength" data-anim>
            <span v-for="i in 4" :key="i" class="strength-bar" :class="{ on: passwordScore >= i }"></span>
            <span class="strength-text">{{ password.length >= MIN_PASSWORD ? 'Подходит' : `от ${MIN_PASSWORD} символов` }}</span>
          </div>
          <p v-if="error" ref="errorEl" class="error">{{ error }}</p>
          <button type="submit" class="pill-btn submit" data-anim :disabled="busy">
            <span v-if="busy" class="dots"><i></i><i></i><i></i></span>
            <span v-else>Получить код</span>
          </button>
        </form>

        <!-- Шаг 3: код из письма -->
        <form v-else-if="step === 'code'" key="code" class="step" novalidate @submit.prevent="submitCode">
          <div class="title-row" data-anim>
            <h2 class="title">Код из письма</h2>
            <button type="button" class="back" aria-label="Назад" @click="go('password', -1)">
              <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M12.5 4.5 7 10l5.5 5.5" /></svg>
            </button>
          </div>
          <p class="note" data-anim>
            Отправили 6-значный код на <strong :title="email.trim()">{{ email.trim() }}</strong>
          </p>
          <div class="otp" data-anim @paste="onOtpPaste">
            <input
              v-for="(_, i) in otp"
              :key="i"
              :ref="(el) => { if (el) otpEls[i] = el as HTMLInputElement }"
              :value="otp[i]"
              class="otp-cell"
              :class="{ filled: otp[i] }"
              type="text"
              inputmode="numeric"
              autocomplete="one-time-code"
              maxlength="6"
              :aria-label="`Цифра ${i + 1}`"
              @input="onOtpInput(i, $event)"
              @keydown="onOtpKeydown(i, $event)"
            />
          </div>
          <div class="row row--muted" data-anim>
            <span v-if="resendLeft > 0">Повтор через {{ resendLabel }}</span>
            <button v-else type="button" class="link" :disabled="busy" @click="requestCode(false)">Отправить снова</button>
            <button type="button" class="link" @click="go('join', -1)">Сменить почту</button>
          </div>
          <p v-if="error" ref="errorEl" class="error">{{ error }}</p>
          <button type="submit" class="pill-btn submit" data-anim :disabled="busy">
            <span v-if="busy" class="dots"><i></i><i></i><i></i></span>
            <span v-else>Подтвердить</span>
          </button>
        </form>

        <!-- Шаг 4: готово -->
        <div v-else-if="step === 'welcome'" key="welcome" class="step">
          <h2 class="title" data-anim>Вы в игре, {{ displayName }}</h2>
          <p class="note" data-anim>Место готово. Вот ваши стартовые фишки.</p>
          <div class="stack" data-anim>
            <span class="chip chip--big">{{ chipsShown }}</span>
          </div>
          <button type="button" class="pill-btn submit" data-anim @click="openLobby">В лобби</button>
        </div>

        <!-- Вход -->
        <form v-else-if="step === 'login'" key="login" class="step" novalidate @submit.prevent="submitLogin">
          <h2 class="title" data-anim>С возвращением</h2>
          <input v-model="email" data-anim type="email" placeholder="Почта" class="field" autocomplete="email" />
          <div class="field-wrap" data-anim>
            <input
              v-model="password"
              :type="showPassword ? 'text' : 'password'"
              placeholder="Пароль"
              class="field field--icon"
              autocomplete="current-password"
            />
            <button type="button" class="eye" :aria-label="showPassword ? 'Скрыть пароль' : 'Показать пароль'" @click="showPassword = !showPassword">
              <svg v-if="!showPassword" viewBox="0 0 24 24" aria-hidden="true"><path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12z" /><circle cx="12" cy="12" r="3" /></svg>
              <svg v-else viewBox="0 0 24 24" aria-hidden="true"><path d="M3 3l18 18M10.6 5.1A10.7 10.7 0 0 1 12 5c6.4 0 10 7 10 7a17 17 0 0 1-3.2 4M6.6 6.6C3.7 8.4 2 12 2 12s3.6 7 10 7c1.7 0 3.2-.5 4.5-1.2M9.9 9.9a3 3 0 0 0 4.2 4.2" /></svg>
            </button>
          </div>
          <div class="row row--muted" data-anim>
            <span>Впервые в Bito?</span>
            <button type="button" class="link" @click="switchTo('join')">Регистрация</button>
          </div>
          <p v-if="error" ref="errorEl" class="error">{{ error }}</p>
          <button type="submit" class="pill-btn submit" data-anim :disabled="busy">
            <span v-if="busy" class="dots"><i></i><i></i><i></i></span>
            <span v-else>Войти</span>
          </button>
        </form>

        <!-- Уже вошёл -->
        <div v-else key="signed" class="step">
          <h2 class="title" data-anim>Привет, {{ displayName }}</h2>
          <p class="note" data-anim>Стол ждёт. Возвращайтесь, когда будете готовы.</p>
          <div class="row" data-anim>
            <span class="chip">{{ chipsShown.replace('+', '') }}</span>
            <button type="button" class="link" @click="logout">Выйти</button>
          </div>
          <button type="button" class="pill-btn submit" data-anim @click="openLobby">В лобби</button>
        </div>
      </Transition>
    </div>
  </div>
</template>

<style scoped>
.join-card {
  box-sizing: border-box;
  min-height: 344px;
  padding: 28.5px 27px 28px 29px;
  border-radius: 22px;
  background: rgba(247, 246, 241, 0.42);
  border: 1px solid rgba(255, 255, 255, 0.9);
  box-shadow:
    0 0 0 1.5px rgba(196, 190, 176, 0.45),
    0 24px 50px -18px rgba(80, 74, 60, 0.18);
  backdrop-filter: blur(16px) saturate(1.05);
  -webkit-backdrop-filter: blur(16px) saturate(1.05);
  color: #1b1916;
}

.join-body {
  overflow: hidden;
  /* место под фокус-кольца и подпрыгивание ячеек кода */
  margin: -6px;
  padding: 6px;
}

.step {
  display: flex;
  flex-direction: column;
}

.title {
  margin: 0 0 18.5px;
  font-size: 32px;
  font-weight: 400;
  line-height: 38px;
  letter-spacing: -0.015em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.title-row .title {
  margin-bottom: 8px;
  font-size: 28px;
}

.back {
  flex: none;
  width: 34px;
  height: 34px;
  margin-top: 2px;
  display: grid;
  place-items: center;
  border: 1.5px solid #ddd7cd;
  border-radius: 50%;
  background: rgba(238, 236, 230, 0.55);
  color: #3a362f;
  cursor: pointer;
  transition: background-color 0.2s, border-color 0.2s;
}

.back:hover {
  background: rgba(250, 249, 245, 0.9);
  border-color: #c9c2b5;
}

.back svg {
  width: 16px;
  height: 16px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.8;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.note {
  margin: 0 0 16px;
  font-size: 14.5px;
  line-height: 21px;
  letter-spacing: -0.006em;
  color: #6b675e;
}

.note strong {
  display: block;
  overflow: hidden;
  font-weight: 500;
  color: #1b1916;
  white-space: nowrap;
  text-overflow: ellipsis;
}

.field {
  width: 100%;
  height: 50px;
  box-sizing: border-box;
  margin-bottom: 13px;
  padding: 0 15px;
  border: 1.5px solid #ddd7cd;
  border-radius: 10px;
  background: rgba(238, 236, 230, 0.55);
  font: inherit;
  font-size: 15.8px;
  letter-spacing: -0.006em;
  color: #1b1916;
  outline: none;
  transition: border-color 0.2s, background-color 0.2s, box-shadow 0.2s;
}

.field::placeholder {
  color: #78756c;
}

.field:focus {
  border-color: #bdb6a8;
  background: rgba(250, 249, 245, 0.85);
  box-shadow: 0 0 0 4px rgba(215, 211, 200, 0.45);
}

.field-wrap {
  position: relative;
}

.field--icon {
  padding-right: 46px;
}

.eye {
  position: absolute;
  right: 6px;
  top: 6px;
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: #78756c;
  cursor: pointer;
}

.eye:hover {
  color: #1b1916;
}

.eye svg {
  width: 19px;
  height: 19px;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.6;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 35px;
  margin-top: 2px;
}

.row--muted {
  margin-top: 0;
  font-size: 14px;
  color: #6b675e;
}

.chip {
  height: 35px;
  padding: 0 12px 0 13px;
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  background: #e4ded2;
  font-size: 16px;
  letter-spacing: -0.005em;
  font-variant-numeric: tabular-nums;
}

.chip--big {
  height: 44px;
  padding: 0 18px;
  font-size: 20px;
}

.stack {
  display: flex;
  margin: 2px 0 4px;
}

.link {
  padding: 4px 2px;
  border: 0;
  background: none;
  font: inherit;
  font-size: 14px;
  color: #5c5850;
  text-decoration: underline;
  text-decoration-color: rgba(92, 88, 80, 0.35);
  text-underline-offset: 3px;
  cursor: pointer;
  transition: color 0.2s, text-decoration-color 0.2s;
}

.link:hover {
  color: #1b1916;
  text-decoration-color: currentColor;
}

.link:disabled {
  opacity: 0.5;
  cursor: default;
}

.strength {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: -1px 0 2px;
  min-height: 22px;
}

.strength-bar {
  width: 30px;
  height: 4px;
  border-radius: 2px;
  background: #e2ddd3;
  transition: background-color 0.35s;
}

.strength-bar.on {
  background: #8f8a7d;
}

.strength-text {
  margin-left: 6px;
  font-size: 13px;
  color: #78756c;
}

.otp {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.otp-cell {
  flex: 1;
  min-width: 0;
  height: 54px;
  box-sizing: border-box;
  border: 1.5px solid #ddd7cd;
  border-radius: 10px;
  background: rgba(238, 236, 230, 0.55);
  font: inherit;
  font-size: 22px;
  text-align: center;
  color: #1b1916;
  outline: none;
  caret-color: #8f8a7d;
  transition: border-color 0.2s, background-color 0.2s, box-shadow 0.2s;
}

.otp-cell:focus {
  border-color: #bdb6a8;
  background: rgba(250, 249, 245, 0.85);
  box-shadow: 0 0 0 4px rgba(215, 211, 200, 0.45);
}

.otp-cell.filled {
  background: rgba(250, 249, 245, 0.9);
}

.error {
  margin: 10px 0 -8px;
  font-size: 13.5px;
  line-height: 18px;
  color: #a2412f;
}

.pill-btn {
  border: 0;
  border-radius: 999px;
  background: #d7d3c8;
  color: #1b1916;
  font: inherit;
  font-size: 15.5px;
  letter-spacing: -0.006em;
  cursor: pointer;
  transition: background-color 0.2s, transform 0.2s;
}

.pill-btn:hover:not(:disabled) {
  background: #cdc8bb;
}

.pill-btn:active:not(:disabled) {
  transform: scale(0.985);
}

.pill-btn:disabled {
  cursor: default;
}

.submit {
  width: 100%;
  height: 46px;
  margin-top: 20px;
}

.dots {
  display: inline-flex;
  gap: 5px;
}

.dots i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #1b1916;
  animation: dot 0.9s ease-in-out infinite;
}

.dots i:nth-child(2) {
  animation-delay: 0.12s;
}

.dots i:nth-child(3) {
  animation-delay: 0.24s;
}

@keyframes dot {
  0%,
  80%,
  100% {
    opacity: 0.25;
    transform: translateY(0);
  }
  40% {
    opacity: 1;
    transform: translateY(-3px);
  }
}

.visually-hidden {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0 0 0 0);
  opacity: 0;
  pointer-events: none;
}
</style>
