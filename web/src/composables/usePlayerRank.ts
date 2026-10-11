const MONTHS = ['января', 'февраля', 'марта', 'апреля', 'мая', 'июня', 'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря']

export function rankTitle(elo: number): string {
  if (elo >= 1600) return 'Мастер стола'
  if (elo >= 1300) return 'Опытный игрок'
  if (elo >= 1100) return 'Игрок'
  return 'Новичок'
}

export function memberSince(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return `${MONTHS[d.getMonth()]} ${d.getFullYear()}`
}

export function winRate(wins: number, losses: number): number | null {
  return wins + losses === 0 ? null : Math.round((wins / (wins + losses)) * 100)
}
