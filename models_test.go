package klaracase

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func decodeExample[T any](t *testing.T, name string) *T {
	t.Helper()
	out := new(T)
	if err := json.Unmarshal(fixture(t, "examples/"+name+".json"), out); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return out
}

func TestCrossTypeSearchSplitsLawAndDecisionRows(t *testing.T) {
	res := decodeExample[CrossTypeSearchResponse](t, "search")
	law := res.Sections.Laws[0]
	if law.ArtifactType != ArtifactTypeLaw || law.Abbreviation != "StGB" || law.Jurisdiction != JurisdictionDE || law.Confidence != ConfidenceHigh {
		t.Errorf("law row %+v", law)
	}
	for _, key := range []string{"aktenzeichen", "court", "date", "document_type"} {
		if string(law.Extra[key]) != "null" {
			t.Errorf("the always null key %q should stay in Extra, got %v", key, law.Extra)
		}
	}
	dec := res.Sections.Decisions[0]
	if dec.ArtifactType != ArtifactTypeDecision || dec.Date != "2001-11-15" || dec.DocumentType != DocumentTypeUrteil || dec.Confidence != ConfidenceMedium {
		t.Errorf("decision row %+v", dec)
	}
	if res.Mode != ModeHybrid || res.TopConfidence != ConfidenceHigh {
		t.Errorf("mode %q top confidence %q", res.Mode, res.TopConfidence)
	}
	if got, ok := res.CreditsRemaining(); !ok || got != 28 {
		t.Errorf("CreditsRemaining = %d, %v", got, ok)
	}
	if got, ok := res.Cost(); !ok || got != 1 {
		t.Errorf("Cost = %d, %v", got, ok)
	}
}

func TestCreditAccessorsReportAnAbsentBalance(t *testing.T) {
	var e Envelope
	if _, ok := e.CreditsRemaining(); ok {
		t.Error("no credits key must report false")
	}
	if _, ok := e.Cost(); ok {
		t.Error("no credits key must report false")
	}
	res := decodeExample[CoverageResponse](t, "../conformance/bodies/corpus.coverage.not_metered")
	if _, ok := res.CreditsRemaining(); ok {
		t.Error("a call that is not metered has a null balance")
	}
	if _, ok := res.Cost(); ok {
		t.Error("a call that is not metered has a null cost")
	}
	if res.Credits == nil || res.Credits.Metered {
		t.Errorf("credits %+v", res.Credits)
	}
	free := decodeExample[CoverageResponse](t, "corpus.coverage")
	if cost, ok := free.Cost(); !ok || cost != 0 {
		t.Errorf("a free call costs 0, got %d, %v", cost, ok)
	}
}

func TestFacetValuesKeepTheServerOrder(t *testing.T) {
	body := `{"field":"court_name","values":{"b":{"value":2,"is_floor":false},"a":{"value":1,"is_floor":true},"§ 543 Abs. 1 BGB":{"value":9,"is_floor":false}}}`
	var res FacetsResponse
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Values) != 3 || res.Values[0].Key != "b" || res.Values[1].Key != "a" || res.Values[2].Key != "§ 543 Abs. 1 BGB" {
		t.Fatalf("values %+v", res.Values)
	}
	if c, ok := res.Values.Lookup("a"); !ok || c.Value != 1 || !c.IsFloor {
		t.Errorf("Lookup(a) = %+v, %v", c, ok)
	}
	if _, ok := res.Values.Lookup("missing"); ok {
		t.Error("Lookup found a missing key")
	}
	out, err := json.Marshal(res.Values)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"b":{"value":2,"is_floor":false},"a":{"value":1,"is_floor":true},"§ 543 Abs. 1 BGB":{"value":9,"is_floor":false}}`
	if string(out) != want {
		t.Errorf("marshal\n got %s\nwant %s", out, want)
	}
	var empty FacetValues
	if err := json.Unmarshal([]byte("null"), &empty); err != nil || empty != nil {
		t.Errorf("null: %v %v", empty, err)
	}
	if err := json.Unmarshal([]byte("[]"), &empty); err == nil {
		t.Error("an array is not a facet object")
	}
	if err := json.Unmarshal([]byte(`{"a":"x"}`), &empty); err == nil {
		t.Error("a bad count must fail")
	}
}

func TestFindingCoverageNullIsNilAndExcerptCutsByRunes(t *testing.T) {
	run := decodeExample[CitationCheckRun](t, "citation-checks.store")
	f := run.Findings[0]
	if f.Coverage != nil || f.CoverageBoundary != nil {
		t.Errorf("null must read as nil, got %q %q", f.Coverage, f.CoverageBoundary)
	}
	if f.Type != ReferenceTypeCase || f.ExternalStatus != ExternalStatusSkipped || f.SubdivisionCheck != SubdivisionNotChecked ||
		f.Evidence.MatchStrength != MatchStrengthExact || f.MatchedDecision.Date != "2001-11-15" {
		t.Errorf("finding %+v", f)
	}
	if run.UniqueCitations[0].KeyBasis != KeyBasisMatchedDecision || run.UniqueCitations[0].ReferenceType != ReferenceTypeCase {
		t.Errorf("unique citation %+v", run.UniqueCitations[0])
	}
	if got := run.VerdictCounts[VerdictVerifiedCorpus]; got != 1 {
		t.Errorf("verdict counts %v", run.VerdictCounts)
	}

	// Offsets count code points. The text starts with 4 letters that take 2 bytes each.
	text := "ääää " + f.RawText + " Ende"
	g := Finding{StartOffset: 5, EndOffset: 5 + len([]rune(f.RawText))}
	if got := g.Excerpt(text); got != f.RawText {
		t.Errorf("Excerpt = %q, want %q", got, f.RawText)
	}
	if text[5:5+len(f.RawText)] == f.RawText {
		t.Fatal("the test text does not tell bytes from runes")
	}
	for _, bad := range []Finding{{StartOffset: -1, EndOffset: 2}, {StartOffset: 3, EndOffset: 2}, {StartOffset: 0, EndOffset: 999}} {
		if bad.Excerpt(text) != "" {
			t.Errorf("offsets %d-%d should give an empty excerpt", bad.StartOffset, bad.EndOffset)
		}
	}
}

func TestTimestampsAreParsed(t *testing.T) {
	run := decodeExample[CitationCheckRun](t, "citation-checks.store")
	want := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
	if !run.SubmittedAt.Equal(want) || run.CompletedAt == nil || !run.CompletedAt.Equal(want) || !run.CreatedAt.Equal(want) {
		t.Errorf("times %v %v %v", run.SubmittedAt, run.CompletedAt, run.CreatedAt)
	}
	if run.FindingsPurgedAt != nil {
		t.Error("a null time must be nil")
	}
	doc := decodeExample[CitationCheckDocument](t, "citation-check-documents.store")
	if !doc.ExpiresAt.Equal(time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)) || doc.CharCount != nil {
		t.Errorf("document %+v", doc)
	}
	norm := decodeExample[NormResponse](t, "norms.show")
	if !norm.Law.UpstreamLastSeenAt.Equal(time.Date(2026, 10, 5, 4, 17, 0, 0, time.UTC)) || norm.Version.LastAmendmentDate != "2024-12-20" {
		t.Errorf("norm %+v %+v", norm.Law, norm.Version)
	}
	if _, err := norm.Version.LastAmendmentDate.Time(); err != nil {
		t.Error(err)
	}
	var bad CitationCheckDocument
	if err := json.Unmarshal([]byte(`{"expires_at":"tomorrow"}`), &bad); err == nil {
		t.Error("an unparsable time must fail, not turn into the zero time")
	}
}

func TestCoveragePointerFieldsTellMissingFromZero(t *testing.T) {
	res := decodeExample[CoverageResponse](t, "corpus.coverage")
	branch := res.Branches[0]
	if branch.CourtFoundingYear == nil || *branch.CourtFoundingYear != 1950 || branch.HasDatesBelowFloor == nil || *branch.HasDatesBelowFloor {
		t.Errorf("branch %+v", branch)
	}
	top := branch.TopCourts[0]
	if top.CourtFoundingYear != nil || top.HasDatesBelowFloor != nil || top.Branch != "" {
		t.Errorf("a top_courts row has no founding year and no floor flag, got %+v", top)
	}
	if res.Courts[0].Branch != BranchOrdentlich || res.Courts[0].Earliest != "1950-10" {
		t.Errorf("court %+v", res.Courts[0])
	}
	if got, err := res.Courts[0].Earliest.Time(); err != nil || got.Month() != time.October {
		t.Errorf("Earliest.Time = %v, %v", got, err)
	}
	if res.CoverageRanges.Ranges[0].Until != nil || res.CoverageRanges.VerifiedAt != "2026-08-13" || res.CoverageRanges.Ranges[0].From != "1950-10-02" {
		t.Errorf("ranges %+v", res.CoverageRanges)
	}
}

func TestCitationEdgeHopIsOmittedWhenZero(t *testing.T) {
	pkg := decodeExample[DatapackageResponse](t, "datapackages.store")
	out, err := json.Marshal(pkg.CitationEdges[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), `"hop"`) {
		t.Errorf("an edge without a hop must not marshal a hop: %s", out)
	}
	if pkg.CitationEdges[0].ResolutionStatus != ResolutionResolved || pkg.TextMode != TextModeExcerpt {
		t.Errorf("package %+v", pkg.CitationEdges[0])
	}
	graph := decodeExample[CitationsResponse](t, "decisions.citations")
	if graph.References[0].Hop != 1 || graph.ServedDirection != DirectionBoth {
		t.Errorf("graph %+v", graph.References[0])
	}
}

func TestDecisionKeysFromTheContractProse(t *testing.T) {
	body := `{"id":"x","returned_range":{"rn_from":20,"rn_to":30,"rn_source":"printed"},"date_source":{"basis":"ocr"},"date_confidence":0.4}`
	var res DecisionResponse
	if err := json.Unmarshal([]byte(body), &res); err != nil {
		t.Fatal(err)
	}
	if res.ReturnedRange == nil || res.ReturnedRange.RnFrom != 20 || res.ReturnedRange.RnSource != RnSourcePrinted {
		t.Errorf("returned range %+v", res.ReturnedRange)
	}
	if string(res.DateSource) != `{"basis":"ocr"}` || string(res.DateConfidence) != "0.4" {
		t.Errorf("date keys %s %s", res.DateSource, res.DateConfidence)
	}
	if len(res.Extra) != 0 {
		t.Errorf("these keys are modeled, extra = %v", res.Extra)
	}
	var cross CrossTypeSearchResponse
	if err := json.Unmarshal([]byte(`{"filters_applied_sections":{"jurisdiction":["laws"]}}`), &cross); err != nil {
		t.Fatal(err)
	}
	if string(cross.FiltersAppliedSections) != `{"jurisdiction":["laws"]}` || len(cross.Extra) != 0 {
		t.Errorf("filters_applied_sections %s extra %v", cross.FiltersAppliedSections, cross.Extra)
	}
}
