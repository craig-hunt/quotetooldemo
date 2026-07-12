import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen, waitFor, fireEvent } from '@testing-library/react'
import { renderWithRouter, stubFetch } from '../test-utils'
import Customers from './Customers'

afterEach(() => vi.unstubAllGlobals())

const twoCustomers = [
  { id: '11111111-1111-1111-1111-111111111111', name: 'Blackstone Paving', contact_name: 'Ellie', email: 'e@x.com', phone: '555', billing_address: { street: '1', city: 'Wilmington', state: 'DE', zip: '19801' }, created_at: '2026-07-12T10:00:00Z', updated_at: '2026-07-12T10:00:00Z' },
  { id: '22222222-2222-2222-2222-222222222222', name: 'Ridgeline Contractors', contact_name: 'Marcus', email: 'm@x.com', phone: '555', billing_address: { street: '2', city: 'Reading', state: 'PA', zip: '19601' }, created_at: '2026-07-12T10:00:00Z', updated_at: '2026-07-12T10:00:00Z' },
]

describe('Customers page', () => {
  it('renders the list from the API', async () => {
    stubFetch({ 'GET http://localhost:8081/customers': { status: 200, body: twoCustomers } })
    renderWithRouter(<Customers />)
    expect(await screen.findByText('Blackstone Paving')).toBeInTheDocument()
    expect(screen.getByText('Ridgeline Contractors')).toBeInTheDocument()
  })

  it('shows empty state when no rows exist', async () => {
    stubFetch({ 'GET http://localhost:8081/customers': { status: 200, body: [] } })
    renderWithRouter(<Customers />)
    expect(await screen.findByText(/No customers yet/)).toBeInTheDocument()
  })

  it('surfaces API errors', async () => {
    stubFetch({ 'GET http://localhost:8081/customers': { status: 500, body: { error: 'db down' } } })
    renderWithRouter(<Customers />)
    expect(await screen.findByText('db down')).toBeInTheDocument()
  })

  it('POSTs a new customer and reloads', async () => {
    let created = false
    const fn = stubFetch({
      'GET http://localhost:8081/customers': () => ({ status: 200, body: created ? twoCustomers : [] }),
      'POST http://localhost:8081/customers': () => {
        created = true
        return { status: 201, body: twoCustomers[0] }
      },
    })
    renderWithRouter(<Customers />)
    fireEvent.click(await screen.findByRole('button', { name: /New Customer/ }))
    fireEvent.change(screen.getByRole('textbox', { name: /Company Name/ }), { target: { value: 'New Co' } })
    fireEvent.click(screen.getByRole('button', { name: /Create Customer/ }))
    await waitFor(() => expect(fn).toHaveBeenCalledWith(
      'http://localhost:8081/customers',
      expect.objectContaining({ method: 'POST' }),
    ))
    expect(await screen.findByText('Blackstone Paving')).toBeInTheDocument()
  })
})
