const MONTHS = ['января', 'февраля', 'марта', 'апреля', 'мая', 'июня', 'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря']

export function rankTitle(games: number): string {
  if (games >= 200) return 'Мастер стола'
  if (games >= 50) return 'Завсегдатай'
  if (games >= 10) return 'Игрок'
  return 'Новичок'
}

export function memberSince(iso?: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return `${d.getDate()} ${MONTHS[d.getMonth()]} ${d.getFullYear()}`
}

export function winRate(wins: number, losses: number): number | null {
  return wins + losses === 0 ? null : Math.round((wins / (wins + losses)) * 100)
}
