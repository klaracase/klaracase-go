package klaracase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"
)

// InstanceMix counts decisions by level of court.
type InstanceMix struct {
	// Federal counts decisions of federal courts.
	Federal *CountValue `json:"federal"`
	// Appellate counts decisions of appellate courts.
	Appellate *CountValue `json:"appellate"`
	// FirstInstanceAndRegional counts decisions of first instance and regional courts.
	FirstInstanceAndRegional *CountValue `json:"first_instance_and_regional"`
	// Other counts the rest.
	Other *CountValue `json:"other"`
	// Note explains the mix in words.
	Note string `json:"note"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CoverageCourt is the coverage of one court.
type CoverageCourt struct {
	// Court is the court short form.
	Court string `json:"court"`
	// Branch is the court branch. It is empty in the top_courts list of a branch.
	Branch Branch `json:"branch"`
	// Count is the number of decisions.
	Count *CountValue `json:"count"`
	// Earliest is the month of the earliest decision.
	Earliest YearMonth `json:"earliest"`
	// Latest is the month of the latest decision.
	Latest YearMonth `json:"latest"`
	// CourtFoundingYear is the year the court was founded, or nil when the row has none (the
	// top_courts rows of a branch carry no founding year).
	CourtFoundingYear *int `json:"court_founding_year"`
	// CourtFoundingBasis says where the founding year comes from.
	CourtFoundingBasis string `json:"court_founding_basis"`
	// HasDatesBelowFloor is true when some decisions carry a date before the founding year. It
	// is nil when the row does not say.
	HasDatesBelowFloor *bool `json:"has_dates_below_floor"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CoverageBranch is the coverage of one court branch.
type CoverageBranch struct {
	// Branch is the branch slug, for example "ordentlich".
	Branch Branch `json:"branch"`
	// Count is the number of decisions.
	Count *CountValue `json:"count"`
	// Earliest is the month of the earliest decision.
	Earliest YearMonth `json:"earliest"`
	// Latest is the month of the latest decision.
	Latest YearMonth `json:"latest"`
	// CourtFoundingYear is the founding year used as the floor of the branch, or nil.
	CourtFoundingYear *int `json:"court_founding_year"`
	// CourtFoundingBasis says where the founding year comes from.
	CourtFoundingBasis string `json:"court_founding_basis"`
	// HasDatesBelowFloor is true when some decisions carry a date before the floor. It is nil
	// when the row does not say.
	HasDatesBelowFloor *bool `json:"has_dates_below_floor"`
	// InstanceMix counts decisions by level of court.
	InstanceMix *InstanceMix `json:"instance_mix"`
	// TopCourts lists the busiest courts of the branch.
	TopCourts []CoverageCourt `json:"top_courts"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CoverageRange is one ingestion range. An out_of_coverage verdict is decided against these.
type CoverageRange struct {
	// Bucket is the name of the range, for example "BGH_STRAFRECHT".
	Bucket string `json:"bucket"`
	// Court is the court short form.
	Court string `json:"court"`
	// Label is a human readable name.
	Label string `json:"label"`
	// From is the first date covered.
	From Date `json:"from"`
	// Until is the last date covered, or nil when the range is open.
	Until *Date `json:"until"`
	// Sources lists the data sources of the range.
	Sources []string `json:"sources"`
	// Exhaustive is true when the range holds every decision.
	Exhaustive bool `json:"exhaustive"`
	// OCR is true when scanned decisions were read with text recognition.
	OCR bool `json:"ocr"`
	// Note explains the range in words.
	Note string `json:"note"`
	// HoldingsEarliest is the earliest decision actually held, or nil.
	HoldingsEarliest *Date `json:"holdings_earliest"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CoverageRanges is the list of ingestion ranges.
type CoverageRanges struct {
	// Note explains the ranges in words.
	Note string `json:"note"`
	// Scope says what the list covers, for example "all".
	Scope string `json:"scope"`
	// VerifiedAt is the date the ranges were last verified.
	VerifiedAt Date `json:"verified_at"`
	// Ranges are the ranges.
	Ranges []CoverageRange `json:"ranges"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// LawCoverage lists the statutes behind the citation check.
type LawCoverage struct {
	// Note explains the coverage in words.
	Note string `json:"note"`
	// EU lists the covered EU instruments.
	EU []string `json:"eu"`
	// International lists the covered international instruments.
	International []string `json:"international"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CoverageResponse is the answer of Corpus.Coverage. It is free.
type CoverageResponse struct {
	Envelope
	// TotalDecisions is the number of decisions in the corpus.
	TotalDecisions *CountValue `json:"total_decisions"`
	// Branches is the coverage per court branch.
	Branches []CoverageBranch `json:"branches"`
	// Courts is the coverage per court, busiest first.
	Courts []CoverageCourt `json:"courts"`
	// CourtsReturned is the number of courts in Courts.
	CourtsReturned int `json:"courts_returned"`
	// CourtsTotal is the number of courts in the corpus.
	CourtsTotal int `json:"courts_total"`
	// CorpusFloors states the earliest decision held per branch.
	CorpusFloors string `json:"corpus_floors"`
	// CoverageRanges lists the ingestion ranges.
	CoverageRanges *CoverageRanges `json:"coverage_ranges"`
	// AsOf is the time of the report. The server writes it with the offset +00:00; it is parsed
	// to the same instant.
	AsOf time.Time `json:"as_of"`
	// FilterCoverageCaveats states known blind spots of the filters.
	FilterCoverageCaveats string `json:"filter_coverage_caveats"`
	// LawCoverage lists the statutes behind the citation check.
	LawCoverage *LawCoverage `json:"law_coverage"`
}

// FacetsResponse is the answer of Corpus.Facets. It is free.
type FacetsResponse struct {
	Envelope
	// Field is the field that was counted.
	Field FacetField `json:"field"`
	// AppliesTo says which artifact type the counts cover, for example "DECISION".
	AppliesTo ArtifactType `json:"applies_to"`
	// Norm is the canonical norm for a cited_norms facet, otherwise empty.
	Norm string `json:"norm"`
	// Values holds each facet value with its count, in the order of the server. Use
	// Values.Lookup to find one value by name.
	Values FacetValues `json:"values"`
	// TotalValues is the number of distinct values.
	TotalValues int `json:"total_values"`
	// Truncated is true when more values exist than were returned.
	Truncated bool `json:"truncated"`
	// ValueLimit is the largest number of values the server returns.
	ValueLimit int `json:"value_limit"`
	// CountBasis says what the counts count, for example "indexed_passages".
	CountBasis string `json:"count_basis"`
	// AsOf is the time of the report. The server writes it with the offset +00:00; it is parsed
	// to the same instant.
	AsOf time.Time `json:"as_of"`
}

// FacetValue is one value of a facet with its count.
type FacetValue struct {
	// Key is the facet value, for example a court name.
	Key string
	// Count is the count of the value.
	Count CountValue
}

// FacetValues is the values object of a facet answer. A Go map has no order, and the server
// sorts the values, so this type keeps them in the order they arrived. It marshals back to a
// JSON object in the same order.
type FacetValues []FacetValue

// Lookup returns the count of one facet value and whether the value is present.
func (v FacetValues) Lookup(key string) (CountValue, bool) {
	for _, item := range v {
		if item.Key == key {
			return item.Count, true
		}
	}
	return CountValue{}, false
}

// UnmarshalJSON implements json.Unmarshaler. It reads the object token by token so that the
// order of the keys survives.
func (v *FacetValues) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	first, err := dec.Token()
	if err != nil {
		return err
	}
	if first == nil {
		*v = nil
		return nil
	}
	if delim, ok := first.(json.Delim); !ok || delim != '{' {
		return fmt.Errorf("klaracase: facet values must be a JSON object, got %v", first)
	}
	out := FacetValues{}
	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("klaracase: facet value key is not a string: %v", keyToken)
		}
		var count CountValue
		if err := dec.Decode(&count); err != nil {
			return fmt.Errorf("klaracase: facet value %q: %w", key, err)
		}
		out = append(out, FacetValue{Key: key, Count: count})
	}
	if _, err := dec.Token(); err != nil {
		return err
	}
	*v = out
	return nil
}

// MarshalJSON implements json.Marshaler. It writes a JSON object in the stored order.
func (v FacetValues) MarshalJSON() ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, item := range v {
		if i > 0 {
			buf.WriteByte(',')
		}
		key, err := json.Marshal(item.Key)
		if err != nil {
			return nil, err
		}
		value, err := json.Marshal(item.Count)
		if err != nil {
			return nil, err
		}
		buf.Write(key)
		buf.WriteByte(':')
		buf.Write(value)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *InstanceMix) UnmarshalJSON(b []byte) error {
	type plain InstanceMix
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CoverageCourt) UnmarshalJSON(b []byte) error {
	type plain CoverageCourt
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CoverageBranch) UnmarshalJSON(b []byte) error {
	type plain CoverageBranch
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CoverageRange) UnmarshalJSON(b []byte) error {
	type plain CoverageRange
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CoverageRanges) UnmarshalJSON(b []byte) error {
	type plain CoverageRanges
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *LawCoverage) UnmarshalJSON(b []byte) error {
	type plain LawCoverage
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CoverageResponse) UnmarshalJSON(b []byte) error {
	type plain CoverageResponse
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *FacetsResponse) UnmarshalJSON(b []byte) error {
	type plain FacetsResponse
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}
