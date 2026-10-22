import { authedFetch } from './http'
import type { WalletTransaction } from '../types/user'

const WALLET = '/api/v1/wallet'

export function topUp(): Promise<{ balance: number }> {
  return authedFetch<{ balance: number }>(`${WALLET}/top-up`, { method: 'POST' })
}

export async function getTransactions(limit = 50): Promise<WalletTransaction[]> {
  return (await authedFetch<WalletTransaction[] | null>(`${WALLET}/transactions?limit=${limit}`)) ?? []
}
