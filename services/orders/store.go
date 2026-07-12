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
	ErrNotFound       = errors.New("order not found")
	ErrInvalidStatus  = errors.New("invalid status transition")
	ErrQuoteNotFound  = errors.New("quote not found")
	ErrQuoteNotAccept = errors.New("quote must be in accepted status")
)

type Store interface {
	Create(ctx context.Context, in CreateInput, q QuoteInfo) (Order, error)
	Get(ctx context.Context, id uuid.UUID) (Order, error)
	List(ctx context.Context, p ListParams) ([]Order, error)
	Update(ctx context.Context, id uuid.UUID, in UpdateInput) (Order, error)
	Transition(ctx context.Context, id uuid.UUID, to OrderStatus) (Order, error)
}

type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) Create(ctx context.Context, in CreateInput, q QuoteInfo) (Order, error) {
	// Generate order number via a sequence for human-readable IDs.
	var seq int64
	if err := s.pool.QueryRow(ctx, "SELECT nextval('"+OrderNumberSequence+"')").Scan(&seq); err != nil {
		return Order{}, fmt.Errorf("order number: %w", err)
	}
	orderNumber := fmt.Sprintf(OrderNumberPattern, OrderNumberPrefix, time.Now().Year(), seq)

	const insertQ = `
		INSERT INTO orders.orders
			(quote_id, customer_id, order_number, status, scheduled_date, total_amount, notes)
		VALUES ($1, $2, $3, 'open', $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	o := Order{
		QuoteID:       q.ID,
		CustomerID:    q.CustomerID,
		OrderNumber:   orderNumber,
		Status:        StatusOpen,
		ScheduledDate: in.ScheduledDate,
		TotalAmount:   q.Total,
		Notes:         in.Notes,
	}
	if err := s.pool.QueryRow(ctx, insertQ,
		o.QuoteID, o.CustomerID, o.OrderNumber, o.ScheduledDate, o.TotalAmount, o.Notes,
	).Scan(&o.ID, &o.CreatedAt, &o.UpdatedAt); err != nil {
		return Order{}, fmt.Errorf("insert order: %w", err)
	}
	return o, nil
}

func (s *PGStore) Get(ctx context.Context, id uuid.UUID) (Order, error) {
	const q = `
		SELECT id, quote_id, customer_id, order_number, status,
		       scheduled_date, fulfilled_date, cancelled_date,
		       total_amount, notes, created_at, updated_at
		FROM orders.orders
		WHERE id = $1
	`
	o, err := scanOrder(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	return o, err
}

func (s *PGStore) List(ctx context.Context, p ListParams) ([]Order, error) {
	sql := `
		SELECT id, quote_id, customer_id, order_number, status,
		       scheduled_date, fulfilled_date, cancelled_date,
		       total_amount, notes, created_at, updated_at
		FROM orders.orders
		WHERE 1=1
	`
	args := []any{}
	argN := 1
	if p.Status != nil {
		sql += fmt.Sprintf(" AND status = $%d", argN)
		args = append(args, *p.Status)
		argN++
	}
	sql += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, p.Limit, p.Offset)

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("list: %w", err)
	}
	defer rows.Close()

	out := make([]Order, 0)
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (s *PGStore) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (Order, error) {
	const q = `
		UPDATE orders.orders
		SET scheduled_date = $2, notes = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, quote_id, customer_id, order_number, status,
		          scheduled_date, fulfilled_date, cancelled_date,
		          total_amount, notes, created_at, updated_at
	`
	o, err := scanOrder(s.pool.QueryRow(ctx, q, id, in.ScheduledDate, in.Notes))
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	return o, err
}

func (s *PGStore) Transition(ctx context.Context, id uuid.UUID, to OrderStatus) (Order, error) {
	o, err := s.Get(ctx, id)
	if err != nil {
		return Order{}, err
	}
	if !allowedTransition(o.Status, to) {
		return Order{}, ErrInvalidStatus
	}

	col := ""
	switch to {
	case StatusFulfilled:
		col = "fulfilled_date"
	case StatusCancelled:
		col = "cancelled_date"
	}

	now := time.Now().UTC()
	sql := "UPDATE orders.orders SET status = $2, updated_at = NOW()"
	if col != "" {
		sql += fmt.Sprintf(", %s = $3", col)
	}
	sql += " WHERE id = $1 RETURNING updated_at"

	args := []any{id, to}
	if col != "" {
		args = append(args, now)
	}
	if err := s.pool.QueryRow(ctx, sql, args...).Scan(&o.UpdatedAt); err != nil {
		return Order{}, fmt.Errorf("transition: %w", err)
	}
	o.Status = to
	switch to {
	case StatusFulfilled:
		o.FulfilledDate = &now
	case StatusCancelled:
		o.CancelledDate = &now
	}
	return o, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanOrder(r rowScanner) (Order, error) {
	var o Order
	err := r.Scan(
		&o.ID, &o.QuoteID, &o.CustomerID, &o.OrderNumber, &o.Status,
		&o.ScheduledDate, &o.FulfilledDate, &o.CancelledDate,
		&o.TotalAmount, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)
	return o, err
}

func allowedTransition(from, to OrderStatus) bool {
	transitions := map[OrderStatus][]OrderStatus{
		StatusOpen:       {StatusInProgress, StatusFulfilled, StatusCancelled},
		StatusInProgress: {StatusFulfilled, StatusCancelled},
		StatusFulfilled:  {},
		StatusCancelled:  {},
	}
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
