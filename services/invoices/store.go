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
	ErrNotFound        = errors.New("invoice not found")
	ErrInvalidStatus   = errors.New("invalid status transition")
	ErrOrderNotFound   = errors.New("order not found")
	ErrOrderNotFulfill = errors.New("order must be fulfilled")
)

type Store interface {
	Create(ctx context.Context, in CreateInput, o OrderInfo) (Invoice, error)
	Get(ctx context.Context, id uuid.UUID) (Invoice, error)
	List(ctx context.Context, p ListParams) ([]Invoice, error)
	Transition(ctx context.Context, id uuid.UUID, to InvoiceStatus) (Invoice, error)
}

type PGStore struct{ pool *pgxpool.Pool }

func NewPGStore(pool *pgxpool.Pool) *PGStore { return &PGStore{pool: pool} }

func (s *PGStore) Create(ctx context.Context, in CreateInput, o OrderInfo) (Invoice, error) {
	var seq int64
	if err := s.pool.QueryRow(ctx, "SELECT nextval('"+InvoiceNumberSequence+"')").Scan(&seq); err != nil {
		return Invoice{}, fmt.Errorf("invoice number: %w", err)
	}
	invoiceNumber := fmt.Sprintf(InvoiceNumberPattern, InvoiceNumberPrefix, time.Now().Year(), seq)

	dueDate := time.Now().UTC().AddDate(0, 0, in.DueDays)

	const insertQ = `
		INSERT INTO invoices.invoices
			(order_id, customer_id, invoice_number, status, amount_due, due_date, notes)
		VALUES ($1, $2, $3, 'draft', $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	inv := Invoice{
		OrderID:       o.ID,
		CustomerID:    o.CustomerID,
		InvoiceNumber: invoiceNumber,
		Status:        StatusDraft,
		AmountDue:     o.TotalAmount,
		DueDate:       dueDate,
		Notes:         in.Notes,
	}
	if err := s.pool.QueryRow(ctx, insertQ,
		inv.OrderID, inv.CustomerID, inv.InvoiceNumber, inv.AmountDue, inv.DueDate, inv.Notes,
	).Scan(&inv.ID, &inv.CreatedAt, &inv.UpdatedAt); err != nil {
		return Invoice{}, fmt.Errorf("insert invoice: %w", err)
	}
	return inv, nil
}

func (s *PGStore) Get(ctx context.Context, id uuid.UUID) (Invoice, error) {
	const q = `
		SELECT id, order_id, customer_id, invoice_number, status,
		       amount_due, amount_paid, due_date, sent_at, paid_date, cancelled_date,
		       notes, created_at, updated_at
		FROM invoices.invoices
		WHERE id = $1
	`
	inv, err := scanInvoice(s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, ErrNotFound
	}
	return inv, err
}

func (s *PGStore) List(ctx context.Context, p ListParams) ([]Invoice, error) {
	sql := `
		SELECT id, order_id, customer_id, invoice_number, status,
		       amount_due, amount_paid, due_date, sent_at, paid_date, cancelled_date,
		       notes, created_at, updated_at
		FROM invoices.invoices
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

	out := make([]Invoice, 0)
	for rows.Next() {
		inv, err := scanInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (s *PGStore) Transition(ctx context.Context, id uuid.UUID, to InvoiceStatus) (Invoice, error) {
	inv, err := s.Get(ctx, id)
	if err != nil {
		return Invoice{}, err
	}
	if !allowedTransition(inv.Status, to) {
		return Invoice{}, ErrInvalidStatus
	}

	now := time.Now().UTC()
	sql := "UPDATE invoices.invoices SET status = $2, updated_at = NOW()"
	args := []any{id, to}
	argN := 3

	switch to {
	case StatusSent:
		sql += fmt.Sprintf(", sent_at = $%d", argN)
		args = append(args, now)
	case StatusPaid:
		sql += fmt.Sprintf(", paid_date = $%d, amount_paid = amount_due", argN)
		args = append(args, now)
	case StatusCancelled:
		sql += fmt.Sprintf(", cancelled_date = $%d", argN)
		args = append(args, now)
	}
	sql += " WHERE id = $1 RETURNING updated_at, amount_paid"

	if err := s.pool.QueryRow(ctx, sql, args...).Scan(&inv.UpdatedAt, &inv.AmountPaid); err != nil {
		return Invoice{}, fmt.Errorf("transition: %w", err)
	}
	inv.Status = to
	switch to {
	case StatusSent:
		inv.SentAt = &now
	case StatusPaid:
		inv.PaidDate = &now
	case StatusCancelled:
		inv.CancelledDate = &now
	}
	return inv, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInvoice(r rowScanner) (Invoice, error) {
	var inv Invoice
	err := r.Scan(
		&inv.ID, &inv.OrderID, &inv.CustomerID, &inv.InvoiceNumber, &inv.Status,
		&inv.AmountDue, &inv.AmountPaid, &inv.DueDate,
		&inv.SentAt, &inv.PaidDate, &inv.CancelledDate,
		&inv.Notes, &inv.CreatedAt, &inv.UpdatedAt,
	)
	return inv, err
}

func allowedTransition(from, to InvoiceStatus) bool {
	transitions := map[InvoiceStatus][]InvoiceStatus{
		StatusDraft:     {StatusSent, StatusCancelled},
		StatusSent:      {StatusPaid, StatusOverdue, StatusCancelled},
		StatusOverdue:   {StatusPaid, StatusCancelled},
		StatusPaid:      {},
		StatusCancelled: {},
	}
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
