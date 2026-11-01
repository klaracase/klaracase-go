package klaracase

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func TestWithHeaderIsSentOnEveryAttempt(t *testing.T) {
	var seen []string
	var n atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Header.Get("X-Trace-Id")+"|"+r.Header.Get("Authorization")+"|"+r.Header.Get("Accept"))
		if n.Add(1) == 1 {
			writeProblem(w, 503, `{"status":503,"code":"backend_unavailable"}`)
			return
		}
		_, _ = io.WriteString(w, "{}")
	})
	_, err := client.Corpus.Coverage(context.Background(), CoverageParams{},
		WithHeader("x-trace-id", "abc"), WithHeader("Accept", "application/json; q=1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "abc|Bearer klc_test_token|application/json; q=1"
	if len(seen) != 2 || seen[0] != want || seen[1] != want {
		t.Errorf("headers per attempt = %q, want two of %q", seen, want)
	}
}

func TestWithHeaderRefusesAuthorizationAndContentType(t *testing.T) {
	var calls atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	for _, name := range []string{"Authorization", "content-type", ""} {
		_, err := client.Corpus.Coverage(context.Background(), CoverageParams{}, WithHeader(name, "x"))
		var cfg *ConfigurationError
		if !errors.As(err, &cfg) {
			t.Errorf("header %q: got %v, want *ConfigurationError", name, err)
		}
	}
	if calls.Load() != 0 {
		t.Errorf("the server saw %d requests", calls.Load())
	}
}

func TestRequestMaxRetriesWinsOverTheClientOption(t *testing.T) {
	fail := func(w http.ResponseWriter, r *http.Request) {
		writeProblem(w, 503, `{"status":503,"code":"backend_unavailable"}`)
	}
	t.Run("zero for one call", func(t *testing.T) {
		var calls atomic.Int32
		client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); fail(w, r) }, WithMaxRetries(2))
		_, err := client.Corpus.Coverage(context.Background(), CoverageParams{}, WithRequestMaxRetries(0))
		var e *ServiceUnavailableError
		if !errors.As(err, &e) || calls.Load() != 1 {
			t.Fatalf("calls %d err %v", calls.Load(), err)
		}
		if _, err := client.Corpus.Coverage(context.Background(), CoverageParams{}); err == nil || calls.Load() != 4 {
			t.Errorf("the client default must stay in force for the next call: calls %d", calls.Load())
		}
	})
	t.Run("more for one call", func(t *testing.T) {
		var calls atomic.Int32
		client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); fail(w, r) }, WithMaxRetries(0))
		_, _ = client.Corpus.Coverage(context.Background(), CoverageParams{}, WithRequestMaxRetries(3))
		if calls.Load() != 4 {
			t.Errorf("calls %d, want 4", calls.Load())
		}
	})
}

func TestRequestTimeoutWinsOverTheClientOption(t *testing.T) {
	slow := func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(300 * time.Millisecond):
			_, _ = io.WriteString(w, "{}")
		case <-r.Context().Done():
		}
	}
	client, _ := newTestClient(t, slow, WithMaxRetries(0), WithTimeout(time.Minute))
	_, err := client.Corpus.Coverage(context.Background(), CoverageParams{}, WithRequestTimeout(40*time.Millisecond))
	var te *TimeoutError
	if !errors.As(err, &te) {
		t.Fatalf("got %v, want *TimeoutError", err)
	}
	short, _ := newTestClient(t, slow, WithMaxRetries(0), WithTimeout(40*time.Millisecond))
	if _, err := short.Corpus.Coverage(context.Background(), CoverageParams{}, WithRequestTimeout(0)); err != nil {
		t.Errorf("a zero request timeout turns the limit off: %v", err)
	}
}

func TestInvalidRequestOptionsAreConfigurationErrors(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { t.Error("a request was sent") })
	for name, opt := range map[string]RequestOption{
		"negative timeout": WithRequestTimeout(-1),
		"negative retries": WithRequestMaxRetries(-1),
	} {
		_, err := client.Corpus.Facets(context.Background(), FacetCourtName, FacetsParams{}, opt)
		var cfg *ConfigurationError
		if !errors.As(err, &cfg) {
			t.Errorf("%s: got %v", name, err)
		}
	}
}
