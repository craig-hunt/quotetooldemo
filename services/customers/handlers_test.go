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
	testCustomerIDStr = "11111111-1111-1111-1111-111111111111"
	testHealthPath    = "/health"
	testCustomersPath = "/customers"
	testInvalidUUID   = "not-a-uuid"
	testValidName     = "Acme Paving"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func newTestServer(m *MockStore) http.Handler {
	return NewHandlers(m, discardLogger()).Routes()
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

func validInput() CustomerInput {
	return CustomerInput{
		Name:        testValidName,
		ContactName: "John Smith",
		Email:       "acme@example.com",
		Phone:       "555-0100",
		BillingAddress: Address{
			Street: "123 Main St",
			City:   "Boston",
			State:  "MA",
			Zip:    "02101",
		},
	}
}

func TestHealth(t *testing.T) {
	srv := newTestServer(&MockStore{})
	rec := doJSON(t, srv, http.MethodGet, testHealthPath, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body[StatusKey] != StatusOK {
		t.Errorf("status = %q, want %q", body[StatusKey], StatusOK)
	}
}

func TestCreate_Success(t *testing.T) {
	stored := Customer{
		ID:   uuid.MustParse(testCustomerIDStr),
		Name: testValidName,
	}
	m := &MockStore{
		CreateFn: func(ctx context.Context, in CustomerInput) (Customer, error) {
			return stored, nil
		},
	}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPost, testCustomersPath, validInput())

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
	if m.LastCreateInput.Name != testValidName {
		t.Errorf("input.Name = %q, want %q", m.LastCreateInput.Name, testValidName)
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	srv := newTestServer(&MockStore{})
	req := httptest.NewRequest(http.MethodPost, testCustomersPath, strings.NewReader("{"))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidJSON {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidJSON)
	}
}

func TestCreate_NameRequired(t *testing.T) {
	srv := newTestServer(&MockStore{})
	in := validInput()
	in.Name = ""
	rec := doJSON(t, srv, http.MethodPost, testCustomersPath, in)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNameRequired {
		t.Errorf("error = %q, want %q", got, ErrMsgNameRequired)
	}
}

func TestCreate_StoreError(t *testing.T) {
	m := &MockStore{CreateFn: func(ctx context.Context, in CustomerInput) (Customer, error) {
		return Customer{}, errors.New("boom")
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPost, testCustomersPath, validInput())

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgCreateFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgCreateFailed)
	}
}

func TestList_LimitBounds(t *testing.T) {
	cases := []struct {
		name  string
		limit string
		want  int
	}{
		{"omitted defaults", "", DefaultPageLimit},
		{"zero rejected", "0", DefaultPageLimit},
		{"negative rejected", "-5", DefaultPageLimit},
		{"non-numeric rejected", "many", DefaultPageLimit},
		{"in-range accepted", "77", 77},
		{"max accepted", "200", MaxPageLimit},
		{"over max rejected", "201", DefaultPageLimit},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Customer, error) {
				return []Customer{}, nil
			}}
			srv := newTestServer(m)
			url := testCustomersPath
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

func TestList_OffsetRejectsNegative(t *testing.T) {
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Customer, error) {
		return []Customer{}, nil
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testCustomersPath+"?"+QueryParamOffset+"=-1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if m.LastListParams.Offset != 0 {
		t.Errorf("negative offset must default to 0, got %d", m.LastListParams.Offset)
	}
}

func TestList_OffsetAccepted(t *testing.T) {
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Customer, error) {
		return []Customer{}, nil
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testCustomersPath+"?"+QueryParamOffset+"=25", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if m.LastListParams.Offset != 25 {
		t.Errorf("offset = %d, want 25", m.LastListParams.Offset)
	}
}

func TestList_StoreError(t *testing.T) {
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Customer, error) {
		return nil, errors.New("boom")
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testCustomersPath, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgListFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgListFailed)
	}
}

func TestGet_InvalidID(t *testing.T) {
	srv := newTestServer(&MockStore{})
	rec := doJSON(t, srv, http.MethodGet, testCustomersPath+"/"+testInvalidUUID, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestGet_NotFound(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Customer, error) {
		return Customer{}, ErrNotFound
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testCustomersPath+"/"+testCustomerIDStr, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgNotFound)
	}
}

func TestGet_StoreError(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Customer, error) {
		return Customer{}, errors.New("boom")
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testCustomersPath+"/"+testCustomerIDStr, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgGetFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgGetFailed)
	}
}

func TestGet_Success(t *testing.T) {
	stored := Customer{ID: uuid.MustParse(testCustomerIDStr), Name: testValidName}
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Customer, error) {
		return stored, nil
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testCustomersPath+"/"+testCustomerIDStr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestUpdate_InvalidID(t *testing.T) {
	srv := newTestServer(&MockStore{})
	rec := doJSON(t, srv, http.MethodPut, testCustomersPath+"/"+testInvalidUUID, validInput())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	srv := newTestServer(&MockStore{})
	req := httptest.NewRequest(http.MethodPut, testCustomersPath+"/"+testCustomerIDStr,
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

func TestUpdate_NameRequired(t *testing.T) {
	srv := newTestServer(&MockStore{})
	in := validInput()
	in.Name = ""
	rec := doJSON(t, srv, http.MethodPut, testCustomersPath+"/"+testCustomerIDStr, in)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNameRequired {
		t.Errorf("error = %q, want %q", got, ErrMsgNameRequired)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	m := &MockStore{UpdateFn: func(ctx context.Context, id uuid.UUID, in CustomerInput) (Customer, error) {
		return Customer{}, ErrNotFound
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPut, testCustomersPath+"/"+testCustomerIDStr, validInput())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgNotFound)
	}
}

func TestUpdate_StoreError(t *testing.T) {
	m := &MockStore{UpdateFn: func(ctx context.Context, id uuid.UUID, in CustomerInput) (Customer, error) {
		return Customer{}, errors.New("boom")
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPut, testCustomersPath+"/"+testCustomerIDStr, validInput())
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgUpdateFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgUpdateFailed)
	}
}

func TestUpdate_Success(t *testing.T) {
	stored := Customer{ID: uuid.MustParse(testCustomerIDStr), Name: testValidName}
	m := &MockStore{UpdateFn: func(ctx context.Context, id uuid.UUID, in CustomerInput) (Customer, error) {
		return stored, nil
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPut, testCustomersPath+"/"+testCustomerIDStr, validInput())
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestDelete_InvalidID(t *testing.T) {
	srv := newTestServer(&MockStore{})
	rec := doJSON(t, srv, http.MethodDelete, testCustomersPath+"/"+testInvalidUUID, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestDelete_NotFound(t *testing.T) {
	m := &MockStore{DeleteFn: func(ctx context.Context, id uuid.UUID) error {
		return ErrNotFound
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodDelete, testCustomersPath+"/"+testCustomerIDStr, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgNotFound)
	}
}

func TestDelete_StoreError(t *testing.T) {
	m := &MockStore{DeleteFn: func(ctx context.Context, id uuid.UUID) error {
		return errors.New("boom")
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodDelete, testCustomersPath+"/"+testCustomerIDStr, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgDeleteFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgDeleteFailed)
	}
}

func TestDelete_Success(t *testing.T) {
	m := &MockStore{DeleteFn: func(ctx context.Context, id uuid.UUID) error {
		return nil
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodDelete, testCustomersPath+"/"+testCustomerIDStr, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("expected empty body, got %q", rec.Body.String())
	}
}
