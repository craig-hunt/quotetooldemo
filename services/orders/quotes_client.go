package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// QuotesClient calls the quotes service to fetch quote details when an order
// is being created. Standard client pattern: struct wraps http.Client + baseURL.
type QuotesClient struct {
	baseURL string
	http    *http.Client
}

func NewQuotesClient(baseURL string) *QuotesClient {
	return &QuotesClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: HTTPClientTimeout},
	}
}

// Get fetches a quote by ID. Returns the subset of fields orders needs.
// context.Context propagates cancellation from the incoming request.
func (c *QuotesClient) Get(ctx context.Context, id uuid.UUID) (QuoteInfo, error) {
	url := fmt.Sprintf("%s/quotes/%s", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return QuoteInfo{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return QuoteInfo{}, fmt.Errorf("call quotes service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return QuoteInfo{}, ErrQuoteNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return QuoteInfo{}, fmt.Errorf("quotes service returned %d", resp.StatusCode)
	}

	var q QuoteInfo
	if err := json.NewDecoder(resp.Body).Decode(&q); err != nil {
		return QuoteInfo{}, fmt.Errorf("decode quote: %w", err)
	}
	return q, nil
}
