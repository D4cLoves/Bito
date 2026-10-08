<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { animate, stagger, utils } from 'animejs'
import { useAuthStore } from '../../stores/auth'
import { LANDING_SECTIONS, requestJoin, scrollToSection } from '../../composables/useLanding'

// Плавающий хедер: появляется, когда шапка первого экрана ушла из вида,
// и подсвечивает раздел, который сейчас на экране.
const SHOW_AFTER = 160

const router = useRouter()
const authStore = useAuthStore()

const rootEl = ref<HTMLElement | null>(null)
const navEl = ref<HTMLElement | null>(null)
const indicatorEl = ref<HTMLElement | null>(null)
const visible = ref(false)
const active = ref<string | null>(null)
const menuOpen = ref(false)
const menuEl = ref<HTMLElement | null>(null)

let observer: IntersectionObserver | null = null

function onScrollWindow() {
  visible.value = window.scrollY > SHOW_AFTER
  if (!visible.value) menuOpen.value = false
}

// Мобильное меню: панель выезжает из-под шапки, ссылки появляются лесенкой
watch(menuOpen, async (open) => {
  await nextTick()
  const el = menuEl.value
  if (!el) return
  if (open) {
    el.style.display = 'flex'
    animate(el, { opacity: [0, 1], y: [-10, 0], duration: 320, ease: 'out(3)' })
    animate(el.querySelectorAll('a'), { opacity: [0, 1], x: [-8, 0], delay: stagger(45, { start: 60 }), duration: 320, ease: 'out(3)' })
  } else {
    animate(el, { opacity: 0, y: -8, duration: 180, ease: 'in(2)', onComplete: () => { if (!menuOpen.value) el.style.display = 'none' } })
  }
})

function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') menuOpen.value = false
}

watch(visible, (show) => {
  if (!rootEl.value) return
  animate(rootEl.value, {
    y: show ? 0 : -90,
    opacity: show ? 1 : 0,
    duration: show ? 520 : 320,
    ease: show ? 'out(4)' : 'in(2)',
  })
  rootEl.value.style.pointerEvents = show ? 'auto' : 'none'
})

// Подсветка-«пилюля» переезжает под активную ссылку
function moveIndicator(instant = false) {
  const nav = navEl.value
  const ind = indicatorEl.value
  if (!nav || !ind) return
  const link = active.value ? nav.querySelector<HTMLElement>(`[data-id="${active.value}"]`) : null
  if (!link) {
    animate(ind, { opacity: 0, duration: 200 })
    return
  }
  const props = { x: link.offsetLeft, width: link.offsetWidth, opacity: 1 }
  if (instant) utils.set(ind, props)
  else animate(ind, { ...props, duration: 480, ease: 'out(4)' })
}

watch(active, () => nextTick(() => moveIndicator()))

function go(id: string) {
  menuOpen.value = false
  scrollToSection(id)
}

function play() {
  menuOpen.value = false
  if (authStore.isAuthenticated) router.push('/lobby')
  else requestJoin()
}

onMounted(() => {
  utils.set(rootEl.value!, { y: -90, opacity: 0 })
  rootEl.value!.style.pointerEvents = 'none'
  onScrollWindow()
  window.addEventListener('scroll', onScrollWindow, { passive: true })
  window.addEventListener('keydown', onKey)

  // Раздел считается активным, когда пересекает полосу в середине экрана
  observer = new IntersectionObserver(
    (entries) => {
      for (const e of entries) {
        if (e.isIntersecting) active.value = e.target.id
        else if (active.value === e.target.id) active.value = null
      }
    },
    { rootMargin: '-45% 0px -50% 0px' },
  )
  LANDING_SECTIONS.forEach((s) => {
    const el = document.getElementById(s.id)
    if (el) observer!.observe(el)
  })
})

onUnmounted(() => {
  window.removeEventListener('scroll', onScrollWindow)
  window.removeEventListener('keydown', onKey)
  observer?.disconnect()
})
</script>

<template>
  <header ref="rootEl" class="site-header">
    <a href="#" class="brand" aria-label="Bito — наверх" @click.prevent="requestJoin">
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
    </a>

    <nav ref="navEl" class="nav">
      <span ref="indicatorEl" class="indicator" aria-hidden="true"></span>
      <a
        v-for="s in LANDING_SECTIONS"
        :key="s.id"
        :href="`#${s.id}`"
        :data-id="s.id"
        :class="{ on: active === s.id }"
        :aria-current="active === s.id ? 'true' : undefined"
        @click.prevent="go(s.id)"
      >
        {{ s.label }}
      </a>
    </nav>

    <button type="button" class="cta" @click="play">
      {{ authStore.isAuthenticated ? 'В лобби' : 'Играть' }}
    </button>

    <button
      type="button"
      class="burger"
      :class="{ open: menuOpen }"
      :aria-expanded="menuOpen"
      aria-controls="site-menu"
      aria-label="Разделы"
      @click="menuOpen = !menuOpen"
    >
      <span></span><span></span>
    </button>

    <nav id="site-menu" ref="menuEl" class="menu" aria-label="Разделы">
      <a
        v-for="(s, i) in LANDING_SECTIONS"
        :key="s.id"
        :href="`#${s.id}`"
        :class="{ on: active === s.id }"
        @click.prevent="go(s.id)"
      >
        <small>0{{ i + 1 }}</small>{{ s.label }}
      </a>
    </nav>
  </header>
</template>

<style scoped>
.site-header {
  position: fixed;
  z-index: 100;
  top: 14px;
  left: 50%;
  translate: -50% 0;
  width: min(1120px, calc(100% - 24px));
  height: 60px;
  box-sizing: border-box;
  padding: 0 8px 0 18px;
  display: flex;
  align-items: center;
  border-radius: 999px;
  background: rgba(247, 246, 241, 0.72);
  box-shadow:
    0 0 0 1px rgba(214, 208, 196, 0.9),
    0 18px 40px -22px rgba(70, 62, 48, 0.35);
  backdrop-filter: blur(18px) saturate(1.1);
  -webkit-backdrop-filter: blur(18px) saturate(1.1);
  font-family: 'Inter', -apple-system, 'Helvetica Neue', Arial, sans-serif;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 21px;
  letter-spacing: -0.01em;
  color: #1b1916;
  text-decoration: none;
}

.brand svg {
  width: 30px;
  height: 29px;
  color: #26231e;
}

.nav {
  position: relative;
  display: flex;
  margin: 0 auto;
  padding: 4px;
}

.indicator {
  position: absolute;
  left: 0;
  top: 4px;
  bottom: 4px;
  width: 0;
  border-radius: 999px;
  background: #e5e0d5;
  opacity: 0;
}

.nav a {
  position: relative;
  padding: 9px 16px;
  border-radius: 999px;
  font-size: 15px;
  letter-spacing: -0.01em;
  color: #57534b;
  text-decoration: none;
  transition: color 0.25s;
}

.nav a:hover,
.nav a.on {
  color: #1b1916;
}

.cta {
  height: 44px;
  padding: 0 22px;
  border: 0;
  border-radius: 999px;
  background: #36322c;
  color: #f5f4ef;
  font: inherit;
  font-size: 15px;
  cursor: pointer;
  transition:
    background-color 0.2s,
    transform 0.2s;
}

.cta:hover {
  background: #1b1916;
}

.cta:active {
  transform: scale(0.98);
}

.burger,
.menu {
  display: none;
}

@media (max-width: 760px) {
  .site-header {
    padding-right: 6px;
  }

  .nav {
    display: none;
  }

  .cta {
    margin-left: auto;
    height: 44px;
    padding: 0 18px;
  }

  .burger {
    display: grid;
    place-content: center;
    gap: 6px;
    width: 44px;
    height: 44px;
    margin-left: 6px;
    border: 0;
    border-radius: 999px;
    background: #e5e0d5;
    cursor: pointer;
  }

  .burger span {
    display: block;
    width: 18px;
    height: 2px;
    border-radius: 2px;
    background: #1b1916;
    transition: transform 0.3s cubic-bezier(0.2, 0.8, 0.2, 1);
  }

  .burger.open span:first-child {
    transform: translateY(4px) rotate(45deg);
  }

  .burger.open span:last-child {
    transform: translateY(-4px) rotate(-45deg);
  }

  .menu {
    position: absolute;
    top: calc(100% + 8px);
    left: 0;
    right: 0;
    flex-direction: column;
    padding: 8px;
    border-radius: 24px;
    background: rgba(247, 246, 241, 0.94);
    box-shadow:
      0 0 0 1px rgba(214, 208, 196, 0.9),
      0 24px 48px -24px rgba(70, 62, 48, 0.45);
    backdrop-filter: blur(18px) saturate(1.1);
    -webkit-backdrop-filter: blur(18px) saturate(1.1);
  }

  .menu a {
    display: flex;
    align-items: baseline;
    gap: 12px;
    padding: 14px 16px;
    border-radius: 16px;
    font-size: 17px;
    letter-spacing: -0.01em;
    color: #1b1916;
    text-decoration: none;
  }

  .menu a small {
    font-size: 12px;
    color: #9a9384;
    font-variant-numeric: tabular-nums;
  }

  .menu a.on,
  .menu a:active {
    background: #e5e0d5;
  }
}
</style>
