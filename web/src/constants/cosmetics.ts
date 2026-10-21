export type CosmeticKind = 'theme' | 'felt' | 'back' | 'avatar'
export type CosmeticFilterValue = 'all' | CosmeticKind

export interface CosmeticItem<Id extends string = string> {
  id: Id
  name: string
  price: number
  note?: string
}

export const THEMES = [
  { id: 'paper', name: 'Бумага', price: 0, note: 'светлая, как на лендинге' },
  { id: 'hall', name: 'Тёмный зал', price: 0, note: 'тёплый свет над столом' },
  { id: 'desert', name: 'Пустыня', price: 2500, note: 'песок и терракота' },
  { id: 'blush', name: 'Пудра', price: 2500, note: 'светлая, розовая' },
  { id: 'forest', name: 'Лес', price: 3000, note: 'тёмно-зелёная' },
  { id: 'ocean', name: 'Океан', price: 3000, note: 'глубокий синий' },
  { id: 'noir', name: 'Нуар', price: 4000, note: 'графит и серебро' },
] as const satisfies readonly CosmeticItem[]

export const FELTS = [
  { id: 'linen', name: 'Льняное', price: 0 },
  { id: 'green', name: 'Зелёное', price: 0 },
  { id: 'sand', name: 'Песочное', price: 1500 },
  { id: 'wine', name: 'Бордо', price: 1500 },
  { id: 'navy', name: 'Синее', price: 1500 },
  { id: 'graphite', name: 'Графит', price: 1500 },
  { id: 'teal', name: 'Бирюза', price: 2000 },
  { id: 'terracotta', name: 'Терракота', price: 2000 },
  { id: 'royal', name: 'Королевское', price: 2500 },
] as const satisfies readonly CosmeticItem[]

export const CARD_BACKS = [
  { id: 'classic', name: 'Классика', price: 0 },
  { id: 'bito', name: 'Bito', price: 0 },
  { id: 'checker', name: 'Шахматка', price: 1500 },
  { id: 'grove', name: 'Роща', price: 1500 },
  { id: 'emerald', name: 'Изумруд', price: 2000 },
  { id: 'sakura', name: 'Сакура', price: 2000 },
  { id: 'waves', name: 'Волны', price: 2500 },
  { id: 'deco', name: 'Ар-деко', price: 2500 },
  { id: 'royal', name: 'Корона', price: 3000 },
] as const satisfies readonly CosmeticItem[]

export const AVATARS = [
  { id: 'initial', name: 'Буква', price: 0, note: 'первая буква ника' },
  { id: 'spade', name: 'Пики', price: 0 },
  { id: 'heart', name: 'Черви', price: 0 },
  { id: 'diamond', name: 'Бубны', price: 0 },
  { id: 'club', name: 'Трефы', price: 0 },
  { id: 'cat', name: 'Кот', price: 1000 },
  { id: 'dog', name: 'Пёс', price: 1000 },
  { id: 'rabbit', name: 'Кролик', price: 1000 },
  { id: 'fish', name: 'Рыба', price: 1000 },
  { id: 'bird', name: 'Птица', price: 1000 },
  { id: 'cactus', name: 'Кактус', price: 1500 },
  { id: 'ghost', name: 'Призрак', price: 1500 },
  { id: 'robot', name: 'Робот', price: 1500 },
  { id: 'alien', name: 'Пришелец', price: 1500 },
  { id: 'sunglasses', name: 'Очки', price: 1500 },
  { id: 'clover', name: 'Клевер', price: 2000 },
  { id: 'lightning', name: 'Молния', price: 2000 },
  { id: 'fire', name: 'Огонь', price: 2000 },
  { id: 'anchor', name: 'Якорь', price: 2000 },
  { id: 'rocket', name: 'Ракета', price: 2000 },
  { id: 'horse', name: 'Конь', price: 2500 },
  { id: 'sword', name: 'Меч', price: 2500 },
  { id: 'detective', name: 'Сыщик', price: 2500 },
  { id: 'skull', name: 'Череп', price: 2500 },
  { id: 'trophy', name: 'Кубок', price: 3000 },
  { id: 'crown', name: 'Корона', price: 3000 },
] as const satisfies readonly CosmeticItem[]

export type AvatarId = (typeof AVATARS)[number]['id']
export type ThemeId = (typeof THEMES)[number]['id']
export type FeltId = (typeof FELTS)[number]['id']
export type CardBackId = (typeof CARD_BACKS)[number]['id']

export const CATALOG: Record<CosmeticKind, readonly CosmeticItem[]> = {
  theme: THEMES,
  felt: FELTS,
  back: CARD_BACKS,
  avatar: AVATARS,
}

export function isKnown<T extends string>(items: readonly CosmeticItem<T>[], id: unknown): id is T {
  return items.some((item) => item.id === id)
}

export interface CatalogEntry extends CosmeticItem {
  kind: CosmeticKind
}

export const ALL_ITEMS: CatalogEntry[] = [
  ...AVATARS.map((i) => ({ ...i, kind: 'avatar' as const })),
  ...CARD_BACKS.map((i) => ({ ...i, kind: 'back' as const })),
  ...FELTS.map((i) => ({ ...i, kind: 'felt' as const })),
  ...THEMES.map((i) => ({ ...i, kind: 'theme' as const })),
]
