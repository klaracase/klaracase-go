package klaracase

import (
	"context"
	"iter"
)

const (
	// maxSearchPages is the last page of a decision search the API serves.
	maxSearchPages = 10
	// maxCitationPages is the last page of a citation graph the API serves.
	maxCitationPages = 20
)

// SearchAll returns an iterator over the hits of a decision search, page after page, from
// p.Page (page 1 when zero) on. Use it with range:
//
//	for hit, err := range client.Decisions.SearchAll(ctx, params, 3) {
//		if err != nil {
//			return err
//		}
//		fmt.Println(hit.Aktenzeichen)
//	}
//
// maxPages is required, at least 1, and is lowered to what the API serves (page 10 at most).
// Each page is a separate request, so a long iteration spends rate limit and may spend
// credits: the contract bills one credit per distinct query and filter set per month, and
// later pages of the same search should cost nothing more, but check Credits on a response
// before you rely on that. The iteration stops after maxPages pages, on an empty page, or when the
// candidate pool is used up. An error is yielded once, and then the iteration ends. Breaking
// out of the loop sends no further request.
func (s *DecisionsService) SearchAll(ctx context.Context, p SearchDecisionsParams, maxPages int, opts ...RequestOption) iter.Seq2[DecisionHit, error] {
	return func(yield func(DecisionHit, error) bool) {
		if maxPages < 1 {
			yield(DecisionHit{}, clientValidation("max_pages", "maxPages is required: pass at least 1"))
			return
		}
		page := max(p.Page, 1)
		budget := min(maxPages, maxSearchPages-page+1)
		for range budget {
			p.Page = page
			res, err := s.Search(ctx, p, opts...)
			if err != nil {
				yield(DecisionHit{}, err)
				return
			}
			for _, hit := range res.Results {
				if !yield(hit, nil) {
					return
				}
			}
			if len(res.Results) == 0 || poolUsedUp(res) {
				return
			}
			page++
		}
	}
}

// poolUsedUp reports whether the pages served so far cover the whole candidate pool.
func poolUsedUp(res *DecisionSearchResponse) bool {
	return res.CandidatePool > 0 && res.Limit > 0 && res.Page > 0 && res.Page*res.Limit >= res.CandidatePool
}

// CitationsAll returns an iterator over the citation edges of one decision, page after page,
// from p.Page (page 1 when zero) on. For each page it yields the outgoing edges (References),
// then the incoming edges (ReferencedBy). Set p.Direction to read one side only. Paging needs
// depth 1, so a p.Depth of 2 is refused.
//
// maxPages is required, at least 1, and is lowered to what the API serves (page 20 at most).
// Each page is a separate request that costs rate limit; the contract bills 2 credits once per
// decision and month, so later pages of the same decision should cost nothing more, but check
// Credits on a response before you rely on that. The iteration stops after maxPages
// pages, on an empty page, or when the answer says there is no further page. An error is yielded
// once, and then the iteration ends.
func (s *DecisionsService) CitationsAll(ctx context.Context, id string, p CitationsParams, maxPages int, opts ...RequestOption) iter.Seq2[CitationEdge, error] {
	return func(yield func(CitationEdge, error) bool) {
		if maxPages < 1 {
			yield(CitationEdge{}, clientValidation("max_pages", "maxPages is required: pass at least 1"))
			return
		}
		if p.Depth > 1 {
			yield(CitationEdge{}, clientValidation("depth", "paging needs depth 1, depth %d cannot be paged", p.Depth))
			return
		}
		page := max(p.Page, 1)
		budget := min(maxPages, maxCitationPages-page+1)
		for range budget {
			p.Page = page
			res, err := s.Citations(ctx, id, p, opts...)
			if err != nil {
				yield(CitationEdge{}, err)
				return
			}
			for _, edges := range [][]CitationEdge{res.References, res.ReferencedBy} {
				for _, edge := range edges {
					if !yield(edge, nil) {
						return
					}
				}
			}
			if len(res.References)+len(res.ReferencedBy) == 0 || noFurtherCitationPage(res) {
				return
			}
			page++
		}
	}
}

// noFurtherCitationPage reports whether the answer says that neither direction has another page.
func noFurtherCitationPage(res *CitationsResponse) bool {
	pg := res.Pagination
	return pg != nil && !pg.HasMoreReferences && !pg.HasMoreReferencedBy
}
