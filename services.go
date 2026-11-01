package klaracase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"time"
)

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
func (s *DecisionsService) Search(ctx context.Context, p SearchDecisionsParams, opts ...RequestOption) (*DecisionSearchResponse, error) {
	if err := validateSearchDecisions(p); err != nil {
		return nil, err
	}
	var q query
	q.str("query", p.Query)
	q.str("court", p.Court)
	q.str("branch", string(p.Branch))
	q.str("document_type", string(p.DocumentType))
	q.str("cited_norm", p.CitedNorm)
	addList(&q, "cited_statutes", p.CitedStatutes)
	q.str("date_from", string(p.DateFrom))
	q.str("date_to", string(p.DateTo))
	q.str("mode", string(p.Mode))
	q.num("limit", p.Limit)
	q.num("page", p.Page)
	q.boolp("bridge", p.Bridge)
	out := new(DecisionSearchResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /decisions/search", path: "/decisions/search", query: q, opts: opts}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get reads one decision (GET /decisions/{id}). The id is a UUID, an Aktenzeichen or an ECLI.
// It costs 1 credit.
func (s *DecisionsService) Get(ctx context.Context, id string, p GetDecisionParams, opts ...RequestOption) (*DecisionResponse, error) {
	if err := validateGetDecision(id, p); err != nil {
		return nil, err
	}
	var q query
	q.num("max_chars", p.MaxChars)
	q.num("rn_from", p.RnFrom)
	q.num("rn_to", p.RnTo)
	q.str("section", string(p.Section))
	q.str("court", p.Court)
	q.str("date", string(p.Date))
	out := new(DecisionResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /decisions/{id}", path: "/decisions/" + escapeSegment(id), query: q, opts: opts}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Citations reads the citation graph around one decision (GET /decisions/{id}/citations).
// It costs 2 credits.
func (s *DecisionsService) Citations(ctx context.Context, id string, p CitationsParams, opts ...RequestOption) (*CitationsResponse, error) {
	if err := validateCitations(id, p); err != nil {
		return nil, err
	}
	var q query
	q.str("direction", string(p.Direction))
	q.num("depth", p.Depth)
	q.str("court", p.Court)
	q.str("date", string(p.Date))
	q.num("page", p.Page)
	out := new(CitationsResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /decisions/{id}/citations", path: "/decisions/" + escapeSegment(id) + "/citations", query: q, opts: opts}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get reads the current text of one statute norm (GET /norms/{reference}), for example
// "§ 535 BGB" or "Art. 6 DSGVO". It costs 1 credit.
func (s *NormsService) Get(ctx context.Context, reference string, p GetNormParams, opts ...RequestOption) (*NormResponse, error) {
	if reference == "" {
		return nil, clientValidation("reference", "reference is required")
	}
	var q query
	q.str("jurisdiction", string(p.Jurisdiction))
	out := new(NormResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /norms/{reference}", path: "/norms/" + escapeSegment(reference), query: q, opts: opts}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Create bundles 1 to 10 decisions and their citations (POST /datapackages). It costs 5 credits.
func (s *DatapackagesService) Create(ctx context.Context, p CreateDatapackageParams, opts ...RequestOption) (*DatapackageResponse, error) {
	if err := validateDatapackage(p); err != nil {
		return nil, err
	}
	req, err := jsonRequest("POST /datapackages", "/datapackages", p, opts)
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
func (s *CitationChecksService) Create(ctx context.Context, p CreateCitationCheckParams, opts ...RequestOption) (*CitationCheckRun, error) {
	if err := validateCitationCheck(p); err != nil {
		return nil, err
	}
	req, err := jsonRequest("POST /citation-checks", "/citation-checks", p, opts)
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
func (s *CitationChecksService) Get(ctx context.Context, runID string, opts ...RequestOption) (*CitationCheckRun, error) {
	if runID == "" {
		return nil, clientValidation("run_id", "run id is required")
	}
	out := new(CitationCheckRun)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /citation-checks/{run}", path: "/citation-checks/" + escapeSegment(runID), opts: opts}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Upload sends a file for a citation check (POST /citation-check-documents, multipart). The
// answer has HTTP status 202 and the status "uploaded". Upload is free. Poll with Get or
// WaitUntilReady until the status is "ready", then pass the id to CitationChecks.Create.
func (s *CitationCheckDocumentsService) Upload(ctx context.Context, p UploadParams, opts ...RequestOption) (*CitationCheckDocument, error) {
	if p.File == nil {
		return nil, clientValidation("file", "File is required")
	}
	if p.Filename == "" {
		return nil, clientValidation("filename", "Filename is required")
	}
	contentType := p.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	data, err := io.ReadAll(p.File)
	if err != nil {
		return nil, clientValidation("file", "read File: %v", err)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{"name": "file", "filename": p.Filename}))
	header.Set("Content-Type", contentType)
	part, err := mw.CreatePart(header)
	if err != nil {
		return nil, clientValidation("file", "build upload: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, clientValidation("file", "build upload: %v", err)
	}
	if err := mw.Close(); err != nil {
		return nil, clientValidation("file", "build upload: %v", err)
	}
	req := request{method: http.MethodPost, endpoint: "POST /citation-check-documents", path: "/citation-check-documents", body: buf.Bytes(), contentType: mw.FormDataContentType(), opts: opts}
	out := new(CitationCheckDocument)
	if err := s.c.do(ctx, req, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Get reads the receipt of an uploaded document (GET /citation-check-documents/{document}). It is free.
func (s *CitationCheckDocumentsService) Get(ctx context.Context, documentID string, opts ...RequestOption) (*CitationCheckDocument, error) {
	if documentID == "" {
		return nil, clientValidation("document_id", "document id is required")
	}
	out := new(CitationCheckDocument)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /citation-check-documents/{document}", path: "/citation-check-documents/" + escapeSegment(documentID), opts: opts}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// WaitOption changes how WaitUntilReady polls.
type WaitOption func(*waitConfig)

type waitConfig struct {
	interval    time.Duration
	requestOpts []RequestOption
	err         error
}

// DefaultPollInterval is the wait between two polls of WaitUntilReady.
const DefaultPollInterval = 2 * time.Second

// WithPollInterval sets the wait between two polls (default 2 seconds). It must be above zero.
func WithPollInterval(d time.Duration) WaitOption {
	return func(c *waitConfig) {
		if d <= 0 && c.err == nil {
			c.err = newConfigurationError("WithPollInterval: the interval must be above zero")
			return
		}
		c.interval = d
	}
}

// WithPollRequestOptions sets request options for every poll of WaitUntilReady, for example a
// header or a shorter timeout per poll.
func WithPollRequestOptions(opts ...RequestOption) WaitOption {
	return func(c *waitConfig) { c.requestOpts = append(c.requestOpts, opts...) }
}

// WaitUntilReady polls Get until the document is "ready" and returns the receipt. The statuses
// "uploaded", "extracting" and "ocr_pending" are not final.
//
// The context is the only deadline. Pass a context with a timeout; without one the wait lasts
// until the document is ready or failed. It returns:
//
//   - a *DocumentFailedError when the status is "failed",
//   - a *DocumentTimeoutError when the deadline of the context passes first. It wraps
//     context.DeadlineExceeded, so errors.Is(err, context.DeadlineExceeded) is true, and it
//     carries the last receipt that was read,
//   - the context error, inside a *TransportError, when the context is canceled,
//   - the usual error for a failed poll.
func (s *CitationCheckDocumentsService) WaitUntilReady(ctx context.Context, documentID string, opts ...WaitOption) (*CitationCheckDocument, error) {
	cfg := waitConfig{interval: DefaultPollInterval}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.err != nil {
		return nil, cfg.err
	}
	var last *CitationCheckDocument
	for {
		doc, err := s.Get(ctx, documentID, cfg.requestOpts...)
		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, waitTimeout(last)
			}
			return nil, err
		}
		last = doc
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
		if err := s.c.sleep(ctx, cfg.interval); err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, waitTimeout(last)
			}
			return nil, s.c.transportError(ctx, err)
		}
	}
}

// waitTimeout builds the error for a wait that the context deadline ended.
func waitTimeout(last *CitationCheckDocument) error {
	return &DocumentTimeoutError{
		Problem:  Problem{Code: CodeWaitTimeout, Detail: "the document was not ready before the deadline of the context"},
		Document: last,
		Err:      context.DeadlineExceeded,
	}
}

// Coverage reports what the corpus holds (GET /corpus/coverage). It is free.
func (s *CorpusService) Coverage(ctx context.Context, p CoverageParams, opts ...RequestOption) (*CoverageResponse, error) {
	var q query
	q.str("branch", string(p.Branch))
	q.num("top", p.Top)
	out := new(CoverageResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /corpus/coverage", path: "/corpus/coverage", query: q, opts: opts}, out); err != nil {
		return nil, err
	}
	return out, nil
}

// Facets counts the values of one field (GET /corpus/facets/{field}). It is free. The field
// cited_norms needs FacetsParams.Norm.
func (s *CorpusService) Facets(ctx context.Context, field FacetField, p FacetsParams, opts ...RequestOption) (*FacetsResponse, error) {
	if field == "" {
		return nil, clientValidation("field", "field is required")
	}
	var q query
	q.str("type", string(p.Type))
	q.str("norm", p.Norm)
	q.str("prefix", p.Prefix)
	out := new(FacetsResponse)
	if err := s.c.do(ctx, request{method: http.MethodGet, endpoint: "GET /corpus/facets/{field}", path: "/corpus/facets/" + escapeSegment(string(field)), query: q, opts: opts}, out); err != nil {
		return nil, err
	}
	return out, nil
}
