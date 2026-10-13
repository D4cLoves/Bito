import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'
import { useGameSound } from './useGameSound'

const ALERT_TITLE = '🃏 Ваш ход!'
const BLINK_MS = 1000
const LAST_CALL_MS = 5000

export function useTurnAlert(myTurn: Ref<boolean>, turnLeft: Ref<number>) {
  const sound = useGameSound()
  let baseTitle = ''
  let blinkTimer: number | null = null
  let lastCallPlayed = false

  function start(withSound: boolean) {
    if (blinkTimer) return
    baseTitle = document.title
    lastCallPlayed = false
    if (withSound) sound.playYourTurn()
    let alt = true
    document.title = ALERT_TITLE
    blinkTimer = window.setInterval(() => {
      alt = !alt
      document.title = alt ? ALERT_TITLE : baseTitle
    }, BLINK_MS)
  }

  function stop() {
    if (!blinkTimer) return
    clearInterval(blinkTimer)
    blinkTimer = null
    document.title = baseTitle
  }

  function onVisibility() {
    if (!document.hidden) stop()
    else if (myTurn.value) start(false)
  }

  watch(myTurn, (turn) => {
    if (turn && document.hidden) start(true)
    else if (!turn) stop()
  })

  watch(turnLeft, (left) => {
    if (!blinkTimer || lastCallPlayed || left <= 0 || left > LAST_CALL_MS) return
    lastCallPlayed = true
    sound.playYourTurn()
  })

  onMounted(() => document.addEventListener('visibilitychange', onVisibility))

  onBeforeUnmount(() => {
    document.removeEventListener('visibilitychange', onVisibility)
    stop()
  })
}
