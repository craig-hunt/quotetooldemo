package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

const (
	testQuoteIDStr    = "11111111-1111-1111-1111-111111111111"
	testCustomerIDStr = "22222222-2222-2222-2222-222222222222"
	testOrderIDStr    = "33333333-3333-3333-3333-333333333333"
	testHealthPath    = "/health"
	testOrdersPath    = "/orders"
	testInvalidUUID   = "not-a-uuid"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeQuotesServer returns an http.Handler that always responds with the given
// status + body. Used to build a QuotesClient wired to a test server.
func fakeQuotesServer(t *testing.T, status int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}))
}

func newTestServer(m *MockStore, qc *QuotesClient) http.Handler {
	return NewHandlers(m, qc, discardLogger()).Routes()
}

func doJSON(t *testing.T, srv http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set(HeaderContentType, ContentTypeJSON)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	return rec
}

func decodeErrorBody(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v (raw=%q)", err, rec.Body.String())
	}
	return body[ErrorKey]
}

func acceptedQuote() QuoteInfo {
	return QuoteInfo{
		ID:         uuid.MustParse(testQuoteIDStr),
		CustomerID: uuid.MustParse(testCustomerIDStr),
		Status:     ExternalQuoteStatusAccepted,
		Total:      2933.22,
	}
}

func TestHealth(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testHealthPath, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestCreate_QuoteIDRequired(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath, CreateInput{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgQuoteIDRequired {
		t.Errorf("error = %q, want %q", got, ErrMsgQuoteIDRequired)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewQuotesClient("http://never"))
	req := httptest.NewRequest(http.MethodPost, testOrdersPath, strings.NewReader("{"))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidJSON {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidJSON)
	}
}

func TestCreate_QuoteNotFound(t *testing.T) {
	qs := fakeQuotesServer(t, http.StatusNotFound, nil)
	defer qs.Close()
	srv := newTestServer(&MockStore{}, NewQuotesClient(qs.URL))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath, CreateInput{QuoteID: uuid.New()})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgQuoteNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgQuoteNotFound)
	}
}

func TestCreate_QuotesServiceUnavailable(t *testing.T) {
	qs := fakeQuotesServer(t, http.StatusInternalServerError, map[string]string{"error": "boom"})
	defer qs.Close()
	srv := newTestServer(&MockStore{}, NewQuotesClient(qs.URL))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath, CreateInput{QuoteID: uuid.New()})
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgQuotesUnavailable {
		t.Errorf("error = %q, want %q", got, ErrMsgQuotesUnavailable)
	}
}

func TestCreate_RejectsNonAcceptedQuote(t *testing.T) {
	q := acceptedQuote()
	q.Status = "draft"
	qs := fakeQuotesServer(t, http.StatusOK, q)
	defer qs.Close()
	srv := newTestServer(&MockStore{}, NewQuotesClient(qs.URL))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath, CreateInput{QuoteID: q.ID})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgQuoteNotAccepted {
		t.Errorf("error = %q, want %q", got, ErrMsgQuoteNotAccepted)
	}
}

func TestCreate_Success(t *testing.T) {
	q := acceptedQuote()
	qs := fakeQuotesServer(t, http.StatusOK, q)
	defer qs.Close()

	created := Order{ID: uuid.MustParse(testOrderIDStr), Status: StatusOpen, QuoteID: q.ID}
	m := &MockStore{CreateFn: func(ctx context.Context, in CreateInput, qi QuoteInfo) (Order, error) {
		return created, nil
	}}
	srv := newTestServer(m, NewQuotesClient(qs.URL))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath, CreateInput{QuoteID: q.ID})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	if m.LastCreateQuote.ID != q.ID {
		t.Errorf("mock got quote id %v, want %v", m.LastCreateQuote.ID, q.ID)
	}
	if m.LastCreateQuote.Total != q.Total {
		t.Errorf("mock got total %v, want %v", m.LastCreateQuote.Total, q.Total)
	}
}

func TestCreate_StoreError(t *testing.T) {
	q := acceptedQuote()
	qs := fakeQuotesServer(t, http.StatusOK, q)
	defer qs.Close()
	m := &MockStore{CreateFn: func(ctx context.Context, in CreateInput, qi QuoteInfo) (Order, error) {
		return Order{}, errors.New("boom")
	}}
	srv := newTestServer(m, NewQuotesClient(qs.URL))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath, CreateInput{QuoteID: q.ID})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgCreateFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgCreateFailed)
	}
}

func TestList_QueryParamsFlow(t *testing.T) {
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Order, error) {
		return []Order{}, nil
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet,
		testOrdersPath+"?"+QueryParamLimit+"=25&"+QueryParamOffset+"=10&"+QueryParamStatus+"=open", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if m.LastListParams.Limit != 25 {
		t.Errorf("limit = %d, want 25", m.LastListParams.Limit)
	}
	if m.LastListParams.Offset != 10 {
		t.Errorf("offset = %d, want 10", m.LastListParams.Offset)
	}
	if m.LastListParams.Status == nil || *m.LastListParams.Status != StatusOpen {
		t.Errorf("status filter not applied: %+v", m.LastListParams.Status)
	}
}

func TestList_LimitBounds(t *testing.T) {
	cases := []struct {
		limit string
		want  int
	}{
		{"", DefaultPageLimit},
		{"0", DefaultPageLimit},
		{"-5", DefaultPageLimit},
		{"200", MaxPageLimit},
		{"201", DefaultPageLimit},
	}
	for _, tc := range cases {
		t.Run(tc.limit, func(t *testing.T) {
			m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Order, error) {
				return []Order{}, nil
			}}
			srv := newTestServer(m, NewQuotesClient("http://never"))
			url := testOrdersPath
			if tc.limit != "" {
				url += "?" + QueryParamLimit + "=" + tc.limit
			}
			rec := doJSON(t, srv, http.MethodGet, url, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d", rec.Code)
			}
			if m.LastListParams.Limit != tc.want {
				t.Errorf("limit = %d, want %d", m.LastListParams.Limit, tc.want)
			}
		})
	}
}

func TestList_StoreError(t *testing.T) {
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Order, error) {
		return nil, errors.New("boom")
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testOrdersPath, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgListFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgListFailed)
	}
}

func TestGet_InvalidID(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testOrdersPath+"/"+testInvalidUUID, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestGet_NotFound(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Order, error) {
		return Order{}, ErrNotFound
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testOrdersPath+"/"+testOrderIDStr, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgOrderNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgOrderNotFound)
	}
}

func TestGet_StoreError(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Order, error) {
		return Order{}, errors.New("boom")
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testOrdersPath+"/"+testOrderIDStr, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgGetFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgGetFailed)
	}
}

func TestGet_Success(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Order, error) {
		return Order{ID: id, Status: StatusOpen}, nil
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testOrdersPath+"/"+testOrderIDStr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestUpdate_InvalidID(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodPut, testOrdersPath+"/"+testInvalidUUID, UpdateInput{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewQuotesClient("http://never"))
	req := httptest.NewRequest(http.MethodPut, testOrdersPath+"/"+testOrderIDStr,
		strings.NewReader("{"))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidJSON {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidJSON)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	m := &MockStore{UpdateFn: func(ctx context.Context, id uuid.UUID, in UpdateInput) (Order, error) {
		return Order{}, ErrNotFound
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodPut, testOrdersPath+"/"+testOrderIDStr, UpdateInput{})
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgOrderNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgOrderNotFound)
	}
}

func TestUpdate_StoreError(t *testing.T) {
	m := &MockStore{UpdateFn: func(ctx context.Context, id uuid.UUID, in UpdateInput) (Order, error) {
		return Order{}, errors.New("boom")
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodPut, testOrdersPath+"/"+testOrderIDStr, UpdateInput{})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgUpdateFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgUpdateFailed)
	}
}

func TestUpdate_Success(t *testing.T) {
	m := &MockStore{UpdateFn: func(ctx context.Context, id uuid.UUID, in UpdateInput) (Order, error) {
		return Order{ID: id, Status: StatusOpen}, nil
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodPut, testOrdersPath+"/"+testOrderIDStr, UpdateInput{})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestTransition_Routes(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		wantStatus OrderStatus
	}{
		{"fulfill", "/fulfill", StatusFulfilled},
		{"cancel", "/cancel", StatusCancelled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to OrderStatus) (Order, error) {
				return Order{ID: id, Status: to}, nil
			}}
			srv := newTestServer(m, NewQuotesClient("http://never"))
			rec := doJSON(t, srv, http.MethodPost, testOrdersPath+"/"+testOrderIDStr+tc.path, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d", rec.Code)
			}
			if m.LastTransitTo != tc.wantStatus {
				t.Errorf("target = %s, want %s", m.LastTransitTo, tc.wantStatus)
			}
		})
	}
}

func TestTransition_InvalidID(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath+"/"+testInvalidUUID+"/fulfill", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestTransition_NotFound(t *testing.T) {
	m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to OrderStatus) (Order, error) {
		return Order{}, ErrNotFound
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath+"/"+testOrderIDStr+"/fulfill", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgOrderNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgOrderNotFound)
	}
}

func TestTransition_InvalidStatus(t *testing.T) {
	m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to OrderStatus) (Order, error) {
		return Order{}, ErrInvalidStatus
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath+"/"+testOrderIDStr+"/fulfill", nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidTransition {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidTransition)
	}
}

func TestTransition_StoreError(t *testing.T) {
	m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to OrderStatus) (Order, error) {
		return Order{}, errors.New("boom")
	}}
	srv := newTestServer(m, NewQuotesClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testOrdersPath+"/"+testOrderIDStr+"/cancel", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgTransitionFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgTransitionFailed)
	}
}
