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

func TestWaitUntilReadyStopsWhenTheContextIsCanceled(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"d1","status":"extracting"}`)
	})
	ctx, cancel := context.WithCancel(context.Background())
	client.sleep = func(ctx context.Context, d time.Duration) error { cancel(); return ctx.Err() }
	_, err := client.CitationCheckDocuments.WaitUntilReady(ctx, "d1")
	var te *DocumentTimeoutError
	if !errors.Is(err, context.Canceled) || errors.As(err, &te) {
		t.Fatalf("got %v, want the context.Canceled error and no DocumentTimeoutError", err)
	}
}

func TestWaitUntilReadyDeadlineIsADocumentTimeoutThatWrapsDeadlineExceeded(t *testing.T) {
	var polls atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		polls.Add(1)
		_, _ = io.WriteString(w, `{"id":"d1","status":"ocr_pending"}`)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	client.sleep = func(ctx context.Context, d time.Duration) error { <-ctx.Done(); return ctx.Err() }
	doc, err := client.CitationCheckDocuments.WaitUntilReady(ctx, "d1")
	if doc != nil {
		t.Error("no document is returned with an error")
	}
	var te *DocumentTimeoutError
	if !errors.As(err, &te) {
		t.Fatalf("got %T %v", err, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Error("errors.Is(err, context.DeadlineExceeded) must work")
	}
	if te.Code != "wait_timeout" || te.Document == nil || te.Document.Status != DocumentOCRPending {
		t.Errorf("error %+v", te)
	}
	if base, ok := AsError(err); !ok || base.Code != CodeWaitTimeout {
		t.Errorf("AsError = %v, %v", base, ok)
	}
	if polls.Load() != 1 {
		t.Errorf("polls = %d, want 1", polls.Load())
	}
}

func TestWaitUntilReadyDeadlineDuringAPoll(t *testing.T) {
	var polls atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if polls.Add(1) == 1 {
			_, _ = io.WriteString(w, `{"id":"d1","status":"extracting"}`)
			return
		}
		<-r.Context().Done()
	}, WithMaxRetries(0))
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	_, err := client.CitationCheckDocuments.WaitUntilReady(ctx, "d1", WithPollInterval(10*time.Millisecond))
	var te *DocumentTimeoutError
	if !errors.As(err, &te) || !errors.Is(err, context.DeadlineExceeded) || te.Document == nil || te.Document.Status != DocumentExtracting {
		t.Fatalf("got %T %v", err, err)
	}

	// A deadline that passes before any answer gives a timeout without a document.
	hang, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }, WithMaxRetries(0))
	ctx2, cancel2 := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel2()
	_, err = hang.CitationCheckDocuments.WaitUntilReady(ctx2, "d1")
	if !errors.As(err, &te) || te.Document != nil {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestWaitUntilReadyUsesThePollInterval(t *testing.T) {
	var polls atomic.Int32
	client, slept := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		status := "extracting"
		if polls.Add(1) == 3 {
			status = "ready"
		}
		_, _ = io.WriteString(w, `{"id":"d1","status":"`+status+`"}`)
	})
	doc, err := client.CitationCheckDocuments.WaitUntilReady(context.Background(), "d1", WithPollInterval(750*time.Millisecond))
	if err != nil || doc.Status != DocumentReady {
		t.Fatalf("doc %v err %v", doc, err)
	}
	if len(*slept) != 2 || (*slept)[0] != 750*time.Millisecond || (*slept)[1] != 750*time.Millisecond {
		t.Errorf("slept %v, want two of 750ms", *slept)
	}
	*slept = nil
	polls.Store(0)
	if _, err := client.CitationCheckDocuments.WaitUntilReady(context.Background(), "d1"); err != nil || (*slept)[0] != DefaultPollInterval {
		t.Errorf("default interval: slept %v err %v", *slept, err)
	}
}

func TestWaitUntilReadyFailedAndInvalidOptions(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"id":"d1","status":"failed","error_code":"encrypted_pdf"}`)
	})
	_, err := client.CitationCheckDocuments.WaitUntilReady(context.Background(), "d1")
	var failed *DocumentFailedError
	if !errors.As(err, &failed) || failed.ErrorCode != "encrypted_pdf" || failed.Code != "encrypted_pdf" || failed.Document == nil {
		t.Fatalf("got %v", err)
	}
	_, err = client.CitationCheckDocuments.WaitUntilReady(context.Background(), "d1", WithPollInterval(0))
	var cfg *ConfigurationError
	if !errors.As(err, &cfg) {
		t.Errorf("a zero interval must be a ConfigurationError, got %v", err)
	}
}

func TestWaitUntilReadyPassesRequestOptionsToEveryPoll(t *testing.T) {
	var headers []string
	var polls atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		headers = append(headers, r.Header.Get("X-Job"))
		status := "uploaded"
		if polls.Add(1) == 2 {
			status = "ready"
		}
		_, _ = io.WriteString(w, `{"id":"d1","status":"`+status+`"}`)
	})
	if _, err := client.CitationCheckDocuments.WaitUntilReady(context.Background(), "d1",
		WithPollRequestOptions(WithHeader("X-Job", "7"))); err != nil {
		t.Fatal(err)
	}
	if len(headers) != 2 || headers[0] != "7" || headers[1] != "7" {
		t.Errorf("headers %v", headers)
	}
}
