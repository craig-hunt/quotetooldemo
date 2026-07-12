package main

import (
	"time"

	"github.com/google/uuid"
)

type InvoiceStatus string

const (
	StatusDraft     InvoiceStatus = "draft"
	StatusSent      InvoiceStatus = "sent"
	StatusPaid      InvoiceStatus = "paid"
	StatusOverdue   InvoiceStatus = "overdue"
	StatusCancelled InvoiceStatus = "cancelled"
)

type Invoice struct {
	ID             uuid.UUID     `json:"id"`
	OrderID        uuid.UUID     `json:"order_id"`
	CustomerID     uuid.UUID     `json:"customer_id"`
	InvoiceNumber  string        `json:"invoice_number"`
	Status         InvoiceStatus `json:"status"`
	AmountDue      float64       `json:"amount_due"`
	AmountPaid     float64       `json:"amount_paid"`
	DueDate        time.Time     `json:"due_date"`
	SentAt         *time.Time    `json:"sent_at,omitempty"`
	PaidDate       *time.Time    `json:"paid_date,omitempty"`
	CancelledDate  *time.Time    `json:"cancelled_date,omitempty"`
	Notes          string        `json:"notes,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

type CreateInput struct {
	OrderID uuid.UUID `json:"order_id"`
	DueDays int       `json:"due_days"` // e.g., 30 for net-30
	Notes   string    `json:"notes,omitempty"`
}

type ListParams struct {
	Limit  int
	Offset int
	Status *InvoiceStatus
}

// OrderInfo captures the subset of an order the invoices service needs.
type OrderInfo struct {
	ID          uuid.UUID `json:"id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	Status      string    `json:"status"`
	TotalAmount float64   `json:"total_amount"`
}
