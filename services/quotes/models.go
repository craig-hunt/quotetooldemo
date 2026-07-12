package main

import (
	"time"

	"github.com/google/uuid"
)

// Mix types match the enum in migrations/02_quotes.sql.
type MixType string

const (
	MixHMABase    MixType = "hma_base"
	MixHMASurface MixType = "hma_surface"
	MixSuperpave  MixType = "superpave"
	MixWarmMix    MixType = "warm_mix"
)

// Quote statuses match the enum in migrations/02_quotes.sql.
type QuoteStatus string

const (
	StatusDraft    QuoteStatus = "draft"
	StatusSent     QuoteStatus = "sent"
	StatusAccepted QuoteStatus = "accepted"
	StatusRejected QuoteStatus = "rejected"
	StatusExpired  QuoteStatus = "expired"
)

// LineItem is one row on a quote. tons and line_total are computed
// server-side from area × depth × mix density × unit price.
type LineItem struct {
	ID              uuid.UUID `json:"id,omitempty"`
	AreaSqft        float64   `json:"area_sqft"`
	DepthInches     float64   `json:"depth_inches"`
	MixType         MixType   `json:"mix_type"`
	UnitPricePerTon float64   `json:"unit_price_per_ton"`
	Tons            float64   `json:"tons"`
	LineTotal       float64   `json:"line_total"`
	Position        int       `json:"position"`
}

// Quote holds the quote header + its line items.
type Quote struct {
	ID              uuid.UUID   `json:"id"`
	CustomerID      uuid.UUID   `json:"customer_id"`
	ProjectName     string      `json:"project_name"`
	ProjectAddress  string      `json:"project_address"`
	Status          QuoteStatus `json:"status"`
	Subtotal        float64     `json:"subtotal"`
	TaxRate         float64     `json:"tax_rate"`
	TaxAmount       float64     `json:"tax_amount"`
	MarkupRate      float64     `json:"markup_rate"`
	MarkupAmount    float64     `json:"markup_amount"`
	Total           float64     `json:"total"`
	Notes           string      `json:"notes,omitempty"`
	AcceptedAt      *time.Time  `json:"accepted_at,omitempty"`
	RejectedAt      *time.Time  `json:"rejected_at,omitempty"`
	SentAt          *time.Time  `json:"sent_at,omitempty"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	LineItems       []LineItem  `json:"line_items"`
}

// QuoteInput is the client-set portion of a create/update request.
type QuoteInput struct {
	CustomerID     uuid.UUID       `json:"customer_id"`
	ProjectName    string          `json:"project_name"`
	ProjectAddress string          `json:"project_address"`
	TaxRate        float64         `json:"tax_rate"`
	MarkupRate     float64         `json:"markup_rate"`
	Notes          string          `json:"notes"`
	LineItems      []LineItemInput `json:"line_items"`
}

// LineItemInput carries the fields a client sets. tons + line_total are computed.
type LineItemInput struct {
	AreaSqft        float64 `json:"area_sqft"`
	DepthInches     float64 `json:"depth_inches"`
	MixType         MixType `json:"mix_type"`
	UnitPricePerTon float64 `json:"unit_price_per_ton"`
}

type ListParams struct {
	Limit      int
	Offset     int
	CustomerID *uuid.UUID
	Status     *QuoteStatus
}
