package klaracase

import "encoding/json"

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
	Branch string `json:"branch"`
	// Count is the number of decisions.
	Count *CountValue `json:"count"`
	// Earliest is the month of the earliest decision, YYYY-MM.
	Earliest string `json:"earliest"`
	// Latest is the month of the latest decision, YYYY-MM.
	Latest string `json:"latest"`
	// CourtFoundingYear is the year the court was founded.
	CourtFoundingYear int `json:"court_founding_year"`
	// CourtFoundingBasis says where the founding year comes from.
	CourtFoundingBasis string `json:"court_founding_basis"`
	// HasDatesBelowFloor is true when some decisions carry a date before the founding year.
	HasDatesBelowFloor bool `json:"has_dates_below_floor"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CoverageBranch is the coverage of one court branch.
type CoverageBranch struct {
	// Branch is the branch slug, for example "ordentlich".
	Branch string `json:"branch"`
	// Count is the number of decisions.
	Count *CountValue `json:"count"`
	// Earliest is the month of the earliest decision, YYYY-MM.
	Earliest string `json:"earliest"`
	// Latest is the month of the latest decision, YYYY-MM.
	Latest string `json:"latest"`
	// CourtFoundingYear is the founding year used as the floor of the branch.
	CourtFoundingYear int `json:"court_founding_year"`
	// CourtFoundingBasis says where the founding year comes from.
	CourtFoundingBasis string `json:"court_founding_basis"`
	// HasDatesBelowFloor is true when some decisions carry a date before the floor.
	HasDatesBelowFloor bool `json:"has_dates_below_floor"`
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
	// From is the first date covered, YYYY-MM-DD.
	From string `json:"from"`
	// Until is the last date covered, or nil when the range is open.
	Until *string `json:"until"`
	// Sources lists the data sources of the range.
	Sources []string `json:"sources"`
	// Exhaustive is true when the range holds every decision.
	Exhaustive bool `json:"exhaustive"`
	// OCR is true when scanned decisions were read with text recognition.
	OCR bool `json:"ocr"`
	// Note explains the range in words.
	Note string `json:"note"`
	// HoldingsEarliest is the earliest decision actually held, or nil.
	HoldingsEarliest *string `json:"holdings_earliest"`
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
	VerifiedAt string `json:"verified_at"`
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
	// AsOf is the time of the report, RFC 3339.
	AsOf string `json:"as_of"`
	// FilterCoverageCaveats states known blind spots of the filters.
	FilterCoverageCaveats string `json:"filter_coverage_caveats"`
	// LawCoverage lists the statutes behind the citation check.
	LawCoverage *LawCoverage `json:"law_coverage"`
}

// FacetsResponse is the answer of Corpus.Facets. It is free.
type FacetsResponse struct {
	Envelope
	// Field is the field that was counted.
	Field string `json:"field"`
	// AppliesTo says which artifact type the counts cover, for example "DECISION".
	AppliesTo string `json:"applies_to"`
	// Norm is the canonical norm for a cited_norms facet, otherwise empty.
	Norm string `json:"norm"`
	// Values maps each facet value to its count. A Go map has no order; use Raw for the order of the server.
	Values map[string]CountValue `json:"values"`
	// TotalValues is the number of distinct values.
	TotalValues int `json:"total_values"`
	// Truncated is true when more values exist than were returned.
	Truncated bool `json:"truncated"`
	// ValueLimit is the largest number of values the server returns.
	ValueLimit int `json:"value_limit"`
	// CountBasis says what the counts count, for example "indexed_passages".
	CountBasis string `json:"count_basis"`
	// AsOf is the time of the report, RFC 3339.
	AsOf string `json:"as_of"`
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
