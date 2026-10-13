import { onMounted, onUnmounted, type Ref } from 'vue'
import Lenis from 'lenis'
import 'lenis/dist/lenis.css'

const SETTINGS = { lerp: 0.12, wheelMultiplier: 0.9, autoRaf: true } as const

let pageLenis: Lenis | null = null

export function useSmoothScroll(wrapper?: Ref<HTMLElement | null>) {
  let lenis: Lenis | null = null

  onMounted(() => {
    if (!wrapper) {
      lenis = new Lenis(SETTINGS)
      pageLenis = lenis
      return
    }
    const el = wrapper.value
    const content = el?.firstElementChild
    if (!el || !content) return
    lenis = new Lenis({ ...SETTINGS, wrapper: el, content })
  })

  onUnmounted(() => {
    if (pageLenis === lenis) pageLenis = null
    lenis?.destroy()
    lenis = null
  })
}

export function scrollPageTo(top: number) {
  if (pageLenis) pageLenis.scrollTo(top)
  else window.scrollTo({ top, behavior: 'smooth' })
}
