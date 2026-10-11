import { onUnmounted, ref, watch, type Ref } from 'vue'

const reduceMotion = () => window.matchMedia('(prefers-reduced-motion: reduce)').matches

export function useCountUp(target: Ref<number | null | undefined>, duration = 900) {
  const value = ref(0)
  let frame = 0

  function run(to: number) {
    cancelAnimationFrame(frame)
    const from = value.value
    if (reduceMotion() || from === to) {
      value.value = to
      return
    }
    const start = performance.now()
    const step = (now: number) => {
      const t = Math.min(1, (now - start) / duration)
      const eased = 1 - Math.pow(1 - t, 4)
      value.value = Math.round(from + (to - from) * eased)
      if (t < 1) frame = requestAnimationFrame(step)
    }
    frame = requestAnimationFrame(step)
  }

  watch(
    target,
    (to) => {
      if (typeof to === 'number') run(to)
    },
    { immediate: true },
  )

  onUnmounted(() => cancelAnimationFrame(frame))

  return value
}
