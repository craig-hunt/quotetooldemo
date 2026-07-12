package main

import "time"

// Env var names — must match docker-compose.yml and fly.toml.
const (
	EnvDatabaseURL = "DATABASE_URL"
	EnvServicePort = "SERVICE_PORT"
	EnvLogLevel    = "LOG_LEVEL"
)

// Runtime defaults.
const (
	DefaultPort            = "8080"
	ReadHeaderTimeout      = 5 * time.Second
	ShutdownTimeout        = 15 * time.Second
	DefaultPageLimit       = 50
	MaxPageLimit           = 200
)

// Log-level tokens accepted in LOG_LEVEL.
const (
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// HTTP header + MIME type strings.
const (
	HeaderContentType                   = "Content-Type"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	ContentTypeJSON                     = "application/json"
	CORSAllowedMethods                  = "GET, POST, PUT, DELETE, OPTIONS"
	CORSAllowedHeaders                  = "Content-Type"
	CORSAllowedOrigin                   = "*"
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
)

// SQL schema / table identifier for this service.
const (
	SchemaTable = "customers.customers"
)

// Error-message strings surfaced to API clients.
const (
	ErrMsgInvalidJSON    = "invalid JSON body"
	ErrMsgNameRequired   = "name required"
	ErrMsgInvalidID      = "invalid id"
	ErrMsgNotFound       = "customer not found"
	ErrMsgCreateFailed   = "create failed"
	ErrMsgListFailed     = "list failed"
	ErrMsgGetFailed      = "get failed"
	ErrMsgUpdateFailed   = "update failed"
	ErrMsgDeleteFailed   = "delete failed"
)
