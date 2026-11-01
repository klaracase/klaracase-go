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
//
// Plan credits are spent first. While the optional prepaid API budget is available, a call that
// the plan credits do not cover is paid from the budget, in whole and never in part. Cost counts
// both, and CreditsRemaining never counts the budget.
type Credits struct {
	// Metered is false when the call was not billed.
	Metered bool `json:"metered"`
	// Cost is the credits this call cost, plan credits and API budget together. It is 0 for a
	// free call or a repeat in the same credit period, and nil when Metered is false.
	Cost *int `json:"cost"`
	// CreditsRemaining is the plan credits the organization has left in the current credit
	// period after the call. It never counts the API budget. It is nil when Metered is false.
	CreditsRemaining *int `json:"credits_remaining"`
	// APIBudget is the prepaid API budget after the call. It is nil when the organization has
	// no budget and may not load one, and when the answer has no api_budget key. Test it for
	// nil before you read it; a nil value is not a budget of zero.
	APIBudget *APIBudget `json:"api_budget,omitempty"`
	// Extra holds unknown keys.
	Extra map[string]json.RawMessage `json:"-"`
}

// APIBudget is the prepaid API budget of an organization after a call. It pays for API and MCP
// calls once the plan credits are used up, and it does not expire. The web app never draws on it.
// The budget exists for the Professional and Enterprise plans.
type APIBudget struct {
	// Cents is the budget after the call, in euro cents. It can be negative after a payment was lost.
	Cents int `json:"cents"`
	// RateCentsPerCredit is what one credit costs from the budget, in whole euro cents. It is the
	// rate for the organization, which a contract can set apart from the list rate.
	RateCentsPerCredit int `json:"rate_cents_per_credit"`
	// CreditsAtRate is how many further credits the spendable part of the budget pays for at that
	// rate. It is 0 when the budget is empty or on hold.
	CreditsAtRate int `json:"credits_at_rate"`
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

// UnmarshalJSON implements json.Unmarshaler and keeps unknown keys in Extra.
func (b *APIBudget) UnmarshalJSON(data []byte) error {
	type plain APIBudget
	return decodeWithExtra(data, (*plain)(b), &b.Extra)
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

// nilIfNull turns the JSON literal null into nil. A json.RawMessage field reads a null key as
// the four bytes "null", which is an odd value to test for.
func nilIfNull(raw json.RawMessage) json.RawMessage {
	if string(raw) == "null" {
		return nil
	}
	return raw
}

// CreditsRemaining returns the plan credits the organization has left after this call and true.
// It never counts the API budget; read Credits.APIBudget for that. It returns false when the
// body carries no credits, or when the call was not metered (a superadmin), where the balance
// is null.
func (e *Envelope) CreditsRemaining() (int64, bool) {
	if e.Credits == nil || e.Credits.CreditsRemaining == nil {
		return 0, false
	}
	return int64(*e.Credits.CreditsRemaining), true
}

// Cost returns the credits this call cost and true. Cost is 0 for a free call and for a repeat
// in the same credit period. It returns false when the body carries no credits, or when the call was
// not metered.
func (e *Envelope) Cost() (int64, bool) {
	if e.Credits == nil || e.Credits.Cost == nil {
		return 0, false
	}
	return int64(*e.Credits.Cost), true
}
