// Package klaracase is the official Go client for the Klaracase REST API v1.
//
// Klaracase searches German court decisions and statutes, follows citations between
// decisions, and checks the citations in a legal text. Klaracase covers German law only.
// It is a research tool. It gives no legal advice and does not replace a lawyer's review.
// A "not_found" verdict means the citation was not found in the corpus; it is not a claim
// that the citation is fabricated.
//
// # Quick start
//
// Create an API token at https://klaracase.de/settings/api-tokens and put it in the
// KLARACASE_API_TOKEN environment variable. Then:
//
//	client, err := klaracase.NewFromEnv()
//	if err != nil {
//		log.Fatal(err)
//	}
//	res, err := client.Decisions.Search(ctx, klaracase.SearchDecisionsParams{
//		Query: "Vermögensbetreuungspflicht Untreue",
//		Court: "BGH",
//		Limit: 5,
//	})
//	if err != nil {
//		log.Fatal(err)
//	}
//	for _, hit := range res.Results {
//		fmt.Println(hit.Court, hit.Aktenzeichen, hit.Date)
//	}
//
// # Optional values
//
// A string, a number and a slice that are unset are not sent. A zero number means "not set".
// A boolean is a *bool so that false can be sent: use [Bool] or [Ptr]. Dates are [Date]
// values (YYYY-MM-DD). The client checks only what needs no server, such as a missing required
// value or a date range that runs backwards, and returns a [ValidationError] with the code
// "client_validation" before it sends anything.
//
// # Responses
//
// Each method returns a struct with the documented keys of the answer. A key that the SDK
// does not know is kept in the Extra map of the struct that holds it. Raw holds the whole
// body unchanged: marshal Raw when you need lossless JSON, because marshaling the typed struct
// writes the modeled keys only. Meta holds the HTTP status, the API release and the duration.
// Credits holds the credit accounting of the call; use CreditsRemaining and Cost to read it.
// A success body that does not fit the types returns a [DecodeError] that keeps the body.
//
// # Per-call options
//
// Every method takes [RequestOption] values after its arguments: [WithHeader],
// [WithRequestTimeout] and [WithRequestMaxRetries] change one call and leave the client alone.
//
// # Pagination
//
// [DecisionsService.SearchAll] and [DecisionsService.CitationsAll] return iterators over all
// rows of a search or of a citation graph. They need a maxPages limit, because each page is a
// separate request.
//
// # Logging
//
// The client is silent unless you pass [WithLogger]. It then writes one debug line per retry
// and never writes the token, the query string or a body.
//
// # Agent tools
//
// The subpackage github.com/klaracase/klaracase-go/agenttools describes five operations as
// tools for a language model and runs a tool call against a [Client].
//
// # Errors
//
// Every failed call returns an error of one of the types in this package. Use errors.As to
// pick the one you want, or [AsError] to read the common problem fields:
//
//	var credits *klaracase.InsufficientCreditsError
//	if errors.As(err, &credits) {
//		fmt.Println("this call costs", credits.Cost, "credits, balance", credits.Balance)
//	}
//
// # Retries and time limits
//
// The client retries a 429 or 503 answer up to two times, on every method, because both
// guarantee that nothing ran and nothing was charged. It never retries a 402 or another 4xx
// answer. A transport error is retried on GET only. A wait that the server states above 30
// seconds is not slept: the call returns a [RateLimitError] and you decide. Every method takes
// a context.Context; use it for the total time limit. [WithTimeout] limits one attempt.
//
// To use an MCP client instead, connect it to https://klaracase.de/mcp. This package is a REST
// client and does not implement MCP. The API reference is at
// https://klaracase.de/docs/api.
package klaracase
