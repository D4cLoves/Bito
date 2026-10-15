import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

export type PanelId = 'profile' | 'settings' | 'leaderboard' | 'shop' | 'friends' | 'tables'

const PANELS: PanelId[] = ['profile', 'settings', 'leaderboard', 'shop', 'friends', 'tables']

export function usePanel() {
  const route = useRoute()
  const router = useRouter()

  const panel = computed<PanelId | null>(() => {
    const p = route.query.panel
    return typeof p === 'string' && PANELS.includes(p as PanelId) ? (p as PanelId) : null
  })

  const section = computed(() => (typeof route.query.section === 'string' ? route.query.section : null))

  function open(id: PanelId, sectionId?: string) {
    router.replace({ query: { panel: id, section: sectionId } })
  }

  function close() {
    router.replace({ query: {} })
  }

  function toggle(id: PanelId) {
    if (panel.value === id) close()
    else open(id)
  }

  function setSection(sectionId: string) {
    router.replace({ query: { ...route.query, section: sectionId } })
  }

  return { panel, section, open, close, toggle, setSection }
}
