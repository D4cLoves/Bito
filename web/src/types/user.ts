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

export type TransactionKind = 'signup_bonus' | 'opening' | 'stake_lock' | 'stake_refund' | 'win_payout' | 'top_up'

export interface WalletTransaction {
  id: number
  amount: number
  kind: TransactionKind
  gameId?: string
  balanceAfter: number
  createdAt: string
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
