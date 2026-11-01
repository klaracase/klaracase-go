package klaracase

// This file runs the shared conformance pack (testdata/conformance/cases/*.json). The same
// cases run in every Klaracase SDK. The runner starts an httptest server that serves the
// "responses" of a case in order and records every request, builds a client, calls the SDK
// method named by "group.method" through a plain switch, and checks the request, the sleeps
// and the result.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	confDir      = "testdata/conformance"
	confBasePath = "/api/v1"
	confToken    = "test_token_123"
)

// ---- case file model ----

type confCase struct {
	Name string `json:"name"`
	SDK  struct {
		Group   *string        `json:"group"`
		Method  string         `json:"method"`
		Args    map[string]any `json:"args"`
		Options map[string]any `json:"options"`
		Env     map[string]string
		Repeat  int `json:"repeat"`
	} `json:"sdk"`
	Expect struct {
		Request      *confExpectRequest `json:"request"`
		RequestCount json.RawMessage    `json:"request_count"`
		Sleeps       *[]map[string]float64
	} `json:"expect"`
	Responses    []confResponse `json:"responses"`
	ExpectResult struct {
		Kind                    string         `json:"kind"`
		Assert                  map[string]any `json:"assert"`
		Class                   string         `json:"class"`
		Code                    *string        `json:"code"`
		Status                  *int           `json:"status"`
		Fields                  map[string]any `json:"fields"`
		BodyRoundtrip           bool           `json:"body_roundtrip"`
		DeprecationWarning      *bool          `json:"deprecation_warning"`
		Deprecation             map[string]string
		DeprecationWarningCount *int `json:"deprecation_warning_count"`
	} `json:"expect_result"`
	Runner struct {
		RealClock          bool `json:"real_clock"`
		RepeatLastResponse bool `json:"repeat_last_response"`
	} `json:"runner"`
}

type confExpectRequest struct {
	Method          string              `json:"method"`
	Path            string              `json:"path"`
	Query           map[string]any      `json:"query"`
	Headers         map[string]string   `json:"headers"`
	JSON            any                 `json:"json"`
	Multipart       []map[string]string `json:"multipart"`
	UserAgentSuffix string              `json:"user_agent_suffix"`
}

type confResponse struct {
	Status         int               `json:"status"`
	Headers        map[string]string `json:"headers"`
	Body           json.RawMessage   `json:"body"`
	BodyFile       string            `json:"body_file"`
	BodyText       *string           `json:"body_text"`
	DelaySeconds   float64           `json:"delay_seconds"`
	DropConnection bool              `json:"drop_connection"`
}

// ---- mock server ----

type recordedRequest struct {
	method  string
	rawPath string
	query   url.Values
	header  http.Header
	body    []byte
}

type confMock struct {
	mu         sync.Mutex
	caseDir    string
	responses  []confResponse
	repeatLast bool
	requests   []recordedRequest
	lastServed int
}

func (m *confMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	rawPath, _, _ := strings.Cut(r.RequestURI, "?")
	m.mu.Lock()
	m.requests = append(m.requests, recordedRequest{r.Method, rawPath, r.URL.Query(), r.Header.Clone(), body})
	idx := len(m.requests) - 1
	if idx >= len(m.responses) && m.repeatLast && len(m.responses) > 0 {
		idx = len(m.responses) - 1
	}
	var resp confResponse
	ok := idx < len(m.responses)
	if ok {
		resp = m.responses[idx]
		m.lastServed = idx
	}
	m.mu.Unlock()

	if !ok {
		http.Error(w, "no response left in the case", 599)
		return
	}
	if resp.DropConnection {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err == nil {
			conn.Close()
		}
		return
	}
	if resp.DelaySeconds > 0 {
		select {
		case <-time.After(time.Duration(resp.DelaySeconds * float64(time.Second))):
		case <-r.Context().Done():
			return
		}
	}
	payload, isJSON := m.payload(resp)
	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	if w.Header().Get("Content-Type") == "" && isJSON {
		w.Header().Set("Content-Type", "application/json")
	}
	w.WriteHeader(resp.Status)
	_, _ = w.Write(payload) // the client may have given up already
}

func (m *confMock) payload(resp confResponse) ([]byte, bool) {
	switch {
	case resp.BodyFile != "":
		data, err := os.ReadFile(filepath.Join(m.caseDir, resp.BodyFile))
		if err != nil {
			panic(err)
		}
		return data, strings.HasSuffix(resp.BodyFile, ".json")
	case resp.BodyText != nil:
		return []byte(*resp.BodyText), false
	case len(resp.Body) > 0:
		return resp.Body, true
	}
	return nil, false
}

func (m *confMock) servedBody() []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastServed >= len(m.responses) {
		return nil
	}
	b, _ := m.payload(m.responses[m.lastServed])
	return b
}

// ---- argument helpers ----

type confArgs map[string]any

func (a confArgs) str(k string) string {
	s, _ := a[k].(string)
	return s
}

func (a confArgs) intp(k string) *int {
	f, ok := a[k].(float64)
	if !ok {
		return nil
	}
	return Int(int(f))
}

func (a confArgs) boolp(k string) *bool {
	b, ok := a[k].(bool)
	if !ok {
		return nil
	}
	return Bool(b)
}

func (a confArgs) strs(k string) []string {
	list, _ := a[k].([]any)
	var out []string
	for _, v := range list {
		out = append(out, v.(string))
	}
	return out
}

func (a confArgs) seconds(k string) time.Duration {
	f, _ := a[k].(float64)
	return time.Duration(f * float64(time.Second))
}

func (a confArgs) file(t *testing.T, k string) io.Reader {
	spec, ok := a[k].(map[string]any)
	if !ok {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(confDir, spec["$file"].(string)))
	if err != nil {
		t.Fatal(err)
	}
	return bytes.NewReader(data)
}

func ret[T any](r *T, err error) (any, error) {
	if err != nil {
		return nil, err
	}
	return r, nil
}

// dispatch calls the SDK method named by group and method. A plain switch, no reflection.
func dispatch(t *testing.T, ctx context.Context, c *Client, group *string, method string, raw map[string]any) (any, error) {
	a := confArgs(raw)
	name := method
	if group != nil {
		name = *group + "." + method
	}
	switch name {
	case "search":
		return ret(c.Search(ctx, CrossTypeSearchParams{
			Query: a.str("query"), Jurisdiction: a.str("jurisdiction"), Court: a.str("court"),
			Branch: Branch(a.str("branch")), DateFrom: a.str("date_from"), DateTo: a.str("date_to"),
			Mode: Mode(a.str("mode")), DocumentType: DocumentType(a.str("document_type")),
			CitedNorm: a.str("cited_norm"), CitedStatutes: a.strs("cited_statutes"),
			Limit: a.intp("limit"), Bridge: a.boolp("bridge"),
		}))
	case "decisions.search":
		return ret(c.Decisions.Search(ctx, SearchDecisionsParams{
			Query: a.str("query"), Court: a.str("court"), Branch: Branch(a.str("branch")),
			DocumentType: DocumentType(a.str("document_type")), CitedNorm: a.str("cited_norm"),
			CitedStatutes: a.strs("cited_statutes"), DateFrom: a.str("date_from"), DateTo: a.str("date_to"),
			Mode: Mode(a.str("mode")), Limit: a.intp("limit"), Page: a.intp("page"), Bridge: a.boolp("bridge"),
		}))
	case "decisions.get":
		return ret(c.Decisions.Get(ctx, a.str("id"), GetDecisionParams{
			MaxChars: a.intp("max_chars"), RnFrom: a.intp("rn_from"), RnTo: a.intp("rn_to"),
			Section: a.str("section"), Court: a.str("court"), Date: a.str("date"),
		}))
	case "decisions.citations":
		return ret(c.Decisions.Citations(ctx, a.str("id"), CitationsParams{
			Direction: Direction(a.str("direction")), Depth: a.intp("depth"), Page: a.intp("page"),
			Court: a.str("court"), Date: a.str("date"),
		}))
	case "norms.get":
		return ret(c.Norms.Get(ctx, a.str("reference"), GetNormParams{Jurisdiction: a.str("jurisdiction")}))
	case "datapackages.create":
		return ret(c.Datapackages.Create(ctx, CreateDatapackageParams{
			IDs: a.strs("ids"), IncludeCitations: a.boolp("include_citations"),
			FullText: a.boolp("full_text"), MaxChars: a.intp("max_chars"),
		}))
	case "citation_checks.create":
		return ret(c.CitationChecks.Create(ctx, CreateCitationCheckParams{
			Text: a.str("text"), DocumentID: a.str("document_id"), ReferenceDate: a.str("reference_date"),
		}))
	case "citation_checks.get":
		return ret(c.CitationChecks.Get(ctx, a.str("run_id")))
	case "citation_check_documents.upload":
		return ret(c.CitationCheckDocuments.Upload(ctx, UploadParams{
			File: a.file(t, "file"), Filename: a.str("filename"), ContentType: a.str("content_type"),
		}))
	case "citation_check_documents.get":
		return ret(c.CitationCheckDocuments.Get(ctx, a.str("document_id")))
	case "citation_check_documents.wait_until_ready":
		return ret(c.CitationCheckDocuments.WaitUntilReady(ctx, a.str("document_id"), WaitParams{
			PollInterval: a.seconds("poll_interval"), Timeout: a.seconds("timeout"),
		}))
	case "corpus.coverage":
		return ret(c.Corpus.Coverage(ctx, CoverageParams{Branch: Branch(a.str("branch")), Top: a.intp("top")}))
	case "corpus.facets":
		return ret(c.Corpus.Facets(ctx, FacetField(a.str("field")), FacetsParams{
			Type: a.str("type"), Norm: a.str("norm"), Prefix: a.str("prefix"),
		}))
	}
	t.Fatalf("the runner does not know %q", name)
	return nil, nil
}

// ---- the test ----

func TestConformance(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(confDir, "cases", "*.json"))
	if err != nil || len(files) != 102 {
		t.Fatalf("expected 102 conformance cases, found %d (err %v)", len(files), err)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		var cs confCase
		if err := json.Unmarshal(data, &cs); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		t.Run(cs.Name, func(t *testing.T) { runConformanceCase(t, filepath.Dir(file), cs) })
	}
}

func runConformanceCase(t *testing.T, caseDir string, cs confCase) {
	// 1. reset state
	t.Setenv(EnvToken, "")
	os.Unsetenv(EnvToken)
	for k, v := range cs.SDK.Env {
		t.Setenv(k, v)
	}
	resetDeprecationState()

	// 2. mock server
	mock := &confMock{caseDir: caseDir, responses: cs.Responses, repeatLast: cs.Runner.RepeatLastResponse}
	srv := httptest.NewServer(mock)
	defer srv.Close()

	// 3. client
	var sleeps []float64
	var sleepMu sync.Mutex
	var notices []DeprecationNotice
	opts := []Option{WithDeprecationHandler(func(n DeprecationNotice) { notices = append(notices, n) })}
	base := srv.URL + confBasePath
	if cs.SDK.Options["base_url_trailing_slash"] == true {
		base += "/"
	}
	opts = append(opts, WithBaseURL(base))
	if v, ok := cs.SDK.Options["max_retries"].(float64); ok {
		opts = append(opts, WithMaxRetries(int(v)))
	}
	if v, ok := cs.SDK.Options["timeout"].(float64); ok {
		opts = append(opts, WithTimeout(time.Duration(v*float64(time.Second))))
	}
	if v, ok := cs.SDK.Options["user_agent_suffix"].(string); ok {
		opts = append(opts, WithUserAgentSuffix(v))
	}
	if !cs.Runner.RealClock {
		opts = append(opts, withSleep(func(ctx context.Context, d time.Duration) error {
			sleepMu.Lock()
			sleeps = append(sleeps, d.Seconds())
			sleepMu.Unlock()
			return ctx.Err()
		}))
	}
	var client *Client
	var err error
	tok, hasTok := cs.SDK.Options["token"]
	switch {
	case !hasTok:
		client, err = New(confToken, opts...)
	case tok == nil:
		client, err = NewFromEnv(opts...)
	default:
		client, err = New(tok.(string), opts...)
	}

	// 4. calls
	repeat := max(cs.SDK.Repeat, 1)
	results := make([]any, 0, repeat)
	errs := make([]error, 0, repeat)
	if err != nil {
		results, errs = append(results, nil), append(errs, err)
	} else {
		for range repeat {
			r, e := dispatch(t, context.Background(), client, cs.SDK.Group, cs.SDK.Method, cs.SDK.Args)
			results, errs = append(results, r), append(errs, e)
		}
	}

	// 5. request assertions
	mock.mu.Lock()
	reqs := append([]recordedRequest(nil), mock.requests...)
	mock.mu.Unlock()
	checkRequestCount(t, cs.Expect.RequestCount, len(reqs))
	for _, rr := range reqs {
		if cs.Expect.Request == nil {
			t.Fatalf("expected no request, the mock saw %s %s", rr.method, rr.rawPath)
		}
		checkRequest(t, *cs.Expect.Request, rr)
	}
	if cs.Expect.Sleeps != nil {
		checkSleeps(t, *cs.Expect.Sleeps, sleeps)
	}

	// 6. result assertions
	for i := range results {
		if cs.ExpectResult.Kind == "success" {
			if errs[i] != nil {
				t.Fatalf("call %d returned an error: %v", i, errs[i])
			}
			checkSuccess(t, cs, results[i])
			continue
		}
		checkError(t, cs, errs[i], mock.servedBody())
	}
	if cs.ExpectResult.DeprecationWarning != nil && (len(notices) > 0) != *cs.ExpectResult.DeprecationWarning {
		t.Errorf("deprecation warning: want %v, got %d notices", *cs.ExpectResult.DeprecationWarning, len(notices))
	}
	if cs.ExpectResult.DeprecationWarningCount != nil && len(notices) != *cs.ExpectResult.DeprecationWarningCount {
		t.Errorf("deprecation handler ran %d times, want %d", len(notices), *cs.ExpectResult.DeprecationWarningCount)
	}
	if want := cs.ExpectResult.Deprecation; want != nil {
		if len(notices) == 0 {
			t.Fatal("no deprecation notice")
		}
		got := map[string]string{"deprecation": notices[0].Deprecation, "sunset": notices[0].Sunset, "link": notices[0].Link}
		for k, v := range want {
			if got[k] != v {
				t.Errorf("deprecation notice %s = %q, want %q", k, got[k], v)
			}
		}
	}
}

func checkRequestCount(t *testing.T, raw json.RawMessage, got int) {
	t.Helper()
	var n int
	if json.Unmarshal(raw, &n) == nil {
		if got != n {
			t.Errorf("request count = %d, want %d", got, n)
		}
		return
	}
	var r struct{ Min, Max int }
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("bad request_count %s", raw)
	}
	if got < r.Min || got > r.Max {
		t.Errorf("request count = %d, want %d to %d", got, r.Min, r.Max)
	}
}

func checkSleeps(t *testing.T, want []map[string]float64, got []float64) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("sleeps = %v, want %d sleeps %v", got, len(want), want)
		return
	}
	for i, w := range want {
		if exact, ok := w["exact"]; ok {
			if math.Abs(got[i]-exact) > 1e-6 {
				t.Errorf("sleep %d = %v, want exactly %v", i, got[i], exact)
			}
			continue
		}
		if got[i] < w["min"]-1e-9 || got[i] > w["max"]+1e-9 {
			t.Errorf("sleep %d = %v, want between %v and %v", i, got[i], w["min"], w["max"])
		}
	}
}

func checkRequest(t *testing.T, want confExpectRequest, got recordedRequest) {
	t.Helper()
	if got.method != want.Method {
		t.Errorf("method = %s, want %s", got.method, want.Method)
	}
	if got.rawPath != confBasePath+want.Path {
		t.Errorf("raw path = %s, want %s", got.rawPath, confBasePath+want.Path)
	}
	// Query: the set of names equals the map, values match, list order matters.
	wantQuery := map[string][]string{}
	for k, v := range want.Query {
		switch x := v.(type) {
		case string:
			wantQuery[k] = []string{x}
		case []any:
			for _, e := range x {
				wantQuery[k] = append(wantQuery[k], e.(string))
			}
		}
	}
	if !reflect.DeepEqual(map[string][]string(got.query), wantQuery) && !(len(got.query) == 0 && len(wantQuery) == 0) {
		t.Errorf("query = %v, want %v", got.query, wantQuery)
	}
	for name, value := range want.Headers {
		have := got.header.Get(name)
		switch name {
		case "user-agent":
			if !strings.HasPrefix(have, value) {
				t.Errorf("user-agent %q does not start with %q", have, value)
			}
			if !strings.HasPrefix(have, "klaracase-go/"+Version+" go/") {
				t.Errorf("user-agent %q is not klaracase-go/%s go/<version>", have, Version)
			}
			if want.UserAgentSuffix != "" && !strings.HasSuffix(have, " "+want.UserAgentSuffix) {
				t.Errorf("user-agent %q does not end with %q", have, " "+want.UserAgentSuffix)
			}
		case "content-type":
			mt, _, _ := mime.ParseMediaType(have)
			if !strings.EqualFold(mt, value) {
				t.Errorf("content-type = %q, want %q", mt, value)
			}
		default:
			if have != value {
				t.Errorf("header %s = %q, want %q", name, have, value)
			}
		}
	}
	if _, ok := want.Headers["content-type"]; !ok && got.method == http.MethodGet && got.header.Get("Content-Type") != "" {
		t.Errorf("a GET sent content-type %q", got.header.Get("Content-Type"))
	}
	if want.JSON != nil {
		var body any
		if err := json.Unmarshal(got.body, &body); err != nil {
			t.Fatalf("request body is not JSON: %v", err)
		}
		if !reflect.DeepEqual(body, want.JSON) {
			t.Errorf("json body = %s, want %v", got.body, want.JSON)
		}
	}
	if want.Multipart != nil {
		mt, params, err := mime.ParseMediaType(got.header.Get("Content-Type"))
		if err != nil || !strings.HasPrefix(mt, "multipart/") {
			t.Fatalf("content-type %q is not multipart", got.header.Get("Content-Type"))
		}
		mr := multipart.NewReader(bytes.NewReader(got.body), params["boundary"])
		var parts []map[string]string
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			parts = append(parts, map[string]string{"name": p.FormName(), "filename": p.FileName(), "content_type": p.Header.Get("Content-Type")})
		}
		if !reflect.DeepEqual(parts, want.Multipart) {
			t.Errorf("multipart parts = %v, want %v", parts, want.Multipart)
		}
	}
}

// ---- success assertions ----

func checkSuccess(t *testing.T, cs confCase, result any) {
	t.Helper()
	env, ok := result.(interface{ envelope() *Envelope })
	if !ok {
		t.Fatalf("result %T has no envelope", result)
	}
	view := resultView(t, result, env.envelope())
	for path, want := range cs.ExpectResult.Assert {
		got, found := lookupPath(view, path)
		if !found {
			t.Errorf("%s: path not found in the result", path)
			continue
		}
		if msg := matchValue(got, want); msg != "" {
			t.Errorf("%s: %s", path, msg)
		}
	}
	if cs.ExpectResult.BodyRoundtrip {
		var want, have any
		_ = json.Unmarshal(mockBodyFor(t, cs), &want)
		if err := json.Unmarshal(env.envelope().Raw, &have); err != nil || !reflect.DeepEqual(have, want) {
			t.Errorf("Raw does not equal the body: %v", err)
		}
	}
}

func mockBodyFor(t *testing.T, cs confCase) []byte {
	t.Helper()
	// The case has one answer or repeats the last one; the roundtrip case has exactly one.
	m := &confMock{caseDir: filepath.Join(confDir, "cases")}
	b, _ := m.payload(cs.Responses[len(cs.Responses)-1])
	return b
}

// resultView turns a typed result into generic JSON, with the roots meta and extra added.
func resultView(t *testing.T, result any, env *Envelope) map[string]any {
	t.Helper()
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	var view map[string]any
	if err := json.Unmarshal(data, &view); err != nil {
		t.Fatal(err)
	}
	var release any
	if env.Meta.APIRelease != "" {
		release = env.Meta.APIRelease
	}
	view["meta"] = map[string]any{
		"status":           float64(env.Meta.Status),
		"api_release":      release,
		"request_duration": env.Meta.Duration.Seconds(),
	}
	view["extra"] = decodeExtra(env.Extra)
	return view
}

func decodeExtra(extra map[string]json.RawMessage) map[string]any {
	out := map[string]any{}
	for k, raw := range extra {
		var v any
		_ = json.Unmarshal(raw, &v)
		out[k] = v
	}
	return out
}

// lookupPath resolves "a.b[0].c" and `values["§ 543 Abs. 1 BGB"].value` against generic JSON.
func lookupPath(root any, path string) (any, bool) {
	cur := root
	for i := 0; i < len(path); {
		switch {
		case path[i] == '.':
			i++
		case strings.HasPrefix(path[i:], `["`):
			end := strings.Index(path[i:], `"]`)
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, false
			}
			cur, ok = m[path[i+2:i+end]]
			if !ok {
				return nil, false
			}
			i += end + 2
		case path[i] == '[':
			end := strings.IndexByte(path[i:], ']')
			n, _ := strconv.Atoi(path[i+1 : i+end])
			list, ok := cur.([]any)
			if !ok || n >= len(list) {
				return nil, false
			}
			cur = list[n]
			i += end + 1
		default:
			end := strings.IndexAny(path[i:], ".[")
			if end < 0 {
				end = len(path) - i
			}
			m, ok := cur.(map[string]any)
			if !ok {
				return nil, false
			}
			cur, ok = m[path[i:i+end]]
			if !ok {
				return nil, false
			}
			i += end
		}
	}
	return cur, true
}

// matchValue compares a value with a literal or a matcher object ({"$type": ...}). It returns
// an empty string on a match.
func matchValue(got, want any) string {
	m, isMap := want.(map[string]any)
	if isMap && len(m) > 0 && allDollar(m) {
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if msg := matchOne(got, k, m[k]); msg != "" {
				return msg
			}
		}
		return ""
	}
	if !reflect.DeepEqual(got, want) {
		return fmt.Sprintf("got %v (%T), want %v (%T)", got, got, want, want)
	}
	return ""
}

func allDollar(m map[string]any) bool {
	for k := range m {
		if !strings.HasPrefix(k, "$") {
			return false
		}
	}
	return true
}

func matchOne(got any, matcher string, arg any) string {
	switch matcher {
	case "$type":
		var kind string
		switch got.(type) {
		case float64:
			kind = "number"
		case string:
			kind = "string"
		case map[string]any:
			kind = "object"
		case []any:
			kind = "array"
		case bool:
			kind = "boolean"
		default:
			kind = "null"
		}
		if kind != arg {
			return fmt.Sprintf("type is %s, want %v", kind, arg)
		}
	case "$min":
		f, ok := got.(float64)
		if !ok || f < arg.(float64) {
			return fmt.Sprintf("%v is not at least %v", got, arg)
		}
	case "$length":
		n := -1
		switch x := got.(type) {
		case string:
			n = len([]rune(x))
		case []any:
			n = len(x)
		}
		if float64(n) != arg.(float64) {
			return fmt.Sprintf("length is %d, want %v", n, arg)
		}
	case "$contains":
		if s, ok := got.(string); !ok || !strings.Contains(s, arg.(string)) {
			return fmt.Sprintf("%q does not contain %q", got, arg)
		}
	case "$starts_with":
		if s, ok := got.(string); !ok || !strings.HasPrefix(s, arg.(string)) {
			return fmt.Sprintf("%q does not start with %q", got, arg)
		}
	default:
		return "unknown matcher " + matcher
	}
	return ""
}

// ---- error assertions ----

func errorHasClass(err error, class string) bool {
	switch class {
	case "AuthenticationError":
		var e *AuthenticationError
		return errors.As(err, &e)
	case "PermissionError":
		var e *PermissionError
		return errors.As(err, &e)
	case "NotFoundError":
		var e *NotFoundError
		return errors.As(err, &e)
	case "ConflictError":
		var e *ConflictError
		return errors.As(err, &e)
	case "ValidationError":
		var e *ValidationError
		return errors.As(err, &e)
	case "InsufficientCreditsError":
		var e *InsufficientCreditsError
		return errors.As(err, &e)
	case "RateLimitError":
		var e *RateLimitError
		return errors.As(err, &e)
	case "ServiceUnavailableError":
		var e *ServiceUnavailableError
		return errors.As(err, &e)
	case "ApiError":
		var e *APIError
		return errors.As(err, &e)
	case "TransportError":
		var e *TransportError
		return errors.As(err, &e)
	case "DocumentFailedError":
		var e *DocumentFailedError
		return errors.As(err, &e)
	case "DocumentTimeoutError":
		var e *DocumentTimeoutError
		return errors.As(err, &e)
	case "ConfigurationError":
		var e *ConfigurationError
		return errors.As(err, &e)
	}
	return false
}

// errorView builds the object that "fields" paths are resolved against.
func errorView(err error) map[string]any {
	view := map[string]any{}
	nilIf := func(s string) any {
		if s == "" {
			return nil
		}
		return s
	}
	if base, ok := AsError(err); ok {
		view["type"], view["title"], view["detail"] = nilIf(base.Type), nilIf(base.Title), base.Detail
		view["code"], view["api_release"] = nilIf(base.Code), nilIf(base.APIRelease)
		if base.Status != 0 {
			view["status"] = float64(base.Status)
		} else {
			view["status"] = nil
		}
		view["extra"] = decodeExtra(base.Extra)
	}
	var cfg *ConfigurationError
	if errors.As(err, &cfg) {
		view["message"] = cfg.Message
	}
	var credits *InsufficientCreditsError
	if errors.As(err, &credits) {
		view["cost"], view["balance"] = float64(credits.Cost), float64(credits.Balance)
	}
	var rate *RateLimitError
	if errors.As(err, &rate) {
		view["retry_after_seconds"] = float64(rate.RetryAfterSeconds)
	}
	var perm *PermissionError
	if errors.As(err, &perm) {
		view["required_ability"] = perm.RequiredAbility
	}
	var val *ValidationError
	if errors.As(err, &val) && val.Errors != nil {
		data, _ := json.Marshal(val.Errors)
		var generic any
		_ = json.Unmarshal(data, &generic)
		view["errors"] = generic
	}
	var failed *DocumentFailedError
	if errors.As(err, &failed) {
		view["error_code"] = failed.ErrorCode
		view["document"] = asGeneric(failed.Document)
	}
	var timedOut *DocumentTimeoutError
	if errors.As(err, &timedOut) {
		view["document"] = asGeneric(timedOut.Document)
	}
	return view
}

func asGeneric(v any) any {
	data, _ := json.Marshal(v)
	var out any
	_ = json.Unmarshal(data, &out)
	return out
}

func checkError(t *testing.T, cs confCase, err error, servedBody []byte) {
	t.Helper()
	want := cs.ExpectResult
	if err == nil {
		t.Fatalf("expected %s, the call succeeded", want.Class)
	}
	if !errorHasClass(err, want.Class) {
		t.Fatalf("error %T (%v) is not %s", err, err, want.Class)
	}
	if want.Class != "ConfigurationError" {
		base, ok := AsError(err)
		if !ok {
			t.Fatalf("errors.As(*Error) failed for %T", err)
		}
		if want.Code == nil && base.Code != "" {
			t.Errorf("code = %q, want none", base.Code)
		}
		if want.Code != nil && base.Code != *want.Code {
			t.Errorf("code = %q, want %q", base.Code, *want.Code)
		}
		if want.Status == nil && base.Status != 0 {
			t.Errorf("status = %d, want none", base.Status)
		}
		if want.Status != nil && base.Status != *want.Status {
			t.Errorf("status = %d, want %d", base.Status, *want.Status)
		}
		checkProblemMembers(t, base, servedBody)
	}
	view := errorView(err)
	for path, wantValue := range want.Fields {
		got, found := lookupPath(view, path)
		if !found {
			got, found = lookupPath(view["extra"], path)
		}
		if !found {
			t.Errorf("field %s not found on the error", path)
			continue
		}
		if msg := matchValue(got, wantValue); msg != "" {
			t.Errorf("field %s: %s", path, msg)
		}
	}
}

// checkProblemMembers runs the checks the pack asks for without the case saying so.
func checkProblemMembers(t *testing.T, base *Error, body []byte) {
	t.Helper()
	var members map[string]any
	if json.Unmarshal(body, &members) != nil || members == nil {
		return
	}
	if _, isProblem := members["code"].(string); !isProblem || base.Status == 0 {
		return
	}
	str := func(k string) string { s, _ := members[k].(string); return s }
	if base.Type != str("type") || base.Title != str("title") || base.Detail != str("detail") || base.Code != str("code") {
		t.Errorf("error members %q/%q/%q/%q differ from the body", base.Type, base.Title, base.Detail, base.Code)
	}
	if status, _ := members["status"].(float64); int(status) != base.Status {
		t.Errorf("status %d differs from the body status %v", base.Status, members["status"])
	}
	wantExtra := map[string]any{}
	for k, v := range members {
		switch k {
		case "type", "title", "status", "detail", "code":
		default:
			wantExtra[k] = v
		}
	}
	gotExtra := decodeExtra(base.Extra)
	if !reflect.DeepEqual(gotExtra, wantExtra) && !(len(gotExtra) == 0 && len(wantExtra) == 0) {
		t.Errorf("extra = %v, want %v", gotExtra, wantExtra)
	}
}
