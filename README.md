# klaracase-go

The official Go client for the [Klaracase](https://klaracase.de) REST API v1.

Klaracase covers German law only. It is a research tool. It gives no legal advice and does not
replace a lawyer's review. A `not_found` verdict means the citation was not found in the corpus;
it is not a claim that the citation is fabricated.

This client is small and uses only the Go standard library. It needs Go 1.24 or newer.

## Install

```sh
go get github.com/klaracase/klaracase-go
```

The package name is `klaracase`:

```go
import "github.com/klaracase/klaracase-go"
```

## Get a token

Create an API token at <https://klaracase.de/settings/api-tokens>. You need to be signed in with a
verified email address.

- A token has up to five abilities: `search`, `read`, `graph`, `datapackage` and `citation-check`.
  All five are ticked by default.
- A token expires after 30, 90, 180 or 365 days. The default is 365 days.
- A token belongs to the organization that was open when you created it. Every call is billed to
  that organization.

Keep the token out of your source code. Put it in the `KLARACASE_API_TOKEN` environment variable:

```sh
export KLARACASE_API_TOKEN=klc_your_token_here
```

Then pick one way to build the client:

- `klaracase.NewFromEnv()` reads `KLARACASE_API_TOKEN`. A missing or empty variable returns a
  `*klaracase.ConfigurationError`.
- `klaracase.New(token)` takes the token as an argument. An empty token returns a
  `*klaracase.ConfigurationError`. It does not fall back to the environment.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/klaracase/klaracase-go"
)

func main() {
	client, err := klaracase.NewFromEnv() // reads KLARACASE_API_TOKEN
	if err != nil {
		log.Fatal(err)
	}
	res, err := client.Decisions.Search(context.Background(), klaracase.SearchDecisionsParams{
		Query: "Vermögensbetreuungspflicht Untreue",
		Court: "BGH",
		Limit: 5,
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
}
```

`klaracase.New("klc_your_token_here")` takes the token as an argument instead. An empty token
returns a `*klaracase.ConfigurationError` at once, not a 401 later.

### Optional values

A string, a number and a slice that are unset are not sent. A zero number means "not set", because
no integer parameter of the API accepts zero. A boolean is a `*bool`, so that `false` can be sent
and "not set" stays different. Use `klaracase.Bool(b)` or the generic `klaracase.Ptr(v)`:

```go
klaracase.SearchDecisionsParams{Query: "Untreue", Bridge: klaracase.Bool(false), Limit: 5}
```

A list such as `CitedStatutes` is sent as `cited_statutes[]=StGB&cited_statutes[]=StPO`.
Dates are `klaracase.Date` values in the form `YYYY-MM-DD`. Enums such as `Branch`, `Section`,
`Jurisdiction` and `Statute` are typed strings with constants. A value that is not a constant is
sent as it is. The server decides what it accepts.

The client checks only what needs no server and sends nothing when a check fails: a missing
required value, a date that is not `YYYY-MM-DD`, `DateFrom` after `DateTo`, `RnFrom` after `RnTo`,
and a datapackage that does not hold 1 to 10 ids. The error is a `*klaracase.ValidationError` with
the code `client_validation`.

## Operations

| Call | HTTP | Credits |
|---|---|---|
| `client.Decisions.Search(ctx, SearchDecisionsParams{...})` | `GET /decisions/search` | 1 |
| `client.Search(ctx, CrossTypeSearchParams{...})` | `GET /search` | 1 |
| `client.Decisions.Get(ctx, id, GetDecisionParams{...})` | `GET /decisions/{id}` | 1 |
| `client.Decisions.Citations(ctx, id, CitationsParams{...})` | `GET /decisions/{id}/citations` | 2 |
| `client.Norms.Get(ctx, "§ 535 BGB", GetNormParams{...})` | `GET /norms/{reference}` | 1 |
| `client.Datapackages.Create(ctx, CreateDatapackageParams{IDs: ...})` | `POST /datapackages` | 5 |
| `client.CitationChecks.Create(ctx, CreateCitationCheckParams{...})` | `POST /citation-checks` | from 2 |
| `client.CitationChecks.Get(ctx, runID)` | `GET /citation-checks/{run}` | free |
| `client.CitationCheckDocuments.Upload(ctx, UploadParams{...})` | `POST /citation-check-documents` | free |
| `client.CitationCheckDocuments.Get(ctx, id)` | `GET /citation-check-documents/{document}` | free |
| `client.CitationCheckDocuments.WaitUntilReady(ctx, id, WithPollInterval(d))` | polls the call above | free |
| `client.Corpus.Coverage(ctx, CoverageParams{...})` | `GET /corpus/coverage` | free |
| `client.Corpus.Facets(ctx, field, FacetsParams{...})` | `GET /corpus/facets/{field}` | free |

The decision `id` is a UUID, an Aktenzeichen or an ECLI. The client encodes each path parameter as
one path segment, so `VIII ZR 271/17` becomes `VIII%20ZR%20271%2F17`.

The full reference is at <https://klaracase.de/docs/api>.

## Check the citations in a text

`CitationChecks.Create` answers with the finished run. Each finding has a verdict.

```go
run, err := client.CitationChecks.Create(ctx, klaracase.CreateCitationCheckParams{
	Text:          brief, // 50 to 20,000 characters of German legal text
	ReferenceDate: "2026-10-01",
})
if err != nil {
	log.Fatal(err)
}
for _, f := range run.Findings {
	switch f.Verdict {
	case klaracase.VerdictVerifiedCorpus:
		// The citation exists in the corpus. This does not prove that a quote
		// or a legal claim is right.
		fmt.Println("found:", f.Canonical)
	case klaracase.VerdictNotFound:
		// The citation was not found in the corpus. This does not mean that
		// it is invented. Check it by hand.
		fmt.Printf("not found in the corpus: %q at %d-%d\n", f.RawText, f.StartOffset, f.EndOffset)
	default:
		fmt.Println(f.Verdict, f.Canonical)
	}
}
```

The verdicts are `verified_corpus`, `verified_external`, `not_found`, `out_of_coverage`,
`ambiguous`, `unverifiable` and `discrepancy`. Each has a constant, such as
`klaracase.VerdictNotFound`. The offsets count Unicode code points in the text you sent.

The API checks against the corpus only. It does not ask an external register, so `not_found`
is weaker evidence here than in the Klaracase web app.

The price is `max(2, ceil(characters / 5000))` credits: 2 up to 10,000 characters, 3 up to 15,000
and 4 at 20,000. The same text checked again in the same month costs nothing more.

### Check a file

```go
doc, err := client.CitationCheckDocuments.Upload(ctx, klaracase.UploadParams{
	File:        file, // an io.Reader: PDF, DOCX, Markdown or plain text, up to 25 MB
	Filename:    "schriftsatz.pdf",
	ContentType: "application/pdf",
})
if err != nil {
	log.Fatal(err)
}
ctx, cancel := context.WithTimeout(ctx, 2*time.Minute) // the context is the only deadline
defer cancel()
ready, err := client.CitationCheckDocuments.WaitUntilReady(ctx, doc.ID,
	klaracase.WithPollInterval(2*time.Second)) // the default
if err != nil {
	log.Fatal(err) // *DocumentFailedError or *DocumentTimeoutError, see below
}
// ready.QuotedCredits is the price of the check, or nil above 20,000 characters.
run, err := client.CitationChecks.Create(ctx, klaracase.CreateCitationCheckParams{DocumentID: ready.ID})
```

Set exactly one of `Text` and `DocumentID`. If you set both or neither, `Create` returns a
`*klaracase.ValidationError` with the code `client_validation` and sends nothing.

The client reads the whole file into memory before the first attempt. A retry then sends the same
bytes.

## Handle errors

Every failed call returns an error from this package. Pick the type with `errors.As`, or use a
helper for the common cases:

```go
res, err := client.Decisions.Search(ctx, klaracase.SearchDecisionsParams{Query: "Untreue"})

var credits *klaracase.InsufficientCreditsError
switch {
case err == nil:
	_ = res
case errors.As(err, &credits):
	fmt.Printf("this call costs %d credits, the balance is %d\n", credits.Cost, credits.Balance)
case klaracase.IsRateLimited(err):
	e, _ := klaracase.AsError(err)
	fmt.Println("try again in", e.RetryAfter())
case klaracase.IsNotFound(err):
	fmt.Println("nothing found")
default:
	log.Fatal(err)
}
```

The helpers are `IsNotFound`, `IsRateLimited`, `IsInsufficientCredits`, `IsValidation` and
`IsPermission`.

| Type | When | Extra fields |
|---|---|---|
| `*AuthenticationError` | 401 | |
| `*InsufficientCreditsError` | 402 | `Cost`, `Balance` |
| `*PermissionError` | 403 | `RequiredAbility` |
| `*NotFoundError` | 404 | |
| `*ConflictError` | 409 | |
| `*ValidationError` | 422, or a client side check (code `client_validation`) | `Errors` (field to messages) |
| `*RateLimitError` | 429 | `RetryAfterSeconds` |
| `*ServiceUnavailableError` | 503 | |
| `*APIError` | any other status, or a body that is not a problem document | |
| `*TransportError` | no HTTP answer; wraps the cause (`errors.Is(err, context.Canceled)` works) | `Err` |
| `*TimeoutError` | a timeout; also matches `*TransportError` | |
| `*DecodeError` | a success status, but the body does not fit the types | `Err`, `Body`, `Raw` |
| `*DocumentFailedError` | `WaitUntilReady` saw the status `failed` | `ErrorCode`, `Document` |
| `*DocumentTimeoutError` | `WaitUntilReady` hit the deadline of the context; wraps `context.DeadlineExceeded` | `Document` |
| `*ConfigurationError` | no token, a bad base URL, a bad request option | `Message` |

Every type, `*ConfigurationError` included, holds the common problem fields in an `*Error`.
Read them with `klaracase.AsError`:

```go
if e, ok := klaracase.AsError(err); ok {
	fmt.Println(e.Status, e.Code, e.Detail, e.APIRelease)
	fmt.Println(e.Method, e.URL)         // the call that failed; the URL has no query string
	fmt.Println(e.RetryAfter())          // the wait the server asks for, or 0
	fmt.Println(e.Retryable())           // true for 429 and 503
	fmt.Println(string(e.Extra["unit"])) // every other member of the problem document
}
```

`Code` comes from the closed list of the API. The SDK has a constant for each, such as
`klaracase.CodeNotFound`. A code the SDK does not know is kept as it is. If the server sends
something that is not a problem document (an HTML 502 page, for example), the code is `unknown`
and `Detail` holds the first 1,000 characters of the body. `Candidates`, `Successor`, `Unit`,
`Cap` and `ResetsAt` hold the extra members of the codes that carry them, for example the list of
decisions behind an ambiguous Aktenzeichen.

`RetryAfter` reads `retry_after_seconds` from the body first. It then reads the `Retry-After`
header, as seconds or as an HTTP date.

An `*Error` is also JSON. `json.Marshal(err)` writes the code, status, title, detail, the wait and
the extra members. It never holds the token, the query string or the request body, so it is safe
to log or to give to a language model.

### The decode error

A success status with a body that does not fit the types returns a `*klaracase.DecodeError` and
no response. The call may already be charged, so the error keeps the whole body in `Raw`. This is
a choice: a typed client that hid a wrong body would give you wrong values, and a zero value that
looks like data is worse than an error you can handle. If you need to carry on anyway, parse
`Raw` yourself.

## Retries

- The client retries a `429` or `503` answer up to `2` times. Both mean that nothing ran and
  nothing was charged, so a retry is safe on every call, including `POST`.
- The wait is `retry_after_seconds` from the body, or the `Retry-After` header in whole seconds.
  Without either, the wait is a random time up to 0.5 s, then 1 s, then 2 s, doubling to a cap
  of 8 s.
- A wait of more than 30 seconds is not slept. The call returns a `*RateLimitError` at once and
  you decide. A daily volume cap is an example.
- The client never retries `402` or any other `4xx`. It does not retry `500` or `502` either.
- A network error is retried on `GET` only. A `POST` whose connection dropped after the request
  was sent is not retried: the request may have run. Repeating a paid call by hand is free in the
  same month, because one charge covers the same operation on the same subject.

Change the limit with `klaracase.WithMaxRetries(n)`, or for one call with `WithRequestMaxRetries(n)`. Zero turns retries off.

## Time limits

Every method takes a `context.Context`. Use it for the total time of a call, retries included:

```go
ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
defer cancel()
res, err := client.Decisions.Search(ctx, klaracase.SearchDecisionsParams{Query: "Untreue"})
```

`klaracase.WithTimeout(d)` limits one HTTP attempt (the default is 30 seconds). When the context
ends, the call returns a `*TransportError` or `*TimeoutError` that wraps `context.Canceled` or
`context.DeadlineExceeded`, and the client does not retry.

## Responses

Each method returns a struct with the documented keys of the answer, with nothing renamed or
reordered. Three things come with every response:

- `res.Credits` has `Metered`, `Cost` and `CreditsRemaining`. `Cost` and `CreditsRemaining` are
  pointers, because a superadmin call is not metered and both are `null`. Use
  `res.CreditsRemaining()` and `res.Cost()` instead: each returns `(int64, bool)`, and the bool is
  false when the value is absent.
- `res.Meta` has `Status`, `APIRelease` (the `X-Klaracase-Api-Release` header) and `Duration`.
- `res.Extra` holds every top level key the client does not model, as `json.RawMessage`. Nested rows
  have their own `Extra`. `res.Raw` is the whole body, unchanged. Marshal `Raw` when you need
  lossless JSON. Marshaling the typed struct writes the modeled keys only.

Values have types where the API has a fixed shape:

- Decision dates are `klaracase.Date` (`YYYY-MM-DD`, with a `Time()` method). Months in the
  coverage report are `klaracase.YearMonth`.
- Timestamps are `time.Time`. The server writes them with the offset `+00:00`, and they parse to
  the same instant. Marshaling writes `Z`. `Raw` keeps the original text.
- Enum-like values are typed strings with constants, such as `Verdict`, `Confidence`,
  `ResolutionStatus` and `DocumentType`. A value the SDK does not know is kept as it is.
- A cross type search returns `LawHit` rows in `Sections.Laws` and `DecisionHit` rows in
  `Sections.Decisions`. A statute row has no Aktenzeichen, court, date or document type.
- `FacetsResponse.Values` is a slice in the order of the server. `Values.Lookup("BGH")` finds one.
- `Finding.Excerpt(text)` returns the cited part of the text you sent. The offsets count code
  points, and `Excerpt` cuts by runes.

If a success body does not fit the types (a value of the wrong type, or a body that is not JSON),
the call returns a `*klaracase.DecodeError` and no response. The call may already be charged, so
the error keeps the whole body in `Raw`, and the first 1,000 characters in `Body`.

The API only adds keys within v1, so a new release does not break your code.

Some keys carry the note `fortgeltung_note`. It says that the API does not check whether a decision
is still good law. Show it to your users.

## Per-call options

Every method takes `...RequestOption` after its arguments. An option changes one call and leaves
the client as it was.

```go
res, err := client.Corpus.Coverage(ctx, klaracase.CoverageParams{Top: 5},
	klaracase.WithHeader("X-Trace-Id", "job-42"),       // not Authorization or Content-Type
	klaracase.WithRequestTimeout(5*time.Second),        // one attempt of this call
	klaracase.WithRequestMaxRetries(0),                 // no retries for this call
)
```

An invalid option returns a `*klaracase.ConfigurationError` before any request.

## Pagination

`Decisions.SearchAll` and `Decisions.CitationsAll` return iterators (`iter.Seq2`) that fetch page
after page:

```go
const maxPages = 3 // required: there is no way to ask for "all"
for hit, err := range client.Decisions.SearchAll(ctx, klaracase.SearchDecisionsParams{Query: "Untreue"}, maxPages) {
	if err != nil {
		return err
	}
	fmt.Println(hit.Aktenzeichen)
}
```

The iteration stops after `maxPages` pages, on an empty page, when the candidate pool is used up,
at the last page the API serves (10 for a search, 20 for a citation graph), after the first error,
or when you break out of the loop. `CitationsAll` yields the `references`, then the `referenced_by`
rows of each page, and only works at depth 1.

Credits: each page is a separate request and counts against the rate limit of 60 per minute. By
the contract, one charge covers one operation on one subject per organization and calendar month,
so later pages of the same query should cost nothing more. Look at `res.Credits` on a response if
you need to be sure. Keep `maxPages` low.

## Uploads and polling

See "Check a file" above. `WaitUntilReady` polls every 2 seconds by default. Change that with
`WithPollInterval`. The context is the only deadline. When it passes, you get a
`*DocumentTimeoutError` that holds the last receipt, and `errors.Is(err, context.DeadlineExceeded)`
is true. A document with the status `failed` returns a `*DocumentFailedError`. Pass
`WithPollRequestOptions(...)` to give every poll call its own request options.

## Client options

```go
client, err := klaracase.New(token,
	klaracase.WithBaseURL("https://klaracase.de/api/v1"), // the default
	klaracase.WithHTTPClient(myHTTPClient),               // proxy, transport, tracing
	klaracase.WithTimeout(20*time.Second),
	klaracase.WithMaxRetries(3),
	klaracase.WithUserAgentSuffix("my-app/1.2"),
	klaracase.WithLogger(slog.Default()),
	klaracase.WithDeprecationHandler(func(n klaracase.DeprecationNotice) {
		log.Printf("%s is deprecated, sunset %s", n.Endpoint, n.Sunset)
	}),
)
```

The user agent is `klaracase-go/0.1.0 go/<version>`, followed by your suffix.

When an endpoint is deprecated, the API announces it with `Deprecation` and `Sunset` headers. The
client calls your handler once per endpoint and client. Without a handler it writes one warning
with `log/slog`.

A `Client` is safe for use by many goroutines.

## Logging

The client is silent by default. Pass `WithLogger` to get debug lines. A retry writes one line
with the method, the path, the status, the attempt, the wait and the cause. The client never logs
the token, the query string, a request body or a response body. The deprecation warning goes to
this logger too (or to `slog.Default()` when you set none).

## Credits

Most calls cost credits. Search, `Decisions.Get` and `Norms.Get` cost 1 credit,
`Decisions.Citations` costs 2, `Datapackages.Create` costs 5, and a citation check costs as shown
above. Uploads, `CitationChecks.Get`, `Corpus.Coverage` and `Corpus.Facets` are free. One charge
covers one operation on one subject per organization and calendar month (UTC). The same call
again in that month costs 0. The API sends no `Idempotency-Key` header.

The rate limit is 60 requests per minute per token. The calls `Datapackages.Create`,
`CitationChecks.Create` and `CitationCheckDocuments.Upload` have an extra limit of 10 per minute.

## Agent tools

The subpackage `agenttools` describes five operations as tools for a language model and runs a
tool call for you. It has no dependency on an agent framework.

```go
import "github.com/klaracase/klaracase-go/agenttools"

tools := agenttools.Tools() // Name, Description and a JSON Schema for each tool
// ... give them to your model API; the model answers with a tool call ...
result, err := agenttools.Call(ctx, client, call.Name, call.Input)
// give result back to the model as the tool result
```

The tools are `search_decisions`, `cross_type_search`, `get_decision`, `get_norm` and
`check_citations`. The result is short JSON: `get_decision` returns at most 4,000 characters of
text unless the input sets `max_chars`, and `check_citations` returns the verdicts with their
offsets, not the whole run. An API error is not a Go error. It becomes the text
`{"error":{"code":...,"status":...,"detail":...}}`, so the model can read it and act. `Call`
returns a Go error only for an unknown tool name and for a canceled context.

To connect an MCP client instead, use `https://klaracase.de/mcp`. The same token works there.
This package does not implement MCP.

## Testing your code

Give the client a `*http.Client` with your own `RoundTripper`, or point it at an `httptest` server.
Neither needs a network or a token:

```go
type stub func(*http.Request) (*http.Response, error)

func (f stub) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

client, _ := klaracase.New("klc_your_token_here", klaracase.WithHTTPClient(&http.Client{
	Transport: stub(func(r *http.Request) (*http.Response, error) {
		body := `{"results":[],"page":1,"limit":10}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"application/json"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    r,
		}, nil
	}),
}))
```

For an `httptest` server, pass `klaracase.WithBaseURL(srv.URL + "/api/v1")`. The examples in
`example_test.go` do this and run with `go test`.

## Develop

```sh
make test   # go test -race -shuffle=on -cover, no network
make check  # gofmt, vet, go mod tidy, staticcheck, golangci-lint and the tests
make smoke  # calls the two free endpoints; needs KLARACASE_API_TOKEN; not part of CI
```

The tests replay the shared conformance cases in `testdata/conformance/` against a local
`httptest` server. Tests of the drift guards read the example answers in `testdata/examples/`.
See `CONTRIBUTING.md`.

## Requirements

Go 1.24 or newer. CI runs on 1.24, the previous release and the current release. The module has
no dependencies.

## Versioning

This SDK is 0.x and tracks API release 0.95.0. Until 1.0, a minor version (0.1 to 0.2) may change
the public surface, and a patch version (0.1.0 to 0.1.1) never does. Every change is listed in
[CHANGELOG.md](CHANGELOG.md). All Klaracase SDKs bump together when the API contract changes.

## Privacy

The SDK sends no telemetry. The only thing it adds to a request is the User-Agent header
`klaracase-go/<version> go/<version>`, for example `klaracase-go/0.1.0 go/1.24.2`.

Klaracase is not affiliated with any court or with the federal government.

## Links

- API reference: <https://klaracase.de/docs/api>
- MCP tools: <https://klaracase.de/docs/mcp>
- Support: support@klaracase.de
- License: MIT, see `LICENSE`

Klaracase covers German law only. It is a research tool. It gives no legal advice and does not
replace a lawyer's review.
