const BASE = '/api'

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const token = localStorage.getItem('token')
  const headers: Record<string, string> = {
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
  // Only set Content-Type for requests with a body
  if (options?.body) {
    headers['Content-Type'] = 'application/json'
  }
  const res = await fetch(`${BASE}${path}`, {
    ...options,
    headers: { ...headers, ...options?.headers },
  })
  // Only an authenticated (non-login) request getting 401 means the session expired.
  // A 401 from /auth/* is a failed login — let the real error ("invalid credentials") surface
  // instead of wiping state and showing a misleading "Session expired".
  if (res.status === 401 && !path.startsWith('/auth/')) {
    localStorage.removeItem('token')
    window.location.href = '/login'
    throw new Error('Session expired')
  }
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }))
    throw new Error(err.message || res.statusText)
  }
  const data = await res.json()
  return data ?? ([] as unknown as T)
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body: unknown) => request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  put: <T>(path: string, body: unknown) => request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  patch: <T>(path: string, body: unknown) => request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}
