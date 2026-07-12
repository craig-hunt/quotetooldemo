import { useState } from 'react'
import { API, apiFetch } from '../api'
import { useResource } from '../hooks/useResource'
import {
  Button, Card, Empty, ErrorBanner, Field, Input, Loading, PageHeader,
  Select, StatusBadge, Textarea,
} from '../components/ui'
import { fmtDate, fmtMoney } from '../lib/format'
import { ORDER_TRANSITIONS, OrderStatus, QUOTE_STATUS } from '../lib/constants'

type Order = {
  id: string
  quote_id: string
  customer_id: string
  order_number: string
  status: OrderStatus
  scheduled_date: string | null
  fulfilled_date: string | null
  total_amount: number
  notes: string
  created_at: string
}

type Quote = {
  id: string
  customer_id: string
  project_name: string
  status: string
  total: number
}

type Customer = { id: string; name: string }

export default function Orders() {
  const [showForm, setShowForm] = useState(false)
  const orders = useResource<Order[]>(
    () => apiFetch<Order[]>(`${API.orders}/orders`)
  )
  const quotes = useResource<Quote[]>(
    () => apiFetch<Quote[]>(`${API.quotes}/quotes`)
  )
  const customers = useResource<Customer[]>(
    () => apiFetch<Customer[]>(`${API.customers}/customers`)
  )

  const customerName = (id: string) =>
    customers.data?.find(c => c.id === id)?.name ?? id.slice(0, 8)
  const quoteName = (id: string) =>
    quotes.data?.find(q => q.id === id)?.project_name ?? id.slice(0, 8)

  const acceptedQuotes = (quotes.data ?? []).filter(q => q.status === QUOTE_STATUS.ACCEPTED)

  async function transition(orderID: string, action: string) {
    try {
      await apiFetch(`${API.orders}/orders/${orderID}/${action}`, { method: 'POST' })
      orders.reload()
    } catch (e) {
      alert(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div>
      <PageHeader
        title="Orders"
        subtitle="Sales orders converted from accepted quotes"
        actions={
          <Button
            onClick={() => setShowForm(v => !v)}
            disabled={acceptedQuotes.length === 0 && !showForm}
          >
            {showForm ? 'Cancel' : 'New Order'}
          </Button>
        }
      />

      {orders.error && <ErrorBanner message={orders.error} />}

      {acceptedQuotes.length === 0 && !orders.loading && (
        <div className="text-xs text-slate-500 mb-4">
          Orders derive from accepted quotes. Accept a quote first, then create an order from it.
        </div>
      )}

      {showForm && (
        <div className="mb-6">
          <NewOrderForm
            quotes={acceptedQuotes}
            onCreated={() => { setShowForm(false); orders.reload() }}
          />
        </div>
      )}

      {orders.loading ? (
        <Loading />
      ) : !orders.data || orders.data.length === 0 ? (
        <Empty>No orders yet.</Empty>
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left text-xs uppercase tracking-wide text-slate-500 border-b border-slate-800">
                <tr>
                  <th className="px-4 py-3">Order #</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Customer</th>
                  <th className="px-4 py-3">Quote</th>
                  <th className="px-4 py-3 text-right">Amount</th>
                  <th className="px-4 py-3">Scheduled</th>
                  <th className="px-4 py-3">Fulfilled</th>
                  <th className="px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {orders.data.map(o => (
                  <tr key={o.id} className="text-slate-200">
                    <td className="px-4 py-3 font-mono text-xs">{o.order_number}</td>
                    <td className="px-4 py-3"><StatusBadge status={o.status} /></td>
                    <td className="px-4 py-3">{customerName(o.customer_id)}</td>
                    <td className="px-4 py-3 text-slate-400">{quoteName(o.quote_id)}</td>
                    <td className="px-4 py-3 text-right font-medium">{fmtMoney(o.total_amount)}</td>
                    <td className="px-4 py-3 text-slate-400">{fmtDate(o.scheduled_date)}</td>
                    <td className="px-4 py-3 text-slate-400">{fmtDate(o.fulfilled_date)}</td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex justify-end gap-2">
                        {(ORDER_TRANSITIONS[o.status] ?? []).map(t => (
                          <Button
                            key={t.action}
                            variant="secondary"
                            onClick={() => transition(o.id, t.action)}
                          >
                            {t.label}
                          </Button>
                        ))}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  )
}

function NewOrderForm({ quotes, onCreated }: {
  quotes: Quote[]
  onCreated: () => void
}) {
  const [quoteID, setQuoteID] = useState(quotes[0]?.id ?? '')
  const [scheduledDate, setScheduledDate] = useState('')
  const [notes, setNotes] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!quoteID) {
      setErr('Choose an accepted quote')
      return
    }
    setSubmitting(true)
    setErr(null)
    try {
      const body: Record<string, unknown> = { quote_id: quoteID, notes }
      if (scheduledDate) {
        body.scheduled_date = new Date(scheduledDate).toISOString()
      }
      await apiFetch(`${API.orders}/orders`, {
        method: 'POST',
        body: JSON.stringify(body),
      })
      onCreated()
    } catch (e2) {
      setErr(e2 instanceof Error ? e2.message : String(e2))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Card className="p-6">
      <form onSubmit={submit} className="space-y-4">
        {err && <ErrorBanner message={err} />}
        <div className="grid md:grid-cols-2 gap-4">
          <Field label="Accepted Quote">
            <Select
              required
              value={quoteID}
              onChange={e => setQuoteID(e.target.value)}
            >
              {quotes.length === 0 && <option value="">No accepted quotes available</option>}
              {quotes.map(q => (
                <option key={q.id} value={q.id}>
                  {q.project_name} · {fmtMoney(q.total)}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Scheduled Date" hint="Optional">
            <Input
              type="date"
              value={scheduledDate}
              onChange={e => setScheduledDate(e.target.value)}
            />
          </Field>
        </div>
        <Field label="Notes">
          <Textarea
            rows={2}
            value={notes}
            onChange={e => setNotes(e.target.value)}
          />
        </Field>
        <div className="flex justify-end">
          <Button type="submit" disabled={submitting || !quoteID}>
            {submitting ? 'Saving…' : 'Create Order'}
          </Button>
        </div>
      </form>
    </Card>
  )
}
