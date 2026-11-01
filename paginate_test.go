package klaracase

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync/atomic"
	"testing"
)

// searchPageHandler serves pages of two hits each. total is the number of pages that have hits;
// pool is the candidate_pool the answer states.
func searchPageHandler(total, pool int, pages *[]int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		*pages = append(*pages, page)
		hits := ""
		if page <= total {
			hits = fmt.Sprintf(`{"aktenzeichen":"az-%d-a"},{"aktenzeichen":"az-%d-b"}`, page, page)
		}
		_, _ = io.WriteString(w, fmt.Sprintf(`{"results":[%s],"page":%d,"limit":2,"candidate_pool":%d}`, hits, page, pool))
	}
}

func collectSearch(t *testing.T, c *Client, p SearchDecisionsParams, maxPages int) ([]string, error) {
	t.Helper()
	var got []string
	var last error
	for hit, err := range c.Decisions.SearchAll(context.Background(), p, maxPages) {
		if err != nil {
			last = err
			continue
		}
		got = append(got, hit.Aktenzeichen)
	}
	return got, last
}

func TestSearchAllStopsOnAnEmptyPage(t *testing.T) {
	var pages []int
	c, _ := newTestClient(t, searchPageHandler(2, 0, &pages))
	got, err := collectSearch(t, c, SearchDecisionsParams{Query: "q"}, 8)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[0] != "az-1-a" || got[3] != "az-2-b" {
		t.Errorf("hits %v", got)
	}
	if fmt.Sprint(pages) != "[1 2 3]" {
		t.Errorf("pages requested %v, want [1 2 3]: the empty page 3 ends the loop", pages)
	}
}

func TestSearchAllRespectsMaxPagesAndTheServedCap(t *testing.T) {
	var pages []int
	c, _ := newTestClient(t, searchPageHandler(100, 0, &pages))
	if got, _ := collectSearch(t, c, SearchDecisionsParams{Query: "q"}, 2); len(got) != 4 || fmt.Sprint(pages) != "[1 2]" {
		t.Errorf("maxPages 2: hits %d pages %v", len(got), pages)
	}
	pages = nil
	collectSearch(t, c, SearchDecisionsParams{Query: "q"}, 500)
	if len(pages) != 10 || pages[9] != 10 {
		t.Errorf("a huge maxPages is clamped to page 10, requested %v", pages)
	}
	pages = nil
	collectSearch(t, c, SearchDecisionsParams{Query: "q", Page: 8}, 500)
	if fmt.Sprint(pages) != "[8 9 10]" {
		t.Errorf("starting at page 8 stops at 10, requested %v", pages)
	}
}

func TestSearchAllStopsWhenThePoolIsUsedUp(t *testing.T) {
	var pages []int
	c, _ := newTestClient(t, searchPageHandler(100, 4, &pages)) // limit 2, pool 4: two pages hold everything
	got, _ := collectSearch(t, c, SearchDecisionsParams{Query: "q"}, 10)
	if len(got) != 4 || fmt.Sprint(pages) != "[1 2]" {
		t.Errorf("hits %d pages %v", len(got), pages)
	}
}

func TestSearchAllNeedsMaxPagesAndYieldsAnErrorOnce(t *testing.T) {
	var calls atomic.Int32
	c, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("page") == "2" {
			writeProblem(w, 402, `{"status":402,"code":"insufficient_credits","cost":1,"balance":0}`)
			return
		}
		_, _ = io.WriteString(w, `{"results":[{"aktenzeichen":"a"}],"page":1,"limit":1}`)
	}, WithMaxRetries(0))
	got, err := collectSearch(t, c, SearchDecisionsParams{Query: "q"}, 5)
	if len(got) != 1 || !IsInsufficientCredits(err) || calls.Load() != 2 {
		t.Errorf("hits %v err %v calls %d", got, err, calls.Load())
	}
	calls.Store(0)
	for range []int{0, -1} {
		_, err = collectSearch(t, c, SearchDecisionsParams{Query: "q"}, 0)
		if !IsValidation(err) || calls.Load() != 0 {
			t.Errorf("maxPages 0: err %v calls %d", err, calls.Load())
		}
	}
	_, err = collectSearch(t, c, SearchDecisionsParams{}, 3)
	if !IsValidation(err) || calls.Load() != 0 {
		t.Errorf("a missing query is refused before any request: %v", err)
	}
}

func TestSearchAllBreakSendsNoFurtherRequest(t *testing.T) {
	var pages []int
	c, _ := newTestClient(t, searchPageHandler(100, 0, &pages))
	for hit, err := range c.Decisions.SearchAll(context.Background(), SearchDecisionsParams{Query: "q"}, 5) {
		if err != nil {
			t.Fatal(err)
		}
		if hit.Aktenzeichen == "az-1-a" {
			break
		}
	}
	if fmt.Sprint(pages) != "[1]" {
		t.Errorf("requested %v", pages)
	}
}

func citationsPageHandler(lastPage int, pages *[]int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		*pages = append(*pages, page)
		more := page < lastPage
		_, _ = io.WriteString(w, fmt.Sprintf(`{"references":[{"reference_canonical":"out-%d"}],"referenced_by":[{"reference_canonical":"in-%d"}],
			"pagination":{"page":%d,"has_more_references":%t,"has_more_referenced_by":%t}}`, page, page, page, more, more))
	}
}

func TestCitationsAllYieldsBothDirectionsAndStopsWhenNoPageIsLeft(t *testing.T) {
	var pages []int
	c, _ := newTestClient(t, citationsPageHandler(2, &pages))
	var got []string
	for edge, err := range c.Decisions.CitationsAll(context.Background(), "1 StR 185/01", CitationsParams{}, 9) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, edge.ReferenceCanonical)
	}
	if fmt.Sprint(got) != "[out-1 in-1 out-2 in-2]" || fmt.Sprint(pages) != "[1 2]" {
		t.Errorf("edges %v pages %v", got, pages)
	}
}

func TestCitationsAllClampsAndRefusesDepthTwo(t *testing.T) {
	var pages []int
	c, _ := newTestClient(t, citationsPageHandler(1000, &pages))
	for _, err := range c.Decisions.CitationsAll(context.Background(), "x", CitationsParams{Direction: DirectionOutgoing}, 500) {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(pages) != 20 || pages[19] != 20 {
		t.Errorf("clamp to page 20, requested %v", pages)
	}
	pages = nil
	for _, err := range c.Decisions.CitationsAll(context.Background(), "x", CitationsParams{Depth: 2}, 3) {
		if !IsValidation(err) {
			t.Errorf("depth 2 must be refused, got %v", err)
		}
	}
	for _, err := range c.Decisions.CitationsAll(context.Background(), "x", CitationsParams{}, 0) {
		if !IsValidation(err) {
			t.Errorf("maxPages 0 must be refused, got %v", err)
		}
	}
	if len(pages) != 0 {
		t.Errorf("refused calls sent %v", pages)
	}
}
