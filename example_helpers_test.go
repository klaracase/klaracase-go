package klaracase_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/klaracase/klaracase-go"
)

// The examples in this package run against a local test server, so they need no token and
// no network. A reader of the documentation can ignore the helpers: replace the client with
// klaracase.NewFromEnv() to run the same code against the real API.

// exampleClient starts a test server that answers every request with the given status and
// the body. It returns a client for it and a function that stops the server.
func exampleClient(status int, body []byte, opts ...klaracase.Option) (*klaracase.Client, func()) {
	return exampleClientFunc(func(w http.ResponseWriter, _ *http.Request) {
		if status >= http.StatusBadRequest {
			w.Header().Set("Content-Type", "application/problem+json")
		} else {
			w.Header().Set("Content-Type", "application/json")
		}
		w.Header().Set("X-Klaracase-Api-Release", "0.95.0")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}, opts...)
}

// exampleClientFunc is exampleClient with your own handler.
func exampleClientFunc(h http.HandlerFunc, opts ...klaracase.Option) (*klaracase.Client, func()) {
	srv := httptest.NewServer(h)
	all := append([]klaracase.Option{klaracase.WithBaseURL(srv.URL + "/api/v1")}, opts...)
	client, err := klaracase.New("klc_your_token_here", all...)
	if err != nil {
		panic(err)
	}
	return client, srv.Close
}

// exampleBody reads one of the example answers of the API from testdata.
func exampleBody(name string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata", "examples", name))
	if err != nil {
		panic(err)
	}
	return data
}
