package models

// Template is a persisted programmatic template. Form / Transform / Layout /
// Exports hold the JSON-encoded sub-structures of template.Template.
type Template struct {
	ID          string
	Name        string
	Description string
	Form        string
	Transform   string
	Layout      string
	Exports     string
	Model       string
	Builtin     bool
	CreatedTS   int64
}

// TableName is the templates table.
func (Template) TableName() string {
	return "templates"
}
