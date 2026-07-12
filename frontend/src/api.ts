// Service base URLs. Read from env vars at build time so we can point at
// localhost in dev and Fly.io URLs in production without code changes.

const env = import.meta.env

export const API = {
  customers: env.VITE_CUSTOMERS_URL ?? 'http://localhost:8081',
  quotes:    env.VITE_QUOTES_URL    ?? 'http://localhost:8082',
  orders:    env.VITE_ORDERS_URL    ?? 'http://localhost:8083',
  invoices:  env.VITE_INVOICES_URL  ?? 'http://localhost:8084',
  reports:   env.VITE_REPORTS_URL   ?? 'http://localhost:8085',
}

export async function apiFetch<T>(url: string, options: RequestInit = {}): Promise<T> {
  const resp = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
    ...options,
  })
  if (!resp.ok) {
    let msg = `HTTP ${resp.status}`
    try {
      const body = await resp.json()
      if (body?.error) msg = body.error
    } catch {
      // ignore parse error
    }
    throw new Error(msg)
  }
  if (resp.status === 204) return undefined as T
  return resp.json() as Promise<T>
}
