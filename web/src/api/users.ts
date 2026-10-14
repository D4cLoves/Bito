import { authedFetch } from './http'
import type { UserResponse } from '../types/auth'
import type { ChangePasswordRequest, LeaderboardEntry, Profile } from '../types/user'
import type { GameHistoryEntry } from '../types/social'

const ME = '/api/v1/users/me'

export function getProfile(): Promise<Profile> {
  return authedFetch<Profile>(ME)
}

export function updateUsername(username: string): Promise<UserResponse> {
  return authedFetch<UserResponse>(ME, {
    method: 'PATCH',
    body: JSON.stringify({ username }),
  })
}

export async function changePassword(req: ChangePasswordRequest): Promise<void> {
  await authedFetch<null>(`${ME}/password`, {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

export async function getGameHistory(limit = 20): Promise<GameHistoryEntry[]> {
  return (await authedFetch<GameHistoryEntry[] | null>(`${ME}/games?limit=${limit}`)) ?? []
}

export async function deleteAccount(): Promise<void> {
  await authedFetch<null>(ME, { method: 'DELETE' })
}

export function getLeaderboard(limit = 10): Promise<LeaderboardEntry[]> {
  return authedFetch<LeaderboardEntry[]>(`/api/v1/users/top?limit=${limit}`)
}
