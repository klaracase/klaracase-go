package klaracase

import (
	"errors"
	"time"
)

const (
	dateLayout      = "2006-01-02"
	yearMonthLayout = "2006-01"
)

// Date is a calendar date in the form YYYY-MM-DD, exactly as the API writes it. It is a string,
// so a date you read from a response and a date you send are the same value, and the empty
// Date means "not set". Use [NewDate] to build one from a time.Time and [Date.Time] to read it.
type Date string

// NewDate returns the calendar date of t in t's own time zone.
func NewDate(t time.Time) Date { return Date(t.Format(dateLayout)) }

// Time parses d as a UTC midnight. It returns an error when d is not exactly YYYY-MM-DD.
func (d Date) Time() (time.Time, error) {
	if len(d) != len(dateLayout) {
		return time.Time{}, errors.New("klaracase: a date must have the form YYYY-MM-DD")
	}
	t, err := time.Parse(dateLayout, string(d))
	if err != nil {
		return time.Time{}, errors.New("klaracase: a date must have the form YYYY-MM-DD")
	}
	return t, nil
}

// String returns d unchanged.
func (d Date) String() string { return string(d) }

// YearMonth is a month in the form YYYY-MM, as the corpus coverage report writes it.
type YearMonth string

// Time parses m as midnight UTC on the first day of the month. It returns an error when m is
// not exactly YYYY-MM.
func (m YearMonth) Time() (time.Time, error) {
	if len(m) != len(yearMonthLayout) {
		return time.Time{}, errors.New("klaracase: a month must have the form YYYY-MM")
	}
	t, err := time.Parse(yearMonthLayout, string(m))
	if err != nil {
		return time.Time{}, errors.New("klaracase: a month must have the form YYYY-MM")
	}
	return t, nil
}

// String returns m unchanged.
func (m YearMonth) String() string { return string(m) }
