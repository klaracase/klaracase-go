package klaracase

import (
	"fmt"
	"sort"
	"strings"
)

// fieldErrors collects the problems found by the checks that need no server: a missing
// required value, a malformed date, a range that runs backwards. The keys are the wire names
// of the fields, so a result printed for a person or a language model names the field the
// same way the API does.
type fieldErrors map[string][]string

func (f fieldErrors) add(field, format string, args ...any) {
	f[field] = append(f[field], fmt.Sprintf(format, args...))
}

// date checks the form of an optional date.
func (f fieldErrors) date(field string, d Date) {
	if d == "" {
		return
	}
	if _, err := d.Time(); err != nil {
		f.add(field, "%q is not a date in the form YYYY-MM-DD", string(d))
	}
}

// dateRange checks that from is not after to. A malformed date is reported by date, not here.
func (f fieldErrors) dateRange(fromField string, from Date, toField string, to Date) {
	if from == "" || to == "" {
		return
	}
	a, errA := from.Time()
	b, errB := to.Time()
	if errA != nil || errB != nil {
		return
	}
	if a.After(b) {
		f.add(fromField, "%s %s is after %s %s", fromField, from, toField, to)
	}
}

// err returns nil when nothing was collected, otherwise a *ValidationError with the code
// "client_validation", status 0 and the messages in Errors.
func (f fieldErrors) err() error {
	if len(f) == 0 {
		return nil
	}
	names := make([]string, 0, len(f))
	for name := range f {
		names = append(names, name)
	}
	sort.Strings(names)
	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+": "+strings.Join(f[name], "; "))
	}
	return &ValidationError{
		Problem: Problem{Code: CodeClientValidation, Detail: strings.Join(parts, ". ")},
		Errors:  map[string][]string(f),
	}
}

// clientValidation builds the error for a call that is wrong before any request is sent.
func clientValidation(field, format string, args ...any) error {
	f := fieldErrors{}
	f.add(field, format, args...)
	return f.err()
}

// The checks below cover only what the API itself would refuse for a reason that does not
// depend on the server: a missing required value, a malformed date, a range that runs
// backwards, a list of the wrong size. They never check enum members, number ranges or text
// length. The server owns those and may change them.

func validateSearchDecisions(p SearchDecisionsParams) error {
	f := fieldErrors{}
	if p.Query == "" {
		f.add("query", "Query is required")
	}
	f.date("date_from", p.DateFrom)
	f.date("date_to", p.DateTo)
	f.dateRange("date_from", p.DateFrom, "date_to", p.DateTo)
	return f.err()
}

func validateCrossTypeSearch(p CrossTypeSearchParams) error {
	f := fieldErrors{}
	if p.Query == "" {
		f.add("query", "Query is required")
	}
	f.date("date_from", p.DateFrom)
	f.date("date_to", p.DateTo)
	f.dateRange("date_from", p.DateFrom, "date_to", p.DateTo)
	return f.err()
}

func validateGetDecision(id string, p GetDecisionParams) error {
	f := fieldErrors{}
	if id == "" {
		f.add("id", "id is required")
	}
	f.date("date", p.Date)
	if p.RnFrom != 0 && p.RnTo != 0 && p.RnFrom > p.RnTo {
		f.add("rn_from", "rn_from %d is after rn_to %d", p.RnFrom, p.RnTo)
	}
	return f.err()
}

func validateCitations(id string, p CitationsParams) error {
	f := fieldErrors{}
	if id == "" {
		f.add("id", "id is required")
	}
	f.date("date", p.Date)
	return f.err()
}

// maxDatapackageIDs is the most decisions one datapackage holds.
const maxDatapackageIDs = 10

func validateDatapackage(p CreateDatapackageParams) error {
	f := fieldErrors{}
	switch {
	case len(p.IDs) == 0:
		f.add("ids", "IDs is required: pass 1 to %d ids", maxDatapackageIDs)
	case len(p.IDs) > maxDatapackageIDs:
		f.add("ids", "IDs holds %d ids, the most is %d", len(p.IDs), maxDatapackageIDs)
	}
	for i, id := range p.IDs {
		if strings.TrimSpace(id) == "" {
			f.add("ids", "id %d is empty", i+1)
		}
	}
	return f.err()
}

func validateCitationCheck(p CreateCitationCheckParams) error {
	f := fieldErrors{}
	if (p.Text == "") == (p.DocumentID == "") {
		f.add("text", "set exactly one of Text and DocumentID")
	}
	f.date("reference_date", p.ReferenceDate)
	return f.err()
}
