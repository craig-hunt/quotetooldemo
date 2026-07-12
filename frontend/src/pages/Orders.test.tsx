import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter, stubFetch } from '../test-utils'
import Orders from './Orders'

afterEach(() => vi.unstubAllGlobals())

const customer = { id: '11111111-1111-1111-1111-111111111111', name: 'Blackstone Paving' }
const acceptedQuote = {
  id: 'q1', customer_id: customer.id, project_name: 'Repave', status: 'accepted', total: 5000,
}
const openOrder = {
  id: 'o1', quote_id: 'q1', customer_id: customer.id, order_number: 'SO-2026-0001',
  status: 'open', scheduled_date: null, fulfilled_date: null, total_amount: 5000,
  notes: '', created_at: '2026-07-12T10:00:00Z',
}

describe('Orders page', () => {
  it('renders the list with order number and status', async () => {
    stubFetch({
      'GET http://localhost:8083/orders': { status: 200, body: [openOrder] },
      'GET http://localhost:8082/quotes': { status: 200, body: [acceptedQuote] },
      'GET http://localhost:8081/customers': { status: 200, body: [customer] },
    })
    renderWithRouter(<Orders />)
    expect(await screen.findByText('SO-2026-0001')).toBeInTheDocument()
    expect(screen.getByText('open')).toBeInTheDocument()
  })

  it('fires the fulfill transition', async () => {
    const fn = stubFetch({
      'GET http://localhost:8083/orders': { status: 200, body: [openOrder] },
      'GET http://localhost:8082/quotes': { status: 200, body: [acceptedQuote] },
      'GET http://localhost:8081/customers': { status: 200, body: [customer] },
      'POST http://localhost:8083/orders/o1/fulfill': { status: 200, body: { ...openOrder, status: 'fulfilled' } },
    })
    renderWithRouter(<Orders />)
    fireEvent.click(await screen.findByRole('button', { name: /^Fulfill$/ }))
    await waitFor(() => expect(fn).toHaveBeenCalledWith(
      'http://localhost:8083/orders/o1/fulfill',
      expect.objectContaining({ method: 'POST' }),
    ))
  })

  it('disables New Order button when no accepted quotes exist', async () => {
    stubFetch({
      'GET http://localhost:8083/orders': { status: 200, body: [] },
      'GET http://localhost:8082/quotes': { status: 200, body: [{ ...acceptedQuote, status: 'draft' }] },
      'GET http://localhost:8081/customers': { status: 200, body: [customer] },
    })
    renderWithRouter(<Orders />)
    const btn = await screen.findByRole('button', { name: /New Order/ })
    expect(btn).toBeDisabled()
  })
})
