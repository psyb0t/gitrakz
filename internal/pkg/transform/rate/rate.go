// Package rate implements the "rate" transform primitive: it derives a
// dollars value for every row by multiplying its hours value by an hourly
// rate resolved from the run's form values, falling back to a params
// default.
package rate

import (
	"context"
	"encoding/json"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
)

// Name is the primitive's registry key.
const Name = "rate"

const (
	defaultHoursField   = "hours"
	defaultDollarsField = "dollars"
	defaultFormKey      = "rate"
)

// stepParams is the JSON shape a "rate" pipeline step configures itself
// from. Every field is optional.
type stepParams struct {
	HoursField   string  `json:"hoursField"`
	DollarsField string  `json:"dollarsField"`
	FormKey      string  `json:"formKey"`
	Rate         float64 `json:"rate"`
}

// primitive multiplies each row's hours value by an hourly rate to derive
// a dollars value, writing the result back onto the row.
type primitive struct {
	hoursField   string
	dollarsField string
	formKey      string
	fallbackRate float64
}

// New builds a "rate" primitive from its JSON params. Unset fields default
// to hoursField="hours", dollarsField="dollars", formKey="rate", and
// rate=0 — the fallback hourly rate used when State.Form[formKey] is
// absent or not numeric.
func New(params []byte) (transform.Primitive, error) {
	sp := stepParams{
		HoursField:   defaultHoursField,
		DollarsField: defaultDollarsField,
		FormKey:      defaultFormKey,
	}

	if len(params) > 0 {
		if err := json.Unmarshal(params, &sp); err != nil {
			return nil, ctxerrors.Wrap(err, "unmarshal rate params")
		}
	}

	return primitive{
		hoursField:   sp.HoursField,
		dollarsField: sp.DollarsField,
		formKey:      sp.FormKey,
		fallbackRate: sp.Rate,
	}, nil
}

// Name returns the primitive's registry key.
func (primitive) Name() string {
	return Name
}

// Apply resolves the hourly rate and sets Values[dollarsField] =
// hours * rate for every row that has an hoursField value. Rows without
// it are left untouched.
func (p primitive) Apply(_ context.Context, s *transform.State) error {
	rate := p.resolveRate(s.Form)

	for i := range s.Rows {
		hours, ok := s.Rows[i].Values[p.hoursField]
		if !ok {
			continue
		}

		s.Rows[i].Values[p.dollarsField] = hours * rate
	}

	return nil
}

// resolveRate reads the hourly rate from form[formKey], coercing it to
// float64 (float64, json.Number, and int are all accepted). It falls back
// to the params rate when the key is absent or not numeric.
func (p primitive) resolveRate(form map[string]any) float64 {
	raw, ok := form[p.formKey]
	if !ok {
		return p.fallbackRate
	}

	rate, ok := toFloat64(raw)
	if !ok {
		return p.fallbackRate
	}

	return rate
}

// toFloat64 coerces the common numeric shapes a form value arrives as
// (float64, json.Number, int) into a float64.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, false
		}

		return f, true
	case int:
		return float64(n), true
	default:
		return 0, false
	}
}
