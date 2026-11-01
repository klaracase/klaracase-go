package klaracase

// The drift guards. The SDK keeps every key it does not model in an Extra map, which is the
// right behavior for a key the server adds later. The same mechanism would also hide a key
// that the server already sends today and that nobody modeled. These tests make that visible:
// every example body, every conformance body and every JSON block of the contract must decode
// with an empty Extra everywhere, and must marshal back with all its keys.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

// ---- which Go type decodes which body ----

// exampleTypes maps a file in testdata/examples to the type that decodes it.
var exampleTypes = map[string]reflect.Type{
	"citation-check-documents.show.json":  reflect.TypeFor[CitationCheckDocument](),
	"citation-check-documents.store.json": reflect.TypeFor[CitationCheckDocument](),
	"citation-checks.show.json":           reflect.TypeFor[CitationCheckRun](),
	"citation-checks.store.json":          reflect.TypeFor[CitationCheckRun](),
	"corpus.coverage.json":                reflect.TypeFor[CoverageResponse](),
	"corpus.facets.json":                  reflect.TypeFor[FacetsResponse](),
	"datapackages.store.json":             reflect.TypeFor[DatapackageResponse](),
	"decisions.citations.json":            reflect.TypeFor[CitationsResponse](),
	"decisions.search.json":               reflect.TypeFor[DecisionSearchResponse](),
	"decisions.show.json":                 reflect.TypeFor[DecisionResponse](),
	"norms.show.json":                     reflect.TypeFor[NormResponse](),
	"search.json":                         reflect.TypeFor[CrossTypeSearchResponse](),
}

// bodyTypes maps a file in testdata/conformance/bodies to its type. A body that is not
// listed fails the test, so a new body cannot skip the guard.
var bodyTypes = map[string]reflect.Type{
	"corpus.coverage.not_metered.json":    reflect.TypeFor[CoverageResponse](),
	"corpus.facets.cited_norms.json":      reflect.TypeFor[FacetsResponse](),
	"decisions.search.future_key.json":    reflect.TypeFor[DecisionSearchResponse](),
	"documents.extracting.json":           reflect.TypeFor[CitationCheckDocument](),
	"documents.failed.encrypted_pdf.json": reflect.TypeFor[CitationCheckDocument](),
	"documents.ocr_pending.json":          reflect.TypeFor[CitationCheckDocument](),
	"documents.ready.json":                reflect.TypeFor[CitationCheckDocument](),
	"documents.uploaded.json":             reflect.TypeFor[CitationCheckDocument](),
	"norms.show.dsgvo.json":               reflect.TypeFor[NormResponse](),
}

// contractTypes maps "METHOD path" of a heading in CONTRACT.md to the type of its 200 or 202 body.
var contractTypes = map[string]reflect.Type{
	"POST /citation-checks":                    reflect.TypeFor[CitationCheckRun](),
	"GET /citation-checks/{run}":               reflect.TypeFor[CitationCheckRun](),
	"POST /citation-check-documents":           reflect.TypeFor[CitationCheckDocument](),
	"GET /citation-check-documents/{document}": reflect.TypeFor[CitationCheckDocument](),
	"GET /decisions/{id}/citations":            reflect.TypeFor[CitationsResponse](),
	"GET /corpus/coverage":                     reflect.TypeFor[CoverageResponse](),
	"GET /corpus/facets/{field}":               reflect.TypeFor[FacetsResponse](),
	"POST /datapackages":                       reflect.TypeFor[DatapackageResponse](),
	"GET /decisions/{id}":                      reflect.TypeFor[DecisionResponse](),
	"GET /norms/{reference}":                   reflect.TypeFor[NormResponse](),
	"GET /decisions/search":                    reflect.TypeFor[DecisionSearchResponse](),
	"GET /search":                              reflect.TypeFor[CrossTypeSearchResponse](),
}

// ---- the allow list ----

// allowedExtra says whether a key may sit in Extra at this place. There are two reasons, and
// each is named here:
//
//  1. A statute row of a cross type search carries four keys that are always null (a statute
//     has no Aktenzeichen, court, date or document type). LawHit does not model them on
//     purpose, so a caller cannot mistake a null for a value.
//  2. The conformance body decisions.search.future_key.json carries invented keys to prove
//     that unknown keys survive.
func allowedExtra(file, path, key string) bool {
	if strings.HasPrefix(path, "Sections.Laws[") && !strings.Contains(path[len("Sections.Laws["):], ".") {
		switch key {
		case "aktenzeichen", "court", "date", "document_type":
			return true
		}
	}
	if file == "decisions.search.future_key.json" {
		switch {
		case path == "" && (key == "future_key" || key == "another_future_key"):
			return true
		case path == "Results[0]" && key == "future_result_key":
			return true
		}
	}
	return false
}

// ---- the walker ----

var (
	timeType       = reflect.TypeFor[time.Time]()
	rawMessageType = reflect.TypeFor[json.RawMessage]()
	extraType      = reflect.TypeFor[map[string]json.RawMessage]()
)

// extraFinding is one unknown key that a decoded value holds.
type extraFinding struct{ path, key string }

func (f extraFinding) String() string {
	if f.path == "" {
		return f.key
	}
	return f.path + "." + f.key
}

// extraKeys walks a decoded value and returns every key that sits in an Extra map. The path
// is made of Go field names and indexes, so it points at the struct that holds the key.
func extraKeys(v reflect.Value, path string) []extraFinding {
	var out []extraFinding
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			out = append(out, extraKeys(v.Elem(), path)...)
		}
	case reflect.Slice, reflect.Array:
		if v.Type() == rawMessageType || v.Type().Elem().Kind() == reflect.Uint8 {
			return nil
		}
		for i := range v.Len() {
			out = append(out, extraKeys(v.Index(i), fmt.Sprintf("%s[%d]", path, i))...)
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			out = append(out, extraKeys(v.MapIndex(key), fmt.Sprintf("%s[%v]", path, key))...)
		}
	case reflect.Struct:
		if v.Type() == timeType {
			return nil
		}
		for i := range v.NumField() {
			field := v.Type().Field(i)
			if !field.IsExported() {
				continue
			}
			if field.Name == "Extra" && field.Type == extraType {
				keys := make([]string, 0, v.Field(i).Len())
				for _, k := range v.Field(i).MapKeys() {
					keys = append(keys, k.String())
				}
				sort.Strings(keys)
				for _, k := range keys {
					out = append(out, extraFinding{path: path, key: k})
				}
				continue
			}
			child := field.Name
			if path != "" && !field.Anonymous {
				child = path + "." + field.Name
			} else if field.Anonymous {
				child = path
			}
			out = append(out, extraKeys(v.Field(i), child)...)
		}
	}
	return out
}

// unexpectedExtra decodes body into a new value of type t and returns the unknown keys that
// the allow list does not cover.
func unexpectedExtra(t *testing.T, file string, typ reflect.Type, body []byte) []extraFinding {
	t.Helper()
	target := reflect.New(typ)
	if err := json.Unmarshal(body, target.Interface()); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	var bad []extraFinding
	for _, f := range extraKeys(target, "") {
		if !allowedExtra(file, f.path, f.key) {
			bad = append(bad, f)
		}
	}
	return bad
}

// missingKeys lists the key paths of original that remarshaled lacks.
func missingKeys(original, remarshaled any, path string) []string {
	var missing []string
	switch o := original.(type) {
	case map[string]any:
		r, ok := remarshaled.(map[string]any)
		if !ok {
			return []string{path + " (not an object after marshal)"}
		}
		keys := make([]string, 0, len(o))
		for k := range o {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := path + "." + k
			rv, present := r[k]
			if !present {
				missing = append(missing, child)
				continue
			}
			missing = append(missing, missingKeys(o[k], rv, child)...)
		}
	case []any:
		r, ok := remarshaled.([]any)
		if !ok || len(r) != len(o) {
			return []string{path + " (array changed after marshal)"}
		}
		for i := range o {
			missing = append(missing, missingKeys(o[i], r[i], fmt.Sprintf("%s[%d]", path, i))...)
		}
	}
	return missing
}

// allowedMissing says whether a key may be absent after a marshal. These are the keys that
// the allow list lets sit in Extra, because Extra is not marshaled; Raw is the lossless copy.
func allowedMissing(file, path string) bool {
	segs := strings.Split(strings.TrimPrefix(path, "."), ".")
	last := segs[len(segs)-1]
	if strings.HasPrefix(path, ".sections.laws[") && len(segs) == 3 {
		switch last {
		case "aktenzeichen", "court", "date", "document_type":
			return true
		}
	}
	if file == "decisions.search.future_key.json" {
		switch path {
		case ".future_key", ".another_future_key", ".results[0].future_result_key":
			return true
		}
	}
	return false
}

func unexpectedMissing(t *testing.T, file string, typ reflect.Type, body []byte) []string {
	t.Helper()
	target := reflect.New(typ)
	if err := json.Unmarshal(body, target.Interface()); err != nil {
		t.Fatalf("%s: %v", file, err)
	}
	out, err := json.Marshal(target.Interface())
	if err != nil {
		t.Fatalf("%s: marshal: %v", file, err)
	}
	var original, remarshaled any
	if err := json.Unmarshal(body, &original); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(out, &remarshaled); err != nil {
		t.Fatal(err)
	}
	var bad []string
	for _, p := range missingKeys(original, remarshaled, "") {
		if !allowedMissing(file, p) {
			bad = append(bad, p)
		}
	}
	return bad
}

// ---- the sources of bodies ----

type namedBody struct {
	name string
	typ  reflect.Type
	body []byte
}

func exampleBodies(t *testing.T) []namedBody {
	t.Helper()
	var out []namedBody
	files, err := filepath.Glob("testdata/examples/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no examples found: %v", err)
	}
	for _, file := range files {
		typ, ok := exampleTypes[filepath.Base(file)]
		if !ok {
			t.Fatalf("%s has no entry in exampleTypes", file)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, namedBody{filepath.Base(file), typ, data})
	}
	if len(out) != len(exampleTypes) {
		t.Fatalf("exampleTypes lists %d files, found %d", len(exampleTypes), len(out))
	}
	return out
}

func conformanceBodies(t *testing.T) []namedBody {
	t.Helper()
	var out []namedBody
	files, err := filepath.Glob("testdata/conformance/bodies/*.json")
	if err != nil || len(files) == 0 {
		t.Fatalf("no conformance bodies found: %v", err)
	}
	for _, file := range files {
		typ, ok := bodyTypes[filepath.Base(file)]
		if !ok {
			t.Fatalf("%s has no entry in bodyTypes", file)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, namedBody{filepath.Base(file), typ, data})
	}
	return out
}

// contractBlocks returns the JSON body of the 200 or 202 answer of every operation in CONTRACT.md.
// The file has one "## METHOD path" heading per operation. Under it, the line "- **200** ..."
// is followed by a fenced json block.
func contractBlocks(t *testing.T) []namedBody {
	t.Helper()
	file, err := os.Open("testdata/contract/CONTRACT.md")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	var (
		out     []namedBody
		heading string
		success bool
		inBlock bool
		block   bytes.Buffer
	)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case inBlock && strings.HasPrefix(line, "```"):
			inBlock = false
			if typ, ok := contractTypes[heading]; ok && success {
				out = append(out, namedBody{"CONTRACT.md " + heading, typ, append([]byte(nil), block.Bytes()...)})
			}
			success = false
			block.Reset()
		case inBlock:
			block.WriteString(line)
			block.WriteByte('\n')
		case strings.HasPrefix(line, "## "):
			heading = strings.TrimPrefix(line, "## ")
			success = false
		case strings.HasPrefix(line, "- **"):
			success = strings.HasPrefix(line, "- **200**") || strings.HasPrefix(line, "- **202**")
		case strings.HasPrefix(line, "```json"):
			inBlock = true
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if len(out) != len(contractTypes) {
		t.Fatalf("found %d success blocks in CONTRACT.md, want %d", len(out), len(contractTypes))
	}
	return out
}

func allBodies(t *testing.T) []namedBody {
	t.Helper()
	bodies := exampleBodies(t)
	bodies = append(bodies, conformanceBodies(t)...)
	return append(bodies, contractBlocks(t)...)
}

// ---- the guards ----

func TestExamplesHaveNoExtra(t *testing.T) {
	for _, b := range allBodies(t) {
		t.Run(b.name, func(t *testing.T) {
			for _, f := range unexpectedExtra(t, b.name, b.typ, b.body) {
				t.Errorf("the body carries a key the SDK does not model: %s", f)
			}
		})
	}
}

func TestRemarshalKeepsExampleKeys(t *testing.T) {
	for _, b := range allBodies(t) {
		t.Run(b.name, func(t *testing.T) {
			for _, p := range unexpectedMissing(t, b.name, b.typ, b.body) {
				t.Errorf("a key of the body is lost when the typed value is marshaled: %s", p)
			}
		})
	}
}

// modelRoots are the response types. Every other model type is reachable from them.
var modelRoots = []reflect.Type{
	reflect.TypeFor[DecisionSearchResponse](), reflect.TypeFor[CrossTypeSearchResponse](),
	reflect.TypeFor[DecisionResponse](), reflect.TypeFor[CitationsResponse](),
	reflect.TypeFor[NormResponse](), reflect.TypeFor[DatapackageResponse](),
	reflect.TypeFor[CitationCheckRun](), reflect.TypeFor[CitationCheckDocument](),
	reflect.TypeFor[CoverageResponse](), reflect.TypeFor[FacetsResponse](),
}

// reachableStructs collects the struct types reachable from the roots through fields.
func reachableStructs(roots []reflect.Type) map[reflect.Type]bool {
	seen := map[reflect.Type]bool{}
	var visit func(t reflect.Type)
	visit = func(t reflect.Type) {
		switch t.Kind() {
		case reflect.Pointer, reflect.Slice, reflect.Array:
			visit(t.Elem())
		case reflect.Map:
			visit(t.Elem())
		case reflect.Struct:
			if seen[t] || t == timeType {
				return
			}
			seen[t] = true
			for i := range t.NumField() {
				if f := t.Field(i); f.IsExported() {
					visit(f.Type)
				}
			}
		}
	}
	for _, r := range roots {
		visit(r)
	}
	return seen
}

func hasOwnExtra(t reflect.Type) bool {
	f, ok := t.FieldByName("Extra")
	return ok && f.Type == extraType
}

// TestEveryExtraStructDecodesItself checks that each model struct with an Extra map has its own
// UnmarshalJSON. Without it the standard decoder fills the known fields and drops the rest, and
// Extra stays empty without any error.
func TestEveryExtraStructDecodesItself(t *testing.T) {
	unmarshaler := reflect.TypeFor[json.Unmarshaler]()
	reachable := reachableStructs(modelRoots)
	names := map[string]bool{}
	for typ := range reachable {
		if typ.Name() == "Envelope" {
			continue // embedded in every response, which decodes itself
		}
		if !hasOwnExtra(typ) {
			continue
		}
		names[typ.Name()] = true
		if !reflect.PointerTo(typ).Implements(unmarshaler) {
			t.Errorf("%s has an Extra map but no UnmarshalJSON", typ.Name())
		}
	}

	// A model type that no root reaches would escape the loop above. Find every struct with an
	// Extra field in the source and require that it was reached.
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi os.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				spec, ok := n.(*ast.TypeSpec)
				if !ok {
					return true
				}
				st, ok := spec.Type.(*ast.StructType)
				if !ok {
					return true
				}
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						if name.Name == "Extra" && !names[spec.Name.Name] && spec.Name.Name != "Envelope" && spec.Name.Name != "Error" {
							t.Errorf("%s has an Extra field but no model root reaches it; add its root to modelRoots", spec.Name.Name)
						}
					}
				}
				return true
			})
		}
	}
	if len(names) < 30 {
		t.Errorf("only %d model structs with Extra found, the walker is broken", len(names))
	}
}

// ---- the guards must be seen to fail ----

func TestDriftGuardSeesANestedUnknownKey(t *testing.T) {
	var body map[string]any
	if err := json.Unmarshal(fixture(t, "examples/citation-checks.store.json"), &body); err != nil {
		t.Fatal(err)
	}
	evidence := body["findings"].([]any)[0].(map[string]any)["evidence"].(map[string]any)
	evidence["invented_key"] = "x"
	body["findings"].([]any)[0].(map[string]any)["check_scope"].(map[string]any)["another_invented_key"] = 1
	data, _ := json.Marshal(body)

	bad := unexpectedExtra(t, "citation-checks.store.json", reflect.TypeFor[CitationCheckRun](), data)
	var got []string
	for _, f := range bad {
		got = append(got, f.String())
	}
	want := []string{"Findings[0].CheckScope.another_invented_key", "Findings[0].Evidence.invented_key"}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the walker reported %v, want %v", got, want)
	}

	// The allow list must not swallow a key on a decision row or on a law row outside the four.
	var cross map[string]any
	_ = json.Unmarshal(fixture(t, "examples/search.json"), &cross)
	law := cross["sections"].(map[string]any)["laws"].([]any)[0].(map[string]any)
	law["brand_new"] = true
	data, _ = json.Marshal(cross)
	bad = unexpectedExtra(t, "search.json", reflect.TypeFor[CrossTypeSearchResponse](), data)
	if len(bad) != 1 || bad[0].key != "brand_new" {
		t.Errorf("the walker reported %v, want only brand_new", bad)
	}
}

func TestRemarshalGuardSeesALostKey(t *testing.T) {
	original := map[string]any{"a": map[string]any{"b": 1.0, "c": []any{map[string]any{"d": 2.0}}}}
	kept := map[string]any{"a": map[string]any{"b": 1.0, "c": []any{map[string]any{"d": 2.0}}}}
	if got := missingKeys(original, kept, ""); len(got) != 0 {
		t.Errorf("nothing was lost, got %v", got)
	}
	lost := map[string]any{"a": map[string]any{"b": 1.0, "c": []any{map[string]any{}}}}
	if got := missingKeys(original, lost, ""); !reflect.DeepEqual(got, []string{".a.c[0].d"}) {
		t.Errorf("got %v, want the lost nested key", got)
	}
	// A body with a key the type does not model loses that key on marshal, and the guard says so.
	body := []byte(`{"id":"x","status":"ready","brand_new_key":1}`)
	if got := unexpectedMissing(t, "x.json", reflect.TypeFor[CitationCheckDocument](), body); !reflect.DeepEqual(got, []string{".brand_new_key"}) {
		t.Errorf("got %v", got)
	}
}

// TestExtraGuardNeedsTheDecoder shows why TestEveryExtraStructDecodesItself exists: a struct
// with an Extra map and no UnmarshalJSON decodes without error and loses the unknown key.
func TestExtraGuardNeedsTheDecoder(t *testing.T) {
	type naive struct {
		Known string                     `json:"known"`
		Extra map[string]json.RawMessage `json:"-"`
	}
	var n naive
	if err := json.Unmarshal([]byte(`{"known":"a","unknown":1}`), &n); err != nil {
		t.Fatal(err)
	}
	if len(n.Extra) != 0 {
		t.Fatal("the plain decoder is not supposed to fill Extra")
	}
	if reflect.PointerTo(reflect.TypeFor[naive]()).Implements(reflect.TypeFor[json.Unmarshaler]()) {
		t.Fatal("naive must not implement json.Unmarshaler")
	}
}
