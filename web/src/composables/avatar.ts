export function avatarHue(name: string) {
  let h = 0
  for (const ch of name) h = (h * 31 + (ch.codePointAt(0) ?? 0)) % 360
  return h
}

export const avatarInitial = (name: string) => (name[0] ?? '?').toUpperCase()
