import type { AuthResponse, LoginRequest, RegisterRequest } from '../types/auth'

export async function loginUser(data: LoginRequest): Promise<AuthResponse> {
  const res = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  })

  const body = await res.json()

  if (!res.ok) {
    throw new Error(body.error || 'Ошибка входа')
  }

  return body as AuthResponse
}

export async function registerUser(data: RegisterRequest): Promise<AuthResponse> {
  const res = await fetch('/api/v1/auth/register', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify(data),
  })

  const body = await res.json()

  if (!res.ok) {
    throw new Error(body.error || 'Ошибка регистрации')
  }

  return body as AuthResponse
}
