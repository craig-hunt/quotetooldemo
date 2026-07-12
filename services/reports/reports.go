package main

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// The reports service reads directly across the quotes / orders / invoices
// schemas. At production scale, this pattern moves to materialized views or
// an eventing pipeline. For demo scale, direct SQL joins across schemas keep
// the code short.

type Reports struct {
	pool *pgxpool.Pool
}

func NewReports(pool *pgxpool.Pool) *Reports { return &Reports{pool: pool} }

type FunnelStage struct {
	Stage       string  `json:"stage"`
	Count       int64   `json:"count"`
	AmountTotal float64 `json:"amount_total"`
}

type QuoteToCash struct {
	Stages []FunnelStage `json:"stages"`
}

// QuoteToCash returns the funnel: quotes → sent → accepted → orders → fulfilled → invoices paid.
func (r *Reports) QuoteToCash(ctx context.Context) (QuoteToCash, error) {
	q := fmt.Sprintf(`
		SELECT '%s' AS stage, COUNT(*) AS n, COALESCE(SUM(total), 0) AS total
		FROM %s
		UNION ALL
		SELECT '%s', COUNT(*), COALESCE(SUM(total), 0)
		FROM %s WHERE sent_at IS NOT NULL
		UNION ALL
		SELECT '%s', COUNT(*), COALESCE(SUM(total), 0)
		FROM %s WHERE accepted_at IS NOT NULL
		UNION ALL
		SELECT '%s', COUNT(*), COALESCE(SUM(total_amount), 0)
		FROM %s
		UNION ALL
		SELECT '%s', COUNT(*), COALESCE(SUM(total_amount), 0)
		FROM %s WHERE fulfilled_date IS NOT NULL
		UNION ALL
		SELECT '%s', COUNT(*), COALESCE(SUM(amount_paid), 0)
		FROM %s WHERE status = '%s'
	`,
		StageQuotesCreated, SchemaQuotesTable,
		StageQuotesSent, SchemaQuotesTable,
		StageQuotesAccepted, SchemaQuotesTable,
		StageOrdersCreated, SchemaOrdersTable,
		StageOrdersFulfilled, SchemaOrdersTable,
		StageInvoicesPaid, SchemaInvoicesTable, SQLInvoiceStatusPaid,
	)

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return QuoteToCash{}, fmt.Errorf("quote-to-cash query: %w", err)
	}
	defer rows.Close()

	out := QuoteToCash{Stages: make([]FunnelStage, 0, 6)}
	for rows.Next() {
		var s FunnelStage
		if err := rows.Scan(&s.Stage, &s.Count, &s.AmountTotal); err != nil {
			return QuoteToCash{}, fmt.Errorf("scan: %w", err)
		}
		out.Stages = append(out.Stages, s)
	}
	return out, rows.Err()
}

type CycleTimes struct {
	QuoteCreatedToAccepted   float64 `json:"quote_created_to_accepted_days"`
	QuoteAcceptedToOrder     float64 `json:"quote_accepted_to_order_days"`
	OrderCreatedToFulfilled  float64 `json:"order_created_to_fulfilled_days"`
	OrderFulfilledToPaid     float64 `json:"order_fulfilled_to_paid_days"`
}

// CycleTimes returns average days at each stage transition.
func (r *Reports) CycleTimes(ctx context.Context) (CycleTimes, error) {
	q := fmt.Sprintf(`
		SELECT
			COALESCE(AVG(EXTRACT(EPOCH FROM (accepted_at - created_at))/86400), 0) AS quote_created_to_accepted,
			COALESCE((
				SELECT AVG(EXTRACT(EPOCH FROM (o.created_at - q.accepted_at))/86400)
				FROM %s o JOIN %s q ON q.id = o.quote_id
				WHERE q.accepted_at IS NOT NULL
			), 0) AS quote_accepted_to_order,
			COALESCE((
				SELECT AVG(EXTRACT(EPOCH FROM (fulfilled_date::timestamptz - created_at))/86400)
				FROM %s WHERE fulfilled_date IS NOT NULL
			), 0) AS order_created_to_fulfilled,
			COALESCE((
				SELECT AVG(EXTRACT(EPOCH FROM (inv.paid_date::timestamptz - o.fulfilled_date::timestamptz))/86400)
				FROM %s inv JOIN %s o ON o.id = inv.order_id
				WHERE inv.paid_date IS NOT NULL AND o.fulfilled_date IS NOT NULL
			), 0) AS order_fulfilled_to_paid
		FROM %s WHERE accepted_at IS NOT NULL
	`,
		SchemaOrdersTable, SchemaQuotesTable,
		SchemaOrdersTable,
		SchemaInvoicesTable, SchemaOrdersTable,
		SchemaQuotesTable,
	)

	var ct CycleTimes
	if err := r.pool.QueryRow(ctx, q).Scan(
		&ct.QuoteCreatedToAccepted,
		&ct.QuoteAcceptedToOrder,
		&ct.OrderCreatedToFulfilled,
		&ct.OrderFulfilledToPaid,
	); err != nil {
		return CycleTimes{}, fmt.Errorf("cycle times: %w", err)
	}
	return ct, nil
}

type AgingBucket struct {
	Bucket    string  `json:"bucket"`
	Count     int64   `json:"count"`
	AmountDue float64 `json:"amount_due"`
}

// Aging returns invoice aging buckets: current, 1-30, 31-60, 61-90, 90+ days overdue.
func (r *Reports) Aging(ctx context.Context) ([]AgingBucket, error) {
	q := fmt.Sprintf(`
		SELECT
			CASE
				WHEN status = '%s' THEN '%s'
				WHEN status = '%s' THEN '%s'
				WHEN due_date >= CURRENT_DATE THEN '%s'
				WHEN CURRENT_DATE - due_date <= 30 THEN '%s'
				WHEN CURRENT_DATE - due_date <= 60 THEN '%s'
				WHEN CURRENT_DATE - due_date <= 90 THEN '%s'
				ELSE '%s'
			END AS bucket,
			COUNT(*) AS n,
			COALESCE(SUM(amount_due - amount_paid), 0) AS amount_due
		FROM %s
		GROUP BY bucket
		ORDER BY bucket
	`,
		SQLInvoiceStatusPaid, BucketPaid,
		SQLInvoiceStatusCancelled, BucketCancelled,
		BucketCurrent,
		Bucket1To30Days,
		Bucket31To60Days,
		Bucket61To90Days,
		Bucket90PlusDays,
		SchemaInvoicesTable,
	)

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("aging: %w", err)
	}
	defer rows.Close()

	out := make([]AgingBucket, 0)
	for rows.Next() {
		var b AgingBucket
		if err := rows.Scan(&b.Bucket, &b.Count, &b.AmountDue); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

type MixBreakdown struct {
	MixType   string  `json:"mix_type"`
	Tons      float64 `json:"tons"`
	Revenue   float64 `json:"revenue"`
	LineItems int64   `json:"line_items"`
}

// MixBreakdown returns quote line-item revenue grouped by mix type.
func (r *Reports) MixBreakdown(ctx context.Context) ([]MixBreakdown, error) {
	q := fmt.Sprintf(`
		SELECT mix_type::text,
		       COALESCE(SUM(tons), 0),
		       COALESCE(SUM(line_total), 0),
		       COUNT(*)
		FROM %s
		GROUP BY mix_type
		ORDER BY SUM(line_total) DESC
	`, SchemaQuoteLineItemsTable)

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("mix breakdown: %w", err)
	}
	defer rows.Close()

	out := make([]MixBreakdown, 0)
	for rows.Next() {
		var m MixBreakdown
		if err := rows.Scan(&m.MixType, &m.Tons, &m.Revenue, &m.LineItems); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
