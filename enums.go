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

// Section is a named part of a decision that GetDecisionParams.Section can select. The server
// matches the name without regard to case and umlauts.
type Section string

// Decision sections. DecisionResponse.AvailableSections lists the ones a decision has.
const (
	// SectionTenor is the operative part of the decision.
	SectionTenor Section = "Tenor"
	// SectionLeitsatz is the headnote.
	SectionLeitsatz Section = "Leitsatz"
	// SectionTatbestand is the statement of facts.
	SectionTatbestand Section = "Tatbestand"
	// SectionGruende is the reasoning.
	SectionGruende Section = "Gründe"
)

// Jurisdiction is a statute jurisdiction code. A value this SDK does not know is still kept.
type Jurisdiction string

// Statute jurisdictions.
const (
	// JurisdictionDE is German law.
	JurisdictionDE Jurisdiction = "DE"
	// JurisdictionEU is EU law and the Treaties.
	JurisdictionEU Jurisdiction = "EU"
	// JurisdictionXI is international law.
	JurisdictionXI Jurisdiction = "XI"
)

// ArtifactKind selects decisions or statutes in a facet request (FacetsParams.Type).
type ArtifactKind string

// Artifact kinds for facet counts.
const (
	// ArtifactKindDecision counts decisions.
	ArtifactKindDecision ArtifactKind = "decision"
	// ArtifactKindLaw counts statutes.
	ArtifactKindLaw ArtifactKind = "law"
)

// Statute is a statute abbreviation that SearchDecisionsParams.CitedStatutes accepts. The
// server holds a fixed list. A value outside the list is sent as it is and the server decides.
type Statute string

// Statutes that the cited_statutes filter accepts.
const (
	StatuteStGB   Statute = "StGB"
	StatuteStPO   Statute = "StPO"
	StatuteInsO   Statute = "InsO"
	StatuteDSGVO  Statute = "DSGVO"
	StatuteGDPR   Statute = "GDPR"
	StatuteBDSG   Statute = "BDSG"
	StatuteBauGB  Statute = "BauGB"
	StatuteBBauG  Statute = "BBauG"
	StatuteBauNVO Statute = "BauNVO"
	StatuteStVG   Statute = "StVG"
	StatuteStVO   Statute = "StVO"
)

// ArtifactType is the kind of a search hit: a statute or a decision. A value this SDK does not
// know is still kept as a plain string.
type ArtifactType string

// Artifact types.
const (
	// ArtifactTypeLaw is a statute hit.
	ArtifactTypeLaw ArtifactType = "LAW"
	// ArtifactTypeDecision is a decision hit.
	ArtifactTypeDecision ArtifactType = "DECISION"
)

// Confidence bands the relevance of a search hit.
type Confidence string

// Confidence bands. An empty value means the server did not measure it for the request.
const (
	// ConfidenceHigh is the top band.
	ConfidenceHigh Confidence = "high"
	// ConfidenceMedium is the middle band.
	ConfidenceMedium Confidence = "medium"
	// ConfidenceLow is the lowest band.
	ConfidenceLow Confidence = "low"
)

// CitationDateSource says which date a statute citation was checked against.
type CitationDateSource string

// Citation date sources.
const (
	// CitationDateSourceSentence means the sentence of the citation named its own date.
	CitationDateSourceSentence CitationDateSource = "sentence"
	// CitationDateSourceReferenceDate means the reference_date of the request was used.
	CitationDateSourceReferenceDate CitationDateSource = "reference_date"
	// CitationDateSourceNone means no date was available.
	CitationDateSourceNone CitationDateSource = "none"
)

// ExternalStatus says what the check against an external register did for a finding.
type ExternalStatus string

// External statuses. The API checks the corpus only, so the status is "skipped" today.
const (
	// ExternalStatusSkipped means no external register was asked.
	ExternalStatusSkipped ExternalStatus = "skipped"
)

// SubdivisionCheck says how far down a citation (Absatz, Satz) the check looked.
type SubdivisionCheck string

// Subdivision checks.
const (
	// SubdivisionNotChecked means only the section or the decision was checked.
	SubdivisionNotChecked SubdivisionCheck = "not_checked"
	// SubdivisionAbsatzUnverified means an Absatz was named and could not be verified.
	SubdivisionAbsatzUnverified SubdivisionCheck = "absatz_unverified"
)

// MatchStrength says how strong the match of a finding is.
type MatchStrength string

// Match strengths.
const (
	// MatchStrengthExact is an exact match.
	MatchStrengthExact MatchStrength = "exact"
)

// ReferenceType says whether a citation names a decision or a statute.
type ReferenceType string

// Reference types.
const (
	// ReferenceTypeCase is a citation of a decision.
	ReferenceTypeCase ReferenceType = "case"
	// ReferenceTypeStatute is a citation of a statute.
	ReferenceTypeStatute ReferenceType = "statute"
)

// KeyBasis says how a unique citation was keyed.
type KeyBasis string

// Key bases.
const (
	// KeyBasisMatchedDecision means the citation is keyed by the decision it matched.
	KeyBasisMatchedDecision KeyBasis = "matched_decision"
	// KeyBasisCanonical means the citation is keyed by its canonical form.
	KeyBasisCanonical KeyBasis = "canonical"
)

// ResolutionStatus says whether a citation edge points to a known decision.
type ResolutionStatus string

// Resolution statuses. An edge that is not resolved never counts as verified.
const (
	// ResolutionResolved means the edge points to a decision.
	ResolutionResolved ResolutionStatus = "resolved"
	// ResolutionUnresolvedAmbiguous means more than one decision fits.
	ResolutionUnresolvedAmbiguous ResolutionStatus = "unresolved_ambiguous"
	// ResolutionUnresolvedPending means the edge waits for resolution.
	ResolutionUnresolvedPending ResolutionStatus = "unresolved_pending"
	// ResolutionUnresolvedNoMatch means no decision fits.
	ResolutionUnresolvedNoMatch ResolutionStatus = "unresolved_no_match"
	// ResolutionUnresolvedOutOfCoverage means the cited decision lies outside the corpus coverage.
	ResolutionUnresolvedOutOfCoverage ResolutionStatus = "unresolved_out_of_coverage"
)

// TextMode says what text a datapackage serves.
type TextMode string

// Text modes.
const (
	// TextModeExcerpt serves excerpts.
	TextModeExcerpt TextMode = "excerpt"
	// TextModeFullText serves whole judgments, cut at max_chars.
	TextModeFullText TextMode = "full_text"
)

// RnSource says whether the Randnummern of a decision are the court's own.
type RnSource string

// Randnummer sources. Other values are kept as they are.
const (
	// RnSourcePrinted means the Randnummern are the court's own printed numbers.
	RnSourcePrinted RnSource = "printed"
)
