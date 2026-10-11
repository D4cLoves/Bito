import { authedFetch } from './http'
import type { CreateTableRequest, TableInfo } from '../types/game'

export function listTables(): Promise<TableInfo[]> {
  return authedFetch<TableInfo[]>('/api/v1/tables')
}

export function createTable(req: CreateTableRequest): Promise<TableInfo> {
  return authedFetch<TableInfo>('/api/v1/tables', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}

export function quickMatch(req: CreateTableRequest): Promise<TableInfo> {
  return authedFetch<TableInfo>('/api/v1/tables/quick', {
    method: 'POST',
    body: JSON.stringify(req),
  })
}
