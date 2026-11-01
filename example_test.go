package klaracase_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/klaracase/klaracase-go"
)

// The token comes from the KLARACASE_API_TOKEN environment variable. This example has no
// output because it needs that variable.
func ExampleNewFromEnv() {
	client, err := klaracase.NewFromEnv(
		klaracase.WithTimeout(20*time.Second),
		klaracase.WithMaxRetries(3),
		klaracase.WithUserAgentSuffix("my-app/1.0"),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

func ExampleDecisionsService_Search() {
	client, stop := exampleClient(http.StatusOK, exampleBody("decisions.search.json"))
	defer stop()

	res, err := client.Decisions.Search(context.Background(), klaracase.SearchDecisionsParams{
		Query: "Vermögensbetreuungspflicht Untreue",
		Court: "BGH",
		Limit: 5,
		// Bridge false is sent as bridge=false: the server adds no register terms.
		Bridge: klaracase.Bool(false),
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, hit := range res.Results {
		fmt.Println(hit.Court, hit.Aktenzeichen, hit.Date)
	}
	if left, ok := res.CreditsRemaining(); ok {
		fmt.Println("credits left:", left)
	}
	// Output:
	// BGH 1 StR 185/01 2001-11-15
	// credits left: 29
}

func ExampleDecisionsService_SearchAll() {
	pages := map[string]string{
		"1": `{"results":[{"aktenzeichen":"1 StR 1/20"},{"aktenzeichen":"1 StR 2/20"}],"page":1,"limit":2,"candidate_pool":4}`,
		"2": `{"results":[{"aktenzeichen":"1 StR 3/20"},{"aktenzeichen":"1 StR 4/20"}],"page":2,"limit":2,"candidate_pool":4}`,
	}
	client, stop := exampleClientFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, pages[r.URL.Query().Get("page")])
	})
	defer stop()

	// maxPages is required. Every page is one request, so keep it as low as your task allows.
	const maxPages = 3
	for hit, err := range client.Decisions.SearchAll(context.Background(), klaracase.SearchDecisionsParams{Query: "Untreue"}, maxPages) {
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(hit.Aktenzeichen)
	}
	// Output:
	// 1 StR 1/20
	// 1 StR 2/20
	// 1 StR 3/20
	// 1 StR 4/20
}

func ExampleDecisionsService_Get() {
	client, stop := exampleClient(http.StatusOK, exampleBody("decisions.show.json"))
	defer stop()

	// The id may be a UUID, an Aktenzeichen or an ECLI. The client encodes it as one path segment.
	res, err := client.Decisions.Get(context.Background(), "1 StR 185/01", klaracase.GetDecisionParams{
		Section: klaracase.SectionGruende,
		Court:   "BGH",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Aktenzeichen, res.Metadata.DocumentType, res.TextLength)
	fmt.Println("show this note to your user:", strings.HasPrefix(res.FortgeltungNote, "Rechtskraft/Fortgeltung"))
	// Output:
	// 1 StR 185/01 Urteil 48213
	// show this note to your user: true
}

func ExampleNormsService_Get() {
	client, stop := exampleClient(http.StatusOK, exampleBody("norms.show.json"))
	defer stop()

	res, err := client.Norms.Get(context.Background(), "§ 535 BGB", klaracase.GetNormParams{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Norm.Number, res.Norm.Title)
	// Output:
	// § 535 Inhalt und Hauptpflichten des Mietvertrags
}

func ExampleCitationChecksService_Create() {
	client, stop := exampleClient(http.StatusOK, exampleBody("citation-checks.store.json"))
	defer stop()

	text := strings.Repeat("Der Senat folgt BGH, Urt. v. 15.11.2001 – 1 StR 185/01. ", 3)
	run, err := client.CitationChecks.Create(context.Background(), klaracase.CreateCitationCheckParams{
		Text:          text,
		ReferenceDate: "2026-10-01",
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range run.Findings {
		switch f.Verdict {
		case klaracase.VerdictVerifiedCorpus:
			// The citation exists in the corpus. This does not prove that a quote or a legal claim is right.
			fmt.Println("found:", f.Canonical)
		case klaracase.VerdictNotFound:
			// Not found in the corpus. This does not mean the citation is invented. Check it by hand.
			fmt.Println("not found in the corpus:", f.RawText)
		default:
			fmt.Println(f.Verdict, f.Canonical)
		}
	}
	fmt.Println(run.VerdictCounts[klaracase.VerdictVerifiedCorpus], "verified,", run.VerdictCounts[klaracase.VerdictNotFound], "not found")
	// Output:
	// found: BGH 1 StR 185/01
	// 1 verified, 1 not found
}

func ExampleCitationCheckDocumentsService_WaitUntilReady() {
	// The mock answers "extracting" once and then "ready".
	polls := 0
	client, stop := exampleClientFunc(func(w http.ResponseWriter, _ *http.Request) {
		polls++
		status := "extracting"
		if polls > 1 {
			status = "ready"
		}
		_, _ = fmt.Fprintf(w, `{"id":"doc-1","status":%q,"quoted_credits":2}`, status)
	})
	defer stop()

	// The context is the only deadline of the wait.
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	ready, err := client.CitationCheckDocuments.WaitUntilReady(ctx, "doc-1", klaracase.WithPollInterval(10*time.Millisecond))
	var failed *klaracase.DocumentFailedError
	switch {
	case errors.As(err, &failed):
		log.Fatalf("the file cannot be read: %s", failed.ErrorCode)
	case errors.Is(err, context.DeadlineExceeded):
		log.Fatal("the document was not ready in time")
	case err != nil:
		log.Fatal(err)
	}
	if ready.QuotedCredits != nil {
		fmt.Println("the check costs", *ready.QuotedCredits, "credits")
	}
	// Output:
	// the check costs 2 credits
}

func ExampleCitationCheckDocumentsService_Upload() {
	client, stop := exampleClient(http.StatusAccepted, exampleBody("citation-check-documents.store.json"))
	defer stop()

	doc, err := client.CitationCheckDocuments.Upload(context.Background(), klaracase.UploadParams{
		File:        strings.NewReader("... file bytes ..."),
		Filename:    "schriftsatz.pdf",
		ContentType: "application/pdf",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(doc.Status, doc.Meta.Status)
	// Output:
	// uploaded 202
}

func Example_errorHandling() {
	body := []byte(`{"type":"https://klaracase.de/docs/api/errors#insufficient_credits","title":"Payment Required","status":402,
		"detail":"Insufficient credits.","code":"insufficient_credits","cost":2,"balance":0}`)
	client, stop := exampleClient(http.StatusPaymentRequired, body)
	defer stop()

	_, err := client.Decisions.Search(context.Background(), klaracase.SearchDecisionsParams{Query: "Untreue"})

	switch {
	case err == nil:
	case klaracase.IsInsufficientCredits(err):
		var credits *klaracase.InsufficientCreditsError
		if errors.As(err, &credits) {
			fmt.Printf("this call costs %d credits, the balance is %d\n", credits.Cost, credits.Balance)
		}
	case klaracase.IsRateLimited(err):
		if apiErr, ok := klaracase.AsError(err); ok {
			fmt.Println("try again in", apiErr.RetryAfter())
		}
	case klaracase.IsNotFound(err):
		fmt.Println("nothing found")
	default:
		fmt.Println(err)
	}

	// Every API error has the common fields. The code constants name the closed vocabulary.
	if apiErr, ok := klaracase.AsError(err); ok {
		fmt.Println(apiErr.Status, apiErr.Code == klaracase.CodeInsufficientCredits, apiErr.Retryable())
	}

	// An error is also JSON that fits a tool result for a language model.
	data, _ := json.Marshal(err)
	fmt.Println(string(data))
	// Output:
	// this call costs 2 credits, the balance is 0
	// 402 true false
	// {"code":"insufficient_credits","status":402,"title":"Payment Required","detail":"Insufficient credits.","balance":0,"cost":2}
}

func ExampleWithDeprecationHandler() {
	client, stop := exampleClientFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Sunset", "Sat, 01 Aug 2026 00:00:00 GMT")
		_, _ = io.WriteString(w, `{}`)
	}, klaracase.WithDeprecationHandler(func(n klaracase.DeprecationNotice) {
		fmt.Println(n.Endpoint, "is deprecated, sunset", n.Sunset)
	}))
	defer stop()

	// The handler runs once per endpoint and client, however often you call it.
	for range 3 {
		if _, err := client.Corpus.Coverage(context.Background(), klaracase.CoverageParams{}); err != nil {
			log.Fatal(err)
		}
	}
	// Output:
	// GET /corpus/coverage is deprecated, sunset Sat, 01 Aug 2026 00:00:00 GMT
}

func ExampleWithLogger() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey || a.Key == "wait" {
				return slog.Attr{} // keep the output stable for this example
			}
			return a
		},
	}))
	calls := 0
	client, stop := exampleClientFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}, klaracase.WithLogger(logger))
	defer stop()

	if _, err := client.Corpus.Coverage(context.Background(), klaracase.CoverageParams{}); err != nil {
		log.Fatal(err)
	}
	// The log has the method, the path, the status, the attempt and the wait. It never has the token.
	// Output:
	// level=DEBUG msg="klaracase: retrying" method=GET path=/corpus/coverage status=503 attempt=1
}

func ExampleWithHeader() {
	client, stop := exampleClientFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, `{"as_of":"2026-10-01T09:58:12+00:00","trace":%q}`, r.Header.Get("X-Trace-Id"))
	})
	defer stop()

	// A request option changes one call. The client option of the same kind stays as it was.
	res, err := client.Corpus.Coverage(context.Background(), klaracase.CoverageParams{Top: 5},
		klaracase.WithHeader("X-Trace-Id", "job-42"),
		klaracase.WithRequestTimeout(5*time.Second),
		klaracase.WithRequestMaxRetries(0),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(string(res.Extra["trace"]))
	// Output:
	// "job-42"
}

func ExampleCorpusService_Facets() {
	client, stop := exampleClient(http.StatusOK, exampleBody("corpus.facets.json"))
	defer stop()

	res, err := client.Corpus.Facets(context.Background(), klaracase.FacetCourtName, klaracase.FacetsParams{})
	if err != nil {
		log.Fatal(err)
	}
	// Values keep the order of the server.
	for _, v := range res.Values {
		fmt.Println(v.Key, v.Count.Value)
	}
	if bgh, ok := res.Values.Lookup("BGH"); ok {
		fmt.Println("BGH has", bgh.Value, "passages")
	}
	// Output:
	// BGH 90000
	// BVerfG 20000
	// BVerwG 15000
	// BGH has 90000 passages
}

func ExamplePtr() {
	params := klaracase.SearchDecisionsParams{Query: "Untreue", Bridge: klaracase.Ptr(false)}
	data, _ := json.Marshal(params)
	fmt.Println(string(data))
	// Output:
	// {"query":"Untreue","bridge":false}
}

func ExampleDate() {
	d := klaracase.Date("2026-10-01")
	t, err := d.Time()
	fmt.Println(t.Weekday(), err, klaracase.NewDate(t))
	// Output:
	// Thursday <nil> 2026-10-01
}
