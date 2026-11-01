package klaracase

import (
	"net/http"
	"time"
)

// RequestOption changes one call. Pass it as the last arguments of any service method. A
// request option wins over the client option of the same kind for that call only.
type RequestOption func(*requestConfig)

// requestConfig is the result of the request options of one call.
type requestConfig struct {
	header     http.Header
	timeout    *time.Duration
	maxRetries *int
	err        error
}

func (c *requestConfig) fail(msg string) {
	if c.err == nil {
		c.err = &ConfigurationError{Message: msg}
	}
}

// newRequestConfig applies the request options in order. The first invalid option makes the
// call return a *ConfigurationError before it sends anything.
func newRequestConfig(opts []RequestOption) (requestConfig, error) {
	var cfg requestConfig
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg, cfg.err
}

// WithHeader sets a header on every attempt of one call, for example a trace id of your own.
// It replaces a header of the same name that the client would send, except Authorization and
// Content-Type, which a call may not change: they return a *ConfigurationError.
func WithHeader(key, value string) RequestOption {
	return func(c *requestConfig) {
		canonical := http.CanonicalHeaderKey(key)
		switch canonical {
		case "":
			c.fail("WithHeader: the header name is empty")
			return
		case "Authorization", "Content-Type":
			c.fail("WithHeader: the " + canonical + " header cannot be set per call")
			return
		}
		if c.header == nil {
			c.header = http.Header{}
		}
		c.header.Set(canonical, value)
	}
}

// WithRequestTimeout sets the time limit for one HTTP attempt of one call. It replaces the
// client option WithTimeout. Zero turns the per attempt limit off, so only the context limits
// the call.
func WithRequestTimeout(d time.Duration) RequestOption {
	return func(c *requestConfig) {
		if d < 0 {
			c.fail("WithRequestTimeout: the timeout must not be negative")
			return
		}
		c.timeout = &d
	}
}

// WithRequestMaxRetries sets how many times one call is retried after a 429 or 503 answer
// (and after a transport error on a GET call). It replaces the client option WithMaxRetries.
// Zero turns retries off for the call.
func WithRequestMaxRetries(n int) RequestOption {
	return func(c *requestConfig) {
		if n < 0 {
			c.fail("WithRequestMaxRetries: the number of retries must not be negative")
			return
		}
		c.maxRetries = &n
	}
}
