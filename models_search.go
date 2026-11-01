package klaracase

import "encoding/json"

// CountValue is a count that may be a lower bound.
type CountValue struct {
	// Value is the count.
	Value int `json:"value"`
	// IsFloor is true when the real count may be higher.
	IsFloor bool `json:"is_floor"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// Snippet is one passage of a hit, with the matching terms wrapped in <mark> tags.
type Snippet struct {
	// Text is the passage.
	Text string `json:"text"`
	// Rn is the Randnummer (margin number) of the passage, when the decision has one.
	Rn *int `json:"rn"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// QueryExpansion says whether the server added register terms to the query.
type QueryExpansion struct {
	// Status is the outcome, for example "abstained" or "applied".
	Status string `json:"status"`
	// Applied is true when terms were added.
	Applied bool `json:"applied"`
	// Reason says why the server did or did not expand the query.
	Reason string `json:"reason"`
	// Note explains the outcome in words.
	Note string `json:"note"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// DecisionHit is one row of a decision search.
type DecisionHit struct {
	// ID is the UUID of the decision.
	ID string `json:"id"`
	// Aktenzeichen is the file number.
	Aktenzeichen string `json:"aktenzeichen"`
	// Court is the court short form.
	Court string `json:"court"`
	// ArtifactType is "DECISION" in a cross type search and empty in a decision search.
	ArtifactType ArtifactType `json:"artifact_type,omitzero"`
	// Date is the decision date.
	Date Date `json:"date"`
	// DocumentType is the document type, for example "Urteil".
	DocumentType DocumentType `json:"document_type"`
	// Title is the title of the decision.
	Title string `json:"title"`
	// Snippet is the best passage, with <mark> tags.
	Snippet string `json:"snippet"`
	// Snippets lists the matching passages.
	Snippets []Snippet `json:"snippets"`
	// ChunkTitle names the section of the best passage, for example "Gründe".
	ChunkTitle *string `json:"chunk_title"`
	// RnRange is the range of margin numbers of the best passage, for example "21-23".
	RnRange *string `json:"rn_range"`
	// Fundstelle is the printed source of the decision, when known.
	Fundstelle *string `json:"fundstelle"`
	// FortgeltungChecked is always false today: the API does not check whether a decision is still good law.
	FortgeltungChecked bool `json:"fortgeltung_checked"`
	// FortgeltungNote warns that the decision may have been quashed or superseded. Show it.
	FortgeltungNote string `json:"fortgeltung_note"`
	// DateSource says where the stored date comes from. The server sends it for a scanned
	// decision whose date may be wrong. The value is kept raw because its shape is not fixed.
	DateSource json.RawMessage `json:"date_source,omitempty"`
	// DateConfidence says how far the stored date can be trusted. It is kept raw like DateSource.
	DateConfidence json.RawMessage `json:"date_confidence,omitempty"`
	// Confidence is the confidence band of the hit.
	Confidence Confidence `json:"confidence"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// DecisionSearchResponse is the answer of Decisions.Search.
type DecisionSearchResponse struct {
	Envelope
	// Results are the hits, best first.
	Results []DecisionHit `json:"results"`
	// PayloadTruncated is true when the server cut the payload.
	PayloadTruncated bool `json:"payload_truncated"`
	// TotalAvailable is the number of matching passages.
	TotalAvailable *CountValue `json:"total_available"`
	// TotalAvailableBasis says what TotalAvailable counts.
	TotalAvailableBasis string `json:"total_available_basis"`
	// CandidatePool is the size of the candidate pool.
	CandidatePool int `json:"candidate_pool"`
	// PoolExhaustive is true when the pool holds every match.
	PoolExhaustive bool `json:"pool_exhaustive"`
	// Truncated is true when more results exist than were returned.
	Truncated bool `json:"truncated"`
	// Page is the page that was served.
	Page int `json:"page"`
	// Limit is the limit that was applied.
	Limit int `json:"limit"`
	// Mode is the retrieval mode that was used.
	Mode Mode `json:"mode"`
	// FrequentlyCitedByResults lists decisions that many hits cite. The rows are kept raw.
	FrequentlyCitedByResults []json.RawMessage `json:"frequently_cited_by_results"`
	// QueryExpansion reports the query expansion outcome.
	QueryExpansion *QueryExpansion `json:"query_expansion"`
	// TopConfidence is the confidence band of the best hit. It is empty when not measured.
	TopConfidence Confidence `json:"top_confidence"`
}

// LawHit is one statute row of a cross type search. A statute row carries no Aktenzeichen,
// court, date or document type. The server sends those four keys as null, and they stay in
// Extra.
type LawHit struct {
	// ID is the UUID of the statute section.
	ID string `json:"id"`
	// ArtifactType is "LAW".
	ArtifactType ArtifactType `json:"artifact_type"`
	// Title is the title of the hit.
	Title string `json:"title"`
	// Snippet is the best passage, with <mark> tags.
	Snippet string `json:"snippet"`
	// Snippets lists the matching passages.
	Snippets []Snippet `json:"snippets"`
	// ChunkTitle names the section of the best passage, or nil.
	ChunkTitle *string `json:"chunk_title"`
	// RnRange is nil for a statute row.
	RnRange *string `json:"rn_range"`
	// Fundstelle is nil for a statute row.
	Fundstelle *string `json:"fundstelle"`
	// Breadcrumb is the position of the statute section, or nil.
	Breadcrumb *string `json:"breadcrumb"`
	// Abbreviation is the statute abbreviation, for example "StGB".
	Abbreviation string `json:"abbreviation"`
	// LawType is the kind of statute, for example "Gesetz".
	LawType string `json:"law_type"`
	// Jurisdiction is DE, EU or XI.
	Jurisdiction Jurisdiction `json:"jurisdiction"`
	// Confidence is the confidence band of the hit.
	Confidence Confidence `json:"confidence"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// SearchSections holds the two result lists of a cross type search.
type SearchSections struct {
	// Laws are statute hits.
	Laws []LawHit `json:"laws"`
	// Decisions are decision hits.
	Decisions []DecisionHit `json:"decisions"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// SectionTotal is the accounting of one section of a cross type search.
type SectionTotal struct {
	// TotalAvailable is the number of matches in the section.
	TotalAvailable *CountValue `json:"total_available"`
	// TotalAvailableBasis says what TotalAvailable counts.
	TotalAvailableBasis string `json:"total_available_basis"`
	// CandidatePool is the size of the candidate pool.
	CandidatePool int `json:"candidate_pool"`
	// PoolExhaustive is true when the pool holds every match.
	PoolExhaustive bool `json:"pool_exhaustive"`
	// Truncated is true when more results exist than were returned.
	Truncated bool `json:"truncated"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// SectionTotals holds the accounting of both sections.
type SectionTotals struct {
	// Laws is the accounting of the statute section.
	Laws *SectionTotal `json:"laws"`
	// Decisions is the accounting of the decision section.
	Decisions *SectionTotal `json:"decisions"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CrossTypeSearchResponse is the answer of Client.Search.
type CrossTypeSearchResponse struct {
	Envelope
	// Sections holds the statute and decision hits.
	Sections *SearchSections `json:"sections"`
	// SectionTotals holds the accounting per section.
	SectionTotals *SectionTotals `json:"section_totals"`
	// Limit is the limit per section that was applied.
	Limit int `json:"limit"`
	// TotalAvailable is the number of matches over both sections.
	TotalAvailable *CountValue `json:"total_available"`
	// TotalAvailableBasis says what TotalAvailable counts.
	TotalAvailableBasis string `json:"total_available_basis"`
	// PoolExhaustive is true when the pools hold every match.
	PoolExhaustive bool `json:"pool_exhaustive"`
	// Truncated is true when more results exist than were returned.
	Truncated bool `json:"truncated"`
	// Mode is the retrieval mode that was used.
	Mode Mode `json:"mode"`
	// FiltersAppliedSections names, for each filter, the sections it ran on. A section that a
	// filter does not name was searched without that filter. It is kept raw.
	FiltersAppliedSections json.RawMessage `json:"filters_applied_sections,omitempty"`
	// QueryExpansion reports the query expansion outcome.
	QueryExpansion *QueryExpansion `json:"query_expansion"`
	// FrequentlyCitedByResults lists decisions that many hits cite. The rows are kept raw.
	FrequentlyCitedByResults []json.RawMessage `json:"frequently_cited_by_results"`
	// NormsCitedByDecisionResults lists norms that the decision hits cite. The rows are kept raw.
	NormsCitedByDecisionResults []json.RawMessage `json:"norms_cited_by_decision_results"`
	// TopConfidence is the confidence band of the best hit. It is empty when not measured.
	TopConfidence Confidence `json:"top_confidence"`
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CountValue) UnmarshalJSON(b []byte) error {
	type plain CountValue
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *Snippet) UnmarshalJSON(b []byte) error {
	type plain Snippet
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *QueryExpansion) UnmarshalJSON(b []byte) error {
	type plain QueryExpansion
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *DecisionHit) UnmarshalJSON(b []byte) error {
	type plain DecisionHit
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *DecisionSearchResponse) UnmarshalJSON(b []byte) error {
	type plain DecisionSearchResponse
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *LawHit) UnmarshalJSON(b []byte) error {
	type plain LawHit
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *SearchSections) UnmarshalJSON(b []byte) error {
	type plain SearchSections
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *SectionTotal) UnmarshalJSON(b []byte) error {
	type plain SectionTotal
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *SectionTotals) UnmarshalJSON(b []byte) error {
	type plain SectionTotals
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CrossTypeSearchResponse) UnmarshalJSON(b []byte) error {
	type plain CrossTypeSearchResponse
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}
