import { authedFetch } from './http'
import type { CreateTableRequest, QuickMatchRequest, TableInfo } from '../types/game'

export function listTables(): Promise<TableInfo[]> {
  return authedFetch<TableInfo[]>('/api/v1/tables')
}

export function listLiveTables(): Promise<TableInfo[]> {
  return authedFetch<TableInfo[]>('/api/v1/tables/live')
}

export function getTable(id: string): Promise<TableInfo> {
  return authedFetch<TableInfo>(`/api/v1/tables/${encodeURIComponent(id)}`)
}

export function createTable(req: CreateTableRequest): Promise<TableInfo> {
  return authedFetch<TableInfo>('/api/v1/tables', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

export async function getActiveTable(): Promise<TableInfo | null> {
  return (await authedFetch<TableInfo | null>('/api/v1/users/me/active-table')) ?? null
}

export function quickMatch(req: QuickMatchRequest): Promise<TableInfo> {
  return authedFetch<TableInfo>('/api/v1/tables/quick', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

export function inviteToTable(tableId: string, userId: string): Promise<{ tableId: string; expiresAt: string }> {
  return authedFetch(`/api/v1/tables/${encodeURIComponent(tableId)}/invites`, {
    method: 'POST',
    body: JSON.stringify({ userId }),
  })
}

export async function declineInvite(tableId: string): Promise<void> {
  await authedFetch<null>(`/api/v1/tables/${encodeURIComponent(tableId)}/invites/decline`, { method: 'POST' })
}
