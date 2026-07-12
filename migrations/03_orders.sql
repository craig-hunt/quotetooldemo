-- orders schema
-- Owned by quotetooldemo-orders service

CREATE SCHEMA IF NOT EXISTS orders;

CREATE TYPE orders.order_status AS ENUM (
    'open', 'in_progress', 'fulfilled', 'cancelled'
);

CREATE TABLE IF NOT EXISTS orders.orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    quote_id UUID NOT NULL,
    customer_id UUID NOT NULL,
    order_number TEXT NOT NULL UNIQUE,
    status orders.order_status NOT NULL DEFAULT 'open',
    scheduled_date DATE,
    fulfilled_date DATE,
    cancelled_date DATE,
    total_amount NUMERIC(14, 2) NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE SEQUENCE IF NOT EXISTS orders.order_number_seq START 1;

CREATE INDEX IF NOT EXISTS idx_orders_customer_id
    ON orders.orders (customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_quote_id
    ON orders.orders (quote_id);
CREATE INDEX IF NOT EXISTS idx_orders_status
    ON orders.orders (status);
