// Общие вещи лендинга: разделы для навигации и прокрутка к ним.

export interface LandingSection {
  id: string
  label: string
}

export const LANDING_SECTIONS: LandingSection[] = [
  { id: 'how-to-play', label: 'Как играть' },
  { id: 'modes', label: 'Режимы' },
  { id: 'fair-play', label: 'Честная игра' },
  { id: 'faq', label: 'Вопросы' },
]

// Русское склонение по числу: plural(5, ['карта', 'карты', 'карт']) → 'карт'
export function plural(n: number, forms: [string, string, string]) {
  const a = Math.abs(n) % 100
  const b = a % 10
  if (a > 10 && a < 20) return forms[2]
  if (b === 1) return forms[0]
  if (b >= 2 && b <= 4) return forms[1]
  return forms[2]
}

const JOIN_EVENT = 'bito:join'

export function scrollToSection(id: string) {
  const el = document.getElementById(id)
  if (!el) return
  const top = el.getBoundingClientRect().top + window.scrollY
  window.scrollTo({ top, behavior: 'smooth' })
}

// Любая кнопка «играть» на странице: наверх к карточке регистрации.
export function requestJoin() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
  window.dispatchEvent(new CustomEvent(JOIN_EVENT))
}

export function onJoinRequest(handler: () => void) {
  window.addEventListener(JOIN_EVENT, handler)
  return () => window.removeEventListener(JOIN_EVENT, handler)
}
