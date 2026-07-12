const UPSTREAM = {
  customers: 'https://quotedemo-customers.fly.dev',
  quotes:    'https://quotedemo-quotes.fly.dev',
  orders:    'https://quotedemo-orders.fly.dev',
  invoices:  'https://quotedemo-invoices.fly.dev',
  reports:   'https://quotedemo-reports.fly.dev',
}

const CORS_HEADERS = {
  'Access-Control-Allow-Origin': '*',
  'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
  'Access-Control-Allow-Headers': 'Content-Type',
  'Access-Control-Max-Age': '86400',
}

export default {
  async fetch(request) {
    if (request.method === 'OPTIONS') {
      return new Response(null, { status: 204, headers: CORS_HEADERS })
    }

    const url = new URL(request.url)
    const match = url.pathname.match(/^\/api\/([^\/]+)(\/.*)?$/)
    if (!match) {
      return jsonError(404, 'route not matched: expected /api/<service>/<path>')
    }

    const [, service, rest] = match
    const upstream = UPSTREAM[service]
    if (!upstream) {
      return jsonError(404, `unknown service: ${service}`)
    }

    const targetUrl = upstream + (rest || '/') + (url.search || '')

    let upstreamResp
    try {
      upstreamResp = await fetch(new Request(targetUrl, request))
    } catch (err) {
      return jsonError(502, `upstream fetch failed: ${err.message}`)
    }

    const respHeaders = new Headers(upstreamResp.headers)
    for (const [k, v] of Object.entries(CORS_HEADERS)) {
      respHeaders.set(k, v)
    }
    return new Response(upstreamResp.body, {
      status: upstreamResp.status,
      statusText: upstreamResp.statusText,
      headers: respHeaders,
    })
  },
}

function jsonError(status, message) {
  return new Response(JSON.stringify({ error: message }), {
    status,
    headers: {
      'Content-Type': 'application/json',
      ...CORS_HEADERS,
    },
  })
}
