package klaracase

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Copy lint: banned wording must not appear in the README, CHANGELOG, manifest or source.

var forbiddenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)hallucinat`),
	regexp.MustCompile(`(?i)halluzin`),
	regexp.MustCompile(`(?i)fabricat`),
	regexp.MustCompile(`(?i)fabrizier`),
	regexp.MustCompile(`(?i)\bfake\b`),
	regexp.MustCompile(`(?i)official source`),
	regexp.MustCompile(`(?i)offizielle quelle`),
	regexp.MustCompile(`(?i)amtlich`),
}

// The required disclaimer says "fabricated" once. Only this exact sentence is exempt.
var (
	allowedSentence = regexp.MustCompile(`(?i)it is not a claim that the citation is fabricated\.`)
	commentMarks    = regexp.MustCompile(`(?m)^\s*(//|\*|#)\s?`)
)

func findForbidden(text string) []string {
	flat := strings.Join(strings.Fields(commentMarks.ReplaceAllString(text, "")), " ")
	cleaned := allowedSentence.ReplaceAllString(flat, "")
	var hits []string
	for _, pattern := range forbiddenPatterns {
		if pattern.MatchString(cleaned) {
			hits = append(hits, pattern.String())
		}
	}
	return hits
}

func TestCopyLintFindsNoForbiddenWording(t *testing.T) {
	files := []string{"README.md", "CHANGELOG.md", "go.mod"}
	goFiles, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	internal, err := filepath.Glob("internal/*/*.go")
	if err != nil {
		t.Fatal(err)
	}
	agenttools, err := filepath.Glob("agenttools/*.go")
	if err != nil {
		t.Fatal(err)
	}
	files = append(files, goFiles...)
	files = append(files, internal...)
	files = append(files, agenttools...)
	for _, file := range files {
		if file == "copylint_test.go" || strings.HasSuffix(file, "_test.go") && strings.HasPrefix(file, "agenttools/") {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, hit := range findForbidden(string(data)) {
			t.Errorf("%s: %s", file, hit)
		}
	}
}

func TestCopyLintFlagsEveryForbiddenPattern(t *testing.T) {
	samples := []string{
		"This stops hallucinations.",
		"Das verhindert Halluzinationen.",
		"The citation is Fabricated.",
		"Das ist fabriziert.",
		"A fake case.",
		"The official source of law.",
		"Die offizielle Quelle.",
		"Der amtliche Leitsatz.",
	}
	for _, sample := range samples {
		if len(findForbidden(sample)) == 0 {
			t.Errorf("not flagged: %q", sample)
		}
	}
}

func TestCopyLintAllowsTheRequiredDisclaimerOnly(t *testing.T) {
	sentence := "// A `not_found` verdict means the citation was not found in the corpus; it is not a\n// claim that the citation is fabricated."
	if hits := findForbidden(sentence); len(hits) != 0 {
		t.Errorf("disclaimer flagged: %v", hits)
	}
	if len(findForbidden(sentence+" It is fabricated.")) == 0 {
		t.Error("text around the disclaimer was not flagged")
	}
}
