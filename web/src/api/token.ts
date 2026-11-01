export function tokenExpiresAt(token: string): number {
  try {
    const payload = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')))
    return typeof payload.exp === 'number' ? payload.exp * 1000 : 0
  } catch {
    return 0
  }
}

export function expiresSoon(token: string): boolean {
  return tokenExpiresAt(token) - Date.now() < 60_000
}
