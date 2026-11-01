package klaracase

import (
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"time"
)

// ResponseMeta describes the HTTP answer behind a response.
type ResponseMeta struct {
	// Status is the HTTP status (200, or 202 for an upload).
	Status int
	// APIRelease is the X-Klaracase-Api-Release header, or "" when the header was absent.
	APIRelease string
	// Duration is the time of the final HTTP exchange, without the time spent in retries.
	Duration time.Duration
}

// Credits is the credit accounting the API adds to every answer. For a superadmin the call is
// not metered and Cost and CreditsRemaining are nil.
type Credits struct {
	// Metered is false when the call was not billed.
	Metered bool `json:"metered"`
	// Cost is the credits this call cost, 0 for a free call or a repeat in the same month.
	Cost *int `json:"cost"`
	// CreditsRemaining is the balance of the organization after the call.
	CreditsRemaining *int `json:"credits_remaining"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// Envelope is embedded in every response. It holds the keys that every answer carries and the
// parts the SDK adds: Meta, Extra and Raw. Because it is embedded, you read res.Credits,
// res.Meta and res.Extra directly.
type Envelope struct {
	// APIVersion is the api_version key of the body.
	APIVersion string `json:"api_version"`
	// ChangelogURL is the changelog_url key of the body.
	ChangelogURL string `json:"changelog_url"`
	// Credits is the credit accounting of the call. It is nil when the body has no credits key.
	Credits *Credits `json:"credits"`
	// Meta describes the HTTP answer. It is not part of the JSON body.
	Meta ResponseMeta `json:"-"`
	// Extra holds every top level key of the body that this SDK does not model, unchanged.
	Extra map[string]json.RawMessage `json:"-"`
	// Raw is the body exactly as the server sent it. Use it for a lossless copy.
	Raw json.RawMessage `json:"-"`
}

func (e *Envelope) setEnvelope(meta ResponseMeta, raw []byte) {
	e.Meta = meta
	e.Raw = append(json.RawMessage(nil), raw...)
}

// envelope lets code in this package reach the Envelope of any response.
func (e *Envelope) envelope() *Envelope { return e }

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (c *Credits) UnmarshalJSON(b []byte) error {
	type plain Credits
	return decodeWithExtra(b, (*plain)(c), &c.Extra)
}

// knownKeysCache maps a struct type to the set of JSON keys it models.
var knownKeysCache sync.Map

// knownKeys returns the JSON keys that struct type t models, looking into embedded structs.
func knownKeys(t reflect.Type) map[string]struct{} {
	if cached, ok := knownKeysCache.Load(t); ok {
		return cached.(map[string]struct{})
	}
	keys := map[string]struct{}{}
	for i := range t.NumField() {
		f := t.Field(i)
		tag, hasTag := f.Tag.Lookup("json")
		name, _, _ := strings.Cut(tag, ",")
		if name == "-" {
			continue
		}
		if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct {
			for k := range knownKeys(f.Type) {
				keys[k] = struct{}{}
			}
			continue
		}
		if !f.IsExported() {
			continue
		}
		if !hasTag || name == "" {
			name = f.Name
		}
		keys[name] = struct{}{}
	}
	knownKeysCache.Store(t, keys)
	return keys
}

// decodeWithExtra decodes data into target (a pointer to a method-free copy of the real
// type) and stores every key that the type does not model in *extra.
func decodeWithExtra(data []byte, target any, extra *map[string]json.RawMessage) error {
	if err := json.Unmarshal(data, target); err != nil {
		return err
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(data, &all); err != nil {
		return err
	}
	known := knownKeys(reflect.TypeOf(target).Elem())
	for k := range known {
		delete(all, k)
	}
	if len(all) == 0 {
		all = nil
	}
	*extra = all
	return nil
}
