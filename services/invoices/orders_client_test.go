package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestOrdersClient_Success(t *testing.T) {
	want := OrderInfo{
		ID:          uuid.New(),
		CustomerID:  uuid.New(),
		Status:      ExternalOrderStatusFulfilled,
		TotalAmount: 9800.00,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewOrdersClient(srv.URL)
	got, err := c.Get(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != want.ID || got.TotalAmount != want.TotalAmount || got.Status != want.Status {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestOrdersClient_NotFoundMapsToSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewOrdersClient(srv.URL)
	_, err := c.Get(context.Background(), uuid.New())
	if !errors.Is(err, ErrOrderNotFound) {
		t.Errorf("err = %v, want ErrOrderNotFound", err)
	}
}

func TestOrdersClient_ServerErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewOrdersClient(srv.URL)
	_, err := c.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error")
	}
	if errors.Is(err, ErrOrderNotFound) {
		t.Errorf("500 must not map to ErrOrderNotFound")
	}
}

func TestOrdersClient_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json"))
	}))
	defer srv.Close()

	c := NewOrdersClient(srv.URL)
	_, err := c.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected decode error")
	}
}

func TestOrdersClient_ConnectionRefused(t *testing.T) {
	c := NewOrdersClient("http://127.0.0.1:1")
	_, err := c.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for dead endpoint")
	}
	if errors.Is(err, ErrOrderNotFound) {
		t.Errorf("dial error must not map to ErrOrderNotFound")
	}
}
