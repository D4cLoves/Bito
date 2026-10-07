// Общие вещи лендинга: разделы для навигации и прокрутка к ним.

export interface LandingSection {
  id: string
  label: string
}

export const LANDING_SECTIONS: LandingSection[] = [
  { id: 'how-to-play', label: 'How to Play' },
  { id: 'modes', label: 'Game Modes' },
  { id: 'fair-play', label: 'Fair Play' },
  { id: 'faq', label: 'FAQ' },
]

const JOIN_EVENT = 'bito:join'

export function scrollToSection(id: string) {
  const el = document.getElementById(id)
  if (!el) return
  const top = el.getBoundingClientRect().top + window.scrollY
  window.scrollTo({ top, behavior: 'smooth' })
}

// Любая кнопка «играть» на странице: наверх к карточке Join Game.
export function requestJoin() {
  window.scrollTo({ top: 0, behavior: 'smooth' })
  window.dispatchEvent(new CustomEvent(JOIN_EVENT))
}

export function onJoinRequest(handler: () => void) {
  window.addEventListener(JOIN_EVENT, handler)
  return () => window.removeEventListener(JOIN_EVENT, handler)
}
