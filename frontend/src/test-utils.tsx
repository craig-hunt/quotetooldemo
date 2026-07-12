import type { ReactElement } from 'react'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { vi } from 'vitest'

export function renderWithRouter(ui: ReactElement, initialEntries: string[] = ['/']) {
  return render(<MemoryRouter initialEntries={initialEntries}>{ui}</MemoryRouter>)
}

type FetchResp = { status: number; body: unknown }

export function stubFetch(routes: Record<string, FetchResp | ((init?: RequestInit) => FetchResp)>) {
  const fn = vi.fn(async (url: string | URL, init?: RequestInit) => {
    const u = typeof url === 'string' ? url : url.toString()
    const method = init?.method ?? 'GET'
    const key = `${method} ${u}`
    let handler = routes[key]
    if (!handler) {
      const pathKey = Object.keys(routes).find(k => k.startsWith(method + ' ') && u.endsWith(k.slice(method.length + 1)))
      if (pathKey) handler = routes[pathKey]
    }
    if (!handler) {
      return new Response(JSON.stringify({ error: `unrouted: ${key}` }), { status: 404 })
    }
    const resolved = typeof handler === 'function' ? handler(init) : handler
    return new Response(JSON.stringify(resolved.body), {
      status: resolved.status,
      headers: { 'Content-Type': 'application/json' },
    })
  })
  vi.stubGlobal('fetch', fn)
  return fn
}
