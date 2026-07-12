package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// parseLogLevel is the only pure text-to-value function on this service.
// It maps the LOG_LEVEL env token to a slog.Level; unknown values must
// fall back to slog.LevelInfo (default) rather than error.
func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		in   string
		want slog.Level
	}{
		{LogLevelDebug, slog.LevelDebug},
		{LogLevelInfo, slog.LevelInfo},
		{LogLevelWarn, slog.LevelWarn},
		{LogLevelError, slog.LevelError},
		{"", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
		{"DEBUG", slog.LevelInfo}, // case-sensitive; only lowercase tokens map
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			if got := parseLogLevel(tc.in); got != tc.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestWriteJSON_SetsHeadersAndStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	writeJSON(rec, http.StatusCreated, map[string]string{"hello": "world"})

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if got := rec.Header().Get(HeaderContentType); got != ContentTypeJSON {
		t.Errorf("Content-Type = %q, want %q", got, ContentTypeJSON)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["hello"] != "world" {
		t.Errorf("body[hello] = %q, want world", body["hello"])
	}
}

func TestWriteError_BodyShape(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, http.StatusInternalServerError, ErrMsgReportFailed)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body[ErrorKey] != ErrMsgReportFailed {
		t.Errorf("body[%s] = %q, want %q", ErrorKey, body[ErrorKey], ErrMsgReportFailed)
	}
}

func TestWithCORS_SetsHeadersAndForwardsGET(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("body"))
	})
	req := httptest.NewRequest(http.MethodGet, "/anything", nil)
	rec := httptest.NewRecorder()
	withCORS(inner).ServeHTTP(rec, req)

	if got := rec.Header().Get(HeaderAccessControlAllowOrigin); got != CORSAllowedOrigin {
		t.Errorf("Allow-Origin = %q, want %q", got, CORSAllowedOrigin)
	}
	if got := rec.Header().Get(HeaderAccessControlAllowMethods); got != CORSAllowedMethods {
		t.Errorf("Allow-Methods = %q, want %q", got, CORSAllowedMethods)
	}
	if got := rec.Header().Get(HeaderAccessControlAllowHeaders); got != CORSAllowedHeaders {
		t.Errorf("Allow-Headers = %q, want %q", got, CORSAllowedHeaders)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200 (GET must reach inner)", rec.Code)
	}
	if rec.Body.String() != "body" {
		t.Errorf("body = %q, want inner body", rec.Body.String())
	}
}

func TestWithCORS_OptionsShortCircuits(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("inner handler must not run for OPTIONS")
	})
	req := httptest.NewRequest(http.MethodOptions, "/anything", nil)
	rec := httptest.NewRecorder()
	withCORS(inner).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}

// captureHandler collects every slog.Record so tests can assert message +
// attributes. slog.Handler interface: Enabled / Handle / WithAttrs / WithGroup.
type captureHandler struct{ records *[]slog.Record }

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	*h.records = append(*h.records, r)
	return nil
}
func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler      { return h }

func TestWithLogging_EmitsRequestRecord(t *testing.T) {
	var records []slog.Record
	log := slog.New(&captureHandler{records: &records})

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/reports/aging", nil)
	rec := httptest.NewRecorder()
	withLogging(inner, log).ServeHTTP(rec, req)

	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	if records[0].Message != MetricRequest {
		t.Errorf("message = %q, want %q", records[0].Message, MetricRequest)
	}
	attrs := map[string]any{}
	records[0].Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	if attrs["method"] != http.MethodGet {
		t.Errorf("method attr = %v, want GET", attrs["method"])
	}
	if attrs["path"] != "/reports/aging" {
		t.Errorf("path attr = %v, want /reports/aging", attrs["path"])
	}
	if _, ok := attrs["duration_ms"]; !ok {
		t.Errorf("missing duration_ms attribute")
	}
}
