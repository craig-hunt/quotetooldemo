import { useEffect, useState } from 'react'
import { API, apiFetch } from '../api'
import { useResource } from '../hooks/useResource'
import {
  Button, Card, Empty, ErrorBanner, Field, Input, Loading, PageHeader,
  Select, StatusBadge, Textarea,
} from '../components/ui'
import { fmtDate, fmtMoney, fmtTons } from '../lib/format'
import {
  MIX_TYPES, MixType, QUOTE_STATUS, QUOTE_TRANSITIONS, QuoteStatus, computeTons,
} from '../lib/constants'

type Customer = {
  id: string
  name: string
}

type LineItem = {
  id?: string
  area_sqft: number
  depth_inches: number
  mix_type: MixType
  unit_price_per_ton: number
  tons: number
  line_total: number
}

type Quote = {
  id: string
  customer_id: string
  project_name: string
  project_address: string
  status: QuoteStatus
  subtotal: number
  tax_rate: number
  tax_amount: number
  markup_rate: number
  markup_amount: number
  total: number
  notes: string
  created_at: string
  line_items: LineItem[]
}

export default function Quotes() {
  const [showForm, setShowForm] = useState(false)
  const quotes = useResource<Quote[]>(
    () => apiFetch<Quote[]>(`${API.quotes}/quotes`)
  )
  const customers = useResource<Customer[]>(
    () => apiFetch<Customer[]>(`${API.customers}/customers`)
  )

  const customerName = (id: string) =>
    customers.data?.find(c => c.id === id)?.name ?? id.slice(0, 8)

  async function transition(quoteID: string, action: string) {
    try {
      await apiFetch(`${API.quotes}/quotes/${quoteID}/${action}`, { method: 'POST' })
      quotes.reload()
    } catch (e) {
      alert(e instanceof Error ? e.message : String(e))
    }
  }

  return (
    <div>
      <PageHeader
        title="Quotes"
        subtitle="Line-item pricing with tons and totals computed server-side"
        actions={
          <Button onClick={() => setShowForm(v => !v)}>
            {showForm ? 'Cancel' : 'New Quote'}
          </Button>
        }
      />

      {quotes.error && <ErrorBanner message={quotes.error} />}
      {customers.error && <ErrorBanner message={customers.error} />}

      {showForm && customers.data && (
        <div className="mb-6">
          <QuoteForm
            mode="create"
            customers={customers.data}
            onSaved={() => { setShowForm(false); quotes.reload() }}
          />
        </div>
      )}

      {quotes.loading ? (
        <Loading />
      ) : !quotes.data || quotes.data.length === 0 ? (
        <Empty>No quotes yet. Click <em>New Quote</em> to draft one.</Empty>
      ) : (
        <div className="space-y-4">
          {quotes.data.map(q => (
            <QuoteRow
              key={q.id}
              quoteHeader={q}
              customers={customers.data ?? []}
              customerName={customerName(q.customer_id)}
              onAction={a => transition(q.id, a)}
              onSaved={() => quotes.reload()}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function QuoteRow({ quoteHeader, customers, customerName, onAction, onSaved }: {
  quoteHeader: Quote
  customers: Customer[]
  customerName: string
  onAction: (action: string) => void
  onSaved: () => void
}) {
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState(false)
  const [detail, setDetail] = useState<Quote | null>(null)
  const [detailErr, setDetailErr] = useState<string | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const transitions = QUOTE_TRANSITIONS[quoteHeader.status] ?? []
  const canEdit = quoteHeader.status === QUOTE_STATUS.DRAFT

  async function loadDetail() {
    setDetailLoading(true)
    setDetailErr(null)
    try {
      const full = await apiFetch<Quote>(`${API.quotes}/quotes/${quoteHeader.id}`)
      setDetail(full)
    } catch (e) {
      setDetailErr(e instanceof Error ? e.message : String(e))
    } finally {
      setDetailLoading(false)
    }
  }

  useEffect(() => {
    if (open && !detail && !detailLoading) {
      loadDetail()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const quote = detail ?? quoteHeader

  return (
    <Card className="p-4">
      <div className="flex items-center justify-between gap-4 flex-wrap">
        <div className="flex items-center gap-3 flex-wrap">
          <StatusBadge status={quote.status} />
          <div>
            <div className="font-medium text-slate-100">{quote.project_name}</div>
            <div className="text-xs text-slate-500">
              {customerName} · {fmtDate(quote.created_at)}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-3 flex-wrap">
          <span className="text-slate-400 text-sm">Total</span>
          <span className="text-slate-100 font-semibold">{fmtMoney(quote.total)}</span>
          {transitions.map(t => (
            <Button key={t.action} variant="secondary" onClick={() => onAction(t.action)}>
              {t.label}
            </Button>
          ))}
          {canEdit && (
            <Button
              variant="secondary"
              onClick={async () => {
                if (!detail) await loadDetail()
                setOpen(true)
                setEditing(true)
              }}
            >
              Edit
            </Button>
          )}
          <Button variant="ghost" onClick={() => { setOpen(v => !v); setEditing(false) }}>
            {open ? 'Hide' : 'Details'}
          </Button>
        </div>
      </div>

      {open && (
        <div className="mt-4 border-t border-slate-800 pt-4">
          {detailErr && <ErrorBanner message={detailErr} />}
          {detailLoading && !detail ? (
            <Loading />
          ) : editing && detail ? (
            <QuoteForm
              mode="edit"
              customers={customers}
              existing={detail}
              onSaved={() => {
                setEditing(false)
                setDetail(null)
                onSaved()
              }}
              onCancel={() => setEditing(false)}
            />
          ) : (
            <>
              <div className="grid md:grid-cols-3 gap-3 text-sm mb-4">
                <Detail label="Project Address" value={quote.project_address || '—'} />
                <Detail label="Tax Rate" value={`${(quote.tax_rate * 100).toFixed(2)}%`} />
                <Detail label="Markup Rate" value={`${(quote.markup_rate * 100).toFixed(2)}%`} />
              </div>
              <div className="text-xs uppercase tracking-wide text-slate-500 mb-2">
                Line Items ({quote.line_items?.length ?? 0})
              </div>
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead className="text-left text-xs text-slate-500">
                    <tr>
                      <th className="py-2">Mix Type</th>
                      <th className="py-2">Area (sqft)</th>
                      <th className="py-2">Depth (in)</th>
                      <th className="py-2">Tons</th>
                      <th className="py-2">Unit Price</th>
                      <th className="py-2 text-right">Line Total</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-800/60">
                    {(quote.line_items ?? []).map((li, i) => (
                      <tr key={li.id ?? i} className="text-slate-300">
                        <td className="py-2 capitalize">{li.mix_type.replace(/_/g, ' ')}</td>
                        <td className="py-2">{li.area_sqft}</td>
                        <td className="py-2">{li.depth_inches}</td>
                        <td className="py-2">{fmtTons(li.tons)}</td>
                        <td className="py-2">{fmtMoney(li.unit_price_per_ton)}/ton</td>
                        <td className="py-2 text-right font-medium">{fmtMoney(li.line_total)}</td>
                      </tr>
                    ))}
                  </tbody>
                  <tfoot className="text-sm">
                    <tr className="text-slate-400">
                      <td colSpan={5} className="py-1 text-right">Subtotal</td>
                      <td className="py-1 text-right">{fmtMoney(quote.subtotal)}</td>
                    </tr>
                    <tr className="text-slate-400">
                      <td colSpan={5} className="py-1 text-right">Tax</td>
                      <td className="py-1 text-right">{fmtMoney(quote.tax_amount)}</td>
                    </tr>
                    <tr className="text-slate-400">
                      <td colSpan={5} className="py-1 text-right">Markup</td>
                      <td className="py-1 text-right">{fmtMoney(quote.markup_amount)}</td>
                    </tr>
                    <tr className="text-slate-100 font-semibold">
                      <td colSpan={5} className="py-1 text-right">Total</td>
                      <td className="py-1 text-right">{fmtMoney(quote.total)}</td>
                    </tr>
                  </tfoot>
                </table>
              </div>
              {quote.notes && (
                <div className="mt-3 text-sm text-slate-400">
                  <span className="text-xs uppercase tracking-wide text-slate-500">Notes:</span>{' '}
                  {quote.notes}
                </div>
              )}
            </>
          )}
        </div>
      )}
    </Card>
  )
}

function Detail({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div className="text-xs uppercase tracking-wide text-slate-500">{label}</div>
      <div className="text-slate-200">{value}</div>
    </div>
  )
}

type FormLine = {
  area_sqft: string
  depth_inches: string
  mix_type: MixType
  unit_price_per_ton: string
}

const BLANK_LINE: FormLine = {
  area_sqft: '',
  depth_inches: '',
  mix_type: 'hma_surface',
  unit_price_per_ton: '',
}

function linesFromQuote(q: Quote): FormLine[] {
  return q.line_items.map(li => ({
    area_sqft: String(li.area_sqft),
    depth_inches: String(li.depth_inches),
    mix_type: li.mix_type,
    unit_price_per_ton: String(li.unit_price_per_ton),
  }))
}

function QuoteForm({ mode, customers, existing, onSaved, onCancel }: {
  mode: 'create' | 'edit'
  customers: Customer[]
  existing?: Quote
  onSaved: () => void
  onCancel?: () => void
}) {
  const [customerID, setCustomerID] = useState(existing?.customer_id ?? customers[0]?.id ?? '')
  const [projectName, setProjectName] = useState(existing?.project_name ?? '')
  const [projectAddress, setProjectAddress] = useState(existing?.project_address ?? '')
  const [taxRate, setTaxRate] = useState(String(existing?.tax_rate ?? 0.06))
  const [markupRate, setMarkupRate] = useState(String(existing?.markup_rate ?? 0.15))
  const [notes, setNotes] = useState(existing?.notes ?? '')
  const [lines, setLines] = useState<FormLine[]>(
    existing ? linesFromQuote(existing) : [BLANK_LINE]
  )
  const [submitting, setSubmitting] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  const previewTotals = (() => {
    let subtotal = 0
    lines.forEach(l => {
      const area = Number(l.area_sqft)
      const depth = Number(l.depth_inches)
      const price = Number(l.unit_price_per_ton)
      const tons = computeTons(area, depth)
      subtotal += tons * price
    })
    const tax = subtotal * Number(taxRate)
    const markup = (subtotal + tax) * Number(markupRate)
    return { subtotal, tax, markup, total: subtotal + tax + markup }
  })()

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setErr(null)
    try {
      const url = mode === 'edit' && existing
        ? `${API.quotes}/quotes/${existing.id}`
        : `${API.quotes}/quotes`
      await apiFetch(url, {
        method: mode === 'edit' ? 'PUT' : 'POST',
        body: JSON.stringify({
          customer_id: customerID,
          project_name: projectName,
          project_address: projectAddress,
          tax_rate: Number(taxRate),
          markup_rate: Number(markupRate),
          notes,
          line_items: lines.map(l => ({
            area_sqft: Number(l.area_sqft),
            depth_inches: Number(l.depth_inches),
            mix_type: l.mix_type,
            unit_price_per_ton: Number(l.unit_price_per_ton),
          })),
        }),
      })
      onSaved()
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
          <Field label="Customer">
            <Select
              required
              value={customerID}
              onChange={e => setCustomerID(e.target.value)}
            >
              {customers.map(c => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </Select>
          </Field>
          <Field label="Project Name">
            <Input
              required
              value={projectName}
              onChange={e => setProjectName(e.target.value)}
            />
          </Field>
          <Field label="Project Address">
            <Input
              value={projectAddress}
              onChange={e => setProjectAddress(e.target.value)}
            />
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field label="Tax Rate" hint="e.g. 0.06 for 6%">
              <Input
                type="number" step="0.01"
                value={taxRate}
                onChange={e => setTaxRate(e.target.value)}
              />
            </Field>
            <Field label="Markup Rate" hint="e.g. 0.15 for 15%">
              <Input
                type="number" step="0.01"
                value={markupRate}
                onChange={e => setMarkupRate(e.target.value)}
              />
            </Field>
          </div>
        </div>

        <div className="border-t border-slate-800 pt-4">
          <div className="flex items-center justify-between mb-3">
            <span className="text-xs uppercase tracking-wide text-slate-400">Line Items</span>
            <Button
              type="button"
              variant="secondary"
              onClick={() => setLines([...lines, { ...BLANK_LINE }])}
            >
              Add Line
            </Button>
          </div>
          {lines.map((line, i) => {
            const tons = computeTons(Number(line.area_sqft), Number(line.depth_inches))
            const lineTotal = tons * Number(line.unit_price_per_ton || 0)
            return (
              <div key={i} className="grid md:grid-cols-6 gap-3 mb-3 items-end">
                <Field label="Mix Type">
                  <Select
                    value={line.mix_type}
                    onChange={e => {
                      const next = [...lines]
                      next[i] = { ...line, mix_type: e.target.value as MixType }
                      setLines(next)
                    }}
                  >
                    {MIX_TYPES.map(m => (
                      <option key={m.value} value={m.value}>{m.label}</option>
                    ))}
                  </Select>
                </Field>
                <Field label="Area (sqft)">
                  <Input
                    type="number" step="1" required
                    value={line.area_sqft}
                    onChange={e => {
                      const next = [...lines]
                      next[i] = { ...line, area_sqft: e.target.value }
                      setLines(next)
                    }}
                  />
                </Field>
                <Field label="Depth (in)">
                  <Input
                    type="number" step="0.25" required
                    value={line.depth_inches}
                    onChange={e => {
                      const next = [...lines]
                      next[i] = { ...line, depth_inches: e.target.value }
                      setLines(next)
                    }}
                  />
                </Field>
                <Field label="$/Ton">
                  <Input
                    type="number" step="0.01" required
                    value={line.unit_price_per_ton}
                    onChange={e => {
                      const next = [...lines]
                      next[i] = { ...line, unit_price_per_ton: e.target.value }
                      setLines(next)
                    }}
                  />
                </Field>
                <div className="text-sm">
                  <div className="text-xs uppercase tracking-wide text-slate-500">Preview</div>
                  <div className="text-slate-200">{fmtTons(tons)}</div>
                  <div className="text-slate-400">{fmtMoney(lineTotal)}</div>
                </div>
                <div>
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => setLines(lines.filter((_, j) => j !== i))}
                    disabled={lines.length === 1}
                  >
                    Remove
                  </Button>
                </div>
              </div>
            )
          })}
        </div>

        <Field label="Notes">
          <Textarea
            rows={2}
            value={notes}
            onChange={e => setNotes(e.target.value)}
          />
        </Field>

        <div className="border-t border-slate-800 pt-4 flex items-end justify-between">
          <div className="text-sm">
            <div className="text-xs uppercase tracking-wide text-slate-500">Preview</div>
            <div className="text-slate-400">Subtotal {fmtMoney(previewTotals.subtotal)} · Tax {fmtMoney(previewTotals.tax)} · Markup {fmtMoney(previewTotals.markup)}</div>
            <div className="text-slate-100 font-semibold text-lg">Total {fmtMoney(previewTotals.total)}</div>
          </div>
          <div className="flex gap-2">
            {onCancel && (
              <Button type="button" variant="ghost" onClick={onCancel}>
                Cancel
              </Button>
            )}
            <Button type="submit" disabled={submitting}>
              {submitting ? 'Saving…' : mode === 'edit' ? 'Save Changes' : 'Create Quote'}
            </Button>
          </div>
        </div>
      </form>
    </Card>
  )
}
