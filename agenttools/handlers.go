package agenttools

import (
	"context"
	"encoding/json"

	"github.com/klaracase/klaracase-go"
)

// fortgeltungNote is the warning the API attaches to every decision. It is repeated once per
// result so a model passes it on to the user.
const fortgeltungNote = "The API does not check whether a decision is still good law. A decision may have been overturned or replaced. Check this before you rely on it."

// credits is the part of the credit accounting that helps a model plan its next calls.
type credits struct {
	Cost             *int64     `json:"cost,omitempty"`
	CreditsRemaining *int64     `json:"credits_remaining,omitempty"`
	APIBudget        *apiBudget `json:"api_budget,omitempty"`
}

// apiBudget is the prepaid API budget, present only when the organization has one or may load one.
type apiBudget struct {
	Cents              int `json:"cents"`
	RateCentsPerCredit int `json:"rate_cents_per_credit"`
	CreditsAtRate      int `json:"credits_at_rate"`
}

func creditsOf(e *klaracase.Envelope) *credits {
	cost, okCost := e.Cost()
	left, okLeft := e.CreditsRemaining()
	var budget *apiBudget
	if e.Credits != nil && e.Credits.APIBudget != nil {
		b := e.Credits.APIBudget
		budget = &apiBudget{Cents: b.Cents, RateCentsPerCredit: b.RateCentsPerCredit, CreditsAtRate: b.CreditsAtRate}
	}
	if !okCost && !okLeft && budget == nil {
		return nil
	}
	out := &credits{APIBudget: budget}
	if okCost {
		out.Cost = &cost
	}
	if okLeft {
		out.CreditsRemaining = &left
	}
	return out
}

type decisionHit struct {
	ID           string `json:"id"`
	Aktenzeichen string `json:"aktenzeichen"`
	Court        string `json:"court"`
	Date         string `json:"date"`
	DocumentType string `json:"document_type"`
	Title        string `json:"title"`
	Snippet      string `json:"snippet"`
	ChunkTitle   string `json:"chunk_title,omitempty"`
	RnRange      string `json:"rn_range,omitempty"`
	Fundstelle   string `json:"fundstelle,omitempty"`
	Confidence   string `json:"confidence,omitempty"`
}

func slimDecisionHits(in []klaracase.DecisionHit) []decisionHit {
	out := make([]decisionHit, 0, len(in))
	for _, h := range in {
		hit := decisionHit{
			ID: h.ID, Aktenzeichen: h.Aktenzeichen, Court: h.Court, Date: string(h.Date),
			DocumentType: string(h.DocumentType), Title: h.Title, Snippet: h.Snippet, Confidence: string(h.Confidence),
		}
		if h.ChunkTitle != nil {
			hit.ChunkTitle = *h.ChunkTitle
		}
		if h.RnRange != nil {
			hit.RnRange = *h.RnRange
		}
		if h.Fundstelle != nil {
			hit.Fundstelle = *h.Fundstelle
		}
		out = append(out, hit)
	}
	return out
}

type count struct {
	Value   int  `json:"value"`
	IsFloor bool `json:"is_floor"`
}

func countOf(c *klaracase.CountValue) *count {
	if c == nil {
		return nil
	}
	return &count{Value: c.Value, IsFloor: c.IsFloor}
}

func searchDecisions(ctx context.Context, c *klaracase.Client, input json.RawMessage) (any, error) {
	var p klaracase.SearchDecisionsParams
	if err := decodeInput(input, &p); err != nil {
		return nil, err
	}
	res, err := c.Decisions.Search(ctx, p)
	if err != nil {
		return nil, err
	}
	return struct {
		Results        []decisionHit `json:"results"`
		TotalAvailable *count        `json:"total_available,omitempty"`
		Page           int           `json:"page,omitempty"`
		Truncated      bool          `json:"truncated"`
		Note           string        `json:"fortgeltung_note"`
		Credits        *credits      `json:"credits,omitempty"`
	}{slimDecisionHits(res.Results), countOf(res.TotalAvailable), res.Page, res.Truncated, fortgeltungNote, creditsOf(&res.Envelope)}, nil
}

type lawHit struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Abbreviation string `json:"abbreviation"`
	Jurisdiction string `json:"jurisdiction"`
	Snippet      string `json:"snippet"`
	Confidence   string `json:"confidence,omitempty"`
}

func crossTypeSearch(ctx context.Context, c *klaracase.Client, input json.RawMessage) (any, error) {
	var p klaracase.CrossTypeSearchParams
	if err := decodeInput(input, &p); err != nil {
		return nil, err
	}
	res, err := c.Search(ctx, p)
	if err != nil {
		return nil, err
	}
	laws := []lawHit{}
	decisions := []decisionHit{}
	if res.Sections != nil {
		for _, l := range res.Sections.Laws {
			laws = append(laws, lawHit{
				ID: l.ID, Title: l.Title, Abbreviation: l.Abbreviation, Jurisdiction: string(l.Jurisdiction),
				Snippet: l.Snippet, Confidence: string(l.Confidence),
			})
		}
		decisions = slimDecisionHits(res.Sections.Decisions)
	}
	return struct {
		Laws           []lawHit      `json:"laws"`
		Decisions      []decisionHit `json:"decisions"`
		TotalAvailable *count        `json:"total_available,omitempty"`
		Truncated      bool          `json:"truncated"`
		Note           string        `json:"fortgeltung_note"`
		Credits        *credits      `json:"credits,omitempty"`
	}{laws, decisions, countOf(res.TotalAvailable), res.Truncated, fortgeltungNote, creditsOf(&res.Envelope)}, nil
}

// getDecisionInput is the input of get_decision: the id and the optional slice parameters.
type getDecisionInput struct {
	ID string `json:"id"`
	klaracase.GetDecisionParams
}

func getDecision(ctx context.Context, c *klaracase.Client, input json.RawMessage) (any, error) {
	var in getDecisionInput
	if err := decodeInput(input, &in); err != nil {
		return nil, err
	}
	if in.MaxChars == 0 {
		in.MaxChars = DefaultMaxChars
	}
	res, err := c.Decisions.Get(ctx, in.ID, in.GetDecisionParams)
	if err != nil {
		return nil, err
	}
	out := struct {
		ID                string   `json:"id"`
		Aktenzeichen      string   `json:"aktenzeichen"`
		Court             string   `json:"court"`
		Date              string   `json:"date"`
		DocumentType      string   `json:"document_type,omitempty"`
		ECLI              string   `json:"ecli,omitempty"`
		Text              string   `json:"text"`
		TextLength        int      `json:"text_length"`
		Truncated         bool     `json:"truncated"`
		AvailableSections []string `json:"available_sections,omitempty"`
		RnSource          string   `json:"rn_source,omitempty"`
		Note              string   `json:"fortgeltung_note"`
		Credits           *credits `json:"credits,omitempty"`
	}{
		ID: res.ID, Aktenzeichen: res.Aktenzeichen, Court: res.Court, Date: string(res.Date),
		Text: res.FullText, TextLength: res.TextLength, Truncated: res.Truncated,
		AvailableSections: res.AvailableSections, RnSource: string(res.RnSource),
		Note: fortgeltungNote, Credits: creditsOf(&res.Envelope),
	}
	if res.Metadata != nil {
		out.DocumentType = string(res.Metadata.DocumentType)
		if res.Metadata.ECLI != nil {
			out.ECLI = *res.Metadata.ECLI
		}
	}
	return out, nil
}

func getNorm(ctx context.Context, c *klaracase.Client, input json.RawMessage) (any, error) {
	var in struct {
		Reference string `json:"reference"`
		klaracase.GetNormParams
	}
	if err := decodeInput(input, &in); err != nil {
		return nil, err
	}
	res, err := c.Norms.Get(ctx, in.Reference, in.GetNormParams)
	if err != nil {
		return nil, err
	}
	out := struct {
		Reference          string   `json:"reference"`
		RequestedReference string   `json:"requested_reference,omitempty"`
		Law                string   `json:"law,omitempty"`
		Number             string   `json:"number,omitempty"`
		Title              string   `json:"title,omitempty"`
		Text               string   `json:"text,omitempty"`
		LastAmendment      string   `json:"last_amendment_date,omitempty"`
		Previous           string   `json:"previous,omitempty"`
		Next               string   `json:"next,omitempty"`
		Credits            *credits `json:"credits,omitempty"`
	}{Reference: res.Reference, RequestedReference: res.RequestedReference, Credits: creditsOf(&res.Envelope)}
	if res.Law != nil {
		out.Law = res.Law.OfficialName
	}
	if res.Norm != nil {
		out.Number, out.Title, out.Text = res.Norm.Number, res.Norm.Title, res.Norm.Text
	}
	if res.Version != nil {
		out.LastAmendment = string(res.Version.LastAmendmentDate)
	}
	if res.Neighbours != nil {
		if res.Neighbours.Previous != nil {
			out.Previous = *res.Neighbours.Previous
		}
		if res.Neighbours.Next != nil {
			out.Next = *res.Neighbours.Next
		}
	}
	return out, nil
}

type finding struct {
	ID          string `json:"id"`
	Canonical   string `json:"canonical"`
	Verdict     string `json:"verdict"`
	RawText     string `json:"raw_text"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
}

func checkCitations(ctx context.Context, c *klaracase.Client, input json.RawMessage) (any, error) {
	var p klaracase.CreateCitationCheckParams
	if err := decodeInput(input, &p); err != nil {
		return nil, err
	}
	run, err := c.CitationChecks.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	findings := make([]finding, 0, len(run.Findings))
	for _, f := range run.Findings {
		findings = append(findings, finding{
			ID: f.ID, Canonical: f.Canonical, Verdict: string(f.Verdict), RawText: f.RawText,
			StartOffset: f.StartOffset, EndOffset: f.EndOffset,
		})
	}
	unique := run.UniqueCitations
	if unique == nil {
		unique = []klaracase.UniqueCitation{}
	}
	return struct {
		ID              string                     `json:"id"`
		Status          string                     `json:"status"`
		VerdictCounts   map[klaracase.Verdict]int  `json:"verdict_counts"`
		UniqueCitations []klaracase.UniqueCitation `json:"unique_citations"`
		Findings        []finding                  `json:"findings"`
		Note            string                     `json:"note"`
		Credits         *credits                   `json:"credits,omitempty"`
	}{run.ID, string(run.Status), run.VerdictCounts, unique, findings,
		verifiedMeaning + " " + notFoundMeaning, creditsOf(&run.Envelope)}, nil
}
