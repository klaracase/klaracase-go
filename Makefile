.PHONY: test vet fmt smoke

# Run all tests with the race detector. No test touches the network.
test:
	go test -race -count=1 ./...

# Static checks from the Go toolchain.
vet:
	go vet ./...

# Fail when a file is not gofmt-clean. Prints the offending files.
fmt:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "$$out"; echo "gofmt: run 'gofmt -w .'"; exit 1; fi

# Call the two free endpoints with a real token. Needs KLARACASE_API_TOKEN. Not part of CI.
smoke:
	go run ./internal/smoke
