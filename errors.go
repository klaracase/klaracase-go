package klaracase

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Error carries one answer of the API that is an error: an RFC 9457 problem document, or a
// body that is not one. Every error the client returns for a failed call has an Error inside
// it, so errors.As(err, &e) with e of type *Error works for all of them, and so does AsError.
//
// Status is the HTTP status, or 0 when the call got no HTTP answer. Code is the machine
// readable code of the API (for example "not_found"); it is "unknown" when the body was not a
// problem document. Extra holds every member of the problem document other than type, title,
// status, detail and code, including the members that the concrete error types also expose.
//
// The concrete error types of this package (NotFoundError and the others) embed Error as a
// field named Problem. Use the concrete type to read typed members, and Error to read the
// common ones.
type Error struct {
	// Type is the problem type URI. It is a name, not a page: do not fetch it.
	Type string
	// Title is the short title of the problem.
	Title string
	// Status is the HTTP status, or 0 when there was no HTTP answer.
	Status int
	// Detail is the human readable explanation. For a body that is not a problem document it is
	// the raw body, cut to 1000 characters.
	Detail string
	// Code is the closed vocabulary code of the API, or "unknown", or "" when no answer arrived.
	Code string
	// APIRelease is the value of the X-Klaracase-Api-Release header, or "" when it was absent.
	APIRelease string
	// Extra holds every problem member that is not type, title, status, detail or code.
	Extra map[string]json.RawMessage
}

// Problem is another name for Error. The concrete error types embed it under this name,
// because a field cannot be called Error when the type also has an Error method.
type Problem = Error

// Error implements the error interface.
func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("klaracase:")
	if e.Status != 0 {
		fmt.Fprintf(&b, " HTTP %d", e.Status)
	}
	if e.Code != "" {
		b.WriteString(" ")
		b.WriteString(e.Code)
	}
	if e.Detail != "" {
		b.WriteString(": ")
		b.WriteString(e.Detail)
	}
	return b.String()
}

// AuthenticationError is an HTTP 401 answer: the token is missing, wrong, expired or revoked.
type AuthenticationError struct{ Problem }

// PermissionError is an HTTP 403 answer. RequiredAbility is set when the token lacks an ability.
type PermissionError struct {
	Problem
	// RequiredAbility names the missing token ability (for example "citation-check").
	RequiredAbility string
}

// NotFoundError is an HTTP 404 answer.
type NotFoundError struct{ Problem }

// ConflictError is an HTTP 409 answer.
type ConflictError struct{ Problem }

// ValidationError is an HTTP 422 answer. The client also returns it, with Code
// "client_validation" and Status 0, when a call is wrong before any request is sent.
type ValidationError struct {
	Problem
	// Errors maps a request field to its list of messages. It is nil for a client side error.
	Errors map[string][]string
}

// InsufficientCreditsError is an HTTP 402 answer. Nothing ran and nothing was charged.
type InsufficientCreditsError struct {
	Problem
	// Cost is the price of the call in credits.
	Cost int
	// Balance is the credits the organization has left this month.
	Balance int
}

// RateLimitError is an HTTP 429 answer. The client already retried, or the wait was too long
// to sleep (above 30 seconds). Nothing ran and nothing was charged.
type RateLimitError struct {
	Problem
	// RetryAfterSeconds is the wait the server asks for, from the body or the Retry-After header.
	RetryAfterSeconds int
}

// ServiceUnavailableError is an HTTP 503 answer. The client already retried.
type ServiceUnavailableError struct{ Problem }

// APIError is any other HTTP error answer, including an answer whose body is not a problem
// document (an HTML 502 page from a proxy, for example).
type APIError struct{ Problem }

// TransportError means no HTTP answer arrived: the connection failed, was dropped, or a
// timeout passed. Err is the underlying error and is reachable with errors.Unwrap and
// errors.Is (for example errors.Is(err, context.Canceled)).
type TransportError struct {
	Problem
	// Err is the underlying network, TLS, DNS or context error.
	Err error
}

// TimeoutError is a TransportError for a call that ran out of time, either the per attempt
// timeout of the client or the deadline of the context.
type TimeoutError struct{ TransportError }

// DocumentFailedError is returned by WaitUntilReady when the upload ends in the status
// "failed". Code and ErrorCode both equal the error_code of the document.
type DocumentFailedError struct {
	Problem
	// ErrorCode is the error_code of the document (for example "encrypted_pdf").
	ErrorCode string
	// Document is the last receipt of the document.
	Document *CitationCheckDocument
}

// DocumentTimeoutError is returned by WaitUntilReady when the timeout passes before the
// document reaches a final status. Code is "wait_timeout".
type DocumentTimeoutError struct {
	Problem
	// Document is the last receipt of the document.
	Document *CitationCheckDocument
}

// ConfigurationError means the client cannot be built: no token, a bad base URL or an
// invalid option. It is not an API problem and has no code or status.
type ConfigurationError struct {
	// Message says what is wrong and how to fix it.
	Message string
}

// Error implements the error interface.
func (e *ConfigurationError) Error() string { return "klaracase: configuration error: " + e.Message }

// Error implements the error interface.
func (e *TransportError) Error() string {
	if e.Err == nil {
		return "klaracase: transport error"
	}
	return "klaracase: transport error: " + e.Err.Error()
}

// Error implements the error interface.
func (e *TimeoutError) Error() string {
	if e.Err == nil {
		return "klaracase: request timed out"
	}
	return "klaracase: request timed out: " + e.Err.Error()
}

// Error implements the error interface.
func (e *DocumentFailedError) Error() string {
	return "klaracase: the document could not be processed: " + e.ErrorCode
}

// Error implements the error interface.
func (e *DocumentTimeoutError) Error() string {
	return "klaracase: the document was not ready before the timeout"
}

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *AuthenticationError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *PermissionError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *NotFoundError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *ConflictError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *ValidationError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *InsufficientCreditsError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *RateLimitError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *ServiceUnavailableError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *APIError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error and the underlying error.
func (e *TransportError) Unwrap() []error {
	if e.Err == nil {
		return []error{&e.Problem}
	}
	return []error{&e.Problem, e.Err}
}

// Unwrap returns the embedded TransportError, so a TimeoutError also matches *TransportError.
func (e *TimeoutError) Unwrap() []error { return []error{&e.TransportError} }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *DocumentFailedError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *DocumentTimeoutError) Unwrap() error { return &e.Problem }

// AsError returns the Error inside err, if there is one. It works for every error type of
// this package except ConfigurationError, and returns false for any other error.
func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// clientValidation builds the error for a call that is wrong before any request is sent.
func clientValidation(format string, args ...any) error {
	return &ValidationError{Problem: Problem{Code: "client_validation", Detail: fmt.Sprintf(format, args...)}}
}
