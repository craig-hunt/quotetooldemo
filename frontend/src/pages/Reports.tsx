import { API, apiFetch } from '../api'
import { useResource } from '../hooks/useResource'
import {
  Card, ErrorBanner, Loading, PageHeader, StatusBadge,
} from '../components/ui'
import { fmtMoney, fmtNumber } from '../lib/format'

type FunnelStage = {
  stage: string
  count: number
  amount_total: number
}

type QuoteToCash = { stages: FunnelStage[] }

type CycleTimes = {
  quote_created_to_accepted_days: number
  quote_accepted_to_order_days: number
  order_created_to_fulfilled_days: number
  order_fulfilled_to_paid_days: number
}

type AgingBucket = {
  bucket: string
  count: number
  amount_due: number
}

type MixBreakdown = {
  mix_type: string
  tons: number
  revenue: number
  line_items: number
}

export default function Reports() {
  const funnel = useResource<QuoteToCash>(
    () => apiFetch<QuoteToCash>(`${API.reports}/reports/quote-to-cash`)
  )
  const cycle = useResource<CycleTimes>(
    () => apiFetch<CycleTimes>(`${API.reports}/reports/cycle-time`)
  )
  const aging = useResource<AgingBucket[]>(
    () => apiFetch<AgingBucket[]>(`${API.reports}/reports/aging`)
  )
  const mix = useResource<MixBreakdown[]>(
    () => apiFetch<MixBreakdown[]>(`${API.reports}/reports/mix-breakdown`)
  )

  return (
    <div>
      <PageHeader
        title="Reports"
        subtitle="Cross-service analytics rolled up from every schema"
      />

      {[funnel, cycle, aging, mix].some(r => r.error) && (
        <ErrorBanner message="One or more reports failed to load. Check the reports service logs." />
      )}

      <div className="grid md:grid-cols-2 gap-6">
        <FunnelCard data={funnel.data} loading={funnel.loading} />
        <CycleTimeCard data={cycle.data} loading={cycle.loading} />
        <AgingCard data={aging.data} loading={aging.loading} />
        <MixCard data={mix.data} loading={mix.loading} />
      </div>
    </div>
  )
}

function ReportShell({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card className="p-6">
      <h2 className="text-sm uppercase tracking-wide text-slate-400 mb-4">{title}</h2>
      {children}
    </Card>
  )
}

function FunnelCard({ data, loading }: { data: QuoteToCash | null; loading: boolean }) {
  const maxCount = Math.max(1, ...(data?.stages ?? []).map(s => s.count))
  return (
    <ReportShell title="Quote-to-Cash Funnel">
      {loading ? (
        <Loading />
      ) : (
        <div className="space-y-2">
          {(data?.stages ?? []).map(s => {
            const pct = Math.round((s.count / maxCount) * 100)
            return (
              <div key={s.stage}>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-slate-300 capitalize">
                    {s.stage.replace(/_/g, ' ')}
                  </span>
                  <span className="text-slate-100 font-medium">
                    {s.count} · {fmtMoney(s.amount_total)}
                  </span>
                </div>
                <div className="bg-slate-800 rounded h-2 overflow-hidden">
                  <div
                    className="h-full bg-violet-500 transition-all"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>
            )
          })}
        </div>
      )}
    </ReportShell>
  )
}

function CycleTimeCard({ data, loading }: { data: CycleTimes | null; loading: boolean }) {
  return (
    <ReportShell title="Cycle Time (Days)">
      {loading ? (
        <Loading />
      ) : (
        <dl className="space-y-3">
          <MetricRow label="Quote Created → Accepted" value={data?.quote_created_to_accepted_days ?? 0} />
          <MetricRow label="Quote Accepted → Order" value={data?.quote_accepted_to_order_days ?? 0} />
          <MetricRow label="Order Created → Fulfilled" value={data?.order_created_to_fulfilled_days ?? 0} />
          <MetricRow label="Order Fulfilled → Paid" value={data?.order_fulfilled_to_paid_days ?? 0} />
        </dl>
      )}
    </ReportShell>
  )
}

function MetricRow({ label, value }: { label: string; value: number }) {
  return (
    <div className="flex justify-between items-baseline">
      <dt className="text-slate-400 text-sm">{label}</dt>
      <dd className="text-slate-100 font-semibold">{fmtNumber(value)}</dd>
    </div>
  )
}

function AgingCard({ data, loading }: { data: AgingBucket[] | null; loading: boolean }) {
  return (
    <ReportShell title="Invoice Aging">
      {loading ? (
        <Loading />
      ) : !data || data.length === 0 ? (
        <div className="text-slate-500 text-sm">No invoices in the system yet.</div>
      ) : (
        <table className="w-full text-sm">
          <thead className="text-xs text-slate-500">
            <tr>
              <th className="text-left pb-2">Bucket</th>
              <th className="text-right pb-2">Count</th>
              <th className="text-right pb-2">Amount Due</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60">
            {data.map(b => (
              <tr key={b.bucket}>
                <td className="py-2"><StatusBadge status={b.bucket.split('_')[0]} /></td>
                <td className="py-2 text-right text-slate-300">{b.count}</td>
                <td className="py-2 text-right text-slate-100 font-medium">
                  {fmtMoney(b.amount_due)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </ReportShell>
  )
}

function MixCard({ data, loading }: { data: MixBreakdown[] | null; loading: boolean }) {
  const totalRevenue = (data ?? []).reduce((sum, m) => sum + m.revenue, 0) || 1
  return (
    <ReportShell title="Mix Breakdown">
      {loading ? (
        <Loading />
      ) : !data || data.length === 0 ? (
        <div className="text-slate-500 text-sm">No line items in the system yet.</div>
      ) : (
        <div className="space-y-3">
          {data.map(m => {
            const pct = Math.round((m.revenue / totalRevenue) * 100)
            return (
              <div key={m.mix_type}>
                <div className="flex justify-between text-sm mb-1">
                  <span className="text-slate-300 capitalize">
                    {m.mix_type.replace(/_/g, ' ')}
                  </span>
                  <span className="text-slate-100">
                    {fmtMoney(m.revenue)} · {pct}%
                  </span>
                </div>
                <div className="bg-slate-800 rounded h-2 overflow-hidden">
                  <div
                    className="h-full bg-emerald-500"
                    style={{ width: `${pct}%` }}
                  />
                </div>
                <div className="text-xs text-slate-500 mt-1">
                  {fmtNumber(m.tons)} tons · {m.line_items} line items
                </div>
              </div>
            )
          })}
        </div>
      )}
    </ReportShell>
  )
}
