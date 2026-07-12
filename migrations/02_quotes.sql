-- quotes schema
-- Owned by quotetooldemo-quotes service

CREATE SCHEMA IF NOT EXISTS quotes;

CREATE TYPE quotes.quote_status AS ENUM (
    'draft', 'sent', 'accepted', 'rejected', 'expired'
);

CREATE TYPE quotes.mix_type AS ENUM (
    'hma_base', 'hma_surface', 'superpave', 'warm_mix'
);

CREATE TABLE IF NOT EXISTS quotes.quotes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id UUID NOT NULL,
    project_name TEXT NOT NULL,
    project_address TEXT NOT NULL,
    status quotes.quote_status NOT NULL DEFAULT 'draft',
    subtotal NUMERIC(14, 2) NOT NULL DEFAULT 0,
    tax_rate NUMERIC(6, 4) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(14, 2) NOT NULL DEFAULT 0,
    markup_rate NUMERIC(6, 4) NOT NULL DEFAULT 0,
    markup_amount NUMERIC(14, 2) NOT NULL DEFAULT 0,
    total NUMERIC(14, 2) NOT NULL DEFAULT 0,
    notes TEXT,
    accepted_at TIMESTAMPTZ,
    rejected_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS quotes.quote_line_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quote_id UUID NOT NULL REFERENCES quotes.quotes(id) ON DELETE CASCADE,
    area_sqft NUMERIC(14, 2) NOT NULL,
    depth_inches NUMERIC(6, 2) NOT NULL,
    mix_type quotes.mix_type NOT NULL,
    unit_price_per_ton NUMERIC(10, 2) NOT NULL,
    tons NUMERIC(14, 4) NOT NULL,
    line_total NUMERIC(14, 2) NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_quotes_customer_id
    ON quotes.quotes (customer_id);
CREATE INDEX IF NOT EXISTS idx_quotes_status
    ON quotes.quotes (status);
CREATE INDEX IF NOT EXISTS idx_quote_line_items_quote_id
    ON quotes.quote_line_items (quote_id);
