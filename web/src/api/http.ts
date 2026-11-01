import { useAuthStore } from '../stores/auth'
import { ApiError } from './errors'
import { expiresSoon } from './token'

export { ApiError }

async function send(url: string, init: RequestInit, token: string | null): Promise<Response> {
  const headers = new Headers(init.headers)
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (init.body) headers.set('Content-Type', 'application/json')
  return fetch(url, { ...init, headers })
}

export async function authedFetch<T>(url: string, init: RequestInit = {}): Promise<T> {
  const auth = useAuthStore()

  let res: Response
  try {
    res = await send(url, init, auth.token)
    if (res.status === 401 && (await auth.refreshSession())) {
      res = await send(url, init, auth.token)
    }
  } catch {
    throw new ApiError('Сервер не отвечает. Проверьте, что бэкенд запущен.', 0)
  }

  const text = await res.text()
  let body: unknown = null
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      body = null
    }
  }

  if (!res.ok) {
    const message =
      body && typeof body === 'object' && 'error' in body ? String((body as { error: unknown }).error) : `Ошибка ${res.status}`
    throw new ApiError(message, res.status)
  }

  return body as T
}

export async function freshAccessToken(): Promise<string | null> {
  const auth = useAuthStore()
  if (!auth.token) return null
  if (expiresSoon(auth.token)) {
    await auth.refreshSession()
  }
  return auth.token
}

export function isNotImplemented(err: unknown): boolean {
  return err instanceof ApiError && (err.status === 404 || err.status === 405 || err.status === 501)
}
