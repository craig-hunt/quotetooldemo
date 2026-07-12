import { useState } from 'react'
import { API, apiFetch } from '../api'
import { useResource } from '../hooks/useResource'
import {
  Button, Card, Empty, ErrorBanner, Field, Input, Loading, PageHeader,
  Select, StatusBadge, Textarea,
} from '../components/ui'
import { fmtDate, fmtMoney } from '../lib/format'
import { INVOICE_TRANSITIONS, InvoiceStatus, ORDER_STATUS } from '../lib/constants'

type Invoice = {
  id: string
  order_id: string
  customer_id: string
  invoice_number: string
  status: InvoiceStatus
  amount_due: number
  amount_paid: number
  due_date: string
  sent_at: string | null
  paid_date: string | null
  notes: string
  created_at: string
}

type Order = {
  id: string
  customer_id: string
  order_number: string
  status: string
  total_amount: number
}

type Customer = { id: string; name: string }

export default function Invoices() {
  const [showForm, setShowForm] = useState(false)
  const invoices = useResource<Invoice[]>(
    () => apiFetch<Invoice[]>(`${API.invoices}/invoices`)
  )
  const orders = useResource<Order[]>(
    () => apiFetch<Order[]>(`${API.orders}/orders`)
  )
  const customers = useResource<Customer[]>(
    () => apiFetch<Customer[]>(`${API.customers}/customers`)
  )

  const customerName = (id: string) =>
    customers.data?.find(c => c.id === id)?.name ?? id.slice(0, 8)
  const orderNumber = (id: string) =>
    orders.data?.find(o => o.id === id)?.order_number ?? id.slice(0, 8)

  const fulfilledOrders = (orders.data ?? []).filter(o => o.status === ORDER_STATUS.FULFILLED)

  async function transition(invoiceID: string, action: string) {
    try {
      await apiFetch(`${API.invoices}/invoices/${invoiceID}/${action}`, { method: 'POST' })
      invoices.reload()
    } catch (e) {
      alert(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div>
      <PageHeader
        title="Invoices"
        subtitle="Invoices billed against fulfilled orders"
        actions={
          <Button
            onClick={() => setShowForm(v => !v)}
            disabled={fulfilledOrders.length === 0 && !showForm}
          >
            {showForm ? 'Cancel' : 'New Invoice'}
          </Button>
        }
      />

      {invoices.error && <ErrorBanner message={invoices.error} />}

      {fulfilledOrders.length === 0 && !invoices.loading && (
        <div className="text-xs text-slate-500 mb-4">
          Invoices derive from fulfilled orders. Fulfill an order first, then invoice it.
        </div>
      )}

      {showForm && (
        <div className="mb-6">
          <NewInvoiceForm
            orders={fulfilledOrders}
            onCreated={() => { setShowForm(false); invoices.reload() }}
          />
        </div>
      )}

      {invoices.loading ? (
        <Loading />
      ) : !invoices.data || invoices.data.length === 0 ? (
        <Empty>No invoices yet.</Empty>
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left text-xs uppercase tracking-wide text-slate-500 border-b border-slate-800">
                <tr>
                  <th className="px-4 py-3">Invoice #</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3">Customer</th>
                  <th className="px-4 py-3">Order</th>
                  <th className="px-4 py-3 text-right">Amount Due</th>
                  <th className="px-4 py-3 text-right">Amount Paid</th>
                  <th className="px-4 py-3">Due Date</th>
                  <th className="px-4 py-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {invoices.data.map(inv => (
                  <tr key={inv.id} className="text-slate-200">
                    <td className="px-4 py-3 font-mono text-xs">{inv.invoice_number}</td>
                    <td className="px-4 py-3"><StatusBadge status={inv.status} /></td>
                    <td className="px-4 py-3">{customerName(inv.customer_id)}</td>
                    <td className="px-4 py-3 font-mono text-xs text-slate-400">
                      {orderNumber(inv.order_id)}
                    </td>
                    <td className="px-4 py-3 text-right font-medium">{fmtMoney(inv.amount_due)}</td>
                    <td className="px-4 py-3 text-right text-slate-400">{fmtMoney(inv.amount_paid)}</td>
                    <td className="px-4 py-3 text-slate-400">{fmtDate(inv.due_date)}</td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex justify-end gap-2">
                        {(INVOICE_TRANSITIONS[inv.status] ?? []).map(t => (
                          <Button
                            key={t.action}
                            variant="secondary"
                            onClick={() => transition(inv.id, t.action)}
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

function NewInvoiceForm({ orders, onCreated }: {
  orders: Order[]
  onCreated: () => void
}) {
  const [orderID, setOrderID] = useState(orders[0]?.id ?? '')
  const [dueDays, setDueDays] = useState('30')
  const [notes, setNotes] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!orderID) {
      setErr('Choose a fulfilled order')
      return
    }
    setSubmitting(true)
    setErr(null)
    try {
      await apiFetch(`${API.invoices}/invoices`, {
        method: 'POST',
        body: JSON.stringify({
          order_id: orderID,
          due_days: Number(dueDays),
          notes,
        }),
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
          <Field label="Fulfilled Order">
            <Select
              required
              value={orderID}
              onChange={e => setOrderID(e.target.value)}
            >
              {orders.length === 0 && <option value="">No fulfilled orders available</option>}
              {orders.map(o => (
                <option key={o.id} value={o.id}>
                  {o.order_number} · {fmtMoney(o.total_amount)}
                </option>
              ))}
            </Select>
          </Field>
          <Field label="Due Days" hint="Days until due (net terms)">
            <Input
              type="number" step="1" min="1"
              value={dueDays}
              onChange={e => setDueDays(e.target.value)}
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
          <Button type="submit" disabled={submitting || !orderID}>
            {submitting ? 'Saving…' : 'Create Invoice'}
          </Button>
        </div>
      </form>
    </Card>
  )
}
