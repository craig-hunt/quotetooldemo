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

func TestQuotesClient_Success(t *testing.T) {
	want := QuoteInfo{
		ID:         uuid.New(),
		CustomerID: uuid.New(),
		Status:     ExternalQuoteStatusAccepted,
		Total:      1234.56,
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

	c := NewQuotesClient(srv.URL)
	got, err := c.Get(context.Background(), want.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID != want.ID || got.Total != want.Total || got.Status != want.Status {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestQuotesClient_NotFoundMapsToSentinel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewQuotesClient(srv.URL)
	_, err := c.Get(context.Background(), uuid.New())
	if !errors.Is(err, ErrQuoteNotFound) {
		t.Errorf("err = %v, want ErrQuoteNotFound", err)
	}
}

func TestQuotesClient_ServerErrorSurfaces(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewQuotesClient(srv.URL)
	_, err := c.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrQuoteNotFound) {
		t.Errorf("500 must not map to ErrQuoteNotFound")
	}
}

func TestQuotesClient_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json"))
	}))
	defer srv.Close()

	c := NewQuotesClient(srv.URL)
	_, err := c.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected decode error, got nil")
	}
}

func TestQuotesClient_ConnectionRefused(t *testing.T) {
	// Point the client at a URL nothing is listening on. http.Client.Do
	// returns a dial error, which our client wraps.
	c := NewQuotesClient("http://127.0.0.1:1")
	_, err := c.Get(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for dead endpoint")
	}
	if errors.Is(err, ErrQuoteNotFound) {
		t.Errorf("dial error must not map to ErrQuoteNotFound")
	}
}
