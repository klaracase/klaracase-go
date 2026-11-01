package klaracase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// newTestClient starts a server with the handler and returns a client that does not sleep.
func newTestClient(t *testing.T, h http.HandlerFunc, opts ...Option) (*Client, *[]time.Duration) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	var mu sync.Mutex
	slept := &[]time.Duration{}
	base := []Option{
		WithBaseURL(srv.URL + "/api/v1"),
		withSleep(func(ctx context.Context, d time.Duration) error {
			mu.Lock()
			*slept = append(*slept, d)
			mu.Unlock()
			return ctx.Err()
		}),
	}
	c, err := New("klc_test_token", append(base, opts...)...)
	if err != nil {
		t.Fatal(err)
	}
	return c, slept
}

func writeProblem(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.Header().Set("X-Klaracase-Api-Release", "0.95.0")
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func fixture(t *testing.T, rel string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", rel))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestBackoffStaysInsideTheCap(t *testing.T) {
	for n := range 12 {
		ceiling := min(8*time.Second, 500*time.Millisecond<<n)
		var max time.Duration
		for range 300 {
			d := backoff(n)
			if d < 0 || d > ceiling {
				t.Fatalf("backoff(%d) = %v, want between 0 and %v", n, d, ceiling)
			}
			max = max2(max, d)
		}
		if max == 0 {
			t.Fatalf("backoff(%d) never left zero: no jitter", n)
		}
	}
}

func max2(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}

func TestConstructionErrors(t *testing.T) {
	cases := map[string][]Option{
		"negative retries": {WithMaxRetries(-1)},
		"negative timeout": {WithTimeout(-time.Second)},
		"nil http client":  {WithHTTPClient(nil)},
		"bad base url":     {WithBaseURL("not a url")},
		"ftp base url":     {WithBaseURL("ftp://klaracase.de/api/v1")},
	}
	for name, opts := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := New("klc_test_token", opts...)
			var cfg *ConfigurationError
			if !errors.As(err, &cfg) {
				t.Fatalf("got %v, want *ConfigurationError", err)
			}
			if _, ok := AsError(err); ok {
				t.Error("a ConfigurationError must not be an *Error")
			}
		})
	}
	if _, err := New("has space"); err == nil || strings.Contains(err.Error(), "has space") {
		t.Errorf("a token with whitespace must be refused without echoing it, got %v", err)
	}
}

func TestEnvTokenWhitespaceIsTrimmed(t *testing.T) {
	t.Setenv(EnvToken, "klc_test_token\n")
	if _, err := NewFromEnv(); err != nil {
		t.Fatal(err)
	}
}

func TestEveryErrorClassIsAnErrorAndKeepsItsTypedFields(t *testing.T) {
	type tc struct {
		status int
		body   string
		check  func(t *testing.T, err error)
	}
	cases := map[string]tc{
		"401": {401, `{"status":401,"code":"unauthenticated","detail":"d"}`, func(t *testing.T, err error) {
			var e *AuthenticationError
			mustAs(t, err, &e)
		}},
		"402": {402, `{"status":402,"code":"insufficient_credits","cost":5,"balance":3}`, func(t *testing.T, err error) {
			var e *InsufficientCreditsError
			mustAs(t, err, &e)
			if e.Cost != 5 || e.Balance != 3 {
				t.Errorf("cost %d balance %d", e.Cost, e.Balance)
			}
		}},
		"403": {403, `{"status":403,"code":"token_ability_missing","required_ability":"read"}`, func(t *testing.T, err error) {
			var e *PermissionError
			mustAs(t, err, &e)
			if e.RequiredAbility != "read" {
				t.Errorf("ability %q", e.RequiredAbility)
			}
		}},
		"404": {404, `{"status":404,"code":"not_found"}`, func(t *testing.T, err error) {
			var e *NotFoundError
			mustAs(t, err, &e)
		}},
		"409": {409, `{"status":409,"code":"token_org_missing"}`, func(t *testing.T, err error) {
			var e *ConflictError
			mustAs(t, err, &e)
		}},
		"422": {422, `{"status":422,"code":"validation_failed","errors":{"query":["too short","bad"]}}`, func(t *testing.T, err error) {
			var e *ValidationError
			mustAs(t, err, &e)
			if got := e.Errors["query"]; len(got) != 2 || got[1] != "bad" {
				t.Errorf("errors %v", e.Errors)
			}
		}},
		"429": {429, `{"status":429,"code":"rate_limited","retry_after_seconds":42}`, func(t *testing.T, err error) {
			var e *RateLimitError
			mustAs(t, err, &e)
			if e.RetryAfterSeconds != 42 {
				t.Errorf("retry after %d", e.RetryAfterSeconds)
			}
		}},
		"503": {503, `{"status":503,"code":"backend_unavailable"}`, func(t *testing.T, err error) {
			var e *ServiceUnavailableError
			mustAs(t, err, &e)
		}},
		"418": {418, `{"status":418,"code":"teapot"}`, func(t *testing.T, err error) {
			var e *APIError
			mustAs(t, err, &e)
		}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { writeProblem(w, c.status, c.body) }, WithMaxRetries(0))
			_, err := client.Corpus.Coverage(context.Background(), CoverageParams{})
			if err == nil {
				t.Fatal("expected an error")
			}
			c.check(t, err)
			base, ok := AsError(err)
			if !ok || base.Status != c.status || base.APIRelease != "0.95.0" {
				t.Errorf("AsError = %+v, %v", base, ok)
			}
			var viaAs *Error
			if !errors.As(err, &viaAs) || viaAs != base {
				t.Error("errors.As(*Error) and AsError disagree")
			}
			if !strings.HasPrefix(err.Error(), "klaracase:") || strings.Contains(err.Error(), "klc_test_token") {
				t.Errorf("error text %q", err.Error())
			}
		})
	}
}

func mustAs[T error](t *testing.T, err error, target *T) {
	t.Helper()
	if !errors.As(err, target) {
		t.Fatalf("%T (%v) is not the expected type", err, err)
	}
}

func TestTimeoutIsATransportErrorAndATimeoutError(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	}, WithMaxRetries(0), WithTimeout(50*time.Millisecond))
	_, err := client.Corpus.Coverage(context.Background(), CoverageParams{})
	var te *TimeoutError
	var tr *TransportError
	if !errors.As(err, &te) || !errors.As(err, &tr) {
		t.Fatalf("got %T %v, want both *TimeoutError and *TransportError", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("the cause should be reachable with errors.Is")
	}
	if base, ok := AsError(err); !ok || base.Status != 0 {
		t.Errorf("AsError = %v, %v", base, ok)
	}
}

func TestCancelledContextIsNotRetriedAndStaysReachable(t *testing.T) {
	var calls atomic.Int32
	client, slept := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		<-r.Context().Done()
	}, WithMaxRetries(3))
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_, err := client.Corpus.Coverage(ctx, CoverageParams{})
	var tr *TransportError
	if !errors.As(err, &tr) || !errors.Is(err, context.Canceled) {
		t.Fatalf("got %T %v", err, err)
	}
	var te *TimeoutError
	if errors.As(err, &te) {
		t.Error("a cancel is not a timeout")
	}
	if calls.Load() != 1 || len(*slept) != 0 {
		t.Errorf("calls %d, sleeps %v: a cancelled call must not retry", calls.Load(), *slept)
	}
}

func TestContextDeadlineEndsTheCall(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }, WithMaxRetries(0))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := client.Corpus.Coverage(ctx, CoverageParams{})
	var te *TimeoutError
	if !errors.As(err, &te) {
		t.Fatalf("got %T %v, want *TimeoutError", err, err)
	}
}

func TestUnknownKeysSurviveInNestedRows(t *testing.T) {
	body := fixture(t, "conformance/bodies/decisions.search.future_key.json")
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	res, err := client.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
	if err != nil {
		t.Fatal(err)
	}
	if string(res.Results[0].Extra["future_result_key"]) != "true" {
		t.Errorf("row extra = %v", res.Results[0].Extra)
	}
	if string(res.Extra["another_future_key"]) != `"x"` {
		t.Errorf("top level extra = %v", res.Extra)
	}
	if _, known := res.Extra["results"]; known {
		t.Error("a modelled key must not appear in Extra")
	}
	if !bytes.Equal(bytes.Join(bytes.Fields(res.Raw), nil), bytes.Join(bytes.Fields(body), nil)) {
		t.Error("Raw must be the body unchanged")
	}
	if res.Results[0].Snippets[0].Rn == nil || *res.Results[0].Snippets[0].Rn != 21 {
		t.Error("the nested snippet rn is not typed")
	}
}

func TestCitationCheckResponseIsTyped(t *testing.T) {
	body := fixture(t, "examples/citation-checks.store.json")
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
	run, err := client.CitationChecks.Create(context.Background(), CreateCitationCheckParams{Text: strings.Repeat("a", 60)})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != RunComplete || run.VerdictCounts[VerdictNotFound] != 1 || run.VerdictCounts[VerdictVerifiedCorpus] != 1 {
		t.Errorf("status %q counts %v", run.Status, run.VerdictCounts)
	}
	if run.UniqueCitations[1].Verdict != VerdictNotFound || run.UniqueCitations[1].TargetArtifactID != nil {
		t.Errorf("unique citation %+v", run.UniqueCitations[1])
	}
	if f := run.Findings[0]; f.MatchedDecision == nil || f.MatchedDecision.Court != "BGH" || f.Evidence.MatchStrength != "exact" {
		t.Errorf("finding %+v", f)
	}
	if run.Credits == nil || *run.Credits.Cost != 2 {
		t.Errorf("credits %+v", run.Credits)
	}
}

func TestDeprecationDefaultHandlerLogsOnce(t *testing.T) {
	resetDeprecationState()
	var mu sync.Mutex
	var lines []string
	old := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(writerFunc(func(p []byte) (int, error) {
		mu.Lock()
		lines = append(lines, string(p))
		mu.Unlock()
		return len(p), nil
	}), nil)))
	defer slog.SetDefault(old)

	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Sunset", "Sat, 01 Aug 2026 00:00:00 GMT")
		_, _ = io.WriteString(w, "{}")
	})
	for range 3 {
		if _, err := client.Corpus.Coverage(context.Background(), CoverageParams{}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := client.Corpus.Facets(context.Background(), FacetCourtName, FacetsParams{}); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(lines) != 2 {
		t.Fatalf("want one warning per endpoint (2), got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "GET /corpus/coverage") || !strings.Contains(lines[0], "WARN") {
		t.Errorf("warning %q", lines[0])
	}
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func TestUploadRetrySendsTheWholeFileAgain(t *testing.T) {
	var bodies [][]byte
	var n atomic.Int32
	client, slept := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		bodies = append(bodies, data)
		if n.Add(1) == 1 {
			writeProblem(w, 429, `{"status":429,"code":"rate_limited","retry_after_seconds":3}`)
			return
		}
		w.WriteHeader(202)
		_, _ = io.WriteString(w, `{"id":"d1","status":"uploaded"}`)
	})
	doc, err := client.CitationCheckDocuments.Upload(context.Background(), UploadParams{
		File: strings.NewReader("hello file"), Filename: `we"ird.txt`, ContentType: "text/plain",
	})
	if err != nil {
		t.Fatal(err)
	}
	if doc.Meta.Status != 202 || doc.Status != DocumentUploaded {
		t.Errorf("doc %+v", doc)
	}
	if len(bodies) != 2 || !bytes.Equal(bodies[0], bodies[1]) || !bytes.Contains(bodies[1], []byte("hello file")) {
		t.Errorf("the retry must resend the same body, got %d bodies", len(bodies))
	}
	if len(*slept) != 1 || (*slept)[0] != 3*time.Second {
		t.Errorf("slept %v, want [3s]", *slept)
	}
}

func TestRetryAfterHTTPDateFallsBackToBackoff(t *testing.T) {
	var n atomic.Int32
	client, slept := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			w.Header().Set("Retry-After", "Wed, 21 Oct 2026 07:28:00 GMT")
			w.WriteHeader(503)
			return
		}
		_, _ = io.WriteString(w, "{}")
	})
	if _, err := client.Corpus.Coverage(context.Background(), CoverageParams{}); err != nil {
		t.Fatal(err)
	}
	if len(*slept) != 1 || (*slept)[0] > 500*time.Millisecond {
		t.Errorf("slept %v, want one backoff of at most 500ms", *slept)
	}
}

func TestClientSideValidationSendsNothing(t *testing.T) {
	var calls atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	ctx := context.Background()
	calls0 := []func() error{
		func() error { _, err := client.Decisions.Search(ctx, SearchDecisionsParams{}); return err },
		func() error { _, err := client.Search(ctx, CrossTypeSearchParams{}); return err },
		func() error { _, err := client.Decisions.Get(ctx, "", GetDecisionParams{}); return err },
		func() error { _, err := client.Decisions.Citations(ctx, "", CitationsParams{}); return err },
		func() error { _, err := client.Norms.Get(ctx, "", GetNormParams{}); return err },
		func() error { _, err := client.Datapackages.Create(ctx, CreateDatapackageParams{}); return err },
		func() error { _, err := client.CitationChecks.Get(ctx, ""); return err },
		func() error { _, err := client.CitationCheckDocuments.Get(ctx, ""); return err },
		func() error {
			_, err := client.CitationCheckDocuments.Upload(ctx, UploadParams{Filename: "a.pdf"})
			return err
		},
		func() error {
			_, err := client.CitationCheckDocuments.Upload(ctx, UploadParams{File: strings.NewReader("x")})
			return err
		},
		func() error { _, err := client.Corpus.Facets(ctx, "", FacetsParams{}); return err },
	}
	for i, f := range calls0 {
		var ve *ValidationError
		err := f()
		if !errors.As(err, &ve) || ve.Code != "client_validation" || ve.Status != 0 {
			t.Errorf("call %d: got %v", i, err)
		}
	}
	if calls.Load() != 0 {
		t.Errorf("the server saw %d requests", calls.Load())
	}
}

func TestCustomHTTPClientIsUsedAndNotChanged(t *testing.T) {
	var used atomic.Int32
	custom := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		used.Add(1)
		return &http.Response{
			StatusCode: 200, Header: http.Header{"X-Klaracase-Api-Release": {"0.95.0"}},
			Body: io.NopCloser(strings.NewReader(`{"total_decisions":{"value":7,"is_floor":true}}`)), Request: r,
		}, nil
	})}
	c, err := New("klc_test_token", WithHTTPClient(custom), WithTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	res, err := c.Corpus.Coverage(context.Background(), CoverageParams{})
	if err != nil {
		t.Fatal(err)
	}
	if used.Load() != 1 || res.TotalDecisions.Value != 7 || !res.TotalDecisions.IsFloor {
		t.Errorf("used %d, res %+v", used.Load(), res.TotalDecisions)
	}
	if custom.Timeout != 0 {
		t.Error("the client must not change the http.Client it was given")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestWaitUntilReadyStopsWhenTheContextEnds(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"d1","status":"extracting"}`)
	})
	ctx, cancel := context.WithCancel(context.Background())
	client.sleep = func(ctx context.Context, d time.Duration) error { cancel(); return ctx.Err() }
	_, err := client.CitationCheckDocuments.WaitUntilReady(ctx, "d1", WaitParams{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestWaitUntilReadyUsesTheInjectedClock(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	client, slept := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"d1","status":"extracting"}`)
	}, withNow(func() time.Time { return now }))
	client.sleep = func(ctx context.Context, d time.Duration) error {
		now = now.Add(d)
		*slept = append(*slept, d)
		return nil
	}
	_, err := client.CitationCheckDocuments.WaitUntilReady(context.Background(), "d1", WaitParams{PollInterval: 10 * time.Second, Timeout: 25 * time.Second})
	var te *DocumentTimeoutError
	if !errors.As(err, &te) || te.Document.Status != DocumentExtracting || te.Code != "wait_timeout" {
		t.Fatalf("got %v", err)
	}
	if len(*slept) != 3 || (*slept)[2] != 5*time.Second {
		t.Errorf("slept %v, want 10s, 10s, 5s (the last one is cut to the remaining time)", *slept)
	}
}

func TestClientIsSafeForConcurrentUse(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("query") == "boom" {
			writeProblem(w, 404, `{"status":404,"code":"not_found"}`)
			return
		}
		_, _ = io.WriteString(w, `{"results":[]}`)
	})
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q := "ok"
			if i%2 == 0 {
				q = "boom"
			}
			_, err := client.Decisions.Search(context.Background(), SearchDecisionsParams{Query: q})
			if (err != nil) != (q == "boom") {
				t.Errorf("query %s: err %v", q, err)
			}
		}()
	}
	wg.Wait()
}

func TestBodyOfAnInvalidSuccessIsADecodeError(t *testing.T) {
	long := `{"total_decisions":{"value":"seven"},"pad":"` + strings.Repeat("ä", 1500) + `"}`
	for name, body := range map[string]string{"not json": "<html>", "type mismatch": long} {
		t.Run(name, func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Klaracase-Api-Release", "0.95.0")
				_, _ = io.WriteString(w, body)
			})
			res, err := client.Corpus.Coverage(context.Background(), CoverageParams{})
			if res != nil {
				t.Error("a decode error returns no response")
			}
			var de *DecodeError
			if !errors.As(err, &de) {
				t.Fatalf("got %T %v, want *DecodeError", err, err)
			}
			if de.Status != 200 || de.Code != "invalid_response" || de.APIRelease != "0.95.0" || de.Err == nil {
				t.Errorf("decode error %+v", de)
			}
			if string(de.Raw) != body {
				t.Error("Raw must hold the whole body, so a paid answer is not lost")
			}
			if n := len([]rune(de.Body)); n > 1000 || (len([]rune(body)) > 1000 && n != 1000) {
				t.Errorf("Body has %d runes, want at most 1000 and exactly 1000 for a long body", n)
			}
			if base, ok := AsError(err); !ok || base.Code != "invalid_response" {
				t.Errorf("AsError = %v, %v", base, ok)
			}
			var typeErr *json.UnmarshalTypeError
			if name == "type mismatch" && !errors.As(err, &typeErr) {
				t.Errorf("the decoder error must stay reachable: %v", err)
			}
		})
	}
}

func TestPathEscapeFollowsRFC3986(t *testing.T) {
	cases := map[string]string{
		"VIII ZR 271/17":   "VIII%20ZR%20271%2F17",
		"§ 535 BGB":        "%C2%A7%20535%20BGB",
		"a!*'():@&=+$,;b":  "a%21%2A%27%28%29%3A%40%26%3D%2B%24%2C%3Bb",
		"keep-._~":         "keep-._~",
		"ECLI:DE:BGH:2001": "ECLI%3ADE%3ABGH%3A2001",
	}
	for in, want := range cases {
		if got := escapeSegment(in); got != want {
			t.Errorf("escapeSegment(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestJSONBodyOmitsUnsetFieldsButKeepsFalse(t *testing.T) {
	var got map[string]any
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = io.WriteString(w, "{}")
	})
	_, err := client.Datapackages.Create(context.Background(), CreateDatapackageParams{
		IDs: []string{"a"}, IncludeCitations: Bool(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := got["include_citations"]; !ok || v != false {
		t.Errorf("include_citations missing or wrong: %v", got)
	}
	for _, key := range []string{"max_chars", "full_text"} {
		if _, ok := got[key]; ok {
			t.Errorf("%s must be absent: %v", key, got)
		}
	}
}
