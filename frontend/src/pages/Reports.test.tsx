import { afterEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithRouter, stubFetch } from '../test-utils'
import Reports from './Reports'

afterEach(() => vi.unstubAllGlobals())

describe('Reports page', () => {
  it('renders all four cards from the reports API', async () => {
    stubFetch({
      'GET http://localhost:8085/reports/quote-to-cash': {
        status: 200,
        body: { stages: [
          { stage: 'quotes_created', count: 5, amount_total: 25000 },
          { stage: 'invoices_paid', count: 2, amount_total: 8000 },
        ]},
      },
      'GET http://localhost:8085/reports/cycle-time': {
        status: 200,
        body: {
          quote_created_to_accepted_days: 3.5,
          quote_accepted_to_order_days: 1.2,
          order_created_to_fulfilled_days: 7.1,
          order_fulfilled_to_paid_days: 25.4,
        },
      },
      'GET http://localhost:8085/reports/aging': {
        status: 200,
        body: [
          { bucket: 'paid', count: 2, amount_due: 0 },
          { bucket: '1-30_days_overdue', count: 1, amount_due: 500 },
        ],
      },
      'GET http://localhost:8085/reports/mix-breakdown': {
        status: 200,
        body: [
          { mix_type: 'hma_surface', tons: 100, revenue: 10000, line_items: 3 },
          { mix_type: 'hma_base', tons: 80, revenue: 7040, line_items: 2 },
        ],
      },
    })

    renderWithRouter(<Reports />)

    expect(await screen.findByText('Quote-to-Cash Funnel')).toBeInTheDocument()
    expect(screen.getByText('Cycle Time (Days)')).toBeInTheDocument()
    expect(screen.getByText('Invoice Aging')).toBeInTheDocument()
    expect(screen.getByText('Mix Breakdown')).toBeInTheDocument()

    expect(await screen.findByText(/quotes created/i)).toBeInTheDocument()
    expect(await screen.findByText(/hma surface/i)).toBeInTheDocument()
  })

  it('surfaces an error banner when any report fails', async () => {
    stubFetch({
      'GET http://localhost:8085/reports/quote-to-cash': { status: 500, body: { error: 'boom' } },
      'GET http://localhost:8085/reports/cycle-time': { status: 200, body: {} },
      'GET http://localhost:8085/reports/aging': { status: 200, body: [] },
      'GET http://localhost:8085/reports/mix-breakdown': { status: 200, body: [] },
    })
    renderWithRouter(<Reports />)
    expect(await screen.findByText(/One or more reports failed to load/)).toBeInTheDocument()
  })
})
