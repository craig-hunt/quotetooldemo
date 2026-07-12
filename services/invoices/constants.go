package main

import "time"

// Env var names.
const (
	EnvDatabaseURL      = "DATABASE_URL"
	EnvServicePort      = "SERVICE_PORT"
	EnvLogLevel         = "LOG_LEVEL"
	EnvOrdersServiceURL = "ORDERS_SERVICE_URL"
)

// Runtime defaults.
const (
	DefaultPort         = "8080"
	ReadHeaderTimeout   = 5 * time.Second
	ShutdownTimeout     = 15 * time.Second
	DefaultPageLimit    = 50
	MaxPageLimit        = 200
	HTTPClientTimeout   = 5 * time.Second
	DefaultDueDays      = 30
	InvoiceNumberPrefix = "INV"
	InvoiceNumberPattern = "%s-%d-%04d" // prefix-year-sequence
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
// Must match the string form of orders.OrderStatus values in the orders service.
const (
	ExternalOrderStatusFulfilled = "fulfilled"
)

// SQL sequence name (schema-qualified).
const (
	InvoiceNumberSequence = "invoices.invoice_number_seq"
)

// Error-message strings surfaced to API clients.
const (
	ErrMsgInvalidJSON         = "invalid JSON body"
	ErrMsgInvalidID           = "invalid id"
	ErrMsgOrderIDRequired     = "order_id required"
	ErrMsgOrderNotFound       = "order not found"
	ErrMsgOrdersUnavailable   = "orders service unavailable"
	ErrMsgOrderNotFulfilled   = "order must be in fulfilled status"
	ErrMsgNotFound            = "invoice not found"
	ErrMsgInvalidTransition   = "invalid status transition"
	ErrMsgCreateFailed        = "create failed"
	ErrMsgListFailed          = "list failed"
	ErrMsgGetFailed           = "get failed"
	ErrMsgTransitionFailed    = "transition failed"
)
