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
// A boolean is a *bool so that false can be sent: use [Bool] or [Ptr].
//
// # Responses
//
// Each method returns a struct with the documented keys of the answer. A key that the SDK
// does not know is kept in the Extra map of the struct that holds it. Raw holds the whole
// body unchanged. Meta holds the HTTP status, the API release and the duration. Credits
// holds the credit accounting of the call.
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
// For agent frameworks, connect an MCP client to https://klaracase.de/mcp instead. This
// package is a REST client and does not implement MCP. The API reference is at
// https://klaracase.de/docs/api.
package klaracase
