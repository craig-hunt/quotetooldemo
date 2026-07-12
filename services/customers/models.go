package main

import (
	"time"

	"github.com/google/uuid"
)

// Address models a customer's billing address. Stored as JSONB in Postgres.
type Address struct {
	Street string `json:"street"`
	City   string `json:"city"`
	State  string `json:"state"`
	Zip    string `json:"zip"`
}

// Customer models a paving contractor's customer.
type Customer struct {
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	ContactName    string    `json:"contact_name,omitempty"`
	Email          string    `json:"email,omitempty"`
	Phone          string    `json:"phone,omitempty"`
	BillingAddress Address   `json:"billing_address"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CustomerInput carries the fields a client can set on create or update.
// Kept separate from Customer so the server owns id + timestamps.
type CustomerInput struct {
	Name           string  `json:"name"`
	ContactName    string  `json:"contact_name"`
	Email          string  `json:"email"`
	Phone          string  `json:"phone"`
	BillingAddress Address `json:"billing_address"`
}

// ListParams governs pagination on the list endpoint.
type ListParams struct {
	Limit  int
	Offset int
}
