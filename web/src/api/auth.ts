import type { AuthResponse, LoginRequest, RegisterRequest, VerifyCodeRequest } from '../types/auth'

async function safeFetchJSON<T>(url: string, payload: unknown, defaultError: string): Promise<T> {
  let res: Response
  try {
    res = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    })
  } catch (err) {
    throw new Error('Не удалось связаться с сервером. Убедитесь, что бэкенд запущен.')
  }

  const rawText = await res.text()
  let body: any = null

  if (rawText && rawText.trim().length > 0) {
    try {
      body = JSON.parse(rawText)
    } catch {
      // Body is not JSON (e.g. proxy HTML error or plain text 502/504)
    }
  }

  if (!res.ok) {
    if (body && typeof body === 'object' && body.error) {
      throw new Error(body.error)
    }
    if (res.status === 502 || res.status === 504) {
      throw new Error('Бэкенд-сервер недоступен (502/504 Gateway). Запустите Go-сервер на порту 8080.')
    }
    if (res.status === 404) {
      throw new Error(`Роут не найден (404 ${url}).`)
    }
    throw new Error(defaultError + (res.status ? ` (HTTP ${res.status})` : ''))
  }

  if (!body) {
    throw new Error('Пустой ответ от сервера.')
  }

  return body as T
}

export async function loginUser(data: LoginRequest): Promise<AuthResponse> {
  return safeFetchJSON<AuthResponse>('/api/v1/auth/login', data, 'Ошибка входа')
}

export async function sendVerificationCode(data: RegisterRequest): Promise<{ message: string }> {
  return safeFetchJSON<{ message: string }>('/api/v1/auth/send-code', data, 'Ошибка отправки кода подтверждения')
}

export async function verifyCodeAndRegister(data: VerifyCodeRequest): Promise<AuthResponse> {
  return safeFetchJSON<AuthResponse>('/api/v1/auth/verify-code', data, 'Ошибка подтверждения кода')
}

export async function refreshAccessToken(refreshToken?: string): Promise<AuthResponse> {
  return safeFetchJSON<AuthResponse>(
    '/api/v1/auth/refresh',
    { refreshToken: refreshToken || '' },
    'Ошибка обновления сессии'
  )
}

export async function registerUser(data: VerifyCodeRequest): Promise<AuthResponse> {
  return verifyCodeAndRegister(data)
}



