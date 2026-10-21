import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getAvatars } from '../api/shop'

const BATCH = 50
const FRESH_MS = 120_000

export const useAvatarsStore = defineStore('avatars', () => {
  const known = ref(new Map<string, string>())
  const fetchedAt = new Map<string, number>()
  const queue = new Set<string>()
  let timer: number | null = null

  function of(id: string | undefined) {
    return id ? known.value.get(id) : undefined
  }

  function want(id: string | undefined) {
    if (!id || queue.has(id)) return
    const at = fetchedAt.get(id)
    if (at && Date.now() - at < FRESH_MS) return
    queue.add(id)
    timer ??= window.setTimeout(flush, 30)
  }

  function remember(id: string, avatar: string) {
    known.value = new Map(known.value).set(id, avatar)
    fetchedAt.set(id, Date.now())
  }

  async function flush() {
    timer = null
    const ids = [...queue].slice(0, BATCH)
    ids.forEach((id) => {
      queue.delete(id)
      fetchedAt.set(id, Date.now())
    })
    if (queue.size) timer = window.setTimeout(flush, 30)

    try {
      const got = await getAvatars(ids)
      const next = new Map(known.value)
      for (const [id, avatar] of Object.entries(got)) next.set(id, avatar)
      known.value = next
    } catch {
      ids.forEach((id) => fetchedAt.delete(id))
    }
  }

  return { of, want, remember }
})
