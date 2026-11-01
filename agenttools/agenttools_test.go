package agenttools

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/klaracase/klaracase-go"
)

// jsonTags returns the json names of a struct, with the fields of embedded structs flattened.
func jsonTags(t reflect.Type) []string {
	var names []string
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			names = append(names, jsonTags(f.Type)...)
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

type schemaDoc struct {
	Type       string                     `json:"type"`
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required"`
}

func properties(t *testing.T, tool Tool) (schemaDoc, []string) {
	t.Helper()
	if !json.Valid(tool.InputSchema) {
		t.Fatalf("%s: the schema is not valid JSON", tool.Name)
	}
	var doc schemaDoc
	if err := json.Unmarshal(tool.InputSchema, &doc); err != nil {
		t.Fatal(err)
	}
	var names []string
	for name := range doc.Properties {
		names = append(names, name)
	}
	sort.Strings(names)
	return doc, names
}

func TestEverySchemaMatchesItsParamsStruct(t *testing.T) {
	type getDecision = getDecisionInput
	structs := map[string]reflect.Type{
		ToolSearchDecisions: reflect.TypeFor[klaracase.SearchDecisionsParams](),
		ToolCrossTypeSearch: reflect.TypeFor[klaracase.CrossTypeSearchParams](),
		ToolGetDecision:     reflect.TypeFor[getDecision](),
		ToolGetNorm: reflect.TypeFor[struct {
			Reference string `json:"reference"`
			klaracase.GetNormParams
		}](),
		ToolCheckCitations: reflect.TypeFor[klaracase.CreateCitationCheckParams](),
	}
	all := Tools()
	if len(all) != len(structs) {
		t.Fatalf("%d tools, %d structs", len(all), len(structs))
	}
	for _, tool := range all {
		typ, ok := structs[tool.Name]
		if !ok {
			t.Fatalf("no params struct for the tool %q", tool.Name)
		}
		doc, names := properties(t, tool)
		if want := jsonTags(typ); !reflect.DeepEqual(names, want) {
			t.Errorf("%s: schema properties %v, json tags of %s %v", tool.Name, names, typ, want)
		}
		if doc.Type != "object" {
			t.Errorf("%s: the schema type is %q", tool.Name, doc.Type)
		}
		for _, r := range doc.Required {
			if _, ok := doc.Properties[r]; !ok {
				t.Errorf("%s: %q is required but not a property", tool.Name, r)
			}
		}
	}
}

func TestSchemaEnumsMatchTheConstants(t *testing.T) {
	enumOf := func(tool, property string) []string {
		for _, tl := range Tools() {
			if tl.Name != tool {
				continue
			}
			doc, _ := properties(t, tl)
			var prop struct {
				Enum  []string `json:"enum"`
				Items struct {
					Enum []string `json:"enum"`
				} `json:"items"`
			}
			if err := json.Unmarshal(doc.Properties[property], &prop); err != nil {
				t.Fatal(err)
			}
			if len(prop.Items.Enum) > 0 {
				return prop.Items.Enum
			}
			return prop.Enum
		}
		t.Fatalf("no tool %s", tool)
		return nil
	}
	branches := []klaracase.Branch{
		klaracase.BranchOrdentlich, klaracase.BranchVerwaltung, klaracase.BranchSozial, klaracase.BranchArbeit,
		klaracase.BranchFinanz, klaracase.BranchVerfassung, klaracase.BranchPatent, klaracase.BranchUnbekannt,
	}
	types := []klaracase.DocumentType{
		klaracase.DocumentTypeUrteil, klaracase.DocumentTypeBeschluss, klaracase.DocumentTypeVerfuegung,
		klaracase.DocumentTypeSonstige, klaracase.DocumentTypeGerichtsbescheid, klaracase.DocumentTypeSchlussantraege,
	}
	modes := []klaracase.Mode{klaracase.ModeKeyword, klaracase.ModeSemantic, klaracase.ModeHybrid}
	statutes := []klaracase.Statute{
		klaracase.StatuteStGB, klaracase.StatuteStPO, klaracase.StatuteInsO, klaracase.StatuteDSGVO, klaracase.StatuteGDPR,
		klaracase.StatuteBDSG, klaracase.StatuteBauGB, klaracase.StatuteBBauG, klaracase.StatuteBauNVO, klaracase.StatuteStVG,
		klaracase.StatuteStVO,
	}
	sections := []klaracase.Section{klaracase.SectionTenor, klaracase.SectionLeitsatz, klaracase.SectionTatbestand, klaracase.SectionGruende}
	str := func(v any) []string {
		rv := reflect.ValueOf(v)
		out := make([]string, rv.Len())
		for i := range out {
			out[i] = rv.Index(i).String()
		}
		return out
	}
	checks := []struct {
		tool, property string
		want           []string
	}{
		{ToolSearchDecisions, "branch", str(branches)}, {ToolCrossTypeSearch, "branch", str(branches)},
		{ToolSearchDecisions, "document_type", str(types)}, {ToolCrossTypeSearch, "document_type", str(types)},
		{ToolSearchDecisions, "mode", str(modes)}, {ToolCrossTypeSearch, "mode", str(modes)},
		{ToolSearchDecisions, "cited_statutes", str(statutes)}, {ToolCrossTypeSearch, "cited_statutes", str(statutes)},
		{ToolGetDecision, "section", str(sections)},
		{ToolCrossTypeSearch, "jurisdiction", []string{"DE", "EU", "XI"}},
	}
	for _, c := range checks {
		if got := enumOf(c.tool, c.property); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s.%s enum %v, want %v", c.tool, c.property, got, c.want)
		}
	}
}

func TestToolsReturnsCopies(t *testing.T) {
	first := Tools()
	first[0].Name = "changed"
	first[0].InputSchema[0] = 'X'
	second := Tools()
	if second[0].Name == "changed" || second[0].InputSchema[0] == 'X' {
		t.Error("Tools must return copies")
	}
	seen := map[string]bool{}
	for _, tool := range second {
		if seen[tool.Name] || tool.Description == "" {
			t.Errorf("duplicate or empty tool %+v", tool.Name)
		}
		seen[tool.Name] = true
	}
}

func TestDescriptionsAreFreeOfBannedWordingAndDashes(t *testing.T) {
	banned := []string{"hallucinat", "halluzin", "fabricat", "fabrizier", "official source", "amtlich", "—", "–"}
	for _, tool := range Tools() {
		text := strings.ToLower(tool.Description + string(tool.InputSchema))
		for _, word := range banned {
			if strings.Contains(text, word) {
				t.Errorf("%s: contains %q", tool.Name, word)
			}
		}
	}
	// The source files obey the same copy rules as the rest of the module.
	files, _ := filepath.Glob("*.go")
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		data, _ := os.ReadFile(file)
		lower := strings.ToLower(string(data))
		for _, word := range []string{"hallucinat", "halluzin", "fabricat", "fabrizier", "official source", "—"} {
			// The doc.go disclaimer carries the one allowed sentence.
			cleaned := strings.ReplaceAll(strings.Join(strings.Fields(strings.ReplaceAll(lower, "//", "")), " "), "it is not a claim that the citation is fabricated.", "")
			if strings.Contains(cleaned, word) {
				t.Errorf("%s: contains %q", file, word)
			}
		}
	}
}

// ---- Call ----

type seenRequest struct {
	path  string
	query map[string][]string
}

func serve(t *testing.T, status int, body string, seen *seenRequest) *klaracase.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if seen != nil {
			seen.path = r.URL.EscapedPath()
			seen.query = r.URL.Query()
		}
		if status >= 400 {
			w.Header().Set("Content-Type", "application/problem+json")
		}
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	c, err := klaracase.New("klc_test_token", klaracase.WithBaseURL(srv.URL+"/api/v1"), klaracase.WithMaxRetries(0))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func fixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "testdata", "examples", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func decodeResult(t *testing.T, result string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(result), &out); err != nil {
		t.Fatalf("the result is not JSON: %v\n%s", err, result)
	}
	return out
}

func TestSearchDecisionsIsSlim(t *testing.T) {
	var seen seenRequest
	c := serve(t, 200, fixture(t, "decisions.search.json"), &seen)
	result, err := Call(context.Background(), c, ToolSearchDecisions,
		json.RawMessage(`{"query":"Untreue","court":"BGH","limit":5,"bridge":false,"cited_statutes":["StGB"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if seen.path != "/api/v1/decisions/search" || seen.query["limit"][0] != "5" || seen.query["bridge"][0] != "false" ||
		seen.query["cited_statutes[]"][0] != "StGB" {
		t.Errorf("request %+v", seen)
	}
	got := decodeResult(t, result)
	hit := got["results"].([]any)[0].(map[string]any)
	if hit["aktenzeichen"] != "1 StR 185/01" || hit["rn_range"] != "21-23" || hit["court"] != "BGH" {
		t.Errorf("hit %v", hit)
	}
	if _, has := hit["snippets"]; has {
		t.Error("the list of snippets is not part of the slim result")
	}
	if got["fortgeltung_note"] == "" || got["fortgeltung_note"] == nil {
		t.Error("the note about good law must be in the result")
	}
	if credits := got["credits"].(map[string]any); credits["credits_remaining"] != float64(29) || credits["cost"] != float64(1) {
		t.Errorf("credits %v", credits)
	}
	if len(result) > len(fixture(t, "decisions.search.json")) {
		t.Errorf("the slim result (%d bytes) is larger than the API answer", len(result))
	}
}

func TestGetDecisionDefaultsMaxCharsTo4000(t *testing.T) {
	var seen seenRequest
	c := serve(t, 200, fixture(t, "decisions.show.json"), &seen)
	result, err := Call(context.Background(), c, ToolGetDecision, json.RawMessage(`{"id":"VIII ZR 271/17","section":"Gründe"}`))
	if err != nil {
		t.Fatal(err)
	}
	if seen.path != "/api/v1/decisions/VIII%20ZR%20271%2F17" || seen.query["max_chars"][0] != "4000" || seen.query["section"][0] != "Gründe" {
		t.Errorf("request %+v", seen)
	}
	got := decodeResult(t, result)
	if got["aktenzeichen"] != "1 StR 185/01" || got["document_type"] != "Urteil" || got["text"] == "" || got["ecli"] == "" {
		t.Errorf("result %v", got)
	}
	if _, err := Call(context.Background(), c, ToolGetDecision, json.RawMessage(`{"id":"x","max_chars":900}`)); err != nil {
		t.Fatal(err)
	}
	if seen.query["max_chars"][0] != "900" {
		t.Errorf("an explicit max_chars must win, got %v", seen.query["max_chars"])
	}
}

func TestGetNormAndCrossTypeSearch(t *testing.T) {
	var seen seenRequest
	c := serve(t, 200, fixture(t, "norms.show.json"), &seen)
	result, err := Call(context.Background(), c, ToolGetNorm, json.RawMessage(`{"reference":"§ 535 BGB"}`))
	if err != nil {
		t.Fatal(err)
	}
	got := decodeResult(t, result)
	if seen.path != "/api/v1/norms/%C2%A7%20535%20BGB" || got["number"] != "§ 535" || got["law"] != "Bürgerliches Gesetzbuch" ||
		got["previous"] != "§ 534 BGB" || got["last_amendment_date"] != "2024-12-20" {
		t.Errorf("path %s result %v", seen.path, got)
	}

	c = serve(t, 200, fixture(t, "search.json"), &seen)
	result, err = Call(context.Background(), c, ToolCrossTypeSearch, json.RawMessage(`{"query":"Untreue","jurisdiction":"DE"}`))
	if err != nil {
		t.Fatal(err)
	}
	got = decodeResult(t, result)
	laws := got["laws"].([]any)
	if seen.path != "/api/v1/search" || seen.query["jurisdiction"][0] != "DE" || len(laws) != 1 ||
		laws[0].(map[string]any)["abbreviation"] != "StGB" || len(got["decisions"].([]any)) != 1 {
		t.Errorf("path %s result %v", seen.path, got)
	}
}

func TestCheckCitationsIsReducedToTheVerdicts(t *testing.T) {
	var seen seenRequest
	c := serve(t, 200, fixture(t, "citation-checks.store.json"), &seen)
	text := strings.Repeat("Der BGH, Urt. v. 15.11.2001 - 1 StR 185/01 hat entschieden. ", 2)
	input, _ := json.Marshal(map[string]string{"text": text, "reference_date": "2026-10-01"})
	result, err := Call(context.Background(), c, ToolCheckCitations, input)
	if err != nil {
		t.Fatal(err)
	}
	got := decodeResult(t, result)
	keys := make([]string, 0, len(got))
	for k := range got {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if want := []string{"credits", "findings", "id", "note", "status", "unique_citations", "verdict_counts"}; !reflect.DeepEqual(keys, want) {
		t.Errorf("keys %v, want %v", keys, want)
	}
	finding := got["findings"].([]any)[0].(map[string]any)
	fkeys := make([]string, 0)
	for k := range finding {
		fkeys = append(fkeys, k)
	}
	sort.Strings(fkeys)
	if want := []string{"canonical", "end_offset", "id", "raw_text", "start_offset", "verdict"}; !reflect.DeepEqual(fkeys, want) {
		t.Errorf("finding keys %v, want %v", fkeys, want)
	}
	if finding["verdict"] != "verified_corpus" || finding["start_offset"] != float64(118) || finding["end_offset"] != float64(157) {
		t.Errorf("finding %v", finding)
	}
	if counts := got["verdict_counts"].(map[string]any); counts["not_found"] != float64(1) {
		t.Errorf("counts %v", counts)
	}
	if !strings.Contains(got["note"].(string), "does not mean the citation is invented") {
		t.Errorf("note %v", got["note"])
	}
	if seen.path != "/api/v1/citation-checks" {
		t.Errorf("path %s", seen.path)
	}
}

func errorOf(t *testing.T, result string) map[string]any {
	t.Helper()
	got := decodeResult(t, result)
	body, ok := got["error"].(map[string]any)
	if !ok || len(got) != 1 {
		t.Fatalf("want exactly {\"error\":{...}}, got %s", result)
	}
	return body
}

func TestApiErrorsBecomeErrorText(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
		input  string
		check  func(t *testing.T, e map[string]any)
	}{
		{"not found", 404, `{"type":"x","title":"Not Found","status":404,"detail":"No such decision.","code":"not_found"}`,
			`{"id":"9 Z 9/99"}`, func(t *testing.T, e map[string]any) {
				if e["code"] != "not_found" || e["status"] != float64(404) || e["detail"] != "No such decision." {
					t.Errorf("error %v", e)
				}
			}},
		{"rate limited", 429, `{"status":429,"code":"rate_limited","detail":"Slow down.","retry_after_seconds":42}`,
			`{"id":"x"}`, func(t *testing.T, e map[string]any) {
				if e["retry_after_seconds"] != float64(42) {
					t.Errorf("error %v", e)
				}
			}},
		{"ambiguous", 422, string(mustRead(t, "../testdata/conformance/problems/ambiguous_aktenzeichen.json")),
			`{"id":"5 U 123/19"}`, func(t *testing.T, e map[string]any) {
				if cands, _ := e["candidates"].([]any); len(cands) != 2 {
					t.Errorf("the model needs the candidates: %v", e)
				}
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := serve(t, tc.status, tc.body, nil)
			result, err := Call(context.Background(), c, ToolGetDecision, json.RawMessage(tc.input))
			if err != nil {
				t.Fatalf("an API error is a result, not a Go error: %v", err)
			}
			tc.check(t, errorOf(t, result))
		})
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRefusedInputsBecomeErrorText(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true }))
	defer srv.Close()
	c, _ := klaracase.New("klc_test_token", klaracase.WithBaseURL(srv.URL+"/api/v1"))
	cases := map[string]struct{ tool, input, code string }{
		"unknown field":        {ToolSearchDecisions, `{"query":"x","colour":"red"}`, "invalid_input"},
		"wrong type":           {ToolSearchDecisions, `{"query":"x","limit":"many"}`, "invalid_input"},
		"not an object":        {ToolSearchDecisions, `[1]`, "invalid_input"},
		"two values":           {ToolSearchDecisions, `{"query":"x"}{"query":"y"}`, "invalid_input"},
		"missing query":        {ToolSearchDecisions, `{}`, "client_validation"},
		"empty input":          {ToolSearchDecisions, ``, "client_validation"},
		"bad date":             {ToolSearchDecisions, `{"query":"x","date_from":"01.02.2020"}`, "client_validation"},
		"inverted rn range":    {ToolGetDecision, `{"id":"x","rn_from":5,"rn_to":2}`, "client_validation"},
		"check without a text": {ToolCheckCitations, `{}`, "client_validation"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result, err := Call(context.Background(), c, tc.tool, json.RawMessage(tc.input))
			if err != nil {
				t.Fatalf("got a Go error: %v", err)
			}
			if e := errorOf(t, result); e["code"] != tc.code || e["detail"] == "" {
				t.Errorf("error %v, want code %s", e, tc.code)
			}
		})
	}
	if hit {
		t.Error("a refused input must not reach the server")
	}
}

func TestUnknownToolAndCanceledContextAreGoErrors(t *testing.T) {
	c := serve(t, 200, "{}", nil)
	if _, err := Call(context.Background(), c, "no_such_tool", nil); err == nil || !strings.Contains(err.Error(), "no_such_tool") {
		t.Errorf("got %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Call(ctx, c, ToolSearchDecisions, json.RawMessage(`{"query":"Untreue"}`)); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v, want context.Canceled", err)
	}
}

func TestNoResultHoldsTheToken(t *testing.T) {
	c := serve(t, 401, `{"status":401,"code":"unauthenticated","detail":"The token is not valid."}`, nil)
	for _, tool := range Tools() {
		result, err := Call(context.Background(), c, tool.Name, json.RawMessage(`{"query":"x","id":"x","reference":"x","text":"x"}`))
		if err != nil {
			continue // some tools refuse the extra keys
		}
		if strings.Contains(result, "klc_test_token") || !utf8.ValidString(result) {
			t.Errorf("%s: %s", tool.Name, result)
		}
	}
}

func TestSlimCreditsCarryTheAPIBudgetOnlyWhenTheAnswerHasIt(t *testing.T) {
	body := string(mustRead(t, "../testdata/conformance/bodies/decisions.search.api_budget.json"))
	result, err := Call(context.Background(), serve(t, 200, body, nil), ToolSearchDecisions, json.RawMessage(`{"query":"Untreue"}`))
	if err != nil {
		t.Fatal(err)
	}
	credits := decodeResult(t, result)["credits"].(map[string]any)
	budget, _ := credits["api_budget"].(map[string]any)
	if budget["cents"] != float64(4991) || budget["rate_cents_per_credit"] != float64(9) || budget["credits_at_rate"] != float64(554) ||
		credits["credits_remaining"] != float64(0) {
		t.Errorf("credits %v", credits)
	}
	plain, err := Call(context.Background(), serve(t, 200, fixture(t, "decisions.search.json"), nil), ToolSearchDecisions, json.RawMessage(`{"query":"Untreue"}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, has := decodeResult(t, plain)["credits"].(map[string]any)["api_budget"]; has {
		t.Error("no api_budget key in the answer, none in the slim result")
	}
}

func TestPaymentRefusalsTellTheModelToStopAndAreNeverRetried(t *testing.T) {
	cases := map[string]struct {
		problem string
		keep    map[string]any
	}{
		"insufficient_credits": {"insufficient_credits.api_budget_empty.json", map[string]any{
			"reason": "api_budget_empty", "top_up_url": "https://klaracase.de/org/billing/api-budget", "cost": float64(1), "balance": float64(0)}},
		"token_limit_reached": {"token_limit_reached.json", map[string]any{
			"spent": float64(500), "cap": float64(500), "unit": "credits", "settings_url": "https://klaracase.de/org/api-keys"}},
	}
	for code, tc := range cases {
		t.Run(code, func(t *testing.T) {
			body := string(mustRead(t, "../testdata/conformance/problems/"+tc.problem))
			var hits int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hits++
				w.Header().Set("Content-Type", "application/problem+json")
				w.WriteHeader(402)
				_, _ = io.WriteString(w, body)
			}))
			defer srv.Close()
			c, err := klaracase.New("klc_test_token", klaracase.WithBaseURL(srv.URL+"/api/v1"), klaracase.WithMaxRetries(3))
			if err != nil {
				t.Fatal(err)
			}
			result, err := Call(context.Background(), c, ToolSearchDecisions, json.RawMessage(`{"query":"Untreue"}`))
			if err != nil {
				t.Fatalf("a refusal is a result, not a Go error: %v", err)
			}
			e := errorOf(t, result)
			if e["code"] != code || e["status"] != float64(402) {
				t.Errorf("error %v", e)
			}
			for k, v := range tc.keep {
				if e[k] != v {
					t.Errorf("%s = %v, want %v", k, e[k], v)
				}
			}
			step, _ := e["next_step"].(string)
			if !strings.Contains(step, "Stop") || !strings.Contains(step, "Do not open any link") {
				t.Errorf("next_step %q", step)
			}
			if _, has := e["retry_after_seconds"]; has || hits != 1 {
				t.Errorf("retry_after_seconds in %v, %d requests", e, hits)
			}
		})
	}
	// Other errors carry no next_step.
	notFound := errorOf(t, mustCall(t, serve(t, 404, `{"status":404,"code":"not_found","detail":"No."}`, nil)))
	if _, has := notFound["next_step"]; has {
		t.Errorf("not_found must not carry next_step: %v", notFound)
	}
}

func mustCall(t *testing.T, c *klaracase.Client) string {
	t.Helper()
	result, err := Call(context.Background(), c, ToolGetDecision, json.RawMessage(`{"id":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	return result
}
