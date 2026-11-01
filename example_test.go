package klaracase_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/klaracase/klaracase-go"
)

// The token comes from the KLARACASE_API_TOKEN environment variable.
func ExampleNewFromEnv() {
	client, err := klaracase.NewFromEnv(
		klaracase.WithTimeout(20*time.Second),
		klaracase.WithMaxRetries(3),
		klaracase.WithUserAgentSuffix("my-app/1.0"),
	)
	if err != nil {
		log.Fatal(err)
	}
	_ = client
}

func ExampleDecisionsService_Search() {
	client, err := klaracase.New("klc_your_token_here")
	if err != nil {
		log.Fatal(err)
	}
	res, err := client.Decisions.Search(context.Background(), klaracase.SearchDecisionsParams{
		Query: "Vermögensbetreuungspflicht Untreue",
		Court: "BGH",
		Limit: 5,
		// Bridge false is sent as bridge=false: the server adds no register terms.
		Bridge: klaracase.Bool(false),
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, hit := range res.Results {
		fmt.Println(hit.Court, hit.Aktenzeichen, hit.Date)
	}
	if left, ok := res.CreditsRemaining(); ok {
		fmt.Println("credits left:", left)
	}
}

func ExampleDecisionsService_Get() {
	client, _ := klaracase.New("klc_your_token_here")
	// The id may be a UUID, an Aktenzeichen or an ECLI. The client encodes it as one path segment.
	res, err := client.Decisions.Get(context.Background(), "VIII ZR 271/17", klaracase.GetDecisionParams{
		Section: klaracase.SectionGruende,
		Court:   "BGH",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Aktenzeichen, res.Metadata.DocumentType, res.TextLength)
	fmt.Println(res.FortgeltungNote) // show this note to your user
}

func ExampleNormsService_Get() {
	client, _ := klaracase.New("klc_your_token_here")
	res, err := client.Norms.Get(context.Background(), "Art. 6 DSGVO", klaracase.GetNormParams{Jurisdiction: klaracase.JurisdictionEU})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(res.Norm.Number, res.Norm.Title)
}

func ExampleCitationChecksService_Create() {
	client, _ := klaracase.New("klc_your_token_here")
	ctx := context.Background()
	text, err := os.ReadFile("schriftsatz.txt")
	if err != nil {
		log.Fatal(err)
	}
	run, err := client.CitationChecks.Create(ctx, klaracase.CreateCitationCheckParams{
		Text:          string(text),
		ReferenceDate: "2026-10-01",
	})
	if err != nil {
		log.Fatal(err)
	}
	for _, f := range run.Findings {
		switch f.Verdict {
		case klaracase.VerdictVerifiedCorpus:
			// The citation exists in the corpus. This does not prove that a quote or a legal claim is right.
			fmt.Println("found:", f.Canonical)
		case klaracase.VerdictNotFound:
			// Not found in the corpus. This does not mean the citation is invented. Check it by hand.
			fmt.Println("not found in the corpus:", f.RawText)
		default:
			fmt.Println(f.Verdict, f.Canonical)
		}
	}
}

func ExampleCitationCheckDocumentsService_WaitUntilReady() {
	client, _ := klaracase.New("klc_your_token_here")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	doc, err := client.CitationCheckDocuments.Upload(ctx, klaracase.UploadParams{
		File:        strings.NewReader("... file bytes ..."),
		Filename:    "schriftsatz.pdf",
		ContentType: "application/pdf",
	})
	if err != nil {
		log.Fatal(err)
	}
	// The context is the only deadline of the wait.
	ready, err := client.CitationCheckDocuments.WaitUntilReady(ctx, doc.ID, klaracase.WithPollInterval(2*time.Second))
	var failed *klaracase.DocumentFailedError
	if errors.As(err, &failed) {
		log.Fatalf("the file cannot be read: %s", failed.ErrorCode)
	}
	if err != nil {
		log.Fatal(err)
	}
	if ready.QuotedCredits != nil {
		fmt.Println("the check costs", *ready.QuotedCredits, "credits")
	}
	run, err := client.CitationChecks.Create(ctx, klaracase.CreateCitationCheckParams{DocumentID: ready.ID})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(run.VerdictCounts)
}

func Example_errorHandling() {
	client, _ := klaracase.New("klc_your_token_here")
	_, err := client.Decisions.Search(context.Background(), klaracase.SearchDecisionsParams{Query: "Untreue"})

	var credits *klaracase.InsufficientCreditsError
	var limited *klaracase.RateLimitError
	var notFound *klaracase.NotFoundError
	switch {
	case err == nil:
	case errors.As(err, &credits):
		fmt.Printf("this call costs %d credits, the balance is %d\n", credits.Cost, credits.Balance)
	case errors.As(err, &limited):
		fmt.Printf("try again in %d seconds\n", limited.RetryAfterSeconds)
	case errors.As(err, &notFound):
		fmt.Println("nothing found:", notFound.Detail)
	default:
		if apiErr, ok := klaracase.AsError(err); ok {
			fmt.Println(apiErr.Status, apiErr.Code, apiErr.APIRelease)
		}
		var cfg *klaracase.ConfigurationError
		if errors.As(err, &cfg) {
			fmt.Println("setup problem:", cfg.Message)
		}
	}
}

func ExampleWithDeprecationHandler() {
	client, _ := klaracase.New("klc_your_token_here",
		klaracase.WithDeprecationHandler(func(n klaracase.DeprecationNotice) {
			log.Printf("%s is deprecated, sunset %s, see %s", n.Endpoint, n.Sunset, n.Link)
		}),
	)
	_ = client
}

func ExampleCorpusService_Facets() {
	client, _ := klaracase.New("klc_your_token_here")
	res, err := client.Corpus.Facets(context.Background(), klaracase.FacetCitedNorms, klaracase.FacetsParams{Norm: "§ 543 BGB"})
	if err != nil {
		log.Fatal(err)
	}
	for _, v := range res.Values {
		fmt.Println(v.Key, v.Count.Value)
	}
}
