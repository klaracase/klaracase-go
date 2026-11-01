# Contributing

Thank you for your interest. This is a small, hand-written client. We keep it that way.

## Rules

- Use the Go standard library only. Do not add a dependency, in the library or in the tests.
- Do not generate code. Write it by hand.
- Request parameters are typed strictly. Response types keep every key the server sends.
- Every exported symbol has a doc comment.
- A new field of a params struct also goes into the schema in `agenttools/schemas.go`, when the tool uses that struct. A test fails if the two differ.
- Every change that alters behavior has a test. Tests use `net/http/httptest`. A test never uses the network.

## Before you open a pull request

```sh
make check # gofmt, go vet, go mod tidy, staticcheck, golangci-lint and the tests
```

Install `staticcheck` and `golangci-lint` v2 first (`go install honnef.co/go/tools/cmd/staticcheck@latest`
and see https://golangci-lint.run/welcome/install/). CI runs the same checks, plus `govulncheck`.

The shared conformance cases live in `testdata/conformance/`. They are the same cases the other
Klaracase SDKs run. Do not edit them here. If a case is wrong, tell the maintainers.

## Reporting a problem

Open an issue with the smallest example you can write. Never paste a real API token.
For a security problem, read `SECURITY.md` instead.
