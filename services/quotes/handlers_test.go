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
	testQuoteIDStr    = "22222222-2222-2222-2222-222222222222"
	testHealthPath    = "/health"
	testQuotesPath    = "/quotes"
	testInvalidUUID   = "not-a-uuid"
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
		t.Fatalf("decode error body: %v (raw=%q)", err, rec.Body.String())
	}
	return body[ErrorKey]
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
		t.Errorf("status key = %q, want %q", body[StatusKey], StatusOK)
	}
}

func TestCreate_Success(t *testing.T) {
	custID := uuid.MustParse(testCustomerIDStr)
	created := Quote{
		ID:          uuid.MustParse(testQuoteIDStr),
		CustomerID:  custID,
		ProjectName: "Main St",
		Status:      StatusDraft,
	}
	m := &MockStore{
		CreateFn: func(ctx context.Context, in QuoteInput) (Quote, error) {
			return created, nil
		},
	}
	srv := newTestServer(m)

	in := validQuoteInput()
	in.CustomerID = custID
	rec := doJSON(t, srv, http.MethodPost, testQuotesPath, in)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201: body=%s", rec.Code, rec.Body.String())
	}
	var got Quote
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("id = %v, want %v", got.ID, created.ID)
	}
	if m.LastCreateInput.ProjectName != in.ProjectName {
		t.Errorf("mock did not receive project name")
	}
}

func TestCreate_InvalidJSON(t *testing.T) {
	srv := newTestServer(&MockStore{})
	req := httptest.NewRequest(http.MethodPost, testQuotesPath, strings.NewReader("{not-json"))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidJSON {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidJSON)
	}
}

func TestCreate_ValidationError(t *testing.T) {
	srv := newTestServer(&MockStore{})
	in := validQuoteInput()
	in.ProjectName = "" // triggers ErrMsgProjectNameRequired
	rec := doJSON(t, srv, http.MethodPost, testQuotesPath, in)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgProjectNameRequired {
		t.Errorf("error = %q, want %q", got, ErrMsgProjectNameRequired)
	}
}

func TestCreate_StoreError(t *testing.T) {
	m := &MockStore{
		CreateFn: func(ctx context.Context, in QuoteInput) (Quote, error) {
			return Quote{}, errors.New("db exploded")
		},
	}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPost, testQuotesPath, validQuoteInput())

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgCreateFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgCreateFailed)
	}
}

func TestList_QueryParamsFlowToStore(t *testing.T) {
	m := &MockStore{
		ListFn: func(ctx context.Context, p ListParams) ([]Quote, error) {
			return []Quote{}, nil
		},
	}
	srv := newTestServer(m)

	url := testQuotesPath + "?" +
		QueryParamLimit + "=25&" +
		QueryParamOffset + "=10&" +
		QueryParamCustomerID + "=" + testCustomerIDStr + "&" +
		QueryParamStatus + "=sent"
	rec := doJSON(t, srv, http.MethodGet, url, nil)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if m.LastListParams.Limit != 25 {
		t.Errorf("limit = %d, want 25", m.LastListParams.Limit)
	}
	if m.LastListParams.Offset != 10 {
		t.Errorf("offset = %d, want 10", m.LastListParams.Offset)
	}
	if m.LastListParams.CustomerID == nil || m.LastListParams.CustomerID.String() != testCustomerIDStr {
		t.Errorf("customer id filter not applied: %+v", m.LastListParams.CustomerID)
	}
	if m.LastListParams.Status == nil || *m.LastListParams.Status != StatusSent {
		t.Errorf("status filter not applied: %+v", m.LastListParams.Status)
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
			m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Quote, error) {
				return []Quote{}, nil
			}}
			srv := newTestServer(m)
			url := testQuotesPath
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
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Quote, error) {
		return []Quote{}, nil
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testQuotesPath+"?"+QueryParamOffset+"=-1", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if m.LastListParams.Offset != 0 {
		t.Errorf("negative offset must default to 0, got %d", m.LastListParams.Offset)
	}
}

func TestList_InvalidCustomerIDIgnored(t *testing.T) {
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Quote, error) {
		return []Quote{}, nil
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet,
		testQuotesPath+"?"+QueryParamCustomerID+"=nope", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if m.LastListParams.CustomerID != nil {
		t.Errorf("invalid customer id must be dropped, got %v", m.LastListParams.CustomerID)
	}
}

func TestList_StoreError(t *testing.T) {
	m := &MockStore{ListFn: func(ctx context.Context, p ListParams) ([]Quote, error) {
		return nil, errors.New("boom")
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testQuotesPath, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgListFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgListFailed)
	}
}

func TestGet_InvalidID(t *testing.T) {
	srv := newTestServer(&MockStore{})
	rec := doJSON(t, srv, http.MethodGet, testQuotesPath+"/"+testInvalidUUID, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestGet_NotFound(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Quote, error) {
		return Quote{}, ErrNotFound
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testQuotesPath+"/"+testQuoteIDStr, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgNotFound)
	}
}

func TestGet_StoreError(t *testing.T) {
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Quote, error) {
		return Quote{}, errors.New("boom")
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testQuotesPath+"/"+testQuoteIDStr, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgGetFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgGetFailed)
	}
}

func TestGet_Success(t *testing.T) {
	want := Quote{ID: uuid.MustParse(testQuoteIDStr), Status: StatusDraft}
	m := &MockStore{GetFn: func(ctx context.Context, id uuid.UUID) (Quote, error) {
		return want, nil
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodGet, testQuotesPath+"/"+testQuoteIDStr, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got Quote
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != want.ID {
		t.Errorf("id = %v, want %v", got.ID, want.ID)
	}
	if m.LastGetID.String() != testQuoteIDStr {
		t.Errorf("mock did not receive id %s", testQuoteIDStr)
	}
}

func TestUpdate_InvalidID(t *testing.T) {
	srv := newTestServer(&MockStore{})
	rec := doJSON(t, srv, http.MethodPut, testQuotesPath+"/"+testInvalidUUID, validQuoteInput())
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestUpdate_InvalidJSON(t *testing.T) {
	srv := newTestServer(&MockStore{})
	req := httptest.NewRequest(http.MethodPut, testQuotesPath+"/"+testQuoteIDStr,
		strings.NewReader("{not-json"))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidJSON {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidJSON)
	}
}

func TestUpdate_Validation(t *testing.T) {
	srv := newTestServer(&MockStore{})
	in := validQuoteInput()
	in.CustomerID = uuid.Nil
	rec := doJSON(t, srv, http.MethodPut, testQuotesPath+"/"+testQuoteIDStr, in)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgCustomerIDRequired {
		t.Errorf("error = %q, want %q", got, ErrMsgCustomerIDRequired)
	}
}

func TestUpdate_NotFound(t *testing.T) {
	m := &MockStore{UpdateFn: func(ctx context.Context, id uuid.UUID, in QuoteInput) (Quote, error) {
		return Quote{}, ErrNotFound
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPut, testQuotesPath+"/"+testQuoteIDStr, validQuoteInput())
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgNotFound)
	}
}

func TestUpdate_StoreError(t *testing.T) {
	m := &MockStore{UpdateFn: func(ctx context.Context, id uuid.UUID, in QuoteInput) (Quote, error) {
		return Quote{}, errors.New("boom")
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPut, testQuotesPath+"/"+testQuoteIDStr, validQuoteInput())
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgUpdateFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgUpdateFailed)
	}
}

func TestUpdate_Success(t *testing.T) {
	updated := Quote{ID: uuid.MustParse(testQuoteIDStr), Status: StatusDraft}
	m := &MockStore{UpdateFn: func(ctx context.Context, id uuid.UUID, in QuoteInput) (Quote, error) {
		return updated, nil
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPut, testQuotesPath+"/"+testQuoteIDStr, validQuoteInput())
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestTransition_TargetStatusRoutes(t *testing.T) {
	cases := []struct {
		name       string
		path       string
		wantStatus QuoteStatus
	}{
		{"send", "/send", StatusSent},
		{"accept", "/accept", StatusAccepted},
		{"reject", "/reject", StatusRejected},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to QuoteStatus) (Quote, error) {
				return Quote{ID: id, Status: to}, nil
			}}
			srv := newTestServer(m)
			rec := doJSON(t, srv, http.MethodPost, testQuotesPath+"/"+testQuoteIDStr+tc.path, nil)
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
	srv := newTestServer(&MockStore{})
	rec := doJSON(t, srv, http.MethodPost, testQuotesPath+"/"+testInvalidUUID+"/send", nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidID {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidID)
	}
}

func TestTransition_NotFound(t *testing.T) {
	m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to QuoteStatus) (Quote, error) {
		return Quote{}, ErrNotFound
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPost, testQuotesPath+"/"+testQuoteIDStr+"/send", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgNotFound {
		t.Errorf("error = %q, want %q", got, ErrMsgNotFound)
	}
}

func TestTransition_InvalidStatus(t *testing.T) {
	m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to QuoteStatus) (Quote, error) {
		return Quote{}, ErrInvalidStatus
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPost, testQuotesPath+"/"+testQuoteIDStr+"/accept", nil)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgInvalidTransition {
		t.Errorf("error = %q, want %q", got, ErrMsgInvalidTransition)
	}
}

func TestTransition_StoreError(t *testing.T) {
	m := &MockStore{TransitionFn: func(ctx context.Context, id uuid.UUID, to QuoteStatus) (Quote, error) {
		return Quote{}, errors.New("boom")
	}}
	srv := newTestServer(m)
	rec := doJSON(t, srv, http.MethodPost, testQuotesPath+"/"+testQuoteIDStr+"/reject", nil)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if got := decodeErrorBody(t, rec); got != ErrMsgTransitionFailed {
		t.Errorf("error = %q, want %q", got, ErrMsgTransitionFailed)
	}
}
