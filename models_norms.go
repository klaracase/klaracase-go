package klaracase

import "encoding/json"

// NormLaw describes the statute that a norm belongs to.
type NormLaw struct {
	// ID is the UUID of the statute.
	ID string `json:"id"`
	// Abbreviation is the abbreviation, for example "BGB".
	Abbreviation string `json:"abbreviation"`
	// SourceAbbreviation is the abbreviation in the source.
	SourceAbbreviation string `json:"source_abbreviation"`
	// OfficialName is the full name, for example "Bürgerliches Gesetzbuch".
	OfficialName string `json:"official_name"`
	// LawLevel is the level of the statute, for example "BUND" or "EU".
	LawLevel string `json:"law_level"`
	// Jurisdiction is DE or EU.
	Jurisdiction string `json:"jurisdiction"`
	// SourceURL is the address of the page the norm text comes from.
	SourceURL string `json:"source_url"`
	// UpstreamStatus is the state of the source, for example "confirmed".
	UpstreamStatus string `json:"upstream_status"`
	// UpstreamLastSeenAt is the last time the source was seen, RFC 3339.
	UpstreamLastSeenAt string `json:"upstream_last_seen_at"`
	// UpstreamMissingSince is the time the source went missing, or nil.
	UpstreamMissingSince *string `json:"upstream_missing_since"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// NormText is the text of one norm.
type NormText struct {
	// Number is the norm number, for example "§ 535" or "Art. 6".
	Number string `json:"number"`
	// Title is the title of the norm.
	Title string `json:"title"`
	// Lifecycle is the state of the norm, for example "current".
	Lifecycle string `json:"lifecycle"`
	// Text is the current consolidated text.
	Text string `json:"text"`
	// AbsatzIdentifiers lists the paragraph identifiers of the norm.
	AbsatzIdentifiers []string `json:"absatz_identifiers"`
	// Chars is the length of Text in characters.
	Chars int `json:"chars"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// NormQualifier reports whether a part of the requested reference (for example "Abs. 1")
// was found in the norm.
type NormQualifier struct {
	// Type is the kind of part, for example "Abs.".
	Type string `json:"type"`
	// Value is the value, for example "1".
	Value string `json:"value"`
	// Checked is true when the API checked this part.
	Checked bool `json:"checked"`
	// Present is true when the part exists in the norm.
	Present bool `json:"present"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// NormVersion describes the version of the statute text that was served.
type NormVersion struct {
	// ID is the UUID of the version.
	ID string `json:"id"`
	// LastAmendmentDate is the date of the last amendment.
	LastAmendmentDate string `json:"last_amendment_date"`
	// LastAmendmentDateBasis says where LastAmendmentDate comes from.
	LastAmendmentDateBasis string `json:"last_amendment_date_basis"`
	// SourceFetchedAt is the time the text was fetched, RFC 3339.
	SourceFetchedAt string `json:"source_fetched_at"`
	// SourceLastSeenAt is the last time the source was seen, RFC 3339.
	SourceLastSeenAt string `json:"source_last_seen_at"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// NormNeighbours names the norms before and after.
type NormNeighbours struct {
	// Previous is the previous norm, or nil.
	Previous *string `json:"previous"`
	// Next is the next norm, or nil.
	Next *string `json:"next"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// NormLink describes a follow-up call as a tool name and its arguments.
type NormLink struct {
	// Tool is the MCP tool name that makes the call.
	Tool string `json:"tool"`
	// Arguments are the tool arguments. The values are kept raw.
	Arguments map[string]json.RawMessage `json:"arguments"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// NormLinks holds follow-up links.
type NormLinks struct {
	// DecisionsCiting describes the search for decisions that cite this norm.
	DecisionsCiting *NormLink `json:"decisions_citing"`
	// DecisionsCitingURL is the same search as a path below the host.
	DecisionsCitingURL string `json:"decisions_citing_url"`
	// SelfURL is the path of this norm.
	SelfURL string `json:"self_url"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// NormResponse is the answer of Norms.Get. It costs 1 credit.
type NormResponse struct {
	Envelope
	// Reference is the canonical reference that was served, for example "§ 535 BGB".
	Reference string `json:"reference"`
	// RequestedReference is the reference as the caller asked, for example "§ 535 Abs. 1 BGB".
	RequestedReference string `json:"requested_reference"`
	// Law describes the statute.
	Law *NormLaw `json:"law"`
	// Norm holds the text.
	Norm *NormText `json:"norm"`
	// RequestedQualifiers reports the parts of the requested reference.
	RequestedQualifiers []NormQualifier `json:"requested_qualifiers"`
	// Version describes the text version.
	Version *NormVersion `json:"version"`
	// Neighbours names the previous and next norm.
	Neighbours *NormNeighbours `json:"neighbours"`
	// Links holds follow-up links.
	Links *NormLinks `json:"links"`
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *NormLaw) UnmarshalJSON(b []byte) error {
	type plain NormLaw
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *NormText) UnmarshalJSON(b []byte) error {
	type plain NormText
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *NormQualifier) UnmarshalJSON(b []byte) error {
	type plain NormQualifier
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *NormVersion) UnmarshalJSON(b []byte) error {
	type plain NormVersion
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *NormNeighbours) UnmarshalJSON(b []byte) error {
	type plain NormNeighbours
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *NormLink) UnmarshalJSON(b []byte) error {
	type plain NormLink
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *NormLinks) UnmarshalJSON(b []byte) error {
	type plain NormLinks
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (x *NormResponse) UnmarshalJSON(b []byte) error {
	type plain NormResponse
	return decodeWithExtra(b, (*plain)(x), &x.Extra)
}
