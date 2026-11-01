// Package agenttools describes the Klaracase operations as tools for a language model, and runs
// a tool call against a [klaracase.Client].
//
// It has no dependency on an agent framework. [Tools] returns the name, the description and a
// JSON Schema for each tool, in the shape that tool use APIs expect. [Call] takes the name and
// the JSON input of a tool call that the model made, calls the API, and returns a short JSON
// text to give back to the model.
//
//	tools := agenttools.Tools() // hand these to your model API
//	// ... the model answers with a tool call: name and input (JSON)
//	result, err := agenttools.Call(ctx, client, name, input)
//	if err != nil {
//		return err // a bug or a canceled context, not an answer for the model
//	}
//	// give result back to the model as the tool result
//
// The tools are search_decisions, cross_type_search, get_decision, get_norm and
// check_citations. Each call can cost credits: search, get_decision and get_norm cost 1 credit,
// check_citations costs 2 or more.
//
// # Errors
//
// An error of the API is not a Go error here. It becomes the text {"error":{...}} with the
// fields code, status, detail and, when the API states a wait, retry_after_seconds. The model can
// read it and act, for example by waiting or by narrowing a search. [Call] returns a Go error
// only for a tool name that does not exist and for a canceled context.
//
// The codes insufficient_credits and token_limit_reached are never retried. Their error text
// also holds next_step, which tells the model to stop and report to its user. The members
// top_up_url and settings_url, when present, are for a human organization admin: a model must
// show them to its user and never open them.
//
// # Results are short
//
// The result holds the fields a model needs, not the whole API answer. get_decision returns at
// most 4000 characters of text unless the input sets max_chars. check_citations returns the
// verdict of each citation, with its text and its offsets in the text that was sent.
//
// Klaracase covers German law only. It is a research tool. It gives no legal advice and does
// not replace a lawyer's review. A "not_found" verdict means the citation was not found in the
// corpus; it is not a claim that the citation is fabricated.
package agenttools
