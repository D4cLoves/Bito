import { authedFetch } from './http'
import type { CosmeticKind } from '../constants/cosmetics'

export interface Look {
  theme: string
  felt: string
  back: string
}

export interface ShopState {
  items: { kind: CosmeticKind; id: string; price: number }[]
  owned: string[]
  look: Look
  lookSaved: boolean
  avatar: string
}

export function getShop(): Promise<ShopState> {
  return authedFetch<ShopState>('/api/v1/shop')
}

export function buyItem(item: string): Promise<{ balance: number; owned: string[] }> {
  return authedFetch('/api/v1/shop/buy', {
    method: 'POST',
    body: JSON.stringify({ item }),
  })
}

export async function saveLook(look: Look): Promise<void> {
  await authedFetch<null>('/api/v1/shop/look', {
    method: 'PUT',
    body: JSON.stringify(look),
  })
}

export async function saveAvatar(avatar: string): Promise<void> {
  await authedFetch<null>('/api/v1/shop/avatar', {
    method: 'PUT',
    body: JSON.stringify({ avatar }),
  })
}

export function getAvatars(ids: string[]): Promise<Record<string, string>> {
  return authedFetch<Record<string, string>>(`/api/v1/shop/avatars?ids=${ids.map(encodeURIComponent).join(',')}`)
}
