package klaracase

import (
	"encoding/json"
	"time"
)

// CheckProgress counts the citations of a run.
type CheckProgress struct {
	// CitationsTotal is the number of citations found.
	CitationsTotal int `json:"citations_total"`
	// CitationsChecked is the number of citations checked.
	CitationsChecked int `json:"citations_checked"`
	// UniqueCitationsTotal is the number of distinct citations.
	UniqueCitationsTotal int `json:"unique_citations_total"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// ReasonCount counts findings per reason category.
type ReasonCount struct {
	// Category is the reason, for example "statute_unknown".
	Category string `json:"category"`
	// Count is the number of findings with that reason.
	Count int `json:"count"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// UniqueCitation is one distinct citation of the text with its verdict.
type UniqueCitation struct {
	// KeyBasis says how the citation was keyed, for example "matched_decision" or "canonical".
	KeyBasis KeyBasis `json:"key_basis"`
	// TargetArtifactID is the UUID of the matched decision, or nil.
	TargetArtifactID *string `json:"target_artifact_id"`
	// Canonical is the canonical form, for example "BGH 1 StR 185/01".
	Canonical string `json:"canonical"`
	// Spellings lists the ways the citation appears in the text.
	Spellings []string `json:"spellings"`
	// Occurrences is the number of times the citation appears.
	Occurrences int `json:"occurrences"`
	// Verdict is the verdict of the citation.
	Verdict Verdict `json:"verdict"`
	// ReferenceType is "case" or "statute".
	ReferenceType ReferenceType `json:"reference_type"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CheckExtraction reports what the extraction found and dropped. The lists are kept raw.
type CheckExtraction struct {
	// Detected is the number of citations detected.
	Detected int `json:"detected"`
	// Dropped is the number of detections dropped.
	Dropped int `json:"dropped"`
	// DroppedUnreadable counts detections dropped as unreadable.
	DroppedUnreadable int `json:"dropped_unreadable"`
	// DroppedTruncation counts detections dropped by truncation.
	DroppedTruncation int `json:"dropped_truncation"`
	// DroppedTrailingNoun counts detections dropped for a trailing noun.
	DroppedTrailingNoun int `json:"dropped_trailing_noun"`
	// DroppedStackedSubdivision counts detections dropped for stacked subdivisions.
	DroppedStackedSubdivision int `json:"dropped_stacked_subdivision"`
	// DroppedRefusedCollision counts detections dropped after a refused collision.
	DroppedRefusedCollision int `json:"dropped_refused_collision"`
	// DroppedByToken lists drops by token. The rows are kept raw.
	DroppedByToken []json.RawMessage `json:"dropped_by_token"`
	// DroppedUnreadableTexts lists the unreadable texts. The rows are kept raw.
	DroppedUnreadableTexts []json.RawMessage `json:"dropped_unreadable_texts"`
	// RefusedContractCollision lists refused collisions. The rows are kept raw.
	RefusedContractCollision []json.RawMessage `json:"refused_contract_collision"`
	// NotVerifiable is the number of citations that cannot be verified.
	NotVerifiable int `json:"not_verifiable"`
	// NotVerifiableByType lists them by type. The rows are kept raw.
	NotVerifiableByType []json.RawMessage `json:"not_verifiable_by_type"`
	// PossibleCitationsUnparsed is the number of possible citations that were not parsed.
	PossibleCitationsUnparsed int `json:"possible_citations_unparsed"`
	// PossibleCitationsUnparsedTexts lists their texts. The rows are kept raw.
	PossibleCitationsUnparsedTexts []json.RawMessage `json:"possible_citations_unparsed_texts"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// MatchedDecision is the decision a finding matched.
type MatchedDecision struct {
	// ArtifactID is the UUID of the decision.
	ArtifactID string `json:"artifact_id"`
	// ECLI is the European Case Law Identifier, when known.
	ECLI *string `json:"ecli"`
	// Court is the court short form.
	Court string `json:"court"`
	// Date is the decision date.
	Date Date `json:"date"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CheckScope says what the check of a finding covered.
type CheckScope struct {
	// Subdivision says how the subdivision (Absatz, Satz) was handled.
	Subdivision string `json:"subdivision"`
	// ArticleTextAvailable is true when the article text was available, or nil when not applicable.
	ArticleTextAvailable *bool `json:"article_text_available"`
	// ArticleTextRead is true when the article text was read.
	ArticleTextRead bool `json:"article_text_read"`
	// Court says how the court was handled, for example "checked".
	Court string `json:"court"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// FindingEvidence explains a verdict.
type FindingEvidence struct {
	// Reason is the reason for the verdict, for example "exact_match".
	Reason string `json:"reason"`
	// MatchStrength is how strong the match is, for example "exact".
	MatchStrength MatchStrength `json:"match_strength"`
	// CitationDateSource says which date a statute citation used: "sentence", "reference_date" or "none".
	CitationDateSource CitationDateSource `json:"citation_date_source"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// Finding is one citation occurrence in the text, with its verdict and its place.
type Finding struct {
	// ID is the UUID of the finding.
	ID string `json:"id"`
	// RawText is the citation as it appears in the text.
	RawText string `json:"raw_text"`
	// Canonical is the canonical form.
	Canonical string `json:"canonical"`
	// Type is "case" or "statute".
	Type ReferenceType `json:"type"`
	// Role is the role of the citation, or nil.
	Role *string `json:"role"`
	// Verdict is the verdict. VerdictNotFound is absence of evidence in the corpus, nothing more.
	Verdict Verdict `json:"verdict"`
	// ExternalStatus says what the external register check did, for example "skipped".
	ExternalStatus ExternalStatus `json:"external_status"`
	// StartOffset is the start of the citation in your text, in Unicode code points.
	StartOffset int `json:"start_offset"`
	// EndOffset is the end of the citation in your text, in Unicode code points.
	EndOffset int `json:"end_offset"`
	// OccurrenceIndex is the position among the occurrences of the same citation.
	OccurrenceIndex int `json:"occurrence_index"`
	// Confidence is a confidence value, or nil.
	Confidence *float64 `json:"confidence"`
	// MatchedDecision is the matched decision, or nil.
	MatchedDecision *MatchedDecision `json:"matched_decision"`
	// SubdivisionCheck says whether a subdivision was checked.
	SubdivisionCheck SubdivisionCheck `json:"subdivision_check"`
	// SubdivisionAdvisory is a note about the subdivision, or nil.
	SubdivisionAdvisory *string `json:"subdivision_advisory"`
	// CheckScope says what the check covered.
	CheckScope *CheckScope `json:"check_scope"`
	// AmbiguityAdvisory is a note about an ambiguous citation, or nil.
	AmbiguityAdvisory *string `json:"ambiguity_advisory"`
	// Coverage is the coverage note of the finding. It is kept raw because its shape varies.
	// A JSON null is read as nil.
	Coverage json.RawMessage `json:"coverage"`
	// CoverageBoundary is the coverage boundary of the finding. It is kept raw. A JSON null is
	// read as nil.
	CoverageBoundary json.RawMessage `json:"coverage_boundary"`
	// CoverageAdvisory is a note about coverage, or nil.
	CoverageAdvisory *string `json:"coverage_advisory"`
	// Evidence explains the verdict.
	Evidence *FindingEvidence `json:"evidence"`
	// DisputedAt is the time the finding was disputed, or nil.
	DisputedAt *time.Time `json:"disputed_at"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CitationCheckRun is the answer of CitationChecks.Create and CitationChecks.Get: one run with
// one verdict per citation. CitationChecks.Create answers with the finished run.
type CitationCheckRun struct {
	Envelope
	// ID is the UUID of the run.
	ID string `json:"id"`
	// Status is the status of the run.
	Status RunStatus `json:"status"`
	// Source says where the run came from, "api_v1" for this API.
	Source string `json:"source"`
	// Progress counts the citations.
	Progress *CheckProgress `json:"progress"`
	// VerdictCounts counts the unique citations per verdict.
	VerdictCounts map[Verdict]int `json:"verdict_counts"`
	// Partial is true when the run covers only part of the text.
	Partial bool `json:"partial"`
	// UnverifiedCount is the number of citations that were not verified.
	UnverifiedCount int `json:"unverified_count"`
	// ReasonCounts counts findings per reason.
	ReasonCounts []ReasonCount `json:"reason_counts"`
	// UniqueCitations lists each distinct citation once.
	UniqueCitations []UniqueCitation `json:"unique_citations"`
	// Extraction reports what the extraction found and dropped.
	Extraction *CheckExtraction `json:"extraction"`
	// Notes are notes of the server. The rows are kept raw.
	Notes []json.RawMessage `json:"notes"`
	// FindingsUnavailableReason says why Findings is empty, or nil.
	FindingsUnavailableReason *string `json:"findings_unavailable_reason"`
	// FindingsPurgedAt is the time the findings were purged, or nil.
	FindingsPurgedAt *time.Time `json:"findings_purged_at"`
	// Findings lists every citation occurrence.
	Findings []Finding `json:"findings"`
	// SubmittedAt is the time the text was submitted. The server writes it with the offset
	// +00:00; it is parsed to the same instant.
	SubmittedAt time.Time `json:"submitted_at"`
	// CompletedAt is the time the run ended, or nil.
	CompletedAt *time.Time `json:"completed_at"`
	// CreatedAt is the time the run was created.
	CreatedAt time.Time `json:"created_at"`
	// PayloadTruncated is true when the server cut the payload.
	PayloadTruncated bool `json:"payload_truncated"`
}

// CitationCheckDocument is the receipt of an uploaded document. The receipt never holds the text.
type CitationCheckDocument struct {
	Envelope
	// ID is the UUID of the document. Pass it to CitationChecks.Create.
	ID string `json:"id"`
	// Filename is the name you sent.
	Filename string `json:"filename"`
	// Format is the detected format: pdf, docx, md or txt.
	Format string `json:"format"`
	// Status is the status of the document.
	Status DocumentStatus `json:"status"`
	// CharCount is the number of characters, or nil until the text is read.
	CharCount *int `json:"char_count"`
	// PageCount is the number of pages, or nil.
	PageCount *int `json:"page_count"`
	// PagesWithoutTextLayer is the number of pages that need text recognition, or nil.
	PagesWithoutTextLayer *int `json:"pages_without_text_layer"`
	// CheckMaxChars is the longest text a citation check accepts.
	CheckMaxChars int `json:"check_max_chars"`
	// QuotedCredits is the price of the check in credits, or nil above the limit or before the text is read.
	QuotedCredits *int `json:"quoted_credits"`
	// ErrorCode is why a failed document failed, or nil.
	ErrorCode *string `json:"error_code"`
	// ExpiresAt is the time an unused upload expires.
	ExpiresAt time.Time `json:"expires_at"`
	// CreatedAt is the time of the upload.
	CreatedAt time.Time `json:"created_at"`
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CheckProgress) UnmarshalJSON(b []byte) error {
	type plain CheckProgress
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *ReasonCount) UnmarshalJSON(b []byte) error {
	type plain ReasonCount
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *UniqueCitation) UnmarshalJSON(b []byte) error {
	type plain UniqueCitation
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CheckExtraction) UnmarshalJSON(b []byte) error {
	type plain CheckExtraction
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *MatchedDecision) UnmarshalJSON(b []byte) error {
	type plain MatchedDecision
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CheckScope) UnmarshalJSON(b []byte) error {
	type plain CheckScope
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *FindingEvidence) UnmarshalJSON(b []byte) error {
	type plain FindingEvidence
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra. It reads a JSON
// null in Coverage and CoverageBoundary as nil.
func (x *Finding) UnmarshalJSON(b []byte) error {
	type plain Finding
	if err := decodeWithExtra(b, (*plain)(x), &x.Extra); err != nil {
		return err
	}
	x.Coverage = nilIfNull(x.Coverage)
	x.CoverageBoundary = nilIfNull(x.CoverageBoundary)
	return nil
}

// Excerpt returns the part of text that this finding covers. Pass the text you sent to
// CitationChecks.Create. The offsets count Unicode code points, not bytes, so the text is cut
// by runes. Excerpt returns the empty string when the offsets do not fit the text.
func (x *Finding) Excerpt(text string) string {
	runes := []rune(text)
	if x.StartOffset < 0 || x.EndOffset < x.StartOffset || x.EndOffset > len(runes) {
		return ""
	}
	return string(runes[x.StartOffset:x.EndOffset])
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CitationCheckRun) UnmarshalJSON(b []byte) error {
	type plain CitationCheckRun
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CitationCheckDocument) UnmarshalJSON(b []byte) error {
	type plain CitationCheckDocument
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}
