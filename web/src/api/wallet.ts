import { authedFetch } from './http'
import type { WalletTransaction } from '../types/user'

const WALLET = '/api/v1/wallet'

export function topUp(): Promise<{ balance: number }> {
  return authedFetch<{ balance: number }>(`${WALLET}/top-up`, { method: 'POST' })
}

export function getTransactions(limit = 50): Promise<WalletTransaction[]> {
  return authedFetch<WalletTransaction[]>(`${WALLET}/transactions?limit=${limit}`)
}
