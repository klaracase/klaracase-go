// Command smoke calls the two free endpoints of the Klaracase API with a real token.
// It needs KLARACASE_API_TOKEN, costs no credits and is not part of CI. Run it with "make smoke".
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/klaracase/klaracase-go"
)

func main() {
	client, err := klaracase.NewFromEnv(klaracase.WithUserAgentSuffix("klaracase-go-smoke"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cov, err := client.Corpus.Coverage(ctx, klaracase.CoverageParams{Top: klaracase.Int(3)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "coverage:", err)
		os.Exit(1)
	}
	fmt.Printf("coverage: ok, status %d, api release %s, %d decisions, %d courts\n",
		cov.Meta.Status, cov.Meta.APIRelease, cov.TotalDecisions.Value, cov.CourtsTotal)

	fac, err := client.Corpus.Facets(ctx, klaracase.FacetCourtName, klaracase.FacetsParams{Prefix: "BG"})
	if err != nil {
		fmt.Fprintln(os.Stderr, "facets:", err)
		os.Exit(1)
	}
	fmt.Printf("facets: ok, status %d, %d values\n", fac.Meta.Status, fac.TotalValues)
}
