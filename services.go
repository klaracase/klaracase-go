package klaracase

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"
)

// SearchDecisionsParams are the parameters of Decisions.Search. Query is required.
// An empty string, a nil pointer and an empty slice are not sent. Use Int and Bool to send
// a zero or a false.
type SearchDecisionsParams struct {
	// Query is the search text in German legal phrasing, 2 to 500 characters. Required.
	Query string
	// Court filters by court short form, for example "BGH".
	Court string
	// Branch filters by court branch (a jurisdiction, not a subject area).
	Branch Branch
	// DocumentType filters by document type.
	DocumentType DocumentType
	// CitedNorm keeps decisions that cite this norm, for example "§ 906 BGB".
	CitedNorm string
	// CitedStatutes keeps decisions that cite any section of these statutes, for example "StGB". Sent as cited_statutes[].
	CitedStatutes []string
	// DateFrom is the earliest decision date, YYYY-MM-DD.
	DateFrom string
	// DateTo is the latest decision date, YYYY-MM-DD.
	DateTo string
	// Mode is the retrieval mode.
	Mode Mode
	// Limit is the number of results, 1 to 20.
	Limit *int
	// Page is the page, 1 to 10.
	Page *int
	// Bridge allows the server to add register terms to the query. The server default is true.
	Bridge *bool
}

// CrossTypeSearchParams are the parameters of Client.Search. Query is required.
type CrossTypeSearchParams struct {
	// Query is the search text in German legal phrasing, 2 to 500 characters. Required.
	Query string
	// Jurisdiction narrows the statute section: DE, EU or XI.
	Jurisdiction string
	// Court filters the decision section by court short form.
	Court string
	// Branch filters the decision section by court branch.
	Branch Branch
	// DateFrom is the earliest decision date, YYYY-MM-DD.
	DateFrom string
	// DateTo is the latest decision date, YYYY-MM-DD.
	DateTo string
	// Mode is the retrieval mode.
	Mode Mode
	// DocumentType filters the decision section by document type.
	DocumentType DocumentType
	// CitedNorm keeps decisions that cite this norm.
	CitedNorm string
	// CitedStatutes keeps decisions that cite any section of these statutes. Sent as cited_statutes[].
	CitedStatutes []string
	// Limit is the number of results per section, 1 to 10.
	Limit *int
	// Bridge allows the server to add register terms to the query. The server default is true.
	Bridge *bool
}

// GetDecisionParams are the optional parameters of Decisions.Get.
type GetDecisionParams struct {
	// MaxChars is the most characters of text to return (minimum 500, server default 40000).
	MaxChars *int
	// RnFrom is the first Randnummer to return.
	RnFrom *int
	// RnTo is the last Randnummer to return, not lower than RnFrom.
	RnTo *int
	// Section returns one named section: Tenor, Leitsatz, Tatbestand or Gründe.
	Section string
	// Court disambiguates an Aktenzeichen, for example "BGH".
	Court string
	// Date disambiguates an Aktenzeichen, YYYY-MM-DD.
	Date string
}

// CitationsParams are the optional parameters of Decisions.Citations.
type CitationsParams struct {
	// Direction selects outgoing, incoming or both (server default).
	Direction Direction
	// Depth is 1 (server default) or 2.
	Depth *int
	// Page is the 1-based page, up to 20. Page above 1 needs depth 1.
	Page *int
	// Court disambiguates an Aktenzeichen.
	Court string
	// Date disambiguates an Aktenzeichen, YYYY-MM-DD.
	Date string
}

// GetNormParams are the optional parameters of Norms.Get.
type GetNormParams struct {
	// Jurisdiction is DE or EU. It is only read when the abbreviation is ambiguous.
	Jurisdiction string
}

// CreateDatapackageParams are the parameters of Datapackages.Create.
type CreateDatapackageParams struct {
	// IDs are 1 to 10 decisions, each a UUID or an Aktenzeichen. Required. Order matters.
	IDs []string
	// IncludeCitations adds the outgoing citation edges. The server default is true.
	IncludeCitations *bool
	// FullText serves whole judgments instead of excerpts. The server default is false.
	FullText *bool
	// MaxChars caps the text per decision when FullText is true (500 to 200000).
	MaxChars *int
}

// CreateCitationCheckParams are the parameters of CitationChecks.Create. Set exactly one of
// Text and DocumentID.
type CreateCitationCheckParams struct {
	// Text is the German legal text to check, 50 to 20000 characters.
	Text string
	// DocumentID is the id of an upload whose status is "ready". The check consumes the upload.
	DocumentID string
	// ReferenceDate is the document date, YYYY-MM-DD. It dates statute citations that name no date.
	ReferenceDate string
}

// UploadParams are the parameters of CitationCheckDocuments.Upload.
type UploadParams struct {
	// File is the content: a PDF, DOCX, Markdown or plain text file up to 25 MB. The client
	// reads it fully into memory before the first attempt, so a retry can send it again.
	File io.Reader
	// Filename is the name sent with the file, for example "schriftsatz.pdf".
	Filename string
	// ContentType is the media type of the file. It defaults to "application/octet-stream".
	ContentType string
}

// WaitParams are the parameters of CitationCheckDocuments.WaitUntilReady.
type WaitParams struct {
	// PollInterval is the wait between two polls. Zero means 2 seconds.
	PollInterval time.Duration
	// Timeout is the longest total wait. Zero means 120 seconds.
	Timeout time.Duration
}

// CoverageParams are the optional parameters of Corpus.Coverage.
type CoverageParams struct {
	// Branch limits the report to one court branch.
	Branch Branch
	// Top caps the number of court rows, 1 to 100 (server default 20).
	Top *int
}

// FacetsParams are the optional parameters of Corpus.Facets.
type FacetsParams struct {
	// Type limits the counts to "decision" or "law". It is ignored for cited_norms.
	Type string
	// Norm is required for the cited_norms field: the norm whose family to list, for example "§ 543 BGB".
	Norm string
	// Prefix keeps values that start with this text, case-insensitive.
	Prefix string
}

// DecisionsService groups the decision operations. Use Client.Decisions.
type DecisionsService struct{ c *Client }

// NormsService groups the norm operations. Use Client.Norms.
type NormsService struct{ c *Client }

// DatapackagesService groups the datapackage operations. Use Client.Datapackages.
type DatapackagesService struct{ c *Client }

// CitationChecksService groups the citation check operations. Use Client.CitationChecks.
type CitationChecksService struct{ c *Client }

// CitationCheckDocumentsService groups the upload operations. Use Client.CitationCheckDocuments.
type CitationCheckDocumentsService struct{ c *Client }

// CorpusService groups the free corpus reports. Use Client.Corpus.
type CorpusService struct{ c *Client }

// Search finds court decisions (GET /decisions/search). It costs 1 credit.
func (s *DecisionsService) Search(ctx context.Context, p SearchDecisionsParams) (*DecisionSearchResponse, error) {
	if p.Query == "" {
		return nil, clientValidation("Query is required")
	}
	var q query
	q.str("query", p.Query)
	q.str("court", p.Court)
	q.str("branch", string(p.Branch))
	q.str("document_type", string(p.DocumentType))
	q.str("cited_norm", p.CitedNorm)
	q.list("cited_statutes", p.CitedStatutes)
	q.str("date_from", p.DateFrom)
	q.str("date_to", p.DateTo)
	q.str("mode", string(p.Mode))
	q.intp("limit", p.Limit)
	q.intp("page", p.Page)
	q.boolp("bridge", p.Bridge)
	out := new(DecisionSearchResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /decisions/search", path: "/decisions/search", query: q}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get reads one decision (GET /decisions/{id}). The id is a UUID, an Aktenzeichen or an ECLI.
// It costs 1 credit.
func (s *DecisionsService) Get(ctx context.Context, id string, p GetDecisionParams) (*DecisionResponse, error) {
	if id == "" {
		return nil, clientValidation("id is required")
	}
	var q query
	q.intp("max_chars", p.MaxChars)
	q.intp("rn_from", p.RnFrom)
	q.intp("rn_to", p.RnTo)
	q.str("section", p.Section)
	q.str("court", p.Court)
	q.str("date", p.Date)
	out := new(DecisionResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /decisions/{id}", path: "/decisions/" + escapeSegment(id), query: q}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Citations reads the citation graph around one decision (GET /decisions/{id}/citations).
// It costs 2 credits.
func (s *DecisionsService) Citations(ctx context.Context, id string, p CitationsParams) (*CitationsResponse, error) {
	if id == "" {
		return nil, clientValidation("id is required")
	}
	var q query
	q.str("direction", string(p.Direction))
	q.intp("depth", p.Depth)
	q.str("court", p.Court)
	q.str("date", p.Date)
	q.intp("page", p.Page)
	out := new(CitationsResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /decisions/{id}/citations", path: "/decisions/" + escapeSegment(id) + "/citations", query: q}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get reads the current text of one statute norm (GET /norms/{reference}), for example
// "§ 535 BGB" or "Art. 6 DSGVO". It costs 1 credit.
func (s *NormsService) Get(ctx context.Context, reference string, p GetNormParams) (*NormResponse, error) {
	if reference == "" {
		return nil, clientValidation("reference is required")
	}
	var q query
	q.str("jurisdiction", p.Jurisdiction)
	out := new(NormResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /norms/{reference}", path: "/norms/" + escapeSegment(reference), query: q}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Create bundles 1 to 10 decisions and their citations (POST /datapackages). It costs 5 credits.
func (s *DatapackagesService) Create(ctx context.Context, p CreateDatapackageParams) (*DatapackageResponse, error) {
	if len(p.IDs) == 0 {
		return nil, clientValidation("IDs is required")
	}
	body := struct {
		IDs              []string `json:"ids"`
		IncludeCitations *bool    `json:"include_citations,omitempty"`
		FullText         *bool    `json:"full_text,omitempty"`
		MaxChars         *int     `json:"max_chars,omitempty"`
	}{p.IDs, p.IncludeCitations, p.FullText, p.MaxChars}
	req, err := jsonRequest("POST /datapackages", "/datapackages", body)
	if err != nil {
		return nil, err
	}
	out := new(DatapackageResponse)
	if err := s.c.do(ctx, req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Create checks the citations of a text or of an uploaded document (POST /citation-checks).
// The answer is the finished run. It costs 2 to 4 credits by length; the same text again in
// the same month costs nothing. Exactly one of Text and DocumentID must be set, otherwise
// Create returns a *ValidationError with code "client_validation" and sends nothing.
func (s *CitationChecksService) Create(ctx context.Context, p CreateCitationCheckParams) (*CitationCheckRun, error) {
	if (p.Text == "") == (p.DocumentID == "") {
		return nil, clientValidation("set exactly one of Text and DocumentID")
	}
	body := struct {
		Text          string `json:"text,omitempty"`
		DocumentID    string `json:"document_id,omitempty"`
		ReferenceDate string `json:"reference_date,omitempty"`
	}{p.Text, p.DocumentID, p.ReferenceDate}
	req, err := jsonRequest("POST /citation-checks", "/citation-checks", body)
	if err != nil {
		return nil, err
	}
	out := new(CitationCheckRun)
	if err := s.c.do(ctx, req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get reads a citation check run of your organization (GET /citation-checks/{run}). It is free.
func (s *CitationChecksService) Get(ctx context.Context, runID string) (*CitationCheckRun, error) {
	if runID == "" {
		return nil, clientValidation("runID is required")
	}
	out := new(CitationCheckRun)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /citation-checks/{run}", path: "/citation-checks/" + escapeSegment(runID)}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Upload sends a file for a citation check (POST /citation-check-documents, multipart). The
// answer has HTTP status 202 and the status "uploaded". Upload is free. Poll with Get or
// WaitUntilReady until the status is "ready", then pass the id to CitationChecks.Create.
func (s *CitationCheckDocumentsService) Upload(ctx context.Context, p UploadParams) (*CitationCheckDocument, error) {
	if p.File == nil {
		return nil, clientValidation("File is required")
	}
	if p.Filename == "" {
		return nil, clientValidation("Filename is required")
	}
	contentType := p.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	data, err := io.ReadAll(p.File)
	if err != nil {
		return nil, clientValidation("read File: %v", err)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": p.Filename}))
	header.Set("Content-Type", contentType)
	part, err := mw.CreatePart(header)
	if err != nil {
		return nil, clientValidation("build upload: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, clientValidation("build upload: %v", err)
	}
	if err := mw.Close(); err != nil {
		return nil, clientValidation("build upload: %v", err)
	}
	req := request{method: http.MethodPost, endpoint: "POST /citation-check-documents", path: "/citation-check-documents", body: buf.Bytes(), contentType: mw.FormDataContentType()}
	out := new(CitationCheckDocument)
	if err := s.c.do(ctx, req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get reads the receipt of an uploaded document (GET /citation-check-documents/{document}). It is free.
func (s *CitationCheckDocumentsService) Get(ctx context.Context, documentID string) (*CitationCheckDocument, error) {
	if documentID == "" {
		return nil, clientValidation("documentID is required")
	}
	out := new(CitationCheckDocument)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /citation-check-documents/{document}", path: "/citation-check-documents/" + escapeSegment(documentID)}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// WaitUntilReady polls Get until the document is "ready" and returns the receipt. The statuses
// "uploaded", "extracting" and "ocr_pending" are not final. It returns a *DocumentFailedError
// when the status is "failed", a *DocumentTimeoutError when Timeout passes first, and the
// usual error for a failed poll. The context ends the wait early.
func (s *CitationCheckDocumentsService) WaitUntilReady(ctx context.Context, documentID string, p WaitParams) (*CitationCheckDocument, error) {
	poll, timeout := p.PollInterval, p.Timeout
	if poll <= 0 {
		poll = 2 * time.Second
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	deadline := s.c.now().Add(timeout)
	for {
		doc, err := s.Get(ctx, documentID)
		if err != nil {
			return nil, err
		}
		switch doc.Status {
		case DocumentReady:
			return doc, nil
		case DocumentFailed:
			code := "document_failed"
			if doc.ErrorCode != nil {
				code = *doc.ErrorCode
			}
			return nil, &DocumentFailedError{
				Problem:   Problem{Code: code, Detail: "the document could not be processed: " + code},
				ErrorCode: code,
				Document:  doc,
			}
		}
		remaining := deadline.Sub(s.c.now())
		if remaining <= 0 {
			return nil, &DocumentTimeoutError{
				Problem:  Problem{Code: "wait_timeout", Detail: "the document was not ready before the timeout"},
				Document: doc,
			}
		}
		if err := s.c.sleep(ctx, min(poll, remaining)); err != nil {
			return nil, s.c.transportError(ctx, err)
		}
	}
}

// Coverage reports what the corpus holds (GET /corpus/coverage). It is free.
func (s *CorpusService) Coverage(ctx context.Context, p CoverageParams) (*CoverageResponse, error) {
	var q query
	q.str("branch", string(p.Branch))
	q.intp("top", p.Top)
	out := new(CoverageResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /corpus/coverage", path: "/corpus/coverage", query: q}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Facets counts the values of one field (GET /corpus/facets/{field}). It is free. The field
// cited_norms needs FacetsParams.Norm.
func (s *CorpusService) Facets(ctx context.Context, field FacetField, p FacetsParams) (*FacetsResponse, error) {
	if field == "" {
		return nil, clientValidation("field is required")
	}
	var q query
	q.str("type", p.Type)
	q.str("norm", p.Norm)
	q.str("prefix", p.Prefix)
	out := new(FacetsResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /corpus/facets/{field}", path: "/corpus/facets/" + escapeSegment(string(field)), query: q}, out); err != nil {
		return nil, err
	}
	return out, nil
}
