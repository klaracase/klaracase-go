# Changelog

All notable changes to this project are written here.
The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).
This project follows [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- First version of the Go client for the Klaracase REST API v1 (version 0.1.0).
- Request parameters have `json` tags and the `omitzero` option. Integer parameters are plain
  `int` (zero means "not set"). Booleans are `*bool`, with the helpers `Ptr`, `Bool`, `Int` and
  `String`.
- `Date` and `YearMonth` types, and typed enums: `Section`, `Jurisdiction`, `ArtifactKind`,
  `Statute`, `ArtifactType`, `Confidence`, `ResolutionStatus`, `MatchStrength`, `ReferenceType`
  and more.
- Client side validation that sends nothing on failure: missing required values, malformed dates,
  backwards date and Randnummer ranges, wrong datapackage size. It returns a `*ValidationError`
  with the code `client_validation`.
- Per-call options: `WithHeader`, `WithRequestTimeout` and `WithRequestMaxRetries`. Every
  service method takes `...RequestOption`.
- Typed response model: enums, `Date`, `time.Time` timestamps, `LawHit` and `DecisionHit`,
  ordered `FacetValues` with `Lookup`, `Finding.Excerpt`, and the accessors
  `CreditsRemaining()` and `Cost()` on every response.
- `*DecodeError` for a success status with a body that does not fit the types. It keeps the body.
- Tests that fail when a response type loses or leaves out a key of the example answers.
- Errors: `Method`, `URL`, `Header`, `Candidates`, `Successor`, `Unit`, `Cap` and `ResetsAt` on
  `*Error`; `RetryAfter()` (also from an HTTP date) and `Retryable()`; `MarshalJSON`; the helpers
  `IsNotFound`, `IsRateLimited`, `IsInsufficientCredits`, `IsValidation` and `IsPermission`; a
  `Code*` constant for each code of the API.
- `WaitUntilReady` with `WithPollInterval` and `WithPollRequestOptions`. The context is the only
  deadline; its end returns `*DocumentTimeoutError` (code `wait_timeout`).
- `WithLogger`: debug lines for retries and the deprecation warning, without token, query or body.
- `Decisions.SearchAll` and `Decisions.CitationsAll`: iterators over pages (`iter.Seq2`) with a
  required page limit.
- `agenttools` subpackage: `Tools()` and `Call()` for five tools for a language model.
- Runnable examples with an `httptest` server and `// Output:` lines.
- CI: Go 1.24, oldstable and stable; staticcheck, govulncheck, golangci-lint v2, `go mod tidy`
  check, shuffled race tests with coverage; actions pinned by SHA; Dependabot for actions.

### Docs

- Docs: credit period wording. Credits renew per credit period: the UTC calendar month for free organizations, the paid billing period for subscribers. Unused credits do not roll over. The docs said "calendar month" for every organization.

### Changed

- Before 0.1.0 is published: `WaitParams` is gone (use `WithPollInterval`), `SearchHit` is split
  into `LawHit` and `DecisionHit`, `*ConfigurationError` now holds an `*Error` so `AsError`
  finds it, and the deprecation handler runs once per endpoint and client, not once per process.
- `Decisions.Get` and the other methods return the typed enums and `Date` where they returned
  plain strings.

### Notes

- Pages of a paginated search count against the rate limit. By the contract, they should cost no
  extra credits in the same credit period. Check `Credits` on a response before you rely on it.
