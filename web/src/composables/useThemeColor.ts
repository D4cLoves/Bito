import { nextTick, onMounted, onUnmounted, watch, type Ref, type WatchSource } from 'vue'

export function useThemeColor(el: Ref<HTMLElement | null>, theme: WatchSource<unknown>) {
  const meta = document.querySelector<HTMLMetaElement>('meta[name="theme-color"]')
  const initial = meta?.content

  async function apply() {
    await nextTick()
    if (!meta || !el.value) return
    meta.content = getComputedStyle(el.value).backgroundColor
  }

  onMounted(apply)
  watch(theme, apply)
  onUnmounted(() => {
    if (meta && initial) meta.content = initial
  })
}
