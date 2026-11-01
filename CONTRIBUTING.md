# Contributing

Thank you for your interest. This is a small, hand-written client. We keep it that way.

## Rules

- Use the Go standard library only. Do not add a dependency, in the library or in the tests.
- Do not generate code. Write it by hand.
- Request parameters are typed strictly. Response types keep every key the server sends.
- Every exported symbol has a doc comment.
- Every change that alters behavior has a test. Tests use `net/http/httptest`. A test never uses the network.

## Before you open a pull request

```sh
make fmt   # gofmt, must print nothing
make vet   # go vet
make test  # go test -race ./...
```

The shared conformance cases live in `testdata/conformance/`. They are the same cases the other
Klaracase SDKs run. Do not edit them here. If a case is wrong, tell the maintainers.

## Reporting a problem

Open an issue with the smallest example you can write. Never paste a real API token.
For a security problem, read `SECURITY.md` instead.
