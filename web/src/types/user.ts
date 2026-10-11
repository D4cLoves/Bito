import type { UserResponse } from './auth'

export interface PlayerStats {
  elo: number
  wins: number
  losses: number
  totalGames: number
}

export interface Wallet {
  balance: number
}

export interface Profile extends UserResponse {
  stats: PlayerStats
  wallet: Wallet
}

export interface ChangePasswordRequest {
  oldPassword: string
  newPassword: string
}

export interface LeaderboardEntry {
  id: string
  username: string
  elo: number
  wins: number
  losses: number
}
