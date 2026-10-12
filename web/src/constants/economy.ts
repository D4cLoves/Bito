export const MIN_STAKE = 100
export const STAKES = [100, 250, 500, 1000, 2500, 5000] as const
export const TABLE_FEE = 0.05
export const TOP_UP_THRESHOLD = 100
export const TOP_UP_TO = 500

export function placeShare(place: number, winners: number): number {
  return (winners - place + 1) / ((winners * (winners + 1)) / 2)
}

export function winnersPool(stake: number): number {
  return Math.floor(stake * (1 - TABLE_FEE))
}

export function bestPayout(stake: number, players: number): number {
  const winners = Math.max(1, players - 1)
  return Math.floor(winnersPool(stake) * placeShare(1, winners))
}

export function formatChips(n: number): string {
  if (n >= 1000 && n % 1000 === 0) return `${n / 1000}K`
  if (n >= 1000 && n % 500 === 0) return `${(n / 1000).toFixed(1)}K`
  return n.toLocaleString('ru-RU')
}
