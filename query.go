package klaracase

import (
	"net/url"
	"strconv"
	"strings"
)

// query builds a query string in the order the parameters were added. A parameter that is
// empty, zero, nil or an empty list is not added. A *bool that points to false is a value and
// is added.
type query struct {
	pairs []string
}

func encodeQueryComponent(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

func (q *query) add(name, value string) {
	q.pairs = append(q.pairs, encodeQueryComponent(name)+"="+encodeQueryComponent(value))
}

func (q *query) str(name, value string) {
	if value != "" {
		q.add(name, value)
	}
}

// num adds an integer parameter. Zero means "not set", because no parameter of the API
// accepts zero as a value.
func (q *query) num(name string, value int) {
	if value != 0 {
		q.add(name, strconv.Itoa(value))
	}
}

func (q *query) boolp(name string, value *bool) {
	if value != nil {
		q.add(name, strconv.FormatBool(*value))
	}
}

// addList adds name[]=a&name[]=b, the form that PHP and Laravel read as an array.
func addList[S ~string](q *query, name string, values []S) {
	for _, v := range values {
		q.add(name+"[]", string(v))
	}
}

func (q query) encode() string {
	return strings.Join(q.pairs, "&")
}

// escapeSegment encodes a path parameter as one segment. Every byte outside the unreserved
// set of RFC 3986 (A-Z a-z 0-9 - . _ ~) becomes %XX with upper case hex, so a space is %20
// and a slash is %2F. The URL package would leave some characters alone, so it is not used.
func escapeSegment(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		ch := s[i]
		switch {
		case ch >= 'A' && ch <= 'Z', ch >= 'a' && ch <= 'z', ch >= '0' && ch <= '9',
			ch == '-', ch == '.', ch == '_', ch == '~':
			b.WriteByte(ch)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[ch>>4])
			b.WriteByte(hex[ch&0x0f])
		}
	}
	return b.String()
}

// Ptr returns a pointer to v. Use it for an optional value that needs an address, such as
// the Bridge, IncludeCitations and FullText parameters, where false must stay different from
// "not set".
func Ptr[T any](v T) *T { return &v }

// Int returns a pointer to n. It is Ptr for an int.
//
// Since Go 1.26 the built in new accepts an expression, so new(5) does the same and these
// helpers are optional there. This module still supports Go 1.24.
func Int(n int) *int { return &n }

// Bool returns a pointer to b. Use it to send false for an optional boolean parameter. It is
// Ptr for a bool. With Go 1.26 or newer you can write new(false) instead.
func Bool(b bool) *bool { return &b }

// String returns a pointer to s. It is Ptr for a string. With Go 1.26 or newer you can write
// new("text") instead.
func String(s string) *string { return &s }
