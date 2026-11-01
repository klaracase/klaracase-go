package klaracase

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	// DefaultBaseURL is the base URL of the Klaracase REST API v1.
	DefaultBaseURL = "https://klaracase.de/api/v1"
	// EnvToken is the name of the environment variable that NewFromEnv reads.
	EnvToken = "KLARACASE_API_TOKEN"
	// DefaultTimeout is the default time limit for one HTTP attempt.
	DefaultTimeout = 30 * time.Second
	// DefaultMaxRetries is the default number of retries after a 429 or 503 answer.
	DefaultMaxRetries = 2
)

// Client is a client for the Klaracase REST API v1. Create it with New or NewFromEnv.
// A Client is safe for concurrent use by several goroutines.
type Client struct {
	// Decisions searches court decisions, reads one decision and follows its citations.
	Decisions *DecisionsService
	// Norms reads one statute norm.
	Norms *NormsService
	// Datapackages bundles several decisions and their citations.
	Datapackages *DatapackagesService
	// CitationChecks checks the citations in a German legal text.
	CitationChecks *CitationChecksService
	// CitationCheckDocuments uploads a file for a citation check and waits until it is ready.
	CitationCheckDocuments *CitationCheckDocumentsService
	// Corpus reports what the corpus holds. Both calls are free.
	Corpus *CorpusService

	token         string
	baseURL       string
	httpClient    *http.Client
	timeout       time.Duration
	maxRetries    int
	userAgent     string
	uaSuffix      string
	onDeprecation func(DeprecationNotice)
	sleep         func(ctx context.Context, d time.Duration) error
	now           func() time.Time
	optionErr     error
}

// Option configures a Client. Pass options to New or NewFromEnv.
type Option func(*Client)

// WithBaseURL sets the base URL, for example "http://127.0.0.1:8080/api/v1" for a test server.
// A trailing slash is removed. The default is DefaultBaseURL.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) { c.baseURL = baseURL }
}

// WithHTTPClient sets the HTTP client. Use it for a proxy, a custom transport or a test double.
// The client never changes the HTTP client you pass; the per attempt timeout is applied with a
// context instead.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h == nil {
			c.setOptionErr("WithHTTPClient: the HTTP client is nil")
			return
		}
		c.httpClient = h
	}
}

// WithTimeout sets the time limit for one HTTP attempt (default 30 seconds). A retry gets a
// new limit. Zero turns the limit off, so only the context of the call limits it.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		if d < 0 {
			c.setOptionErr("WithTimeout: the timeout must not be negative")
			return
		}
		c.timeout = d
	}
}

// WithMaxRetries sets how many times the client retries after a 429 or 503 answer, and after
// a transport error on a GET call (default 2). Zero turns retries off.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n < 0 {
			c.setOptionErr("WithMaxRetries: the number of retries must not be negative")
			return
		}
		c.maxRetries = n
	}
}

// WithUserAgentSuffix adds text after the SDK part of the User-Agent header, for example
// "my-app/1.2". Use it so the support team can tell your traffic apart.
func WithUserAgentSuffix(suffix string) Option {
	return func(c *Client) { c.uaSuffix = strings.TrimSpace(suffix) }
}

// WithDeprecationHandler sets the function that is called when an endpoint answers with a
// Deprecation or Sunset header. The function runs at most once per endpoint in the whole
// process. The default handler writes one warning with log/slog.
func WithDeprecationHandler(h func(DeprecationNotice)) Option {
	return func(c *Client) { c.onDeprecation = h }
}

// withSleep replaces the sleep function used for retries and polling. Tests only.
func withSleep(f func(ctx context.Context, d time.Duration) error) Option {
	return func(c *Client) { c.sleep = f }
}

// withNow replaces the clock used for the WaitUntilReady timeout. Tests only.
func withNow(f func() time.Time) Option {
	return func(c *Client) { c.now = f }
}

func (c *Client) setOptionErr(msg string) {
	if c.optionErr == nil {
		c.optionErr = &ConfigurationError{Message: msg}
	}
}

// New creates a client with the API token. The token must not be empty. Create a token at
// https://klaracase.de/settings/api-tokens. An empty token returns a *ConfigurationError.
func New(token string, opts ...Option) (*Client, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, &ConfigurationError{Message: "no API token: pass one to New, or use NewFromEnv with the " + EnvToken + " environment variable"}
	}
	if strings.ContainsAny(token, " \t\r\n") {
		return nil, &ConfigurationError{Message: "the API token contains whitespace"}
	}
	c := &Client{
		token:      token,
		baseURL:    DefaultBaseURL,
		httpClient: &http.Client{},
		timeout:    DefaultTimeout,
		maxRetries: DefaultMaxRetries,
		sleep:      sleepContext,
		now:        time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.optionErr != nil {
		return nil, c.optionErr
	}
	base, err := normalizeBaseURL(c.baseURL)
	if err != nil {
		return nil, err
	}
	c.baseURL = base
	c.userAgent = userAgent(c.uaSuffix)
	if c.onDeprecation == nil {
		c.onDeprecation = defaultDeprecationHandler
	}
	c.Decisions = &DecisionsService{c: c}
	c.Norms = &NormsService{c: c}
	c.Datapackages = &DatapackagesService{c: c}
	c.CitationChecks = &CitationChecksService{c: c}
	c.CitationCheckDocuments = &CitationCheckDocumentsService{c: c}
	c.Corpus = &CorpusService{c: c}
	return c, nil
}

// NewFromEnv creates a client with the token from the KLARACASE_API_TOKEN environment
// variable. A missing or empty variable returns a *ConfigurationError.
func NewFromEnv(opts ...Option) (*Client, error) {
	token := strings.TrimSpace(os.Getenv(EnvToken))
	if token == "" {
		return nil, &ConfigurationError{Message: "the " + EnvToken + " environment variable is not set or is empty"}
	}
	return New(token, opts...)
}

// Search searches statutes and court decisions together (GET /search). It costs 1 credit.
func (c *Client) Search(ctx context.Context, p CrossTypeSearchParams, opts ...RequestOption) (*CrossTypeSearchResponse, error) {
	if err := validateCrossTypeSearch(p); err != nil {
		return nil, err
	}
	var q query
	q.str("query", p.Query)
	q.str("jurisdiction", string(p.Jurisdiction))
	q.str("court", p.Court)
	q.str("branch", string(p.Branch))
	q.str("date_from", string(p.DateFrom))
	q.str("date_to", string(p.DateTo))
	q.str("mode", string(p.Mode))
	q.str("document_type", string(p.DocumentType))
	q.str("cited_norm", p.CitedNorm)
	addList(&q, "cited_statutes", p.CitedStatutes)
	q.num("limit", p.Limit)
	q.boolp("bridge", p.Bridge)
	out := new(CrossTypeSearchResponse)
	err := c.do(ctx, request{method: http.MethodGet, endpoint: "GET /search", path: "/search", query: q, opts: opts}, out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// normalizeBaseURL checks the base URL and removes trailing slashes.
func normalizeBaseURL(raw string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(raw), "/")
	u, err := url.Parse(trimmed)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", &ConfigurationError{Message: fmt.Sprintf("the base URL %q is not an http or https URL", raw)}
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", &ConfigurationError{Message: "the base URL must not have a query or a fragment"}
	}
	return trimmed, nil
}

// userAgent builds "klaracase-go/<version> go/<go version>" plus the optional suffix.
func userAgent(suffix string) string {
	ua := "klaracase-go/" + Version + " go/" + strings.TrimPrefix(runtime.Version(), "go")
	if suffix != "" {
		ua += " " + suffix
	}
	return ua
}

// sleepContext waits for d, or returns the context error when the context ends first.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func defaultDeprecationHandler(n DeprecationNotice) {
	slog.Warn("klaracase: endpoint is deprecated",
		"endpoint", n.Endpoint, "deprecation", n.Deprecation, "sunset", n.Sunset, "link", n.Link)
}
