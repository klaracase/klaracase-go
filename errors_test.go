package klaracase

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func problemClient(t *testing.T, status int, body string, header map[string]string, opts ...Option) *Client {
	t.Helper()
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		for k, v := range header {
			w.Header().Set(k, v)
		}
		writeProblem(w, status, body)
	}, append([]Option{WithMaxRetries(0)}, opts...)...)
	return c
}

func TestErrorCarriesHeaderMethodAndURLButNeverTheToken(t *testing.T) {
	c := problemClient(t, 429, `{"status":429,"code":"rate_limited","retry_after_seconds":42}`,
		map[string]string{"Retry-After": "42", "X-RateLimit-Remaining": "0", "Set-Cookie": "s=1"})
	_, err := c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "geheimer Suchtext"})
	base, ok := AsError(err)
	if !ok {
		t.Fatal(err)
	}
	if base.Method != http.MethodGet || !strings.HasSuffix(base.URL, "/api/v1/decisions/search") {
		t.Errorf("method %q url %q", base.Method, base.URL)
	}
	if u, perr := url.Parse(base.URL); perr != nil || u.RawQuery != "" || strings.Contains(base.URL, "geheimer") {
		t.Errorf("the URL must not carry the query string: %q", base.URL)
	}
	if base.Header.Get("X-RateLimit-Remaining") != "0" || base.Header.Get("Retry-After") != "42" {
		t.Errorf("header %v", base.Header)
	}
	if base.Header.Get("Set-Cookie") != "" {
		t.Error("Set-Cookie must be dropped")
	}
	for k := range base.Header {
		if strings.EqualFold(k, "Authorization") {
			t.Error("the Authorization header must never be kept")
		}
	}
	data, _ := json.Marshal(err)
	if strings.Contains(string(data)+err.Error(), "klc_test_token") || strings.Contains(string(data), "geheimer") {
		t.Errorf("secret in %s", data)
	}
}

func TestMethodAndURLAreSetOnEveryKindOfFailure(t *testing.T) {
	post, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "<html>") })
	_, err := post.CitationChecks.Create(context.Background(), CreateCitationCheckParams{Text: "x"})
	var de *DecodeError
	if !errors.As(err, &de) || de.Method != http.MethodPost || !strings.HasSuffix(de.URL, "/citation-checks") {
		t.Errorf("decode error %+v", de)
	}
	dropped, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}, WithMaxRetries(0))
	_, err = dropped.Corpus.Coverage(context.Background(), CoverageParams{})
	var te *TransportError
	if !errors.As(err, &te) || te.Method != http.MethodGet || !strings.HasSuffix(te.URL, "/corpus/coverage") {
		t.Errorf("transport error %+v", te)
	}
	_, err = dropped.Corpus.Coverage(context.Background(), CoverageParams{}, WithRequestTimeout(-1))
	if base, _ := AsError(err); base == nil || base.Method != "" || base.URL != "" {
		t.Errorf("a refused option has no request: %+v", base)
	}
}

func TestRetryAfterAndRetryable(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name      string
		status    int
		body      string
		header    map[string]string
		wantAfter time.Duration
		retryable bool
	}{
		{"body seconds", 429, `{"status":429,"code":"rate_limited","retry_after_seconds":42}`, nil, 42 * time.Second, true},
		{"header seconds", 503, `{"status":503,"code":"backend_unavailable"}`, map[string]string{"Retry-After": "7"}, 7 * time.Second, true},
		{"body wins", 429, `{"status":429,"code":"rate_limited","retry_after_seconds":3}`, map[string]string{"Retry-After": "9"}, 3 * time.Second, true},
		{"no wait stated", 429, `{"status":429,"code":"rate_limited"}`, nil, 0, true},
		{"not retryable", 402, `{"status":402,"code":"insufficient_credits","cost":2,"balance":0}`, nil, 0, false},
		{"not found", 404, `{"status":404,"code":"not_found"}`, nil, 0, false},
		{"server error", 500, `{"status":500,"code":"internal_error"}`, nil, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := problemClient(t, tc.status, tc.body, tc.header, withNow(func() time.Time { return now }))
			_, err := c.Corpus.Coverage(context.Background(), CoverageParams{})
			base, ok := AsError(err)
			if !ok {
				t.Fatal(err)
			}
			if got := base.RetryAfter(); got != tc.wantAfter {
				t.Errorf("RetryAfter = %v, want %v", got, tc.wantAfter)
			}
			if base.Retryable() != tc.retryable {
				t.Errorf("Retryable = %v, want %v", base.Retryable(), tc.retryable)
			}
		})
	}
}

func TestRetryAfterHTTPDateIsRead(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	var n atomic.Int32
	client, slept := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			w.Header().Set("Retry-After", now.Add(7*time.Second).UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, "{}")
	}, withNow(func() time.Time { return now }))
	if _, err := client.Corpus.Coverage(context.Background(), CoverageParams{}); err != nil {
		t.Fatal(err)
	}
	if len(*slept) != 1 || (*slept)[0] != 7*time.Second {
		t.Errorf("slept %v, want exactly [7s] from the HTTP date", *slept)
	}
	// A date in the past means "now": no wait, and a bad value falls back to the backoff.
	if got, ok := statedWaitOf(nil, http.Header{"Retry-After": {now.Add(-time.Hour).UTC().Format(http.TimeFormat)}}, now); !ok || got != 0 {
		t.Errorf("past date = %v, %v", got, ok)
	}
	if _, ok := statedWaitOf(nil, http.Header{"Retry-After": {"soon"}}, now); ok {
		t.Error("a bad Retry-After value must be ignored")
	}
}

func TestTypedExtras(t *testing.T) {
	t.Run("ambiguous_aktenzeichen", func(t *testing.T) {
		c := problemClient(t, 422, string(fixture(t, "conformance/problems/ambiguous_aktenzeichen.json")), nil)
		_, err := c.Decisions.Get(context.Background(), "5 U 123/19", GetDecisionParams{})
		base, _ := AsError(err)
		if len(base.Candidates) != 2 || base.Candidates[1].Court != "OLG Köln" || base.Candidates[1].Date != "2019-09-03" ||
			base.Candidates[0].ID == "" || base.Candidates[0].Aktenzeichen != "5 U 123/19" {
			t.Errorf("candidates %+v", base.Candidates)
		}
		if base.Code != CodeAmbiguousAktenzeichen {
			t.Errorf("code %q", base.Code)
		}
	})
	t.Run("statute_superseded", func(t *testing.T) {
		c := problemClient(t, 404, string(fixture(t, "conformance/problems/statute_superseded.json")), nil)
		_, err := c.Norms.Get(context.Background(), "§ 1 BBauG", GetNormParams{})
		base, _ := AsError(err)
		if base.Successor != "BauGB" || base.Code != CodeStatuteSuperseded || !IsNotFound(err) {
			t.Errorf("error %+v", base)
		}
	})
	t.Run("volume_cap", func(t *testing.T) {
		c := problemClient(t, 429, string(fixture(t, "conformance/problems/volume_cap.json")), map[string]string{"Retry-After": "41230"})
		_, err := c.Corpus.Coverage(context.Background(), CoverageParams{})
		base, _ := AsError(err)
		wantReset := time.Date(2026, 11, 1, 23, 0, 0, 0, time.UTC)
		if base.Unit != "rows" || base.Cap != 1000 || !base.ResetsAt.Equal(wantReset) || base.RetryAfter() != 41230*time.Second {
			t.Errorf("error %+v", base)
		}
		// The raw members stay in Extra, as before.
		if string(base.Extra["unit"]) != `"rows"` {
			t.Errorf("extra %v", base.Extra)
		}
	})
	t.Run("a wrongly typed member stays zero", func(t *testing.T) {
		c := problemClient(t, 429, `{"status":429,"code":"volume_cap","cap":"many","resets_at":"never","candidates":"x"}`, nil)
		_, err := c.Corpus.Coverage(context.Background(), CoverageParams{})
		base, _ := AsError(err)
		if base.Cap != 0 || !base.ResetsAt.IsZero() || base.Candidates != nil {
			t.Errorf("error %+v", base)
		}
	})
}

func TestErrorHelpers(t *testing.T) {
	cases := []struct {
		status int
		body   string
		is     func(error) bool
		name   string
	}{
		{404, `{"status":404,"code":"not_found"}`, IsNotFound, "IsNotFound"},
		{429, `{"status":429,"code":"rate_limited"}`, IsRateLimited, "IsRateLimited"},
		{402, `{"status":402,"code":"insufficient_credits"}`, IsInsufficientCredits, "IsInsufficientCredits"},
		{422, `{"status":422,"code":"validation_failed"}`, IsValidation, "IsValidation"},
		{403, `{"status":403,"code":"token_ability_missing"}`, IsPermission, "IsPermission"},
	}
	helpers := []func(error) bool{IsNotFound, IsRateLimited, IsInsufficientCredits, IsValidation, IsPermission}
	for i, tc := range cases {
		c := problemClient(t, tc.status, tc.body, nil)
		_, err := c.Corpus.Coverage(context.Background(), CoverageParams{})
		for j, h := range helpers {
			if got := h(err); got != (i == j) {
				t.Errorf("%s on a %d: helper %d said %v", tc.name, tc.status, j, got)
			}
		}
		wrapped := errors.Join(errors.New("context"), err)
		if !tc.is(wrapped) {
			t.Errorf("%s must see through errors.Join", tc.name)
		}
	}
	if IsNotFound(nil) || IsValidation(errors.New("plain")) {
		t.Error("helpers must be false for nil and for foreign errors")
	}
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {})
	_, vErr := client.Decisions.Search(context.Background(), SearchDecisionsParams{})
	if !IsValidation(vErr) {
		t.Error("a client side validation error is a validation error")
	}
}

func TestRetryableTransportError(t *testing.T) {
	dropped, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		conn, _, _ := w.(http.Hijacker).Hijack()
		conn.Close()
	}, WithMaxRetries(0))
	_, err := dropped.Corpus.Coverage(context.Background(), CoverageParams{})
	var te *TransportError
	if !errors.As(err, &te) || !te.Retryable() {
		t.Fatalf("a dropped connection is retryable: %v", err)
	}
	slow, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }, WithMaxRetries(0))
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(30 * time.Millisecond); cancel() }()
	_, err = slow.Corpus.Coverage(ctx, CoverageParams{})
	if !errors.As(err, &te) || te.Retryable() {
		t.Fatalf("a canceled call must not be retryable: %v", err)
	}
}

func TestErrorMarshalsForAToolResult(t *testing.T) {
	c := problemClient(t, 402, `{"type":"https://klaracase.de/docs/api/errors#insufficient_credits","title":"Payment Required","status":402,"detail":"Not enough credits.","code":"insufficient_credits","cost":2,"balance":0}`, nil)
	_, err := c.CitationChecks.Create(context.Background(), CreateCitationCheckParams{Text: "x"})
	data, mErr := json.Marshal(err)
	if mErr != nil {
		t.Fatal(mErr)
	}
	want := `{"code":"insufficient_credits","status":402,"title":"Payment Required","detail":"Not enough credits.","balance":0,"cost":2}`
	if string(data) != want {
		t.Errorf("got  %s\nwant %s", data, want)
	}
	rate := problemClient(t, 429, `{"status":429,"code":"rate_limited","detail":"slow down"}`, map[string]string{"Retry-After": "12"})
	_, err = rate.Corpus.Coverage(context.Background(), CoverageParams{})
	data, _ = json.Marshal(err)
	var m map[string]any
	if json.Unmarshal(data, &m) != nil || m["retry_after_seconds"] != float64(12) || m["code"] != "rate_limited" {
		t.Errorf("a header-only wait must show as retry_after_seconds: %s", data)
	}
	// A client side error shows its field messages.
	_, vErr := rate.Datapackages.Create(context.Background(), CreateDatapackageParams{})
	data, _ = json.Marshal(vErr)
	if !strings.Contains(string(data), `"code":"client_validation"`) || !strings.Contains(string(data), `"errors":{"ids":[`) {
		t.Errorf("client validation json: %s", data)
	}
	// Every error type marshals through the embedded problem.
	var _ json.Marshaler = (*NotFoundError)(nil)
	var _ json.Marshaler = (*TimeoutError)(nil)
	var _ json.Marshaler = (*DocumentTimeoutError)(nil)
	var _ json.Marshaler = (*ConfigurationError)(nil)
}

func TestAsErrorMatchesConfigurationError(t *testing.T) {
	_, err := New("")
	base, ok := AsError(err)
	if !ok || base.Code != CodeConfiguration || !strings.Contains(base.Detail, "no API token") {
		t.Errorf("AsError = %+v, %v", base, ok)
	}
	var cfg *ConfigurationError
	if !errors.As(err, &cfg) {
		t.Error("errors.As must still find *ConfigurationError")
	}
}

// factsCodes is the closed vocabulary of error codes from the contract facts (_contract/facts.md,
// the tables of /api/v1 codes), copied here so that a missing constant fails a test.
var factsCodes = []string{
	"unauthenticated", "personal_access_token_required", "token_org_missing", "token_org_membership_lost",
	"pseudonymizer_token_refused", "two_factor_required", "token_ability_missing", "validation_failed",
	"rate_limited", "insufficient_credits", "token_limit_reached", "volume_cap", "backend_unavailable", "service_unavailable",
	"internal_error", "bad_request", "forbidden", "not_found", "method_not_allowed", "conflict",
	"payload_too_large", "unsupported_media_type", "client_error",
	"not_found_under_filter", "statute_superseded", "section_not_in_current_text", "document_unavailable",
	"search_timeout", "index_unavailable", "norm_family_unavailable", "ambiguous_aktenzeichen",
	"ids_unresolved", "page_undefined_at_this_depth", "text_too_long", "not_german_legal_text",
	"unrecognisable_norm_token", "invalid_reference", "range_not_supported", "historical_version_unavailable",
	"ambiguous", "sections_unavailable", "section_not_found", "rn_range_not_found",
	"official_headnote_not_located", "document_leitsatz_not_located",
}

func TestEveryCodeOfTheVocabularyHasAConstant(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "codes.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	have := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		spec, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range spec.Names {
			lit, isLit := spec.Values[i].(*ast.BasicLit)
			if strings.HasPrefix(name.Name, "Code") && isLit {
				value, _ := strconv.Unquote(lit.Value)
				have[value] = name.Name
			}
		}
		return true
	})
	for _, code := range factsCodes {
		if have[code] == "" {
			t.Errorf("no Code constant for %q", code)
		}
	}
}

func TestInsufficientCreditsReasonAndTopUpURL(t *testing.T) {
	t.Run("api_budget_empty", func(t *testing.T) {
		c := problemClient(t, 402, string(fixture(t, "conformance/problems/insufficient_credits.api_budget_empty.json")), nil)
		_, err := c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
		var e *InsufficientCreditsError
		if !errors.As(err, &e) {
			t.Fatalf("got %T", err)
		}
		if e.Reason != ReasonAPIBudgetEmpty || e.TopUpURL != "https://klaracase.de/org/billing/api-budget" || e.Cost != 1 || e.Balance != 0 {
			t.Errorf("error %+v", e)
		}
		if IsTokenLimitReached(err) {
			t.Error("insufficient_credits is not a token limit")
		}
	})
	t.Run("api_budget_not_available has no top up URL", func(t *testing.T) {
		c := problemClient(t, 402, string(fixture(t, "conformance/problems/insufficient_credits.api_budget_not_available.json")), nil)
		_, err := c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
		var e *InsufficientCreditsError
		if !errors.As(err, &e) || e.Reason != ReasonAPIBudgetNotAvailable || e.TopUpURL != "" {
			t.Errorf("error %+v", e)
		}
	})
	t.Run("a plain 402 has neither", func(t *testing.T) {
		c := problemClient(t, 402, string(fixture(t, "conformance/problems/insufficient_credits.json")), nil)
		_, err := c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
		var e *InsufficientCreditsError
		if !errors.As(err, &e) || e.Reason != "" || e.TopUpURL != "" {
			t.Errorf("error %+v", e)
		}
	})
	t.Run("an unknown reason is kept", func(t *testing.T) {
		c := problemClient(t, 402, `{"status":402,"code":"insufficient_credits","cost":1,"balance":0,"reason":"a_future_reason"}`, nil)
		_, err := c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
		var e *InsufficientCreditsError
		if !errors.As(err, &e) || e.Reason != "a_future_reason" || string(e.Extra["reason"]) != `"a_future_reason"` {
			t.Errorf("error %+v", e)
		}
	})
}

func TestTokenLimitReachedError(t *testing.T) {
	c := problemClient(t, 402, string(fixture(t, "conformance/problems/token_limit_reached.json")), nil)
	_, err := c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
	var e *TokenLimitReachedError
	if !errors.As(err, &e) {
		t.Fatalf("got %T", err)
	}
	wantReset := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	if e.Unit != "credits" || e.Cap != 500 || e.Spent != 500 || !e.ResetsAt.Equal(wantReset) ||
		e.ResetsAtRaw != "2026-12-01T00:00:00+00:00" || e.SettingsURL != "https://klaracase.de/org/api-keys" {
		t.Errorf("error %+v", e)
	}
	if e.Code != CodeTokenLimitReached || e.Status != 402 || string(e.Extra["spent"]) != "500" {
		t.Errorf("problem %+v", e.Problem)
	}
	if !IsTokenLimitReached(err) || !IsInsufficientCredits(err) {
		t.Error("a token limit is a 402: both helpers must say yes")
	}
	var credits *InsufficientCreditsError
	if !errors.As(err, &credits) || credits.Code != CodeTokenLimitReached || credits.Cost != 0 || credits.Balance != 0 {
		t.Errorf("errors.As for the 402 type: %+v", credits)
	}
	if base, ok := AsError(err); !ok || base.Code != CodeTokenLimitReached {
		t.Errorf("AsError %+v", base)
	}
	if _, ok := e.Extra["cost"]; ok {
		t.Error("the answer carries no cost, so Extra must not have one")
	}
}

func TestTokenLimitReachedToleratesAbsentAndOddMembers(t *testing.T) {
	c := problemClient(t, 402, `{"status":402,"code":"token_limit_reached","detail":"x"}`, nil)
	_, err := c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
	var e *TokenLimitReachedError
	if !errors.As(err, &e) {
		t.Fatalf("got %T", err)
	}
	if e.Unit != "" || e.Cap != 0 || e.Spent != 0 || !e.ResetsAt.IsZero() || e.ResetsAtRaw != "" || e.SettingsURL != "" {
		t.Errorf("absent members must read as zero values: %+v", e)
	}
	c = problemClient(t, 402, `{"status":402,"code":"token_limit_reached","cap":"many","spent":"lots","resets_at":"soon","settings_url":7}`, nil)
	_, err = c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
	if !errors.As(err, &e) {
		t.Fatalf("got %T", err)
	}
	if e.Cap != 0 || e.Spent != 0 || !e.ResetsAt.IsZero() || e.ResetsAtRaw != "soon" || e.SettingsURL != "" {
		t.Errorf("odd members: %+v", e)
	}
}

func TestPaymentRequiredAnswersAreNeverRetried(t *testing.T) {
	for _, body := range []string{
		string(fixture(t, "conformance/problems/token_limit_reached.json")),
		string(fixture(t, "conformance/problems/insufficient_credits.api_budget_empty.json")),
	} {
		var hits atomic.Int32
		c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)
			writeProblem(w, 402, body)
		}, WithMaxRetries(3))
		_, err := c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
		base, ok := AsError(err)
		if !ok {
			t.Fatalf("got %v", err)
		}
		if hits.Load() != 1 || base.Retryable() {
			t.Errorf("%s: %d requests, retryable %v", base.Code, hits.Load(), base.Retryable())
		}
	}
}

func TestTokenLimitReachedMarshalsForAToolResult(t *testing.T) {
	c := problemClient(t, 402, string(fixture(t, "conformance/problems/token_limit_reached.json")), nil)
	_, err := c.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Untreue"})
	data, mErr := json.Marshal(err)
	if mErr != nil {
		t.Fatal(mErr)
	}
	var got map[string]any
	if json.Unmarshal(data, &got) != nil || got["code"] != "token_limit_reached" || got["spent"] != float64(500) ||
		got["settings_url"] != "https://klaracase.de/org/api-keys" {
		t.Errorf("got %s", data)
	}
	if _, ok := got["cost"]; ok {
		t.Errorf("no cost in %s", data)
	}
}
