import type { ReactNode } from 'react'
import { STATUS_BADGE_CLASSES } from '../lib/constants'

export function PageHeader({ title, subtitle, actions }: {
  title: string
  subtitle?: string
  actions?: ReactNode
}) {
  return (
    <div className="flex items-start justify-between mb-6 flex-wrap gap-4">
      <div>
        <h1 className="text-2xl font-bold text-white">{title}</h1>
        {subtitle && <p className="text-slate-400 text-sm mt-1">{subtitle}</p>}
      </div>
      {actions && <div className="flex gap-2">{actions}</div>}
    </div>
  )
}

export function Card({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <div className={`bg-slate-900 border border-slate-800 rounded-lg ${className}`}>
      {children}
    </div>
  )
}

export function Button({ children, variant = 'primary', ...rest }: {
  children: ReactNode
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
} & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const base = 'px-3 py-1.5 rounded text-sm font-medium transition-colors disabled:opacity-40 disabled:cursor-not-allowed'
  const styles = {
    primary:   'bg-violet-600 hover:bg-violet-500 text-white',
    secondary: 'bg-slate-800 hover:bg-slate-700 text-slate-100 border border-slate-700',
    danger:    'bg-rose-700/70 hover:bg-rose-600 text-white',
    ghost:     'text-slate-300 hover:bg-slate-800',
  }[variant]
  return (
    <button className={`${base} ${styles}`} {...rest}>
      {children}
    </button>
  )
}

export function Field({ label, children, hint }: {
  label: string
  children: ReactNode
  hint?: string
}) {
  return (
    <label className="block">
      <span className="text-xs uppercase tracking-wide text-slate-400">{label}</span>
      <div className="mt-1">{children}</div>
      {hint && <p className="text-xs text-slate-500 mt-1">{hint}</p>}
    </label>
  )
}

export function Input(props: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`w-full bg-slate-950 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-100 focus:border-violet-500 focus:outline-none ${props.className ?? ''}`}
    />
  )
}

export function Textarea(props: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      {...props}
      className={`w-full bg-slate-950 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-100 focus:border-violet-500 focus:outline-none ${props.className ?? ''}`}
    />
  )
}

export function Select(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={`w-full bg-slate-950 border border-slate-700 rounded px-3 py-1.5 text-sm text-slate-100 focus:border-violet-500 focus:outline-none ${props.className ?? ''}`}
    />
  )
}

export function StatusBadge({ status }: { status: string }) {
  const cls = STATUS_BADGE_CLASSES[status] ?? 'bg-slate-700/60 text-slate-200'
  return (
    <span className={`inline-block px-2 py-0.5 rounded text-xs font-medium capitalize ${cls}`}>
      {status.replace(/_/g, ' ')}
    </span>
  )
}

export function Empty({ children }: { children: ReactNode }) {
  return (
    <div className="text-center py-12 text-slate-500 text-sm">
      {children}
    </div>
  )
}

export function ErrorBanner({ message }: { message: string }) {
  return (
    <div className="bg-rose-950/40 border border-rose-800 text-rose-200 rounded px-4 py-3 text-sm mb-4">
      {message}
    </div>
  )
}

export function Loading() {
  return <div className="text-slate-500 text-sm py-8 text-center">Loading…</div>
}
