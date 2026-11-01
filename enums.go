package klaracase

// Mode is the retrieval mode of a search.
type Mode string

// Retrieval modes. Unset lets the server choose, which is hybrid unless the query is a
// precise identifier.
const (
	// ModeKeyword requires all query terms and is not reranked.
	ModeKeyword Mode = "keyword"
	// ModeSemantic is a pure vector search.
	ModeSemantic Mode = "semantic"
	// ModeHybrid blends semantic and keyword search.
	ModeHybrid Mode = "hybrid"
)

// Branch is a Gerichtsbarkeit, a court jurisdiction. It is not a legal subject area.
type Branch string

// Court branches.
const (
	// BranchOrdentlich is the ordinary jurisdiction. BGH patent decisions sit here.
	BranchOrdentlich Branch = "ordentlich"
	// BranchVerwaltung is the administrative jurisdiction.
	BranchVerwaltung Branch = "verwaltung"
	// BranchSozial is the social jurisdiction.
	BranchSozial Branch = "sozial"
	// BranchArbeit is the labour jurisdiction.
	BranchArbeit Branch = "arbeit"
	// BranchFinanz is the fiscal jurisdiction.
	BranchFinanz Branch = "finanz"
	// BranchVerfassung is the constitutional jurisdiction.
	BranchVerfassung Branch = "verfassung"
	// BranchPatent is the Bundespatentgericht alone.
	BranchPatent Branch = "patent"
	// BranchUnbekannt is a decision whose branch is unknown.
	BranchUnbekannt Branch = "unbekannt"
)

// DocumentType is the type of a decision document.
type DocumentType string

// Decision document types.
const (
	// DocumentTypeUrteil is a judgment.
	DocumentTypeUrteil DocumentType = "Urteil"
	// DocumentTypeBeschluss is a ruling by order.
	DocumentTypeBeschluss DocumentType = "Beschluss"
	// DocumentTypeVerfuegung is an order of a presiding judge.
	DocumentTypeVerfuegung DocumentType = "Verfügung"
	// DocumentTypeSonstige is any other document.
	DocumentTypeSonstige DocumentType = "Sonstige"
	// DocumentTypeGerichtsbescheid is a court notice.
	DocumentTypeGerichtsbescheid DocumentType = "Gerichtsbescheid"
	// DocumentTypeSchlussantraege is the opinion of an advocate general.
	DocumentTypeSchlussantraege DocumentType = "Schlussanträge"
)

// Direction selects which citation edges to read.
type Direction string

// Citation directions.
const (
	// DirectionOutgoing lists decisions that the source cites.
	DirectionOutgoing Direction = "outgoing"
	// DirectionIncoming lists decisions that cite the source.
	DirectionIncoming Direction = "incoming"
	// DirectionBoth lists both. It is the server default.
	DirectionBoth Direction = "both"
)

// FacetField is a field that Corpus.Facets can count.
type FacetField string

// Facet fields.
const (
	// FacetCourtName counts decisions per court.
	FacetCourtName FacetField = "court_name"
	// FacetCourtBranch counts per court branch.
	FacetCourtBranch FacetField = "court_branch"
	// FacetDocumentType counts per document type.
	FacetDocumentType FacetField = "document_type"
	// FacetJurisdiction counts per jurisdiction.
	FacetJurisdiction FacetField = "jurisdiction"
	// FacetLawType counts laws per type.
	FacetLawType FacetField = "law_type"
	// FacetCitedNorms counts the norms of one statute family that decisions cite. It needs FacetsParams.Norm.
	FacetCitedNorms FacetField = "cited_norms"
)

// Verdict is the result of the check of one citation. The set is closed on the server. A
// value this SDK does not know is still kept as a plain string.
type Verdict string

// Citation verdicts.
const (
	// VerdictVerifiedCorpus means the citation exists in the corpus. It proves existence only,
	// not that a quote or a legal proposition is right.
	VerdictVerifiedCorpus Verdict = "verified_corpus"
	// VerdictVerifiedExternal means an external register confirmed the citation.
	VerdictVerifiedExternal Verdict = "verified_external"
	// VerdictNotFound means the citation was not found in the corpus. It is not a claim that
	// the citation is fabricated.
	VerdictNotFound Verdict = "not_found"
	// VerdictOutOfCoverage means the citation lies outside what the corpus holds.
	VerdictOutOfCoverage Verdict = "out_of_coverage"
	// VerdictAmbiguous means more than one decision or norm fits.
	VerdictAmbiguous Verdict = "ambiguous"
	// VerdictUnverifiable means the citation cannot be checked.
	VerdictUnverifiable Verdict = "unverifiable"
	// VerdictDiscrepancy means the citation exists but a detail differs.
	VerdictDiscrepancy Verdict = "discrepancy"
)

// RunStatus is the status of a citation check run.
type RunStatus string

// Citation check run statuses. A run returned by CitationChecks.Create is already final.
const (
	// RunPending is queued.
	RunPending RunStatus = "pending"
	// RunGating is in the admission checks.
	RunGating RunStatus = "gating"
	// RunExtracting is extracting citations.
	RunExtracting RunStatus = "extracting"
	// RunResolving is resolving citations.
	RunResolving RunStatus = "resolving"
	// RunCheckingExternal is checking an external register.
	RunCheckingExternal RunStatus = "checking_external"
	// RunComplete is final and successful.
	RunComplete RunStatus = "complete"
	// RunRejected is final: the text was refused.
	RunRejected RunStatus = "rejected"
	// RunFailed is final: the run failed.
	RunFailed RunStatus = "failed"
)

// DocumentStatus is the status of an uploaded document.
type DocumentStatus string

// Document statuses. WaitUntilReady stops at DocumentReady and DocumentFailed. The other
// three are not final.
const (
	// DocumentUploaded was received and waits for extraction.
	DocumentUploaded DocumentStatus = "uploaded"
	// DocumentExtracting is being read.
	DocumentExtracting DocumentStatus = "extracting"
	// DocumentOCRPending waits for text recognition. It is not final.
	DocumentOCRPending DocumentStatus = "ocr_pending"
	// DocumentReady can be passed to CitationChecks.Create.
	DocumentReady DocumentStatus = "ready"
	// DocumentFailed could not be processed; see the error_code.
	DocumentFailed DocumentStatus = "failed"
)
