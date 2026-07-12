package main

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	StatusOpen       OrderStatus = "open"
	StatusInProgress OrderStatus = "in_progress"
	StatusFulfilled  OrderStatus = "fulfilled"
	StatusCancelled  OrderStatus = "cancelled"
)

type Order struct {
	ID             uuid.UUID   `json:"id"`
	QuoteID        uuid.UUID   `json:"quote_id"`
	CustomerID     uuid.UUID   `json:"customer_id"`
	OrderNumber    string      `json:"order_number"`
	Status         OrderStatus `json:"status"`
	ScheduledDate  *time.Time  `json:"scheduled_date,omitempty"`
	FulfilledDate  *time.Time  `json:"fulfilled_date,omitempty"`
	CancelledDate  *time.Time  `json:"cancelled_date,omitempty"`
	TotalAmount    float64     `json:"total_amount"`
	Notes          string      `json:"notes,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
}

// CreateInput accepts a quote_id and pulls remaining fields from the quotes service.
type CreateInput struct {
	QuoteID       uuid.UUID  `json:"quote_id"`
	ScheduledDate *time.Time `json:"scheduled_date,omitempty"`
	Notes         string     `json:"notes,omitempty"`
}

type UpdateInput struct {
	ScheduledDate *time.Time `json:"scheduled_date,omitempty"`
	Notes         string     `json:"notes"`
}

type ListParams struct {
	Limit  int
	Offset int
	Status *OrderStatus
}

// QuoteInfo captures the fields fetched from the quotes service.
// Subset of what quotes returns; we only care about what an order needs.
type QuoteInfo struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	Status     string    `json:"status"`
	Total      float64   `json:"total"`
}
