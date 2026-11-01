.PHONY: test vet fmt lint staticcheck tidy check smoke

# Run all tests with the race detector, in random order, with coverage. No test touches the network.
test:
	go test -race -shuffle=on -cover -count=1 ./...

# Static checks from the Go toolchain.
vet:
	go vet ./...

# Fail when a file is not gofmt-clean. Prints the offending files.
fmt:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "$$out"; echo "gofmt: run 'gofmt -w .'"; exit 1; fi

# staticcheck and golangci-lint need to be installed (see CONTRIBUTING.md).
staticcheck:
	staticcheck ./...

lint:
	golangci-lint run

# Fail when go.mod is not tidy.
tidy:
	go mod tidy -diff

# Everything CI runs on the stable Go version, except govulncheck.
check: fmt vet tidy staticcheck lint test

# Call the two free endpoints with a real token. Needs KLARACASE_API_TOKEN. Not part of CI.
smoke:
	go run ./internal/smoke
