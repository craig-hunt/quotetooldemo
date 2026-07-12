import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen, fireEvent, waitFor } from '@testing-library/react'
import { renderWithRouter, stubFetch } from '../test-utils'
import Invoices from './Invoices'

afterEach(() => vi.unstubAllGlobals())

const customer = { id: '11111111-1111-1111-1111-111111111111', name: 'Blackstone Paving' }
const fulfilledOrder = {
  id: 'o1', customer_id: customer.id, order_number: 'SO-2026-0001', status: 'fulfilled', total_amount: 5000,
}
const sentInvoice = {
  id: 'i1', order_id: 'o1', customer_id: customer.id, invoice_number: 'INV-2026-0001',
  status: 'sent', amount_due: 5000, amount_paid: 0, due_date: '2026-08-11', sent_at: '2026-07-12T10:00:00Z',
  paid_date: null, notes: '', created_at: '2026-07-12T10:00:00Z',
}

describe('Invoices page', () => {
  it('renders the list with invoice number and status', async () => {
    stubFetch({
      'GET http://localhost:8084/invoices': { status: 200, body: [sentInvoice] },
      'GET http://localhost:8083/orders': { status: 200, body: [fulfilledOrder] },
      'GET http://localhost:8081/customers': { status: 200, body: [customer] },
    })
    renderWithRouter(<Invoices />)
    expect(await screen.findByText('INV-2026-0001')).toBeInTheDocument()
    expect(screen.getByText('sent')).toBeInTheDocument()
  })

  it('fires the mark_paid transition', async () => {
    const fn = stubFetch({
      'GET http://localhost:8084/invoices': { status: 200, body: [sentInvoice] },
      'GET http://localhost:8083/orders': { status: 200, body: [fulfilledOrder] },
      'GET http://localhost:8081/customers': { status: 200, body: [customer] },
      'POST http://localhost:8084/invoices/i1/mark_paid': { status: 200, body: { ...sentInvoice, status: 'paid' } },
    })
    renderWithRouter(<Invoices />)
    fireEvent.click(await screen.findByRole('button', { name: /Mark Paid/ }))
    await waitFor(() => expect(fn).toHaveBeenCalledWith(
      'http://localhost:8084/invoices/i1/mark_paid',
      expect.objectContaining({ method: 'POST' }),
    ))
  })

  it('disables New Invoice when no fulfilled orders exist', async () => {
    stubFetch({
      'GET http://localhost:8084/invoices': { status: 200, body: [] },
      'GET http://localhost:8083/orders': { status: 200, body: [{ ...fulfilledOrder, status: 'open' }] },
      'GET http://localhost:8081/customers': { status: 200, body: [customer] },
    })
    renderWithRouter(<Invoices />)
    const btn = await screen.findByRole('button', { name: /New Invoice/ })
    expect(btn).toBeDisabled()
  })
})
