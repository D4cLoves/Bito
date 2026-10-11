import { ref } from 'vue'

export type ToastTone = 'info' | 'success' | 'error'

export interface Toast {
  id: number
  text: string
  tone: ToastTone
}

const toasts = ref<Toast[]>([])
let nextId = 1

export function useToast() {
  function show(text: string, tone: ToastTone = 'info', ms = 2800) {
    const id = nextId++
    toasts.value = [...toasts.value, { id, text, tone }]
    window.setTimeout(() => dismiss(id), ms)
  }

  function dismiss(id: number) {
    toasts.value = toasts.value.filter((t) => t.id !== id)
  }

  return { toasts, show, dismiss }
}
