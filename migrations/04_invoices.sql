-- invoices schema
-- Owned by quotetooldemo-invoices service

CREATE SCHEMA IF NOT EXISTS invoices;

CREATE TYPE invoices.invoice_status AS ENUM (
    'draft', 'sent', 'paid', 'overdue', 'cancelled'
);

CREATE TABLE IF NOT EXISTS invoices.invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    invoice_number TEXT NOT NULL UNIQUE,
    status invoices.invoice_status NOT NULL DEFAULT 'draft',
    amount_due NUMERIC(14, 2) NOT NULL,
    amount_paid NUMERIC(14, 2) NOT NULL DEFAULT 0,
    due_date DATE NOT NULL,
    sent_at TIMESTAMPTZ,
    paid_date DATE,
    cancelled_date DATE,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE SEQUENCE IF NOT EXISTS invoices.invoice_number_seq START 1;

CREATE INDEX IF NOT EXISTS idx_invoices_customer_id
    ON invoices.invoices (customer_id);
CREATE INDEX IF NOT EXISTS idx_invoices_order_id
    ON invoices.invoices (order_id);
CREATE INDEX IF NOT EXISTS idx_invoices_status
    ON invoices.invoices (status);
CREATE INDEX IF NOT EXISTS idx_invoices_due_date
    ON invoices.invoices (due_date);
