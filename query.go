package klaracase

import (
	"net/url"
	"strconv"
	"strings"
)

// query builds a query string in the order the parameters were added. A parameter that is
// empty, nil or an empty list is not added. A false bool and a zero int are values and are added.
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

func (q *query) intp(name string, value *int) {
	if value != nil {
		q.add(name, strconv.Itoa(*value))
	}
}

func (q *query) boolp(name string, value *bool) {
	if value != nil {
		q.add(name, strconv.FormatBool(*value))
	}
}

// list adds name[]=a&name[]=b, the form that PHP and Laravel read as an array.
func (q *query) list(name string, values []string) {
	for _, v := range values {
		q.add(name+"[]", v)
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

// Int returns a pointer to n. Use it to set an optional integer parameter, including zero.
func Int(n int) *int { return &n }

// Bool returns a pointer to b. Use it to set an optional boolean parameter, including false.
func Bool(b bool) *bool { return &b }

// String returns a pointer to s. It is a convenience for optional string values.
func String(s string) *string { return &s }
