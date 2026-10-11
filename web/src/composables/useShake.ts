import { animate } from 'motion-v'
import { useGameSound } from './useGameSound'

export function useShake() {
  const sound = useGameSound()

  return function shake(el: HTMLElement | null | undefined) {
    sound.playError()
    if (!el || window.matchMedia('(prefers-reduced-motion: reduce)').matches) return
    animate(el, { x: [0, -9, 8, -6, 4, -2, 0] }, { duration: 0.45, ease: 'easeInOut' })
  }
}
