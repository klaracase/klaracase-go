package agenttools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/klaracase/klaracase-go"
)

// DefaultMaxChars is the most characters of decision text that get_decision returns when the
// input sets no max_chars. The API default is 40000, which is too much for a model context.
const DefaultMaxChars = 4000

// Tool describes one tool for a language model.
type Tool struct {
	// Name is the tool name, for example "search_decisions".
	Name string `json:"name"`
	// Description tells the model what the tool does, what it costs and when to use it.
	Description string `json:"description"`
	// InputSchema is a JSON Schema for the input object.
	InputSchema json.RawMessage `json:"input_schema"`
}

// Tool names.
const (
	// ToolSearchDecisions is the name of the decision search tool.
	ToolSearchDecisions = "search_decisions"
	// ToolGetDecision is the name of the tool that reads one decision.
	ToolGetDecision = "get_decision"
	// ToolGetNorm is the name of the tool that reads one statute norm.
	ToolGetNorm = "get_norm"
	// ToolCheckCitations is the name of the citation check tool.
	ToolCheckCitations = "check_citations"
	// ToolCrossTypeSearch is the name of the search over statutes and decisions together.
	ToolCrossTypeSearch = "cross_type_search"
)

const (
	notFoundMeaning      = "not_found means the citation was not found in the corpus. It does not mean the citation is invented. Check it by hand."
	verifiedMeaning      = "verified_corpus means the citation exists in the corpus. It does not show that a quote or a legal claim is right."
	codeInvalidToolInput = "invalid_input"
)

// tools holds the tool descriptions. Tools returns copies.
var tools = []Tool{
	{
		Name: ToolSearchDecisions,
		Description: "Search German court decisions. Use German legal terms in the query. Returns the best matches " +
			"with file number, court, date and a short passage. Costs 1 credit. Read the full text of a match " +
			"with get_decision. Decisions may have been overturned later, so tell the user to check that.",
		InputSchema: json.RawMessage(searchDecisionsSchema),
	},
	{
		Name: ToolCrossTypeSearch,
		Description: "Search German statutes and court decisions together in one call. Returns up to the limit of " +
			"statute sections and of decisions. Costs 1 credit. Use it when you do not know yet whether the answer " +
			"is in a statute or in case law.",
		InputSchema: json.RawMessage(crossTypeSearchSchema),
	},
	{
		Name: ToolGetDecision,
		Description: "Read one German court decision by its id, file number (Aktenzeichen) or ECLI. Returns at most " +
			"4000 characters of text unless max_chars is set. Use section or rn_from and rn_to to read one part. " +
			"Costs 1 credit. If the file number belongs to more than one decision, the answer lists candidates; " +
			"call again with court or date.",
		InputSchema: json.RawMessage(getDecisionSchema),
	},
	{
		Name: ToolGetNorm,
		Description: "Read the current text of one German or EU statute norm, for example \"§ 535 BGB\" or " +
			"\"Art. 6 DSGVO\". Gives the text as it is in force now, not an older version. Costs 1 credit.",
		InputSchema: json.RawMessage(getNormSchema),
	},
	{
		Name: ToolCheckCitations,
		Description: "Check the citations of decisions and statutes in a German legal text against the Klaracase " +
			"corpus. Returns one verdict per citation with its position in your text. " + verifiedMeaning + " " +
			notFoundMeaning + " Costs 2 credits for up to 10000 characters, more for longer text.",
		InputSchema: json.RawMessage(checkCitationsSchema),
	},
}

// Tools returns the tool descriptions. Each call returns a new slice, so you can change it.
func Tools() []Tool {
	out := make([]Tool, len(tools))
	for i, t := range tools {
		t.InputSchema = bytes.Clone(t.InputSchema)
		out[i] = t
	}
	return out
}

// Call runs the tool named name with the JSON input that the model sent, and returns the JSON
// text to give back to the model.
//
// An error of the API, a refused input and a client side check all become a result of the form
// {"error":{"code":...,"detail":...}}, because the model can read them and try again. Call
// returns a Go error only for an unknown tool name and for a canceled or expired context.
func Call(ctx context.Context, c *klaracase.Client, name string, input json.RawMessage) (string, error) {
	run, ok := handlers[name]
	if !ok {
		return "", fmt.Errorf("agenttools: unknown tool %q", name)
	}
	result, err := run(ctx, c, input)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return errorResult(err)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("agenttools: encode result: %w", err)
	}
	return string(data), nil
}

type handler func(ctx context.Context, c *klaracase.Client, input json.RawMessage) (any, error)

var handlers = map[string]handler{
	ToolSearchDecisions: searchDecisions,
	ToolCrossTypeSearch: crossTypeSearch,
	ToolGetDecision:     getDecision,
	ToolGetNorm:         getNorm,
	ToolCheckCitations:  checkCitations,
}

// errorResult turns an error into the text {"error":{...}}. An error that is not from the API
// or the SDK is a bug in the caller and is returned as a Go error.
func errorResult(err error) (string, error) {
	var inputErr *inputError
	if errors.As(err, &inputErr) {
		data, mErr := json.Marshal(map[string]any{"error": map[string]any{
			"code": codeInvalidToolInput, "detail": inputErr.Error(),
		}})
		return string(data), mErr
	}
	base, ok := klaracase.AsError(err)
	if !ok {
		return "", err
	}
	data, mErr := json.Marshal(struct {
		Error *klaracase.Error `json:"error"`
	}{base})
	if mErr != nil {
		return "", mErr
	}
	return string(data), nil
}

// inputError means the JSON input of a tool call could not be read.
type inputError struct{ msg string }

func (e *inputError) Error() string { return e.msg }

// decodeInput reads the input into target. It refuses a key the tool does not know, so the
// model learns about a misspelled argument instead of getting a silently different search.
func decodeInput(input json.RawMessage, target any) error {
	if len(bytes.TrimSpace(input)) == 0 {
		input = json.RawMessage("{}")
	}
	dec := json.NewDecoder(bytes.NewReader(input))
	dec.DisallowUnknownFields()
	if err := dec.Decode(target); err != nil {
		return &inputError{msg: "the tool input is not valid: " + err.Error()}
	}
	if dec.More() {
		return &inputError{msg: "the tool input has more than one JSON value"}
	}
	return nil
}
