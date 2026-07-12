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
	DefaultPageLimit  = 50
	MaxPageLimit      = 200
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
	QueryParamLimit      = "limit"
	QueryParamOffset     = "offset"
	QueryParamCustomerID = "customer_id"
	QueryParamStatus     = "status"
)

// Error-message strings surfaced to API clients.
const (
	ErrMsgInvalidJSON         = "invalid JSON body"
	ErrMsgInvalidID           = "invalid id"
	ErrMsgNotFound            = "quote not found"
	ErrMsgInvalidTransition   = "invalid status transition"
	ErrMsgCreateFailed        = "create failed"
	ErrMsgListFailed          = "list failed"
	ErrMsgGetFailed           = "get failed"
	ErrMsgUpdateFailed        = "update failed"
	ErrMsgTransitionFailed    = "transition failed"
	ErrMsgCustomerIDRequired  = "customer_id required"
	ErrMsgProjectNameRequired = "project_name required"
	ErrMsgLineItemsRequired   = "at least one line item required"
	ErrMsgInvalidMixType      = "invalid mix_type"
	ErrMsgInvalidArea         = "area_sqft must be > 0"
	ErrMsgInvalidDepth        = "depth_inches must be > 0"
	ErrMsgInvalidUnitPrice    = "unit_price_per_ton must be > 0"
	ErrMsgLinePrefix          = "line "
)
