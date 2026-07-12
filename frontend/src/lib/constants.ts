export const QUOTE_STATUS = {
  DRAFT: 'draft',
  SENT: 'sent',
  ACCEPTED: 'accepted',
  REJECTED: 'rejected',
  EXPIRED: 'expired',
} as const

export type QuoteStatus = typeof QUOTE_STATUS[keyof typeof QUOTE_STATUS]

export const QUOTE_TRANSITIONS: Record<QuoteStatus, { action: string; label: string; target: QuoteStatus }[]> = {
  draft:    [{ action: 'send',   label: 'Send',   target: 'sent' }],
  sent:     [
    { action: 'accept', label: 'Accept', target: 'accepted' },
    { action: 'reject', label: 'Reject', target: 'rejected' },
  ],
  accepted: [],
  rejected: [],
  expired:  [],
}

export const ORDER_STATUS = {
  OPEN: 'open',
  IN_PROGRESS: 'in_progress',
  FULFILLED: 'fulfilled',
  CANCELLED: 'cancelled',
} as const

export type OrderStatus = typeof ORDER_STATUS[keyof typeof ORDER_STATUS]

export const ORDER_TRANSITIONS: Record<OrderStatus, { action: string; label: string; target: OrderStatus }[]> = {
  open: [
    { action: 'fulfill', label: 'Fulfill', target: 'fulfilled' },
    { action: 'cancel',  label: 'Cancel',  target: 'cancelled' },
  ],
  in_progress: [
    { action: 'fulfill', label: 'Fulfill', target: 'fulfilled' },
    { action: 'cancel',  label: 'Cancel',  target: 'cancelled' },
  ],
  fulfilled: [],
  cancelled: [],
}

export const INVOICE_STATUS = {
  DRAFT: 'draft',
  SENT: 'sent',
  PAID: 'paid',
  OVERDUE: 'overdue',
  CANCELLED: 'cancelled',
} as const

export type InvoiceStatus = typeof INVOICE_STATUS[keyof typeof INVOICE_STATUS]

export const INVOICE_TRANSITIONS: Record<InvoiceStatus, { action: string; label: string; target: InvoiceStatus }[]> = {
  draft:     [
    { action: 'send',   label: 'Send',   target: 'sent' },
    { action: 'cancel', label: 'Cancel', target: 'cancelled' },
  ],
  sent:      [
    { action: 'mark_paid', label: 'Mark Paid', target: 'paid' },
    { action: 'cancel',    label: 'Cancel',    target: 'cancelled' },
  ],
  overdue:   [
    { action: 'mark_paid', label: 'Mark Paid', target: 'paid' },
    { action: 'cancel',    label: 'Cancel',    target: 'cancelled' },
  ],
  paid:      [],
  cancelled: [],
}

export const MIX_TYPES = [
  { value: 'hma_base',    label: 'HMA Base' },
  { value: 'hma_surface', label: 'HMA Surface' },
  { value: 'superpave',   label: 'Superpave' },
  { value: 'warm_mix',    label: 'Warm Mix' },
] as const

export type MixType = typeof MIX_TYPES[number]['value']

export const ASPHALT_DENSITY_LBS_PER_SQFT_INCH = 12.5
export const POUNDS_PER_TON = 2000

export function computeTons(areaSqft: number, depthInches: number): number {
  if (!areaSqft || !depthInches) return 0
  return (areaSqft * depthInches * ASPHALT_DENSITY_LBS_PER_SQFT_INCH) / POUNDS_PER_TON
}

export const STATUS_BADGE_CLASSES: Record<string, string> = {
  draft:       'bg-slate-700/60 text-slate-200',
  sent:        'bg-sky-700/40 text-sky-200',
  accepted:    'bg-emerald-700/40 text-emerald-200',
  fulfilled:   'bg-emerald-700/40 text-emerald-200',
  paid:        'bg-emerald-700/40 text-emerald-200',
  rejected:    'bg-rose-700/40 text-rose-200',
  cancelled:   'bg-rose-700/40 text-rose-200',
  expired:     'bg-amber-700/40 text-amber-200',
  overdue:     'bg-amber-700/40 text-amber-200',
  open:        'bg-sky-700/40 text-sky-200',
  in_progress: 'bg-violet-700/40 text-violet-200',
}
