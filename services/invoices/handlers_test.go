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
	testOrderIDStr    = "11111111-1111-1111-1111-111111111111"
	testCustomerIDStr = "22222222-2222-2222-2222-222222222222"
	testInvoiceIDStr  = "33333333-3333-3333-3333-333333333333"
	testHealthPath    = "/health"
	testInvoicesPath  = "/invoices"
	testInvalidUUID   = "not-a-uuid"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func fakeOrdersServer(t *testing.T, status int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, ContentTypeJSON)
		w.WriteHeader(status)
		if body != nil {
			_ = json.NewEncoder(w).Encode(body)
		}
	}))
}

func newTestServer(m *MockStore, oc *OrdersClient) http.Handler {
	return NewHandlers(m, oc, discardLogger()).Routes()
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

func fulfilledOrder() OrderInfo {
	return OrderInfo{
		ID:          uuid.MustParse(testOrderIDStr),
		CustomerID:  uuid.MustParse(testCustomerIDStr),
		Status:      ExternalOrderStatusFulfilled,
		TotalAmount: 2933.22,
	}
}

func TestHealth(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testHealthPath, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestCreate_OrderIDRequired(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath, CreateInput{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgOrderIDRequired {
		t.Errorf("error = %q, want %q", got, ErrMsgOrderIDRequired)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewOrdersClient("http://never"))
	req := httptest.NewRequest(http.MethodPost, testInvoicesPath, strings.NewReader("{"))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidJSON {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidJSON)
	}
}

func TestCreate_OrderNotFound(t *testing.T) {
	os := fakeOrdersServer(t, http.StatusNotFound, nil)
	defer os.Close()
	srv := newTestServer(&MockStore{}, NewOrdersClient(os.URL))
	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath, CreateInput{OrderID: uuid.New()})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgOrderNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgOrderNotFound)
	}
}

func TestCreate_OrdersUnavailable(t *testing.T) {
	os := fakeOrdersServer(t, http.StatusInternalServerError, nil)
	defer os.Close()
	srv := newTestServer(&MockStore{}, NewOrdersClient(os.URL))
	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath, CreateInput{OrderID: uuid.New()})
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want 502", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgOrdersUnavailable {
		t.Errorf("error = %q, want %q", got, ErrMsgOrdersUnavailable)
	}
}

func TestCreate_RejectsNonFulfilledOrder(t *testing.T) {
	o := fulfilledOrder()
	o.Status = "open"
	os := fakeOrdersServer(t, http.StatusOK, o)
	defer os.Close()
	srv := newTestServer(&MockStore{}, NewOrdersClient(os.URL))
	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath, CreateInput{OrderID: o.ID})
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgOrderNotFulfilled {
		t.Errorf("error = %q, want %q", got, ErrMsgOrderNotFulfilled)
	}
}

func TestCreate_DueDaysDefaultAppliedWhenZero(t *testing.T) {
	o := fulfilledOrder()
	os := fakeOrdersServer(t, http.StatusOK, o)
	defer os.Close()
	m := &MockStore{CreateFn: func(ctx context.Context, in CreateInput, oi OrderInfo) (Invoice, error) {
		return Invoice{ID: uuid.New()}, nil
	}}
	srv := newTestServer(m, NewOrdersClient(os.URL))

	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath, CreateInput{OrderID: o.ID, DueDays: 0})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %s", rec.Code, rec.Body.String())
	}
	if m.LastCreateInput.DueDays != DefaultDueDays {
		t.Errorf("DueDays = %d, want default %d", m.LastCreateInput.DueDays, DefaultDueDays)
	}
}

func TestCreate_DueDaysDefaultAppliedWhenNegative(t *testing.T) {
	o := fulfilledOrder()
	os := fakeOrdersServer(t, http.StatusOK, o)
	defer os.Close()
	m := &MockStore{CreateFn: func(ctx context.Context, in CreateInput, oi OrderInfo) (Invoice, error) {
		return Invoice{ID: uuid.New()}, nil
	}}
	srv := newTestServer(m, NewOrdersClient(os.URL))

	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath, CreateInput{OrderID: o.ID, DueDays: -5})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if m.LastCreateInput.DueDays != DefaultDueDays {
		t.Errorf("DueDays = %d, want default %d", m.LastCreateInput.DueDays, DefaultDueDays)
	}
}

func TestCreate_DueDaysExplicit(t *testing.T) {
	o := fulfilledOrder()
	os := fakeOrdersServer(t, http.StatusOK, o)
	defer os.Close()
	m := &MockStore{CreateFn: func(ctx context.Context, in CreateInput, oi OrderInfo) (Invoice, error) {
		return Invoice{ID: uuid.New()}, nil
	}}
	srv := newTestServer(m, NewOrdersClient(os.URL))

	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath, CreateInput{OrderID: o.ID, DueDays: 60})
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if m.LastCreateInput.DueDays != 60 {
		t.Errorf("DueDays = %d, want 60", m.LastCreateInput.DueDays)
	}
}

func TestCreate_StoreError(t *testing.T) {
	o := fulfilledOrder()
	os := fakeOrdersServer(t, http.StatusOK, o)
	defer os.Close()
	m := &MockStore{CreateFn: func(ctx context.Context, in CreateInput, oi OrderInfo) (Invoice, error) {
		return Invoice{}, errors.New("boom")
	}}
	srv := newTestServer(m, NewOrdersClient(os.URL))
	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath, CreateInput{OrderID: o.ID})
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgCreateFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgCreateFailed)
	}
}

func TestList_QueryParamsFlow(t *testing.T) {
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Invoice, error) {
		return []Invoice{}, nil
	}}
	srv := newTestServer(m, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet,
		testInvoicesPath+"?"+QueryParamLimit+"=25&"+QueryParamOffset+"=10&"+QueryParamStatus+"=sent", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if m.LastListParams.Limit != 25 {
		t.Errorf("limit = %d, want 25", m.LastListParams.Limit)
	}
	if m.LastListParams.Offset != 10 {
		t.Errorf("offset = %d, want 10", m.LastListParams.Offset)
	}
	if m.LastListParams.Status == nil || *m.LastListParams.Status != StatusSent {
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
			m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Invoice, error) {
				return []Invoice{}, nil
			}}
			srv := newTestServer(m, NewOrdersClient("http://never"))
			url := testInvoicesPath
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
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Invoice, error) {
		return nil, errors.New("boom")
	}}
	srv := newTestServer(m, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testInvoicesPath, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgListFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgListFailed)
	}
}

func TestGet_InvalidID(t *testing.T) {
	srv := newTestServer(&MockStore{}, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testInvoicesPath+"/"+testInvalidUUID, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestGet_NotFound(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Invoice, error) {
		return Invoice{}, ErrNotFound
	}}
	srv := newTestServer(m, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testInvoicesPath+"/"+testInvoiceIDStr, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgNotFound)
	}
}

func TestGet_StoreError(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Invoice, error) {
		return Invoice{}, errors.New("boom")
	}}
	srv := newTestServer(m, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testInvoicesPath+"/"+testInvoiceIDStr, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgGetFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgGetFailed)
	}
}

func TestGet_Success(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Invoice, error) {
		return Invoice{ID: id, Status: StatusDraft}, nil
	}}
	srv := newTestServer(m, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodGet, testInvoicesPath+"/"+testInvoiceIDStr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestTransition_Routes(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		wantStatus InvoiceStatus
	}{
		{"send", "/send", StatusSent},
		{"mark_paid", "/mark_paid", StatusPaid},
		{"cancel", "/cancel", StatusCancelled},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to InvoiceStatus) (Invoice, error) {
				return Invoice{ID: id, Status: to}, nil
			}}
			srv := newTestServer(m, NewOrdersClient("http://never"))
			rec := doJSON(t, srv, http.MethodPost, testInvoicesPath+"/"+testInvoiceIDStr+tc.path, nil)
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
	srv := newTestServer(&MockStore{}, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath+"/"+testInvalidUUID+"/send", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestTransition_NotFound(t *testing.T) {
	m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to InvoiceStatus) (Invoice, error) {
		return Invoice{}, ErrNotFound
	}}
	srv := newTestServer(m, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath+"/"+testInvoiceIDStr+"/send", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgNotFound)
	}
}

func TestTransition_InvalidStatus(t *testing.T) {
	m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to InvoiceStatus) (Invoice, error) {
		return Invoice{}, ErrInvalidStatus
	}}
	srv := newTestServer(m, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath+"/"+testInvoiceIDStr+"/mark_paid", nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidTransition {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidTransition)
	}
}

func TestTransition_StoreError(t *testing.T) {
	m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to InvoiceStatus) (Invoice, error) {
		return Invoice{}, errors.New("boom")
	}}
	srv := newTestServer(m, NewOrdersClient("http://never"))
	rec := doJSON(t, srv, http.MethodPost, testInvoicesPath+"/"+testInvoiceIDStr+"/cancel", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgTransitionFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgTransitionFailed)
	}
}
