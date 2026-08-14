// Package template defines the programmatic template contract for gitrakz: a
// saved composition of a form, a transform pipeline, and a display layout —
// never a saved prompt. The LLM composes these from the building blocks; the
// engine runs them.
package template

import "encoding/json"

// FieldType is the input widget / value kind a form field collects.
type FieldType string

const (
	FieldTypeString   FieldType = "string"
	FieldTypeNumber   FieldType = "number"
	FieldTypeBool     FieldType = "bool"
	FieldTypeOffHours FieldType = "offHours" // the not-working windows editor
)

// FormField is one input a run collects before the transform runs (e.g. the
// off-hours schedule, an hourly rate, a lead-in padding).
type FormField struct {
	Key      string    `json:"key"`
	Label    string    `json:"label"`
	Type     FieldType `json:"type"`
	Required bool      `json:"required"`
	Default  any       `json:"default,omitempty"`
}

// Step is one entry in the transform pipeline: the primitive's registered name
// plus its raw JSON parameters (looked up in transform.Registry at run time).
type Step struct {
	Name   string          `json:"name"`
	Params json.RawMessage `json:"params,omitempty"`
}

// LayoutBlock maps part of the transform output onto a display block: Type is
// the block type; Source names the State field / Row values feeding it; Params
// carries per-block rendering config.
type LayoutBlock struct {
	Type   string          `json:"type"`
	Source string          `json:"source,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Template is the persisted, user- or LLM-authored composition.
type Template struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Form        []FormField   `json:"form"`
	Transform   []Step        `json:"transform"`
	Layout      []LayoutBlock `json:"layout"`
	Exports     []string      `json:"exports"`
	Model       string        `json:"model,omitempty"`
	Builtin     bool          `json:"builtin"`
}
