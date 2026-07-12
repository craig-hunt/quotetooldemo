import { useState } from 'react'
import { API, apiFetch } from '../api'
import { useResource } from '../hooks/useResource'
import {
  Button, Card, Empty, ErrorBanner, Field, Input, Loading, PageHeader,
} from '../components/ui'
import { fmtDateTime } from '../lib/format'

type Address = {
  street: string
  city: string
  state: string
  zip: string
}

type Customer = {
  id: string
  name: string
  contact_name: string
  email: string
  phone: string
  billing_address: Address
  created_at: string
  updated_at: string
}

const EMPTY_INPUT = {
  name: '',
  contact_name: '',
  email: '',
  phone: '',
  billing_address: { street: '', city: '', state: '', zip: '' },
}

export default function Customers() {
  const [showForm, setShowForm] = useState(false)
  const { data, error, loading, reload } = useResource<Customer[]>(
    () => apiFetch<Customer[]>(`${API.customers}/customers`)
  )

  return (
    <div>
      <PageHeader
        title="Customers"
        subtitle="Master data for every account the shop invoices"
        actions={
          <Button onClick={() => setShowForm(v => !v)}>
            {showForm ? 'Cancel' : 'New Customer'}
          </Button>
        }
      />

      {error && <ErrorBanner message={error} />}

      {showForm && (
        <div className="mb-6">
          <NewCustomerForm
            onCreated={() => { setShowForm(false); reload() }}
          />
        </div>
      )}

      {loading ? (
        <Loading />
      ) : !data || data.length === 0 ? (
        <Empty>No customers yet. Click <em>New Customer</em> to add one.</Empty>
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left text-xs uppercase tracking-wide text-slate-500 border-b border-slate-800">
                <tr>
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">Contact</th>
                  <th className="px-4 py-3">Email</th>
                  <th className="px-4 py-3">Phone</th>
                  <th className="px-4 py-3">Address</th>
                  <th className="px-4 py-3">Created</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800">
                {data.map(c => (
                  <tr key={c.id} className="text-slate-200">
                    <td className="px-4 py-3 font-medium">{c.name}</td>
                    <td className="px-4 py-3">{c.contact_name || '—'}</td>
                    <td className="px-4 py-3">{c.email || '—'}</td>
                    <td className="px-4 py-3">{c.phone || '—'}</td>
                    <td className="px-4 py-3 text-slate-400">
                      {c.billing_address.street
                        ? `${c.billing_address.city}, ${c.billing_address.state}`
                        : '—'}
                    </td>
                    <td className="px-4 py-3 text-slate-500 text-xs">
                      {fmtDateTime(c.created_at)}
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

function NewCustomerForm({ onCreated }: { onCreated: () => void }) {
  const [form, setForm] = useState(EMPTY_INPUT)
  const [submitting, setSubmitting] = useState(false)
  const [err, setErr] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    setSubmitting(true)
    setErr(null)
    try {
      await apiFetch(`${API.customers}/customers`, {
        method: 'POST',
        body: JSON.stringify(form),
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
          <Field label="Company Name">
            <Input
              required
              value={form.name}
              onChange={e => setForm({ ...form, name: e.target.value })}
            />
          </Field>
          <Field label="Contact Name">
            <Input
              value={form.contact_name}
              onChange={e => setForm({ ...form, contact_name: e.target.value })}
            />
          </Field>
          <Field label="Email">
            <Input
              type="email"
              value={form.email}
              onChange={e => setForm({ ...form, email: e.target.value })}
            />
          </Field>
          <Field label="Phone">
            <Input
              value={form.phone}
              onChange={e => setForm({ ...form, phone: e.target.value })}
            />
          </Field>
        </div>
        <div className="grid md:grid-cols-4 gap-4">
          <Field label="Street">
            <Input
              value={form.billing_address.street}
              onChange={e => setForm({
                ...form,
                billing_address: { ...form.billing_address, street: e.target.value },
              })}
            />
          </Field>
          <Field label="City">
            <Input
              value={form.billing_address.city}
              onChange={e => setForm({
                ...form,
                billing_address: { ...form.billing_address, city: e.target.value },
              })}
            />
          </Field>
          <Field label="State">
            <Input
              value={form.billing_address.state}
              onChange={e => setForm({
                ...form,
                billing_address: { ...form.billing_address, state: e.target.value },
              })}
            />
          </Field>
          <Field label="ZIP">
            <Input
              value={form.billing_address.zip}
              onChange={e => setForm({
                ...form,
                billing_address: { ...form.billing_address, zip: e.target.value },
              })}
            />
          </Field>
        </div>
        <div className="flex justify-end">
          <Button type="submit" disabled={submitting}>
            {submitting ? 'Saving…' : 'Create Customer'}
          </Button>
        </div>
      </form>
    </Card>
  )
}
