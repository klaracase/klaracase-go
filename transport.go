package klaracase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// maxServerWait is the longest server stated wait that the client sleeps. A longer wait
	// is returned to the caller as a RateLimitError (or ServiceUnavailableError).
	maxServerWait = 30 * time.Second
	// maxBackoff caps the computed backoff.
	maxBackoff = 8 * time.Second
	// baseBackoff is the backoff before the first retry, before jitter.
	baseBackoff = 500 * time.Millisecond
	// maxErrorDetail is the number of characters of a non-problem body kept in Detail.
	maxErrorDetail = 1000
)

// DeprecationNotice is what the deprecation handler receives. The values are the raw
// header values of the answer.
type DeprecationNotice struct {
	// Endpoint is the route, for example "GET /decisions/{id}".
	Endpoint string
	// Deprecation is the raw Deprecation header ("" when absent).
	Deprecation string
	// Sunset is the raw Sunset header ("" when absent).
	Sunset string
	// Link is the raw Link header ("" when absent), which may point to the migration notes.
	Link string
}

// deprecationSeen holds one *sync.Once per endpoint for the whole process.
var deprecationSeen sync.Map

// request describes one API call before it is sent.
type request struct {
	method      string
	endpoint    string // route template, used as the deprecation key
	path        string // escaped path below the base URL, starting with "/"
	query       query
	body        []byte
	contentType string
	opts        []RequestOption
}

// rawResponse is one HTTP answer, fully read.
type rawResponse struct {
	status   int
	header   http.Header
	body     []byte
	duration time.Duration
}

// do sends the request and decodes a 2xx answer into out. It retries as the design says:
// 429 and 503 on every method, transport errors on GET only.
func (c *Client) do(ctx context.Context, r request, out any) error {
	cfg, err := newRequestConfig(r.opts)
	if err != nil {
		return err
	}
	maxRetries := c.maxRetries
	if cfg.maxRetries != nil {
		maxRetries = *cfg.maxRetries
	}
	for attempt := 0; ; attempt++ {
		resp, err := c.attempt(ctx, r, cfg)
		if err != nil {
			if r.method != http.MethodGet || attempt >= maxRetries || ctx.Err() != nil {
				return err
			}
			if serr := c.sleep(ctx, backoff(attempt)); serr != nil {
				return c.transportError(ctx, serr)
			}
			continue
		}
		c.noteDeprecation(r.endpoint, resp.header)
		if resp.status >= 200 && resp.status < 300 {
			return decodeSuccess(resp, out)
		}
		parsed := parseProblem(resp)
		apiErr := parsed.toError(resp)
		if (resp.status != http.StatusTooManyRequests && resp.status != http.StatusServiceUnavailable) || attempt >= maxRetries {
			return apiErr
		}
		wait, stated := parsed.statedWait(resp.header)
		if stated && wait > maxServerWait {
			return apiErr
		}
		if !stated {
			wait = backoff(attempt)
		}
		if serr := c.sleep(ctx, wait); serr != nil {
			return c.transportError(ctx, serr)
		}
	}
}

// attempt sends the request once, with the per attempt timeout, and reads the whole answer.
func (c *Client) attempt(ctx context.Context, r request, cfg requestConfig) (*rawResponse, error) {
	reqCtx := ctx
	timeout := c.timeout
	if cfg.timeout != nil {
		timeout = *cfg.timeout
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		reqCtx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	target := c.baseURL + r.path
	if enc := r.query.encode(); enc != "" {
		target += "?" + enc
	}
	var body io.Reader
	if r.body != nil {
		body = bytes.NewReader(r.body)
	}
	req, err := http.NewRequestWithContext(reqCtx, r.method, target, body)
	if err != nil {
		return nil, &TransportError{Problem: Problem{Detail: err.Error()}, Err: err}
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if r.contentType != "" {
		req.Header.Set("Content-Type", r.contentType)
	}
	for key, values := range cfg.header {
		req.Header[key] = append([]string(nil), values...)
	}
	start := time.Now()
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, c.transportError(ctx, err)
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, c.transportError(ctx, err)
	}
	return &rawResponse{status: res.StatusCode, header: res.Header, body: data, duration: time.Since(start)}, nil
}

// transportError wraps err as a TransportError, or a TimeoutError when it is a timeout.
func (c *Client) transportError(ctx context.Context, err error) error {
	var netErr net.Error
	isTimeout := errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout())
	if ctx.Err() != nil && errors.Is(ctx.Err(), context.Canceled) {
		isTimeout = false
	}
	te := TransportError{Problem: Problem{Detail: err.Error()}, Err: err}
	if isTimeout {
		return &TimeoutError{TransportError: te}
	}
	return &te
}

// backoff returns the wait before retry n (0 for the first): a random time between 0 and
// min(8 s, 0.5 s * 2^n), which is "full jitter".
func backoff(n int) time.Duration {
	ceiling := float64(baseBackoff) * math.Pow(2, float64(n))
	if ceiling > float64(maxBackoff) {
		ceiling = float64(maxBackoff)
	}
	return time.Duration(rand.Float64() * ceiling)
}

// noteDeprecation calls the deprecation handler once per endpoint and process when the
// answer carries a Deprecation or Sunset header.
func (c *Client) noteDeprecation(endpoint string, h http.Header) {
	dep, sun := h.Get("Deprecation"), h.Get("Sunset")
	if dep == "" && sun == "" {
		return
	}
	once, _ := deprecationSeen.LoadOrStore(endpoint, new(sync.Once))
	once.(*sync.Once).Do(func() {
		c.onDeprecation(DeprecationNotice{Endpoint: endpoint, Deprecation: dep, Sunset: sun, Link: h.Get("Link")})
	})
}

// envelopeSetter is implemented by every response type through the embedded Envelope.
type envelopeSetter interface {
	setEnvelope(meta ResponseMeta, raw []byte)
}

func decodeSuccess(resp *rawResponse, out any) error {
	if err := json.Unmarshal(resp.body, out); err != nil {
		return &APIError{Problem: Problem{
			Status:     resp.status,
			Code:       "invalid_response",
			Detail:     "the answer is not valid JSON: " + err.Error(),
			APIRelease: resp.header.Get("X-Klaracase-Api-Release"),
		}}
	}
	if s, ok := out.(envelopeSetter); ok {
		s.setEnvelope(ResponseMeta{
			Status:     resp.status,
			APIRelease: resp.header.Get("X-Klaracase-Api-Release"),
			Duration:   resp.duration,
		}, resp.body)
	}
	return nil
}

// problem is the parsed body of an error answer.
type problem struct {
	isProblem bool
	members   map[string]json.RawMessage
	Problem
}

// parseProblem reads an error body. A body is a problem document when it is a JSON object
// with a string "code". Anything else gets the code "unknown" and the raw text as detail.
func parseProblem(resp *rawResponse) problem {
	p := problem{Problem: Problem{
		Status:     resp.status,
		APIRelease: resp.header.Get("X-Klaracase-Api-Release"),
	}}
	var members map[string]json.RawMessage
	if json.Unmarshal(resp.body, &members) == nil && members != nil {
		var code string
		if json.Unmarshal(members["code"], &code) == nil && members["code"] != nil {
			p.isProblem = true
			p.members = members
			p.Code = code
			_ = json.Unmarshal(members["type"], &p.Type)
			_ = json.Unmarshal(members["title"], &p.Title)
			_ = json.Unmarshal(members["detail"], &p.Detail)
			for k, v := range members {
				switch k {
				case "type", "title", "status", "detail", "code":
				default:
					if p.Extra == nil {
						p.Extra = map[string]json.RawMessage{}
					}
					p.Extra[k] = v
				}
			}
			return p
		}
	}
	p.Code = "unknown"
	p.Detail = truncateRunes(string(resp.body), maxErrorDetail)
	return p
}

func truncateRunes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

// number reads a numeric problem member (retry_after_seconds, cost, balance).
func (p problem) number(name string) (float64, bool) {
	raw, ok := p.Extra[name]
	if !ok {
		return 0, false
	}
	var f float64
	if json.Unmarshal(raw, &f) != nil {
		return 0, false
	}
	return f, true
}

// statedWait is the wait the server asks for: retry_after_seconds in the body first, then
// the Retry-After header in whole seconds. An HTTP date in the header is ignored.
func (p problem) statedWait(h http.Header) (time.Duration, bool) {
	if f, ok := p.number("retry_after_seconds"); ok && f >= 0 {
		return time.Duration(f * float64(time.Second)), true
	}
	if s := strings.TrimSpace(h.Get("Retry-After")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			return time.Duration(n) * time.Second, true
		}
	}
	return 0, false
}

// toError picks the error type by HTTP status.
func (p problem) toError(resp *rawResponse) error {
	base := p.Problem
	switch resp.status {
	case http.StatusUnauthorized:
		return &AuthenticationError{Problem: base}
	case http.StatusPaymentRequired:
		cost, _ := p.number("cost")
		balance, _ := p.number("balance")
		return &InsufficientCreditsError{Problem: base, Cost: int(cost), Balance: int(balance)}
	case http.StatusForbidden:
		e := &PermissionError{Problem: base}
		_ = json.Unmarshal(p.Extra["required_ability"], &e.RequiredAbility)
		return e
	case http.StatusNotFound:
		return &NotFoundError{Problem: base}
	case http.StatusConflict:
		return &ConflictError{Problem: base}
	case http.StatusUnprocessableEntity:
		e := &ValidationError{Problem: base}
		_ = json.Unmarshal(p.Extra["errors"], &e.Errors)
		return e
	case http.StatusTooManyRequests:
		e := &RateLimitError{Problem: base}
		if wait, ok := p.statedWait(resp.header); ok {
			e.RetryAfterSeconds = int(math.Ceil(wait.Seconds()))
		}
		return e
	case http.StatusServiceUnavailable:
		return &ServiceUnavailableError{Problem: base}
	default:
		return &APIError{Problem: base}
	}
}

// jsonRequest builds a POST request with a JSON body.
func jsonRequest(endpoint, path string, body any, opts []RequestOption) (request, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return request{}, fmt.Errorf("klaracase: encode request body: %w", err)
	}
	return request{method: http.MethodPost, endpoint: endpoint, path: path, body: data, contentType: "application/json", opts: opts}, nil
}
