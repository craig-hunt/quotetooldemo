import type { ReactNode } from 'react'
import { APP_TITLE } from '../lib/constants'

export default function Dashboard() {
  return (
    <div>
      <h1 className="text-3xl font-bold text-white mb-2">{APP_TITLE}</h1>
      <p className="text-slate-400 mb-8">
        Paving-industry quote-to-cash workflow demo.
      </p>
      <div className="grid md:grid-cols-2 gap-6">
        <Card title="What it does">
          <p className="text-slate-300 text-sm leading-relaxed">
            Manages the full commercial workflow: customer → quote → sales order → invoice → paid.
            Every stage generates reporting data that rolls up into a quote-to-cash funnel.
          </p>
        </Card>
        <Card title="Architecture">
          <ul className="text-slate-300 text-sm space-y-1">
            <li>5 Go microservices (customers, quotes, orders, invoices, reports)</li>
            <li>Postgres backing store, one schema per service</li>
            <li>React + TypeScript + Tailwind frontend</li>
            <li>Fly.io for services, Cloudflare Pages for frontend</li>
          </ul>
        </Card>
      </div>
    </div>
  )
}

function Card({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-lg p-6">
      <h2 className="text-lg font-semibold text-white mb-3">{title}</h2>
      {children}
    </div>
  )
}
