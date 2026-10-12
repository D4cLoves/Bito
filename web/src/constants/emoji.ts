export interface EmojiEntry {
  char: string
  code: string
  label: string
}

export const EMOJI: EmojiEntry[] = [
  { char: '😂', code: '1f602', label: 'смех' },
  { char: '👍', code: '1f44d', label: 'класс' },
  { char: '👏', code: '1f44f', label: 'браво' },
  { char: '🔥', code: '1f525', label: 'огонь' },
  { char: '😎', code: '1f60e', label: 'круто' },
  { char: '😮', code: '1f62e', label: 'ого' },
  { char: '🤔', code: '1f914', label: 'хм' },
  { char: '🙈', code: '1f648', label: 'не смотрю' },
  { char: '😭', code: '1f62d', label: 'плачу' },
  { char: '😡', code: '1f621', label: 'злюсь' },
  { char: '🤡', code: '1f921', label: 'клоун' },
  { char: '🤝', code: '1f91d', label: 'мир' },
]

export const QUICK_PHRASES = ['Привет!', 'Удачи!', 'Хорош!', 'Ну ты дал', 'Ой…', 'Спасибо за игру']

const BY_CHAR = new Map(EMOJI.map((e) => [e.char, e]))

const NOTO = 'https://fonts.gstatic.com/s/e/notoemoji/latest'

export function animatedEmojiUrl(code: string) {
  return `${NOTO}/${code}/512.webp`
}

export function staticEmojiUrl(code: string) {
  return `${NOTO}/${code}/emoji.svg`
}

export function emojiOf(text: string): EmojiEntry | null {
  return BY_CHAR.get(text.trim()) ?? null
}
