package klaracase

import (
	"io"
	"time"
)

// A params struct can be a tool input for a language model. Its json tags are the wire names
// of the API, and a field that is not set is left out when the struct is marshaled. For every
// params struct the zero value is allowed; the fields that the API requires are named in the
// comment of the struct.
//
// An integer parameter is a plain int. Zero means "not set", because no integer parameter of
// the API accepts zero. A boolean parameter is a *bool, because false is a value the API
// accepts: use [Bool] or [Ptr].

// SearchDecisionsParams are the parameters of Decisions.Search. Query is required.
// An empty string, a zero number, a nil pointer and an empty slice are not sent. Use Bool to
// send false.
type SearchDecisionsParams struct {
	// Query is the search text in German legal phrasing, 2 to 500 characters. Required.
	Query string `json:"query"`
	// Court filters by court short form, for example "BGH".
	Court string `json:"court,omitzero"`
	// Branch filters by court branch (a jurisdiction, not a subject area).
	Branch Branch `json:"branch,omitzero"`
	// DocumentType filters by document type.
	DocumentType DocumentType `json:"document_type,omitzero"`
	// CitedNorm keeps decisions that cite this norm, for example "§ 906 BGB".
	CitedNorm string `json:"cited_norm,omitzero"`
	// CitedStatutes keeps decisions that cite any section of these statutes, for example StatuteStGB. Sent as cited_statutes[].
	CitedStatutes []Statute `json:"cited_statutes,omitzero"`
	// DateFrom is the earliest decision date.
	DateFrom Date `json:"date_from,omitzero"`
	// DateTo is the latest decision date. It must not be before DateFrom.
	DateTo Date `json:"date_to,omitzero"`
	// Mode is the retrieval mode.
	Mode Mode `json:"mode,omitzero"`
	// Limit is the number of results, 1 to 20. Zero leaves it to the server.
	Limit int `json:"limit,omitzero"`
	// Page is the page, 1 to 10. Zero leaves it to the server, which serves page 1.
	Page int `json:"page,omitzero"`
	// Bridge allows the server to add register terms to the query. The server default is true.
	Bridge *bool `json:"bridge,omitzero"`
}

// CrossTypeSearchParams are the parameters of Client.Search. Query is required.
type CrossTypeSearchParams struct {
	// Query is the search text in German legal phrasing, 2 to 500 characters. Required.
	Query string `json:"query"`
	// Jurisdiction narrows the statute section: DE, EU or XI.
	Jurisdiction Jurisdiction `json:"jurisdiction,omitzero"`
	// Court filters the decision section by court short form.
	Court string `json:"court,omitzero"`
	// Branch filters the decision section by court branch.
	Branch Branch `json:"branch,omitzero"`
	// DateFrom is the earliest decision date.
	DateFrom Date `json:"date_from,omitzero"`
	// DateTo is the latest decision date. It must not be before DateFrom.
	DateTo Date `json:"date_to,omitzero"`
	// Mode is the retrieval mode.
	Mode Mode `json:"mode,omitzero"`
	// DocumentType filters the decision section by document type.
	DocumentType DocumentType `json:"document_type,omitzero"`
	// CitedNorm keeps decisions that cite this norm.
	CitedNorm string `json:"cited_norm,omitzero"`
	// CitedStatutes keeps decisions that cite any section of these statutes. Sent as cited_statutes[].
	CitedStatutes []Statute `json:"cited_statutes,omitzero"`
	// Limit is the number of results per section, 1 to 10. Zero leaves it to the server.
	Limit int `json:"limit,omitzero"`
	// Bridge allows the server to add register terms to the query. The server default is true.
	Bridge *bool `json:"bridge,omitzero"`
}

// GetDecisionParams are the optional parameters of Decisions.Get. The zero value reads the
// whole decision, up to the server default of 40000 characters.
type GetDecisionParams struct {
	// MaxChars is the most characters of text to return (minimum 500, server default 40000).
	MaxChars int `json:"max_chars,omitzero"`
	// RnFrom is the first Randnummer to return.
	RnFrom int `json:"rn_from,omitzero"`
	// RnTo is the last Randnummer to return. It must not be lower than RnFrom.
	RnTo int `json:"rn_to,omitzero"`
	// Section returns one named section instead of the whole text.
	Section Section `json:"section,omitzero"`
	// Court disambiguates an Aktenzeichen, for example "BGH".
	Court string `json:"court,omitzero"`
	// Date disambiguates an Aktenzeichen.
	Date Date `json:"date,omitzero"`
}

// CitationsParams are the optional parameters of Decisions.Citations. The zero value reads
// both directions at depth 1, page 1.
type CitationsParams struct {
	// Direction selects outgoing, incoming or both (server default).
	Direction Direction `json:"direction,omitzero"`
	// Depth is 1 (server default) or 2. Zero leaves it to the server.
	Depth int `json:"depth,omitzero"`
	// Page is the 1-based page, up to 20. Page above 1 needs depth 1. Zero leaves it to the server.
	Page int `json:"page,omitzero"`
	// Court disambiguates an Aktenzeichen.
	Court string `json:"court,omitzero"`
	// Date disambiguates an Aktenzeichen.
	Date Date `json:"date,omitzero"`
}

// GetNormParams are the optional parameters of Norms.Get. The zero value is allowed.
type GetNormParams struct {
	// Jurisdiction is DE or EU. It is only read when the abbreviation is ambiguous.
	Jurisdiction Jurisdiction `json:"jurisdiction,omitzero"`
}

// CreateDatapackageParams are the parameters of Datapackages.Create. IDs is required.
type CreateDatapackageParams struct {
	// IDs are 1 to 10 decisions, each a UUID or an Aktenzeichen. Required. Order matters.
	IDs []string `json:"ids"`
	// IncludeCitations adds the outgoing citation edges. The server default is true.
	IncludeCitations *bool `json:"include_citations,omitzero"`
	// FullText serves whole judgments instead of excerpts. The server default is false.
	FullText *bool `json:"full_text,omitzero"`
	// MaxChars caps the text per decision when FullText is true (500 to 200000).
	MaxChars int `json:"max_chars,omitzero"`
}

// CreateCitationCheckParams are the parameters of CitationChecks.Create. Set exactly one of
// Text and DocumentID.
type CreateCitationCheckParams struct {
	// Text is the German legal text to check, 50 to 20000 characters.
	Text string `json:"text,omitzero"`
	// DocumentID is the id of an upload whose status is "ready". The check consumes the upload.
	DocumentID string `json:"document_id,omitzero"`
	// ReferenceDate is the document date. It dates statute citations that name no date.
	ReferenceDate Date `json:"reference_date,omitzero"`
}

// UploadParams are the parameters of CitationCheckDocuments.Upload. File and Filename are
// required.
type UploadParams struct {
	// File is the content: a PDF, DOCX, Markdown or plain text file up to 25 MB. The client
	// reads it fully into memory before the first attempt, so a retry can send it again.
	File io.Reader `json:"-"`
	// Filename is the name sent with the file, for example "schriftsatz.pdf".
	Filename string `json:"filename,omitzero"`
	// ContentType is the media type of the file. It defaults to "application/octet-stream".
	ContentType string `json:"content_type,omitzero"`
}

// CoverageParams are the optional parameters of Corpus.Coverage. The zero value is allowed.
type CoverageParams struct {
	// Branch limits the report to one court branch.
	Branch Branch `json:"branch,omitzero"`
	// Top caps the number of court rows, 1 to 100 (server default 20). Zero leaves it to the server.
	Top int `json:"top,omitzero"`
}

// FacetsParams are the optional parameters of Corpus.Facets. The zero value is allowed, except
// that the field cited_norms needs Norm.
type FacetsParams struct {
	// Type limits the counts to decisions or statutes. It is ignored for cited_norms.
	Type ArtifactKind `json:"type,omitzero"`
	// Norm is required for the cited_norms field: the norm whose family to list, for example "§ 543 BGB".
	Norm string `json:"norm,omitzero"`
	// Prefix keeps values that start with this text, case-insensitive.
	Prefix string `json:"prefix,omitzero"`
}

// WaitParams are the parameters of CitationCheckDocuments.WaitUntilReady.
type WaitParams struct {
	// PollInterval is the wait between two polls. Zero means 2 seconds.
	PollInterval time.Duration
	// Timeout is the longest total wait. Zero means 120 seconds.
	Timeout time.Duration
}
