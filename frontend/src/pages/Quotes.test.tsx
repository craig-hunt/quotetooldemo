import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter, stubFetch } from '../test-utils'
import Quotes from './Quotes'

afterEach(() => vi.unstubAllGlobals())

const customer = { id: '11111111-1111-1111-1111-111111111111', name: 'Blackstone Paving' }
const draftQuote = {
  id: 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
  customer_id: customer.id,
  project_name: 'Warehouse Lot Repave',
  project_address: '1140 Industrial Way',
  status: 'draft',
  subtotal: 2406.25,
  tax_rate: 0.06,
  tax_amount: 144.38,
  markup_rate: 0.15,
  markup_amount: 382.59,
  total: 2933.22,
  notes: '',
  created_at: '2026-07-12T10:00:00Z',
  line_items: [
    { id: 'l1', area_sqft: 500, depth_inches: 2, mix_type: 'hma_surface', unit_price_per_ton: 100, tons: 6.25, line_total: 625 },
  ],
}

describe('Quotes page', () => {
  it('renders the list with status badge and total', async () => {
    stubFetch({
      'GET http://localhost:8082/quotes': { status: 200, body: [draftQuote] },
      'GET http://localhost:8081/customers': { status: 200, body: [customer] },
    })
    renderWithRouter(<Quotes />)
    expect(await screen.findByText('Warehouse Lot Repave')).toBeInTheDocument()
    expect(screen.getAllByText('draft').length).toBeGreaterThan(0)
    expect(screen.getByText('$2,933.22')).toBeInTheDocument()
  })

  it('fires the send transition and reloads', async () => {
    let sent = false
    const fn = stubFetch({
      'GET http://localhost:8082/quotes': () => ({
        status: 200,
        body: [{ ...draftQuote, status: sent ? 'sent' : 'draft' }],
      }),
      'GET http://localhost:8081/customers': { status: 200, body: [customer] },
      'POST http://localhost:8082/quotes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/send': () => {
        sent = true
        return { status: 200, body: { ...draftQuote, status: 'sent' } }
      },
    })
    renderWithRouter(<Quotes />)
    fireEvent.click(await screen.findByRole('button', { name: /^Send$/ }))
    await waitFor(() => expect(fn).toHaveBeenCalledWith(
      'http://localhost:8082/quotes/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa/send',
      expect.objectContaining({ method: 'POST' }),
    ))
  })

  it('shows empty state', async () => {
    stubFetch({
      'GET http://localhost:8082/quotes': { status: 200, body: [] },
      'GET http://localhost:8081/customers': { status: 200, body: [customer] },
    })
    renderWithRouter(<Quotes />)
    expect(await screen.findByText(/No quotes yet/)).toBeInTheDocument()
  })
})
