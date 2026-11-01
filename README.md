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
		Limit: klaracase.Int(5),
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, hit := range res.Results {
		fmt.Println(hit.Court, hit.Aktenzeichen, hit.Date)
	}
	fmt.Println("credits left:", *res.Credits.CreditsRemaining)
}
```

`klaracase.New("klc_your_token_here")` takes the token as an argument instead. An empty token
returns a `*klaracase.ConfigurationError` at once, not a 401 later.

### Optional values

A string parameter that is empty is not sent. A number or a boolean is a pointer, so that `0` and
`false` can be sent and "not set" stays different. Use `klaracase.Int(n)` and `klaracase.Bool(b)`:

```go
klaracase.SearchDecisionsParams{Query: "Untreue", Bridge: klaracase.Bool(false), Limit: klaracase.Int(5)}
```

An empty slice is not sent. A list such as `CitedStatutes` is sent as `cited_statutes[]=StGB&cited_statutes[]=StPO`.

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
| `client.CitationCheckDocuments.WaitUntilReady(ctx, id, WaitParams{...})` | polls the call above | free |
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
		// The citation was not found in the corpus. This is not a claim that
		// it is fabricated. Check it by hand.
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
ready, err := client.CitationCheckDocuments.WaitUntilReady(ctx, doc.ID, klaracase.WaitParams{
	PollInterval: 2 * time.Second, // the default
	Timeout:      2 * time.Minute, // the default
})
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

Every failed call returns an error from this package. Use `errors.As` to pick the type you want:

```go
res, err := client.Decisions.Search(ctx, klaracase.SearchDecisionsParams{Query: "Untreue"})

var credits *klaracase.InsufficientCreditsError
var limited *klaracase.RateLimitError
var denied *klaracase.PermissionError
switch {
case err == nil:
	_ = res
case errors.As(err, &credits):
	fmt.Printf("this call costs %d credits, the balance is %d\n", credits.Cost, credits.Balance)
case errors.As(err, &limited):
	fmt.Printf("try again in %d seconds\n", limited.RetryAfterSeconds)
case errors.As(err, &denied):
	fmt.Println("the token lacks the ability:", denied.RequiredAbility)
default:
	log.Fatal(err)
}
```

| Type | When | Extra fields |
|---|---|---|
| `*AuthenticationError` | 401 | |
| `*InsufficientCreditsError` | 402 | `Cost`, `Balance` |
| `*PermissionError` | 403 | `RequiredAbility` |
| `*NotFoundError` | 404 | |
| `*ConflictError` | 409 | |
| `*ValidationError` | 422, or a client side check | `Errors` (field to messages) |
| `*RateLimitError` | 429 | `RetryAfterSeconds` |
| `*ServiceUnavailableError` | 503 | |
| `*APIError` | any other status, or a body that is not a problem document | |
| `*TransportError` | no HTTP answer; wraps the cause (`errors.Is(err, context.Canceled)` works) | `Err` |
| `*TimeoutError` | a timeout; also matches `*TransportError` | |
| `*DocumentFailedError` | `WaitUntilReady` saw the status `failed` | `ErrorCode`, `Document` |
| `*DocumentTimeoutError` | `WaitUntilReady` ran out of time | `Document` |
| `*ConfigurationError` | no token, a bad base URL, a bad option | `Message` |

All of them except `*ConfigurationError` carry the common problem fields. Read them with
`klaracase.AsError`:

```go
if e, ok := klaracase.AsError(err); ok {
	fmt.Println(e.Status, e.Code, e.Detail, e.APIRelease)
	fmt.Println(string(e.Extra["unit"])) // every other member of the problem document
}
```

`Code` comes from the closed list of the API, for example `not_found` or `insufficient_credits`.
If the server sends something that is not a problem document (an HTML 502 page, for example), the
code is `unknown` and `Detail` holds the first 1,000 characters of the body.

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

Change the limit with `klaracase.WithMaxRetries(n)`. Zero turns retries off.

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
  pointers, because a superadmin call is not metered and both are `null`.
- `res.Meta` has `Status`, `APIRelease` (the `X-Klaracase-Api-Release` header) and `Duration`.
- `res.Extra` holds every top level key the client does not model, as `json.RawMessage`. Nested rows
  have their own `Extra`. `res.Raw` is the whole body, unchanged.

The API only adds keys within v1, so a new release does not break your code.

Some keys carry the note `fortgeltung_note`. It says that the API does not check whether a decision
is still good law. Show it to your users.

## Other options

```go
client, err := klaracase.New(token,
	klaracase.WithBaseURL("https://klaracase.de/api/v1"), // the default
	klaracase.WithHTTPClient(myHTTPClient),               // proxy, transport, tracing
	klaracase.WithTimeout(20*time.Second),
	klaracase.WithMaxRetries(3),
	klaracase.WithUserAgentSuffix("my-app/1.2"),
	klaracase.WithDeprecationHandler(func(n klaracase.DeprecationNotice) {
		log.Printf("%s is deprecated, sunset %s", n.Endpoint, n.Sunset)
	}),
)
```

The user agent is `klaracase-go/0.1.0 go/<version>`, followed by your suffix.

When an endpoint is deprecated, the API announces it with `Deprecation` and `Sunset` headers. The
client calls your handler once per endpoint and process. Without a handler it writes one warning
with `log/slog`.

A `Client` is safe for use by many goroutines.

## Credits

Most calls cost credits. Search, `Decisions.Get` and `Norms.Get` cost 1 credit,
`Decisions.Citations` costs 2, `Datapackages.Create` costs 5, and a citation check costs as shown
above. Uploads, `CitationChecks.Get`, `Corpus.Coverage` and `Corpus.Facets` are free. One charge
covers one operation on one subject per organization and calendar month (UTC). The same call
again in that month costs 0. The API sends no `Idempotency-Key` header.

The rate limit is 60 requests per minute per token. The calls `Datapackages.Create`,
`CitationChecks.Create` and `CitationCheckDocuments.Upload` have an extra limit of 10 per minute.

## Agents and MCP

This package is a REST client. It does not implement MCP. If you build an agent, connect your MCP
client to `https://klaracase.de/mcp`. The same token works there.

## Test and develop

```sh
make test   # go test -race ./..., no network
make vet
make fmt
make smoke  # calls the two free endpoints; needs KLARACASE_API_TOKEN; not part of CI
```

The tests replay the shared conformance cases in `testdata/conformance/` against a local
`httptest` server. See `CONTRIBUTING.md`.

## Links

- API reference: <https://klaracase.de/docs/api>
- MCP tools: <https://klaracase.de/docs/mcp>
- Support: support@klaracase.de
- License: MIT, see `LICENSE`

Klaracase covers German law only. It is a research tool. It gives no legal advice and does not
replace a lawyer's review.
