package klaracase

import (
	"encoding/json"
	"time"
)

// DecisionMetadata is the metadata block of a decision.
type DecisionMetadata struct {
	// DocumentType is the document type, for example "Urteil".
	DocumentType DocumentType `json:"document_type"`
	// ECLI is the European Case Law Identifier, when the decision has one.
	ECLI *string `json:"ecli"`
	// FileNumbers lists every Aktenzeichen of the decision.
	FileNumbers []string `json:"file_numbers"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// RnRange is a range of Randnummern (margin numbers).
type RnRange struct {
	// RnFrom is the first Randnummer.
	RnFrom int `json:"rn_from"`
	// RnTo is the last Randnummer.
	RnTo int `json:"rn_to"`
	// RnSource says whether the numbers are the court's own ("printed") or not.
	RnSource RnSource `json:"rn_source"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// DecisionResponse is the answer of Decisions.Get: the text of one decision, or a slice of it.
type DecisionResponse struct {
	Envelope
	// ID is the UUID of the decision.
	ID string `json:"id"`
	// Aktenzeichen is the file number.
	Aktenzeichen string `json:"aktenzeichen"`
	// Court is the court short form.
	Court string `json:"court"`
	// Date is the decision date.
	Date Date `json:"date"`
	// Jurisdiction is DE for German decisions.
	Jurisdiction Jurisdiction `json:"jurisdiction"`
	// Metadata holds the document type, the ECLI and the file numbers.
	Metadata *DecisionMetadata `json:"metadata"`
	// FortgeltungChecked is always false today: the API does not check whether a decision is still good law.
	FortgeltungChecked bool `json:"fortgeltung_checked"`
	// FortgeltungNote warns that the decision may have been quashed or superseded. Show it.
	FortgeltungNote string `json:"fortgeltung_note"`
	// FullText is the served text, which may be a slice (see Truncated and the range keys).
	FullText string `json:"full_text"`
	// TextLength is the length of the whole text in characters.
	TextLength int `json:"text_length"`
	// Truncated is true when FullText is shorter than the whole text.
	Truncated bool `json:"truncated"`
	// RnSource says whether the Randnummern are the court's own ("printed") or not.
	RnSource RnSource `json:"rn_source"`
	// RnSourceNote explains RnSource in words.
	RnSourceNote string `json:"rn_source_note"`
	// AvailableSections lists the sections the decision has, for example "Tenor" and "Gründe".
	AvailableSections []string `json:"available_sections"`
	// AvailableRange is the range of Randnummern the decision has.
	AvailableRange *RnRange `json:"available_range"`
	// AvailableRangeScope says what AvailableRange covers.
	AvailableRangeScope string `json:"available_range_scope"`
	// ReturnedRange is the span of Randnummern that FullText covers, when the request asked for
	// a window. Whole chunks come back, so it can be wider than the window. It is nil otherwise.
	ReturnedRange *RnRange `json:"returned_range,omitzero"`
	// DateSource says where the stored date comes from. The server sends it for a scanned
	// decision whose date may be wrong. It is kept raw because its shape is not fixed.
	DateSource json.RawMessage `json:"date_source,omitempty"`
	// DateConfidence says how far the stored date can be trusted. It is kept raw like DateSource.
	DateConfidence json.RawMessage `json:"date_confidence,omitempty"`
	// HasRnAnchors is true when the text carries Randnummer anchors.
	HasRnAnchors bool `json:"has_rn_anchors"`
}

// CorpusRef points to a decision inside the corpus.
type CorpusRef struct {
	// ArtifactID is the UUID of the decision.
	ArtifactID string `json:"artifact_id"`
	// Court is the court short form.
	Court string `json:"court"`
	// Date is the decision date.
	Date Date `json:"date"`
	// Az is the Aktenzeichen.
	Az string `json:"az"`
	// DocumentType is the document type.
	DocumentType DocumentType `json:"document_type"`
	// ECLI is the European Case Law Identifier, when known.
	ECLI *string `json:"ecli"`
	// CorpusURL is the address of the decision on klaracase.de.
	CorpusURL string `json:"corpus_url"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CitationEdge is one citation between two decisions.
type CitationEdge struct {
	// ReferenceCanonical is the cited decision in canonical form, for example "BGH 1 StR 50/15".
	ReferenceCanonical string `json:"reference_canonical"`
	// ReferenceRole is the role of the citation, for example "leitentscheidung", or nil.
	ReferenceRole *string `json:"reference_role"`
	// ResolutionStatus says whether the citation points to a known decision ("resolved").
	ResolutionStatus ResolutionStatus `json:"resolution_status"`
	// InCorpus is true when the cited decision is in the corpus.
	InCorpus bool `json:"in_corpus"`
	// Corpus describes the cited decision when it is in the corpus.
	Corpus *CorpusRef `json:"corpus"`
	// FromArtifactID is the UUID of the decision the edge starts from.
	FromArtifactID string `json:"from_artifact_id"`
	// Hop is the distance from the source decision (1 or 2). It is 0 when the edge has no hop,
	// as in a datapackage.
	Hop int `json:"hop,omitempty"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CitationSource is the decision whose citations were read.
type CitationSource struct {
	// ID is the UUID of the decision.
	ID string `json:"id"`
	// Aktenzeichen is the file number.
	Aktenzeichen string `json:"aktenzeichen"`
	// Court is the court short form.
	Court string `json:"court"`
	// Date is the decision date.
	Date Date `json:"date"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CitationsPagination describes the paging of a citations answer.
type CitationsPagination struct {
	// Page is the page that was served.
	Page int `json:"page"`
	// PageSize is the number of neighbours per page and direction.
	PageSize int `json:"page_size"`
	// MaxPage is the last page that can be requested.
	MaxPage int `json:"max_page"`
	// ReachableDistinctMax is the most distinct decisions that paging can reach.
	ReachableDistinctMax int `json:"reachable_distinct_max"`
	// ReferencesPages is the number of pages of outgoing citations.
	ReferencesPages int `json:"references_pages"`
	// ReferencedByPages is the number of pages of incoming citations.
	ReferencedByPages int `json:"referenced_by_pages"`
	// HasMoreReferences is true when another page of outgoing citations exists.
	HasMoreReferences bool `json:"has_more_references"`
	// HasMoreReferencedBy is true when another page of incoming citations exists.
	HasMoreReferencedBy bool `json:"has_more_referenced_by"`
	// AppliesAtDepth is the depth that paging applies to (1).
	AppliesAtDepth int `json:"applies_at_depth"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// CitationsResponse is the answer of Decisions.Citations. It costs 2 credits.
type CitationsResponse struct {
	Envelope
	// Source is the decision whose citations were read.
	Source *CitationSource `json:"source"`
	// References are the decisions the source cites.
	References []CitationEdge `json:"references"`
	// ReferencedBy are the decisions that cite the source.
	ReferencedBy []CitationEdge `json:"referenced_by"`
	// ServedDirection is the direction that was served.
	ServedDirection Direction `json:"served_direction"`
	// ServedReferencesTotal is the number of outgoing edges served.
	ServedReferencesTotal int `json:"served_references_total"`
	// ServedReferencedByTotal is the number of incoming edges served.
	ServedReferencedByTotal int `json:"served_referenced_by_total"`
	// ReferencesTotal is the number of outgoing edges.
	ReferencesTotal int `json:"references_total"`
	// ReferencesResolvedTotal is the number of outgoing edges that point to a known decision.
	ReferencesResolvedTotal int `json:"references_resolved_total"`
	// ReferencedByTotal is the number of incoming edges.
	ReferencedByTotal int `json:"referenced_by_total"`
	// ReferencesUnresolvedTotal is the number of outgoing edges that point to no known decision.
	ReferencesUnresolvedTotal int `json:"references_unresolved_total"`
	// ReferencesWithheldReasons lists reasons for withheld edges. The rows are kept raw.
	ReferencesWithheldReasons []json.RawMessage `json:"references_withheld_reasons"`
	// TotalsScope says what the totals count.
	TotalsScope string `json:"totals_scope"`
	// PerDirectionCap is the cap of edges per direction and page.
	PerDirectionCap int `json:"per_direction_cap"`
	// UnresolvedCap is the cap of unresolved edges.
	UnresolvedCap int `json:"unresolved_cap"`
	// NodeCap is the cap of distinct neighbours at depth 2.
	NodeCap int `json:"node_cap"`
	// Truncated is true when the answer was cut by a cap.
	Truncated bool `json:"truncated"`
	// NodesOmitted is the number of neighbours left out at depth 2.
	NodesOmitted int `json:"nodes_omitted"`
	// PayloadTruncated is true when the server cut the payload.
	PayloadTruncated bool `json:"payload_truncated"`
	// Pagination describes the paging.
	Pagination *CitationsPagination `json:"pagination"`
	// FortgeltungChecked is always false today.
	FortgeltungChecked bool `json:"fortgeltung_checked"`
	// FortgeltungNote warns that a decision may have been quashed or superseded. Show it.
	FortgeltungNote string `json:"fortgeltung_note"`
}

// DatapackageDecision is one decision of a datapackage.
type DatapackageDecision struct {
	// ID is the UUID of the decision.
	ID string `json:"id"`
	// Aktenzeichen is the file number.
	Aktenzeichen string `json:"aktenzeichen"`
	// Court is the court short form.
	Court string `json:"court"`
	// Date is the decision date.
	Date Date `json:"date"`
	// DocumentType is the document type.
	DocumentType DocumentType `json:"document_type"`
	// ECLI is the European Case Law Identifier, when known.
	ECLI *string `json:"ecli"`
	// Jurisdiction is DE for German decisions.
	Jurisdiction Jurisdiction `json:"jurisdiction"`
	// Excerpt is the served text: an excerpt, or the text when full_text was set.
	Excerpt string `json:"excerpt"`
	// ExcerptSpan says which part of the decision Excerpt covers, for example "document_start".
	ExcerptSpan string `json:"excerpt_span"`
	// TextLength is the length of the whole text in characters.
	TextLength int `json:"text_length"`
	// Truncated is true when Excerpt is shorter than the whole text.
	Truncated bool `json:"truncated"`
	// FortgeltungChecked is always false today.
	FortgeltungChecked bool `json:"fortgeltung_checked"`
	// FortgeltungNote warns that the decision may have been quashed or superseded. Show it.
	FortgeltungNote string `json:"fortgeltung_note"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// EdgeAccounting reports how many citation edges a datapackage holds for one source.
type EdgeAccounting struct {
	// FromArtifactID is the UUID of the source decision.
	FromArtifactID string `json:"from_artifact_id"`
	// Unit is what the counts count, for example "edge_rows".
	Unit string `json:"unit"`
	// EdgesTotal is the number of edges of the source.
	EdgesTotal int `json:"edges_total"`
	// EdgesIncluded is the number of edges in the datapackage.
	EdgesIncluded int `json:"edges_included"`
	// EdgesResolvedIncluded is the number of included edges that point to a known decision.
	EdgesResolvedIncluded int `json:"edges_resolved_included"`
	// EdgesUnresolvedIncluded is the number of included edges that point to no known decision.
	EdgesUnresolvedIncluded int `json:"edges_unresolved_included"`
	// EdgesResolvedTotal is the number of resolved edges of the source.
	EdgesResolvedTotal int `json:"edges_resolved_total"`
	// EdgesUnresolvedTotal is the number of unresolved edges of the source.
	EdgesUnresolvedTotal int `json:"edges_unresolved_total"`
	// TargetsResolvedIncluded is the number of distinct resolved targets included.
	TargetsResolvedIncluded int `json:"targets_resolved_included"`
	// TargetsResolvedTotal is the number of distinct resolved targets of the source.
	TargetsResolvedTotal int `json:"targets_resolved_total"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// DatapackageResponse is the answer of Datapackages.Create. It costs 5 credits.
type DatapackageResponse struct {
	Envelope
	// Decisions are the decisions of the package, in the order of the request.
	Decisions []DatapackageDecision `json:"decisions"`
	// CitationEdges are the outgoing citation edges of the decisions.
	CitationEdges []CitationEdge `json:"citation_edges"`
	// EdgeAccounting reports edge counts per source decision.
	EdgeAccounting []EdgeAccounting `json:"edge_accounting"`
	// PerSourceResolvedTargetCap is the cap of resolved targets per source.
	PerSourceResolvedTargetCap int `json:"per_source_resolved_target_cap"`
	// PerSourceUnresolvedEdgeRowCap is the cap of unresolved edge rows per source.
	PerSourceUnresolvedEdgeRowCap int `json:"per_source_unresolved_edge_row_cap"`
	// Requested is the number of distinct ids requested.
	Requested int `json:"requested"`
	// Included is the number of decisions in the package.
	Included int `json:"included"`
	// Deduped is the number of duplicate ids that collapsed.
	Deduped int `json:"deduped"`
	// TextMode is "excerpt" or "full_text".
	TextMode TextMode `json:"text_mode"`
	// MaxChars is the text cap that was applied per decision.
	MaxChars int `json:"max_chars"`
	// Truncated is true when the response budget dropped content.
	Truncated bool `json:"truncated"`
	// PayloadTruncated is true when the server cut the payload.
	PayloadTruncated bool `json:"payload_truncated"`
	// AssembledAt is the time the package was assembled. The server writes it with the offset
	// +00:00; it is parsed to the same instant.
	AssembledAt time.Time `json:"assembled_at"`
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *DecisionMetadata) UnmarshalJSON(b []byte) error {
	type plain DecisionMetadata
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *RnRange) UnmarshalJSON(b []byte) error {
	type plain RnRange
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *DecisionResponse) UnmarshalJSON(b []byte) error {
	type plain DecisionResponse
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CorpusRef) UnmarshalJSON(b []byte) error {
	type plain CorpusRef
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CitationEdge) UnmarshalJSON(b []byte) error {
	type plain CitationEdge
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CitationSource) UnmarshalJSON(b []byte) error {
	type plain CitationSource
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CitationsPagination) UnmarshalJSON(b []byte) error {
	type plain CitationsPagination
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *CitationsResponse) UnmarshalJSON(b []byte) error {
	type plain CitationsResponse
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *DatapackageDecision) UnmarshalJSON(b []byte) error {
	type plain DatapackageDecision
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *EdgeAccounting) UnmarshalJSON(b []byte) error {
	type plain EdgeAccounting
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *DatapackageResponse) UnmarshalJSON(b []byte) error {
	type plain DatapackageResponse
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}
