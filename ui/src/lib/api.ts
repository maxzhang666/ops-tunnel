import type { ApiErrorBody, AuthCheckResponse, LoginRequest } from '@/types/api'

export class ApiError extends Error {
  status: number
  body: ApiErrorBody

  constructor(status: number, body: ApiErrorBody) {
    super(body.error || `HTTP ${status}`)
    this.status = status
    this.body = body
  }
}

const BASE = '/api/v1'

async function request<T>(path: string, opts?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    ...opts,
    headers: {
      ...(opts?.body ? { 'Content-Type': 'application/json' } : {}),
      ...opts?.headers,
    },
  })

  if (!res.ok) {
    // Redirect to login on 401 (skip for auth endpoints themselves)
    if (res.status === 401 && !path.startsWith('/auth/')) {
      window.location.href = '/login'
      return new Promise(() => {}) // never resolves — page is navigating
    }
    let body: ApiErrorBody
    try {
      body = await res.json()
    } catch {
      body = { error: `HTTP ${res.status}` }
    }
    throw new ApiError(res.status, body)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, data: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(data) }),
  put: <T>(path: string, data: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(data) }),
  patch: <T>(path: string, data: unknown) =>
    request<T>(path, { method: 'PATCH', body: JSON.stringify(data) }),
  del: (path: string) => request<void>(path, { method: 'DELETE' }),
  authCheck: () => request<AuthCheckResponse>('/auth/check'),
  login: (data: LoginRequest) =>
    request<{ ok: boolean }>('/auth/login', { method: 'POST', body: JSON.stringify(data) }),
  logout: () =>
    request<{ ok: boolean }>('/auth/logout', { method: 'POST' }),
}
