package agenttools_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/klaracase/klaracase-go"
	"github.com/klaracase/klaracase-go/agenttools"
)

func ExampleTools() {
	for _, tool := range agenttools.Tools() {
		fmt.Println(tool.Name)
	}
	// Hand the tools to your model API. Each has a Name, a Description and an InputSchema.
	// Output:
	// search_decisions
	// cross_type_search
	// get_decision
	// get_norm
	// check_citations
}

func ExampleCall() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"status":404,"title":"Not Found","code":"not_found","detail":"No decision has this file number."}`)
	}))
	defer srv.Close()
	client, err := klaracase.New("klc_your_token_here", klaracase.WithBaseURL(srv.URL+"/api/v1"))
	if err != nil {
		log.Fatal(err)
	}

	// name and input come from the tool call that your model made.
	name, input := "get_decision", json.RawMessage(`{"id":"9 Z 9/99"}`)
	result, err := agenttools.Call(context.Background(), client, name, input)
	if err != nil {
		log.Fatal(err) // an unknown tool or a canceled context
	}
	// An error of the API is a result for the model, not a Go error.
	fmt.Println(result)
	// Output:
	// {"error":{"code":"not_found","status":404,"title":"Not Found","detail":"No decision has this file number."}}
}
