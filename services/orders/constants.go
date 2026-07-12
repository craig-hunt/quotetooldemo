package main

import "time"

// Env var names.
const (
	EnvDatabaseURL       = "DATABASE_URL"
	EnvServicePort       = "SERVICE_PORT"
	EnvLogLevel          = "LOG_LEVEL"
	EnvQuotesServiceURL  = "QUOTES_SERVICE_URL"
)

// Runtime defaults.
const (
	DefaultPort         = "8080"
	ReadHeaderTimeout   = 5 * time.Second
	ShutdownTimeout     = 15 * time.Second
	DefaultPageLimit    = 50
	MaxPageLimit        = 200
	HTTPClientTimeout   = 5 * time.Second
	OrderNumberPrefix   = "SO"
	OrderNumberPattern  = "%s-%d-%04d" // prefix-year-sequence
)

// Log-level tokens.
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// HTTP header + MIME type strings.
const (
	HeaderContentType               = "Content-Type"
	HeaderAccessControlAllowOrigin  = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders = "Access-Control-Allow-Headers"
	ContentTypeJSON                 = "application/json"
	CORSAllowedMethods              = "GET, POST, PUT, DELETE, OPTIONS"
	CORSAllowedHeaders              = "Content-Type"
	CORSAllowedOrigin               = "*"
)

// Health / response tokens.
const (
	StatusOK      = "ok"
	StatusKey     = "status"
	ErrorKey      = "error"
	MetricRequest = "request"
)

// Query param names.
const (
	QueryParamLimit  = "limit"
	QueryParamOffset = "offset"
	QueryParamStatus = "status"
)

// Cross-service status literals.
// Must match the string form of quotes.QuoteStatus values in the quotes
// service. Documented here so a rename in one service surfaces immediately.
const (
	ExternalQuoteStatusAccepted = "accepted"
)

// SQL sequence name (schema-qualified).
const (
	OrderNumberSequence = "orders.order_number_seq"
)

// Error-message strings surfaced to API clients.
const (
	ErrMsgInvalidJSON          = "invalid JSON body"
	ErrMsgInvalidID            = "invalid id"
	ErrMsgQuoteIDRequired      = "quote_id required"
	ErrMsgQuoteNotFound        = "quote not found"
	ErrMsgQuotesUnavailable    = "quotes service unavailable"
	ErrMsgQuoteNotAccepted     = "quote must be in accepted status"
	ErrMsgOrderNotFound        = "order not found"
	ErrMsgInvalidTransition    = "invalid status transition"
	ErrMsgCreateFailed         = "create failed"
	ErrMsgListFailed           = "list failed"
	ErrMsgGetFailed            = "get failed"
	ErrMsgUpdateFailed         = "update failed"
	ErrMsgTransitionFailed     = "transition failed"
)
