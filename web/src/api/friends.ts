import { authedFetch } from './http'
import type { FriendsOverview } from '../types/social'

const BASE = '/api/v1/friends'

export function getFriends(): Promise<FriendsOverview> {
  return authedFetch<FriendsOverview>(BASE)
}

export function addFriend(username: string): Promise<{ status: 'pending' | 'accepted' }> {
  return authedFetch(`${BASE}/requests`, {
    method: 'POST',
    body: JSON.stringify({ username }),
  })
}

export async function acceptFriend(id: string): Promise<void> {
  await authedFetch<null>(`${BASE}/requests/${encodeURIComponent(id)}/accept`, { method: 'POST' })
}

export async function removeFriend(id: string): Promise<void> {
  await authedFetch<null>(`${BASE}/${encodeURIComponent(id)}`, { method: 'DELETE' })
}
