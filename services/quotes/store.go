package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("quote not found")
	ErrInvalidStatus = errors.New("invalid status transition")
)

type Store interface {
	Create(ctx context.Context, in QuoteInput) (Quote, error)
	Get(ctx context.Context, id uuid.UUID) (Quote, error)
	List(ctx context.Context, p ListParams) ([]Quote, error)
	Update(ctx context.Context, id uuid.UUID, in QuoteInput) (Quote, error)
	Transition(ctx context.Context, id uuid.UUID, to QuoteStatus) (Quote, error)
}

type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (s *PGStore) Create(ctx context.Context, in QuoteInput) (Quote, error) {
	q := Quote{
		CustomerID:     in.CustomerID,
		ProjectName:    in.ProjectName,
		ProjectAddress: in.ProjectAddress,
		Status:         StatusDraft,
		TaxRate:        in.TaxRate,
		MarkupRate:     in.MarkupRate,
		Notes:          in.Notes,
	}
	for _, li := range in.LineItems {
		q.LineItems = append(q.LineItems, LineItem{
			AreaSqft:        li.AreaSqft,
			DepthInches:     li.DepthInches,
			MixType:         li.MixType,
			UnitPricePerTon: li.UnitPricePerTon,
		})
	}
	computeQuoteTotals(&q)

	// Transaction wraps the header insert + all line item inserts.
	// Either all succeed or all roll back. Standard Go/pgx pattern.
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Quote{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx) // rollback is a no-op after a commit
	}()

	const insertQ = `
		INSERT INTO quotes.quotes
			(customer_id, project_name, project_address, status,
			 subtotal, tax_rate, tax_amount, markup_rate, markup_amount, total, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`
	if err := tx.QueryRow(ctx, insertQ,
		q.CustomerID, q.ProjectName, q.ProjectAddress, q.Status,
		q.Subtotal, q.TaxRate, q.TaxAmount, q.MarkupRate, q.MarkupAmount, q.Total, q.Notes,
	).Scan(&q.ID, &q.CreatedAt, &q.UpdatedAt); err != nil {
		return Quote{}, fmt.Errorf("insert quote: %w", err)
	}

	for i := range q.LineItems {
		li := &q.LineItems[i]
		li.Position = i
		const insertLine = `
			INSERT INTO quotes.quote_line_items
				(quote_id, area_sqft, depth_inches, mix_type,
				 unit_price_per_ton, tons, line_total, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`
		if err := tx.QueryRow(ctx, insertLine,
			q.ID, li.AreaSqft, li.DepthInches, li.MixType,
			li.UnitPricePerTon, li.Tons, li.LineTotal, li.Position,
		).Scan(&li.ID); err != nil {
			return Quote{}, fmt.Errorf("insert line %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Quote{}, fmt.Errorf("commit: %w", err)
	}
	return q, nil
}

func (s *PGStore) Get(ctx context.Context, id uuid.UUID) (Quote, error) {
	const q = `
		SELECT id, customer_id, project_name, project_address, status,
		       subtotal, tax_rate, tax_amount, markup_rate, markup_amount, total, notes,
		       accepted_at, rejected_at, sent_at, created_at, updated_at
		FROM quotes.quotes
		WHERE id = $1
	`
	quote, err := scanQuote(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Quote{}, ErrNotFound
	}
	if err != nil {
		return Quote{}, err
	}

	lines, err := s.loadLineItems(ctx, id)
	if err != nil {
		return Quote{}, err
	}
	quote.LineItems = lines
	return quote, nil
}

func (s *PGStore) List(ctx context.Context, p ListParams) ([]Quote, error) {
	// Dynamic filter clauses. Order of args matters; track them in a slice.
	sql := `
		SELECT id, customer_id, project_name, project_address, status,
		       subtotal, tax_rate, tax_amount, markup_rate, markup_amount, total, notes,
		       accepted_at, rejected_at, sent_at, created_at, updated_at
		FROM quotes.quotes
		WHERE 1=1
	`
	args := []any{}
	argN := 1
	if p.CustomerID != nil {
		sql += fmt.Sprintf(" AND customer_id = $%d", argN)
		args = append(args, *p.CustomerID)
		argN++
	}
	if p.Status != nil {
		sql += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, *p.Status)
		argN++
	}
	sql += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list query: %w", err)
	}
	defer rows.Close()

	out := make([]Quote, 0)
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			return nil, err
		}
		// List endpoint returns headers without line items to keep responses small.
		// Client hits GET /quotes/:id for full detail.
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *PGStore) Update(ctx context.Context, id uuid.UUID, in QuoteInput) (Quote, error) {
	// Simple approach: delete existing line items, re-insert new ones inside a tx.
	// Adequate for demo scale. Production version would diff and update-in-place.
	existing, err := s.Get(ctx, id)
	if err != nil {
		return Quote{}, err
	}
	if existing.Status != StatusDraft {
		return Quote{}, fmt.Errorf("cannot update quote in status %s", existing.Status)
	}

	q := Quote{
		ID:             id,
		CustomerID:     in.CustomerID,
		ProjectName:    in.ProjectName,
		ProjectAddress: in.ProjectAddress,
		Status:         StatusDraft,
		TaxRate:        in.TaxRate,
		MarkupRate:     in.MarkupRate,
		Notes:          in.Notes,
	}
	for _, li := range in.LineItems {
		q.LineItems = append(q.LineItems, LineItem{
			AreaSqft:        li.AreaSqft,
			DepthInches:     li.DepthInches,
			MixType:         li.MixType,
			UnitPricePerTon: li.UnitPricePerTon,
		})
	}
	computeQuoteTotals(&q)

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Quote{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const upd = `
		UPDATE quotes.quotes
		SET customer_id = $2, project_name = $3, project_address = $4,
		    subtotal = $5, tax_rate = $6, tax_amount = $7,
		    markup_rate = $8, markup_amount = $9, total = $10, notes = $11,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING created_at, updated_at
	`
	if err := tx.QueryRow(ctx, upd, id,
		q.CustomerID, q.ProjectName, q.ProjectAddress,
		q.Subtotal, q.TaxRate, q.TaxAmount, q.MarkupRate, q.MarkupAmount, q.Total, q.Notes,
	).Scan(&q.CreatedAt, &q.UpdatedAt); err != nil {
		return Quote{}, fmt.Errorf("update quote: %w", err)
	}

	if _, err := tx.Exec(ctx, "DELETE FROM quotes.quote_line_items WHERE quote_id = $1", id); err != nil {
		return Quote{}, fmt.Errorf("delete lines: %w", err)
	}

	for i := range q.LineItems {
		li := &q.LineItems[i]
		li.Position = i
		const ins = `
			INSERT INTO quotes.quote_line_items
				(quote_id, area_sqft, depth_inches, mix_type,
				 unit_price_per_ton, tons, line_total, position)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`
		if err := tx.QueryRow(ctx, ins,
			q.ID, li.AreaSqft, li.DepthInches, li.MixType,
			li.UnitPricePerTon, li.Tons, li.LineTotal, li.Position,
		).Scan(&li.ID); err != nil {
			return Quote{}, fmt.Errorf("insert line %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return Quote{}, fmt.Errorf("commit: %w", err)
	}
	return q, nil
}

// Transition validates the requested state change against the allowed graph,
// updates the row, and stamps the timestamp column for the transition.
func (s *PGStore) Transition(ctx context.Context, id uuid.UUID, to QuoteStatus) (Quote, error) {
	q, err := s.Get(ctx, id)
	if err != nil {
		return Quote{}, err
	}
	if !allowedTransition(q.Status, to) {
		return Quote{}, ErrInvalidStatus
	}

	// Stamp the matching timestamp column so we can report cycle-time later.
	// StatusExpired has no timestamp column, so we build the SQL conditionally.
	col := ""
	switch to {
	case StatusSent:
		col = "sent_at"
	case StatusAccepted:
		col = "accepted_at"
	case StatusRejected:
		col = "rejected_at"
	}

	now := time.Now().UTC()
	sql := "UPDATE quotes.quotes SET status = $2, updated_at = NOW()"
	args := []any{id, to}
	if col != "" {
		sql += fmt.Sprintf(", %s = $3", col)
		args = append(args, now)
	}
	sql += " WHERE id = $1 RETURNING updated_at"

	if err := s.pool.QueryRow(ctx, sql, args...).Scan(&q.UpdatedAt); err != nil {
		return Quote{}, fmt.Errorf("transition: %w", err)
	}
	q.Status = to
	switch to {
	case StatusSent:
		q.SentAt = &now
	case StatusAccepted:
		q.AcceptedAt = &now
	case StatusRejected:
		q.RejectedAt = &now
	}
	return q, nil
}

func (s *PGStore) loadLineItems(ctx context.Context, quoteID uuid.UUID) ([]LineItem, error) {
	const q = `
		SELECT id, area_sqft, depth_inches, mix_type, unit_price_per_ton,
		       tons, line_total, position
		FROM quotes.quote_line_items
		WHERE quote_id = $1
		ORDER BY position ASC
	`
	rows, err := s.pool.Query(ctx, q, quoteID)
	if err != nil {
		return nil, fmt.Errorf("load lines: %w", err)
	}
	defer rows.Close()

	out := make([]LineItem, 0)
	for rows.Next() {
		var li LineItem
		if err := rows.Scan(
			&li.ID, &li.AreaSqft, &li.DepthInches, &li.MixType,
			&li.UnitPricePerTon, &li.Tons, &li.LineTotal, &li.Position,
		); err != nil {
			return nil, fmt.Errorf("scan line: %w", err)
		}
		out = append(out, li)
	}
	return out, rows.Err()
}

// Row abstraction: both pgx.Row and pgx.Rows implement Scan(...).
// This helper accepts either so we don't duplicate scan logic.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanQuote(r rowScanner) (Quote, error) {
	var q Quote
	err := r.Scan(
		&q.ID, &q.CustomerID, &q.ProjectName, &q.ProjectAddress, &q.Status,
		&q.Subtotal, &q.TaxRate, &q.TaxAmount, &q.MarkupRate, &q.MarkupAmount, &q.Total, &q.Notes,
		&q.AcceptedAt, &q.RejectedAt, &q.SentAt, &q.CreatedAt, &q.UpdatedAt,
	)
	return q, err
}

// allowedTransition encodes the quote state machine.
func allowedTransition(from, to QuoteStatus) bool {
	transitions := map[QuoteStatus][]QuoteStatus{
		StatusDraft:    {StatusSent},
		StatusSent:     {StatusAccepted, StatusRejected, StatusExpired},
		StatusAccepted: {},
		StatusRejected: {},
		StatusExpired:  {},
	}
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
