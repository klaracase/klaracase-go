package klaracase

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestClientSideValidationTable(t *testing.T) {
	var calls atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1) })
	ctx := context.Background()
	ten := make([]string, 10)
	eleven := make([]string, 11)
	for i := range eleven {
		eleven[i] = "id"
		if i < 10 {
			ten[i] = "id"
		}
	}
	cases := []struct {
		name  string
		call  func() error
		field string
	}{
		{"decisions search without query", func() error { _, err := client.Decisions.Search(ctx, SearchDecisionsParams{}); return err }, "query"},
		{"decisions search date_from not a date", func() error {
			_, err := client.Decisions.Search(ctx, SearchDecisionsParams{Query: "q", DateFrom: "01.02.2020"})
			return err
		}, "date_from"},
		{"decisions search date_to with one digit month", func() error {
			_, err := client.Decisions.Search(ctx, SearchDecisionsParams{Query: "q", DateTo: "2020-1-05"})
			return err
		}, "date_to"},
		{"decisions search impossible day", func() error {
			_, err := client.Decisions.Search(ctx, SearchDecisionsParams{Query: "q", DateTo: "2021-02-30"})
			return err
		}, "date_to"},
		{"decisions search inverted dates", func() error {
			_, err := client.Decisions.Search(ctx, SearchDecisionsParams{Query: "q", DateFrom: "2021-01-02", DateTo: "2021-01-01"})
			return err
		}, "date_from"},
		{"cross type search inverted dates", func() error {
			_, err := client.Search(ctx, CrossTypeSearchParams{Query: "q", DateFrom: "2022-01-01", DateTo: "2021-01-01"})
			return err
		}, "date_from"},
		{"cross type search without query", func() error { _, err := client.Search(ctx, CrossTypeSearchParams{}); return err }, "query"},
		{"get decision without id", func() error { _, err := client.Decisions.Get(ctx, "", GetDecisionParams{}); return err }, "id"},
		{"get decision bad date", func() error {
			_, err := client.Decisions.Get(ctx, "x", GetDecisionParams{Date: "yesterday"})
			return err
		}, "date"},
		{"get decision rn range backwards", func() error {
			_, err := client.Decisions.Get(ctx, "x", GetDecisionParams{RnFrom: 9, RnTo: 3})
			return err
		}, "rn_from"},
		{"citations without id", func() error { _, err := client.Decisions.Citations(ctx, "", CitationsParams{}); return err }, "id"},
		{"citations bad date", func() error {
			_, err := client.Decisions.Citations(ctx, "x", CitationsParams{Date: "2020-13-01"})
			return err
		}, "date"},
		{"norm without reference", func() error { _, err := client.Norms.Get(ctx, "", GetNormParams{}); return err }, "reference"},
		{"datapackage without ids", func() error { _, err := client.Datapackages.Create(ctx, CreateDatapackageParams{}); return err }, "ids"},
		{"datapackage with an empty id", func() error {
			_, err := client.Datapackages.Create(ctx, CreateDatapackageParams{IDs: []string{"a", "  "}})
			return err
		}, "ids"},
		{"datapackage with 11 ids", func() error {
			_, err := client.Datapackages.Create(ctx, CreateDatapackageParams{IDs: eleven})
			return err
		}, "ids"},
		{"citation check with both", func() error {
			_, err := client.CitationChecks.Create(ctx, CreateCitationCheckParams{Text: "t", DocumentID: "d"})
			return err
		}, "text"},
		{"citation check with neither", func() error {
			_, err := client.CitationChecks.Create(ctx, CreateCitationCheckParams{})
			return err
		}, "text"},
		{"citation check bad reference date", func() error {
			_, err := client.CitationChecks.Create(ctx, CreateCitationCheckParams{Text: "t", ReferenceDate: "2026-10"})
			return err
		}, "reference_date"},
		{"facets without field", func() error { _, err := client.Corpus.Facets(ctx, "", FacetsParams{}); return err }, "field"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.call()
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("got %T %v, want *ValidationError", err, err)
			}
			if ve.Code != "client_validation" || ve.Status != 0 {
				t.Errorf("code %q status %d", ve.Code, ve.Status)
			}
			if len(ve.Errors[c.field]) == 0 {
				t.Errorf("no message for field %q in %v", c.field, ve.Errors)
			}
		})
	}
	if n := calls.Load(); n != 0 {
		t.Errorf("the server saw %d requests, want 0", n)
	}

	// Ten ids and a date range that is equal at both ends are valid and reach the server.
	if err := validateDatapackage(CreateDatapackageParams{IDs: ten}); err != nil {
		t.Errorf("10 ids refused: %v", err)
	}
	if err := validateSearchDecisions(SearchDecisionsParams{Query: "q", DateFrom: "2020-02-29", DateTo: "2020-02-29"}); err != nil {
		t.Errorf("equal dates refused: %v", err)
	}
}

func TestServerOwnedRulesAreNotCheckedOnTheClient(t *testing.T) {
	var calls atomic.Int32
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte(`{}`))
	})
	ctx := context.Background()
	long := strings.Repeat("a", 600)
	_, err := client.Decisions.Search(ctx, SearchDecisionsParams{
		Query: long, Branch: "not-a-branch", Mode: "psychic", Limit: 9999, Page: -3,
		CitedStatutes: []Statute{"NotAStatute"},
	})
	if err != nil {
		t.Fatalf("the client refused a value that the server owns: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls %d, want 1", calls.Load())
	}
}

func TestParamsMarshalAsAgentToolInput(t *testing.T) {
	p := SearchDecisionsParams{
		Query: "Untreue", Court: "BGH", CitedStatutes: []Statute{StatuteStGB}, Limit: 5, Bridge: Bool(false),
		DateFrom: "2020-01-01",
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"query":"Untreue","court":"BGH","cited_statutes":["StGB"],"date_from":"2020-01-01","limit":5,"bridge":false}`
	if string(data) != want {
		t.Errorf("got  %s\nwant %s", data, want)
	}
	var back SearchDecisionsParams
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back.Query != p.Query || back.Limit != 5 || back.Bridge == nil || *back.Bridge || back.DateFrom != "2020-01-01" {
		t.Errorf("round trip %+v", back)
	}
	empty, _ := json.Marshal(CoverageParams{})
	if string(empty) != "{}" {
		t.Errorf("an unset params struct marshals to %s, want {}", empty)
	}
}

func TestDateAndYearMonth(t *testing.T) {
	d := NewDate(time.Date(2026, 11, 1, 23, 59, 0, 0, time.UTC))
	if d != "2026-11-01" {
		t.Errorf("NewDate = %q", d)
	}
	got, err := Date("2024-02-29").Time()
	if err != nil || got.Year() != 2024 || got.Month() != time.February || got.Day() != 29 || got.Location() != time.UTC {
		t.Errorf("Time() = %v, %v", got, err)
	}
	for _, bad := range []Date{"", "2024-2-29", "2023-02-29", "2024-02-29T00:00:00Z", "29.02.2024"} {
		if _, err := bad.Time(); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
	m, err := YearMonth("1950-10").Time()
	if err != nil || m.Month() != time.October || m.Day() != 1 {
		t.Errorf("YearMonth = %v, %v", m, err)
	}
	if _, err := YearMonth("1950-10-01").Time(); err == nil {
		t.Error("a full date parsed as a month")
	}
}

func TestPtrHelpers(t *testing.T) {
	if *Ptr(7) != 7 || *Ptr("x") != "x" || *Int(3) != 3 || *String("s") != "s" || *Bool(false) {
		t.Error("pointer helpers return the wrong value")
	}
	a, b := Ptr(1), Ptr(1)
	if a == b {
		t.Error("each call must return a new pointer")
	}
}
