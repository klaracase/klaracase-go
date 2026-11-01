package klaracase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
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
	// Header holds the response headers of the failed call, for example Retry-After and the
	// X-RateLimit-* headers. It never holds Authorization. It is nil when no answer arrived.
	Header http.Header
	// Method is the HTTP method of the failed call. It is "" for an error found before any request.
	Method string
	// URL is the address of the failed call without its query string, which can hold search
	// text. It is "" for an error found before any request.
	URL string

	// Candidates lists the decisions that carry an ambiguous Aktenzeichen
	// (code "ambiguous_aktenzeichen"). Pass one of the ids to the next call.
	Candidates []Candidate
	// Successor names the statute that replaced a repealed one (code "statute_superseded").
	Successor string
	// Unit says what a volume cap counts, "rows" or "documents" (code "volume_cap").
	Unit string
	// Cap is the daily limit that was reached (code "volume_cap").
	Cap int
	// ResetsAt is the time the daily volume cap resets, midnight Europe/Berlin (code "volume_cap").
	ResetsAt time.Time

	// Extra holds every problem member that is not type, title, status, detail or code.
	Extra map[string]json.RawMessage
}

// Candidate is one decision that carries an ambiguous Aktenzeichen.
type Candidate struct {
	// ID is the UUID of the decision. Pass it as the id of the next call.
	ID string `json:"id"`
	// Court is the court short form.
	Court string `json:"court"`
	// Date is the decision date.
	Date Date `json:"date"`
	// Aktenzeichen is the file number.
	Aktenzeichen string `json:"aktenzeichen"`
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

// RetryAfter returns how long the API asks you to wait before the next call: the
// retry_after_seconds member of the body, or else the Retry-After header, in seconds or as an
// HTTP date. It returns 0 when the API stated no wait. A volume cap states a wait of hours.
func (e *Error) RetryAfter() time.Duration {
	if wait, ok := statedWaitOf(e.Extra, e.Header, time.Now()); ok {
		return wait
	}
	return 0
}

// Retryable reports whether the API guarantees that the failed call ran nothing and charged
// nothing, so the same call may be sent again after RetryAfter: a 429 or a 503 answer. It
// is false for every other error. The client has already retried before it returns such an
// error, so a true here means the retries ran out or the wait was too long.
func (e *Error) Retryable() bool {
	return e.Status == http.StatusTooManyRequests || e.Status == http.StatusServiceUnavailable
}

// MarshalJSON implements json.Marshaler. It writes a compact object with snake_case keys that
// is fit to give a language model as a tool result: code, status, title, detail,
// retry_after_seconds when the API stated a wait, and then every other member of the problem
// document (cost, balance, candidates, successor, errors and so on). The header, method and
// URL are left out.
func (e *Error) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	first := true
	put := func(key string, value any) error {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if !first {
			buf.WriteByte(',')
		}
		first = false
		name, _ := json.Marshal(key)
		buf.Write(name)
		buf.WriteByte(':')
		buf.Write(encoded)
		return nil
	}
	if err := put("code", e.Code); err != nil {
		return nil, err
	}
	if e.Status != 0 {
		if err := put("status", e.Status); err != nil {
			return nil, err
		}
	}
	if e.Title != "" {
		if err := put("title", e.Title); err != nil {
			return nil, err
		}
	}
	if err := put("detail", e.Detail); err != nil {
		return nil, err
	}
	if wait := e.RetryAfter(); wait > 0 {
		if err := put("retry_after_seconds", int(math.Ceil(wait.Seconds()))); err != nil {
			return nil, err
		}
	}
	keys := make([]string, 0, len(e.Extra))
	for key := range e.Extra {
		if key != "retry_after_seconds" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		if err := put(key, e.Extra[key]); err != nil {
			return nil, err
		}
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// statedWaitOf reads the wait that a problem states: retry_after_seconds in the body members,
// then the Retry-After header as whole seconds or as an HTTP date. now is the reference for
// a date.
func statedWaitOf(extra map[string]json.RawMessage, header http.Header, now time.Time) (time.Duration, bool) {
	if raw, ok := extra["retry_after_seconds"]; ok {
		var seconds float64
		if json.Unmarshal(raw, &seconds) == nil && seconds >= 0 {
			return time.Duration(seconds * float64(time.Second)), true
		}
	}
	value := strings.TrimSpace(header.Get("Retry-After"))
	if value == "" {
		return 0, false
	}
	if n, err := strconv.Atoi(value); err == nil && n >= 0 {
		return time.Duration(n) * time.Second, true
	}
	if when, err := http.ParseTime(value); err == nil {
		return max(when.Sub(now), 0), true
	}
	return 0, false
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
// "client_validation" and Status 0, when a call is wrong before any request is sent. The
// client checks only what needs no server: a missing required value, a malformed date, a
// range that runs backwards, a list of the wrong size.
type ValidationError struct {
	Problem
	// Errors maps a request field, by its wire name, to its list of messages.
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

// DocumentTimeoutError is returned by WaitUntilReady when the deadline of the context passes
// before the document reaches a final status. Code is "wait_timeout". It wraps
// context.DeadlineExceeded, so errors.Is(err, context.DeadlineExceeded) is true.
type DocumentTimeoutError struct {
	Problem
	// Document is the last receipt that was read, or nil when the deadline passed before the
	// first answer.
	Document *CitationCheckDocument
	// Err is context.DeadlineExceeded.
	Err error
}

// DecodeError means the API answered with a success status, but the body does not fit the
// types of this SDK, for example a value of the wrong type or a body that is not JSON. The call
// may already have been charged, so the answer is kept: Raw holds the whole body, and Body
// holds its first 1000 characters for a log line. The client returns no response value
// with this error.
//
// Code is "invalid_response" and Status is the HTTP status. Err is the decoder error.
type DecodeError struct {
	Problem
	// Err is the error of the JSON decoder.
	Err error
	// Body is the body, cut to 1000 characters.
	Body string
	// Raw is the whole body as the server sent it.
	Raw json.RawMessage
}

// ConfigurationError means the client cannot be built, or a call has an invalid request
// option: no token, a bad base URL or an invalid option. It is not an API problem. AsError
// finds an *Error inside it with the code "configuration_error", status 0 and the message as
// detail.
type ConfigurationError struct {
	Problem
	// Message says what is wrong and how to fix it.
	Message string
}

// newConfigurationError builds a ConfigurationError with the common problem fields set.
func newConfigurationError(message string) *ConfigurationError {
	return &ConfigurationError{
		Problem: Problem{Code: CodeConfiguration, Detail: message},
		Message: message,
	}
}

// Error implements the error interface.
func (e *ConfigurationError) Error() string { return "klaracase: configuration error: " + e.Message }

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *ConfigurationError) Unwrap() error { return &e.Problem }

// Retryable reports whether the same call may be sent again. A transport error means no answer
// arrived, so the call may or may not have run: a repeated GET is safe, and a repeated paid
// call costs nothing more in the same month. It is false when the cause is a canceled context.
func (e *TransportError) Retryable() bool {
	return !errors.Is(e.Err, context.Canceled)
}

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
func (e *DecodeError) Error() string {
	if e.Err == nil {
		return "klaracase: the answer could not be decoded"
	}
	return "klaracase: the answer could not be decoded: " + e.Err.Error()
}

// Error implements the error interface.
func (e *DocumentFailedError) Error() string {
	return "klaracase: the document could not be processed: " + e.ErrorCode
}

// Error implements the error interface.
func (e *DocumentTimeoutError) Error() string {
	return "klaracase: the document was not ready before the deadline of the context"
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

// Unwrap returns the embedded Error and the decoder error.
func (e *DecodeError) Unwrap() []error {
	if e.Err == nil {
		return []error{&e.Problem}
	}
	return []error{&e.Problem, e.Err}
}

// Unwrap returns the embedded Error so that errors.As finds *Error.
func (e *DocumentFailedError) Unwrap() error { return &e.Problem }

// Unwrap returns the embedded Error and the deadline error.
func (e *DocumentTimeoutError) Unwrap() []error {
	if e.Err == nil {
		return []error{&e.Problem}
	}
	return []error{&e.Problem, e.Err}
}

// AsError returns the Error inside err, if there is one. It works for every error type of
// this package, including ConfigurationError, and returns false for any other error.
func AsError(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}

// IsNotFound reports whether err is an HTTP 404 answer.
func IsNotFound(err error) bool {
	var target *NotFoundError
	return errors.As(err, &target)
}

// IsRateLimited reports whether err is an HTTP 429 answer: the rate limit or the daily volume cap.
func IsRateLimited(err error) bool {
	var target *RateLimitError
	return errors.As(err, &target)
}

// IsInsufficientCredits reports whether err is an HTTP 402 answer.
func IsInsufficientCredits(err error) bool {
	var target *InsufficientCreditsError
	return errors.As(err, &target)
}

// IsValidation reports whether err is an HTTP 422 answer, or a call that the client refused
// before any request (code "client_validation").
func IsValidation(err error) bool {
	var target *ValidationError
	return errors.As(err, &target)
}

// IsPermission reports whether err is an HTTP 403 answer.
func IsPermission(err error) bool {
	var target *PermissionError
	return errors.As(err, &target)
}
