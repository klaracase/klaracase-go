package klaracase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func captureLogger() (*slog.Logger, *bytes.Buffer) {
	var buf bytes.Buffer
	return slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})), &buf
}

func TestLoggerSeesRetriesWithoutTheTokenOrTheQuery(t *testing.T) {
	logger, buf := captureLogger()
	var n atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) <= 2 {
			w.Header().Set("Retry-After", "3")
			writeProblem(w, 503, `{"status":503,"code":"backend_unavailable"}`)
			return
		}
		_, _ = io.WriteString(w, `{"results":[]}`)
	}, WithLogger(logger))
	if _, err := client.Decisions.Search(context.Background(), SearchDecisionsParams{Query: "Geheimer Suchtext Untreue"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 debug lines for 2 retries, got %d:\n%s", len(lines), out)
	}
	for i, line := range lines {
		for _, want := range []string{"level=DEBUG", "method=GET", "path=/decisions/search", "status=503", "attempt=" + string(rune('1'+i)), "wait=3s"} {
			if !strings.Contains(line, want) {
				t.Errorf("line %d lacks %q: %s", i, want, line)
			}
		}
	}
	for _, secret := range []string{"klc_test_token", "Bearer", "Geheimer", "Suchtext", "query="} {
		if strings.Contains(out, secret) {
			t.Errorf("the log must not contain %q:\n%s", secret, out)
		}
	}
}

func TestLoggerSeesATransportRetryWithoutTheURL(t *testing.T) {
	logger, buf := captureLogger()
	var n atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if n.Add(1) == 1 {
			conn, _, _ := w.(http.Hijacker).Hijack()
			conn.Close()
			return
		}
		_, _ = io.WriteString(w, "{}")
	}, WithLogger(logger))
	if _, err := client.Corpus.Facets(context.Background(), FacetCourtName, FacetsParams{Prefix: "Geheim"}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "status=0") || !strings.Contains(out, "cause=") || strings.Contains(out, "Geheim") || strings.Contains(out, "http://") {
		t.Errorf("log:\n%s", out)
	}
}

func TestDeprecationWarnsOncePerEndpointAndClient(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Deprecation", "true")
		w.Header().Set("Sunset", "Sat, 01 Aug 2026 00:00:00 GMT")
		_, _ = io.WriteString(w, "{}")
	}
	logger, buf := captureLogger()
	first, _ := newTestClient(t, handler, WithLogger(logger))
	second, _ := newTestClient(t, handler, WithLogger(logger))
	for range 3 {
		if _, err := first.Corpus.Coverage(context.Background(), CoverageParams{}); err != nil {
			t.Fatal(err)
		}
	}
	if got := strings.Count(buf.String(), "level=WARN"); got != 1 {
		t.Fatalf("one client, one endpoint: %d warnings\n%s", got, buf.String())
	}
	if _, err := second.Corpus.Coverage(context.Background(), CoverageParams{}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(buf.String(), "level=WARN"); got != 2 {
		t.Errorf("the guard is per client, so a second client warns again: %d warnings", got)
	}
	if !strings.Contains(buf.String(), "endpoint=\"GET /corpus/coverage\"") || strings.Contains(buf.String(), "klc_test_token") {
		t.Errorf("log:\n%s", buf.String())
	}
}

func TestCustomDeprecationHandlerWinsOverTheLogger(t *testing.T) {
	logger, buf := captureLogger()
	var notices []DeprecationNotice
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Sunset", "Sat, 01 Aug 2026 00:00:00 GMT")
		_, _ = io.WriteString(w, "{}")
	}, WithLogger(logger), WithDeprecationHandler(func(n DeprecationNotice) { notices = append(notices, n) }))
	for range 2 {
		if _, err := client.Corpus.Coverage(context.Background(), CoverageParams{}); err != nil {
			t.Fatal(err)
		}
	}
	if len(notices) != 1 || notices[0].Sunset == "" || strings.Contains(buf.String(), "WARN") {
		t.Errorf("notices %v log %q", notices, buf.String())
	}
}

func TestNilLoggerIsAConfigurationError(t *testing.T) {
	_, err := New("klc_test_token", WithLogger(nil))
	var cfg *ConfigurationError
	if !errors.As(err, &cfg) {
		t.Errorf("got %v", err)
	}
}
