import { ref } from 'vue'
import { acceptFriend, addFriend, getFriends, removeFriend } from '../api/friends'
import type { Friend, FriendRequest } from '../types/social'

export function useFriends() {
  const friends = ref<Friend[]>([])
  const incoming = ref<FriendRequest[]>([])
  const outgoing = ref<FriendRequest[]>([])
  const loaded = ref(false)
  const error = ref('')

  let running: Promise<void> | null = null
  let again = false

  async function load() {
    const data = await getFriends()
    friends.value = data.friends
    incoming.value = data.incoming
    outgoing.value = data.outgoing
    loaded.value = true
  }

  function refresh(): Promise<void> {
    if (running) {
      again = true
      return running
    }
    running = (async () => {
      try {
        do {
          again = false
          await load()
        } while (again)
        error.value = ''
      } catch (err) {
        error.value = err instanceof Error ? err.message : 'Не удалось загрузить друзей'
      } finally {
        running = null
      }
    })()
    return running
  }

  async function add(username: string) {
    const { status } = await addFriend(username.trim())
    await refresh()
    return status
  }

  async function accept(id: string) {
    await acceptFriend(id)
    await refresh()
  }

  async function remove(id: string) {
    await removeFriend(id)
    await refresh()
  }

  return { friends, incoming, outgoing, loaded, error, refresh, add, accept, remove }
}
