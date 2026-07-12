package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type OrdersClient struct {
	baseURL string
	http    *http.Client
}

func NewOrdersClient(baseURL string) *OrdersClient {
	return &OrdersClient{baseURL: baseURL, http: &http.Client{Timeout: HTTPClientTimeout}}
}

func (c *OrdersClient) Get(ctx context.Context, id uuid.UUID) (OrderInfo, error) {
	url := fmt.Sprintf("%s/orders/%s", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return OrderInfo{}, fmt.Errorf("build request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return OrderInfo{}, fmt.Errorf("call orders: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return OrderInfo{}, ErrOrderNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return OrderInfo{}, fmt.Errorf("orders returned %d", resp.StatusCode)
	}
	var o OrderInfo
	if err := json.NewDecoder(resp.Body).Decode(&o); err != nil {
		return OrderInfo{}, fmt.Errorf("decode order: %w", err)
	}
	return o, nil
}
