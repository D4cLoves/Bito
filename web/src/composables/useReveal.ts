import { onMounted, onUnmounted, type Ref } from 'vue'
import { animate, onScroll, stagger, utils } from 'animejs'
import type { JSAnimation } from 'animejs'

// Элементы с [data-reveal] внутри root плавно поднимаются, когда секция входит в экран.
export function useReveal(root: Ref<HTMLElement | null>) {
  let animation: JSAnimation | null = null

  onMounted(() => {
    if (!root.value || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    const items = root.value.querySelectorAll<HTMLElement>('[data-reveal]')
    utils.set(items, { opacity: 0, y: 28 })
    animation = animate(items, {
      opacity: 1,
      y: 0,
      duration: 900,
      delay: stagger(90),
      ease: 'out(4)',
      autoplay: onScroll({ target: root.value, enter: 'bottom-=120 top' }),
      // после появления возвращаем элементам их собственные :hover/:active
      onComplete: () =>
        items.forEach((el) => {
          el.style.removeProperty('transform')
          el.style.removeProperty('opacity')
        }),
    })
  })

  onUnmounted(() => animation?.revert())
}
