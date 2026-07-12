package main

import "time"

// Env var names.
const (
	EnvDatabaseURL = "DATABASE_URL"
	EnvServicePort = "SERVICE_PORT"
	EnvLogLevel    = "LOG_LEVEL"
)

// Runtime defaults.
const (
	DefaultPort       = "8080"
	ReadHeaderTimeout = 5 * time.Second
	ShutdownTimeout   = 15 * time.Second
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
	CORSAllowedMethods              = "GET, OPTIONS"
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

// Report stage names surfaced in the /reports/quote-to-cash response.
// Consumers (frontend charts) pin these strings, so they are API contract.
const (
	StageQuotesCreated   = "quotes_created"
	StageQuotesSent      = "quotes_sent"
	StageQuotesAccepted  = "quotes_accepted"
	StageOrdersCreated   = "orders_created"
	StageOrdersFulfilled = "orders_fulfilled"
	StageInvoicesPaid    = "invoices_paid"
)

// Aging bucket names surfaced in the /reports/aging response.
const (
	BucketPaid         = "paid"
	BucketCancelled    = "cancelled"
	BucketCurrent      = "current"
	Bucket1To30Days    = "1-30_days_overdue"
	Bucket31To60Days   = "31-60_days_overdue"
	Bucket61To90Days   = "61-90_days_overdue"
	Bucket90PlusDays   = "90+_days_overdue"
)

// SQL identifiers.
const (
	SchemaQuotesTable         = "quotes.quotes"
	SchemaOrdersTable         = "orders.orders"
	SchemaInvoicesTable       = "invoices.invoices"
	SchemaQuoteLineItemsTable = "quotes.quote_line_items"
)

// Invoice SQL status literals used in aging / stage bucket queries.
// Must match invoices.InvoiceStatus values in the invoices service.
const (
	SQLInvoiceStatusPaid      = "paid"
	SQLInvoiceStatusCancelled = "cancelled"
)

// Error-message strings.
const (
	ErrMsgReportFailed = "report failed"
)
