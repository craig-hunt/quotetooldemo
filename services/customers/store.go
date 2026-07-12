package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound signals a lookup against a nonexistent or soft-deleted customer.
// Handlers translate this to HTTP 404.
var ErrNotFound = errors.New("customer not found")

// Store isolates database access from HTTP handlers. Handlers depend on this
// interface, which makes testing straightforward: mock Store, call the handler,
// assert the response.
type Store interface {
	Create(ctx context.Context, in CustomerInput) (Customer, error)
	Get(ctx context.Context, id uuid.UUID) (Customer, error)
	List(ctx context.Context, p ListParams) ([]Customer, error)
	Update(ctx context.Context, id uuid.UUID, in CustomerInput) (Customer, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// PGStore satisfies Store via a Postgres connection pool.
type PGStore struct {
	pool *pgxpool.Pool
}

func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

func (s *PGStore) Create(ctx context.Context, in CustomerInput) (Customer, error) {
	addr, err := json.Marshal(in.BillingAddress)
	if err != nil {
		return Customer{}, fmt.Errorf("marshal billing address: %w", err)
	}

	const q = `
		INSERT INTO customers.customers
			(name, contact_name, email, phone, billing_address)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, contact_name, email, phone, billing_address, created_at, updated_at
	`

	return scanRow(ctx, s.pool.QueryRow(ctx, q,
		in.Name, in.ContactName, in.Email, in.Phone, addr,
	))
}

func (s *PGStore) Get(ctx context.Context, id uuid.UUID) (Customer, error) {
	const q = `
		SELECT id, name, contact_name, email, phone, billing_address, created_at, updated_at
		FROM customers.customers
		WHERE id = $1 AND deleted_at IS NULL
	`
	c, err := scanRow(ctx, s.pool.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	return c, err
}

func (s *PGStore) List(ctx context.Context, p ListParams) ([]Customer, error) {
	const q = `
		SELECT id, name, contact_name, email, phone, billing_address, created_at, updated_at
		FROM customers.customers
		WHERE deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := s.pool.Query(ctx, q, p.Limit, p.Offset)
	if err != nil {
		return nil, fmt.Errorf("list query: %w", err)
	}
	defer rows.Close()

	out := make([]Customer, 0)
	for rows.Next() {
		var (
			c    Customer
			addr []byte
		)
		if err := rows.Scan(
			&c.ID, &c.Name, &c.ContactName, &c.Email, &c.Phone,
			&addr, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		if err := json.Unmarshal(addr, &c.BillingAddress); err != nil {
			return nil, fmt.Errorf("unmarshal address: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PGStore) Update(ctx context.Context, id uuid.UUID, in CustomerInput) (Customer, error) {
	addr, err := json.Marshal(in.BillingAddress)
	if err != nil {
		return Customer{}, fmt.Errorf("marshal billing address: %w", err)
	}

	const q = `
		UPDATE customers.customers
		SET name = $2, contact_name = $3, email = $4, phone = $5,
		    billing_address = $6, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, name, contact_name, email, phone, billing_address, created_at, updated_at
	`
	c, err := scanRow(ctx, s.pool.QueryRow(ctx, q, id, in.Name, in.ContactName, in.Email, in.Phone, addr))
	if errors.Is(err, pgx.ErrNoRows) {
		return Customer{}, ErrNotFound
	}
	return c, err
}

func (s *PGStore) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `
		UPDATE customers.customers
		SET deleted_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	tag, err := s.pool.Exec(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// scanRow reads a single result row into a Customer. Shared by Create + Get + Update.
func scanRow(ctx context.Context, row pgx.Row) (Customer, error) {
	var (
		c    Customer
		addr []byte
	)
	err := row.Scan(
		&c.ID, &c.Name, &c.ContactName, &c.Email, &c.Phone,
		&addr, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return Customer{}, err
	}
	if err := json.Unmarshal(addr, &c.BillingAddress); err != nil {
		return Customer{}, fmt.Errorf("unmarshal address: %w", err)
	}
	return c, nil
}
