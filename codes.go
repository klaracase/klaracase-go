package klaracase

// Codes of the problem documents the API sends. The set is closed and shared with the MCP
// tools. A code that is not listed here is still kept in Error.Code, so compare with these
// constants and keep a default branch.
const (
	// CodeUnauthenticated is a missing, expired or revoked token (401).
	CodeUnauthenticated = "unauthenticated"
	// CodePersonalAccessTokenRequired means an OAuth token or a browser session was used on v1 (401).
	CodePersonalAccessTokenRequired = "personal_access_token_required"
	// CodeTokenOrgMissing means the token was minted before organization pinning (409).
	CodeTokenOrgMissing = "token_org_missing"
	// CodeTokenOrgMembershipLost means the token owner left the organization (403).
	CodeTokenOrgMembershipLost = "token_org_membership_lost"
	// CodePseudonymizerTokenRefused means a token of another product was used (403).
	CodePseudonymizerTokenRefused = "pseudonymizer_token_refused"
	// CodeTwoFactorRequired means the organization requires two-factor sign-in (403).
	CodeTwoFactorRequired = "two_factor_required"
	// CodeTokenAbilityMissing means the token lacks an ability the endpoint needs (403).
	CodeTokenAbilityMissing = "token_ability_missing"
	// CodeValidationFailed is a refused request field (422).
	CodeValidationFailed = "validation_failed"
	// CodeRateLimited is the per-minute rate limit (429).
	CodeRateLimited = "rate_limited"
	// CodeInsufficientCredits means the organization has too few credits (402). A reason can say why; see InsufficientCreditsError.
	CodeInsufficientCredits = "insufficient_credits"
	// CodeTokenLimitReached means this API key reached the credit limit set for it (402).
	CodeTokenLimitReached = "token_limit_reached" //nolint:gosec // an error code of the API, not a credential
	// CodeVolumeCap is the daily volume cap, which resets at midnight Europe/Berlin (429).
	CodeVolumeCap = "volume_cap"
	// CodeBackendUnavailable means the work failed and the charge was given back (503).
	CodeBackendUnavailable = "backend_unavailable"
	// CodeServiceUnavailable is the framework answer for an unavailable service (503).
	CodeServiceUnavailable = "service_unavailable"
	// CodeInternalError is an unexpected server error (500).
	CodeInternalError = "internal_error"

	// CodeBadRequest is the framework fallback for 400.
	CodeBadRequest = "bad_request"
	// CodeForbidden is the framework fallback for 403.
	CodeForbidden = "forbidden"
	// CodeNotFound means the decision, norm, run or document does not exist (404).
	CodeNotFound = "not_found"
	// CodeMethodNotAllowed is the framework fallback for 405.
	CodeMethodNotAllowed = "method_not_allowed"
	// CodeConflict is the framework fallback for 409.
	CodeConflict = "conflict"
	// CodePayloadTooLarge means an upload is over the size limit (413).
	CodePayloadTooLarge = "payload_too_large"
	// CodeUnsupportedMediaType is the framework fallback for 415.
	CodeUnsupportedMediaType = "unsupported_media_type"
	// CodeClientError is the framework fallback for any other 4xx.
	CodeClientError = "client_error"

	// CodeNotFoundUnderFilter means the record exists but not under the filter you gave (404).
	CodeNotFoundUnderFilter = "not_found_under_filter"
	// CodeStatuteSuperseded means a repealed statute was named; Error.Successor names its successor (404).
	CodeStatuteSuperseded = "statute_superseded"
	// CodeSectionNotInCurrentText means the section is not in the current text of the statute (404).
	CodeSectionNotInCurrentText = "section_not_in_current_text"
	// CodeDocumentUnavailable means the upload is not ready, expired, failed or consumed (409).
	CodeDocumentUnavailable = "document_unavailable"
	// CodeSearchTimeout means the search ran out of time (503).
	CodeSearchTimeout = "search_timeout"
	// CodeIndexUnavailable means the search index cannot be reached (503).
	CodeIndexUnavailable = "index_unavailable"
	// CodeNormFamilyUnavailable means the statute family cannot be read now (503).
	CodeNormFamilyUnavailable = "norm_family_unavailable"
	// CodeAmbiguousAktenzeichen means several decisions carry the Aktenzeichen; Error.Candidates lists them (422).
	CodeAmbiguousAktenzeichen = "ambiguous_aktenzeichen"
	// CodeIDsUnresolved means none of the datapackage ids could be resolved (422).
	CodeIDsUnresolved = "ids_unresolved"
	// CodePageUndefinedAtThisDepth means a page above 1 was asked at depth 2 (422).
	CodePageUndefinedAtThisDepth = "page_undefined_at_this_depth"
	// CodeTextTooLong means the text is over the length cap, refused before any charge (422).
	CodeTextTooLong = "text_too_long"
	// CodeNotGermanLegalText means the text does not look like German legal text, refused before any charge (422).
	CodeNotGermanLegalText = "not_german_legal_text"
	// CodeUnrecognisableNormToken means the norm for a cited_norms facet cannot be read (422).
	CodeUnrecognisableNormToken = "unrecognisable_norm_token" //nolint:gosec // an error code of the API, not a credential
	// CodeInvalidReference means the norm reference cannot be read (422).
	CodeInvalidReference = "invalid_reference"
	// CodeRangeNotSupported means a range of norms was given (422).
	CodeRangeNotSupported = "range_not_supported"
	// CodeHistoricalVersionUnavailable means an old version ("a.F.") was asked (422).
	CodeHistoricalVersionUnavailable = "historical_version_unavailable"
	// CodeAmbiguous means the norm reference fits more than one statute (422).
	CodeAmbiguous = "ambiguous"
	// CodeSectionsUnavailable means the decision has no named sections (422).
	CodeSectionsUnavailable = "sections_unavailable"
	// CodeSectionNotFound means the decision has no such section (422).
	CodeSectionNotFound = "section_not_found"
	// CodeRnRangeNotFound means the Randnummer range is outside the decision (422).
	CodeRnRangeNotFound = "rn_range_not_found"
	// CodeOfficialHeadnoteNotLocated means the headnote could not be located (422).
	CodeOfficialHeadnoteNotLocated = "official_headnote_not_located"
	// CodeDocumentLeitsatzNotLocated means the Leitsatz could not be located (422).
	CodeDocumentLeitsatzNotLocated = "document_leitsatz_not_located"
)

// Codes that the SDK sets itself. They are not part of the closed vocabulary of the API.
const (
	// CodeUnknown marks an error body that is not a problem document.
	CodeUnknown = "unknown"
	// CodeClientValidation marks a call that the client refused before it sent a request.
	CodeClientValidation = "client_validation"
	// CodeInvalidResponse marks a success answer whose body does not fit the SDK types.
	CodeInvalidResponse = "invalid_response"
	// CodeWaitTimeout marks a WaitUntilReady call that ran out of time.
	CodeWaitTimeout = "wait_timeout"
	// CodeConfiguration marks a client that cannot be built or a call with an invalid option.
	CodeConfiguration = "configuration_error"
)
