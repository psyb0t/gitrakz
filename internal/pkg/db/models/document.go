package models

// Document is a persisted generated document (one template run's output), kept
// so a run is re-exportable without recomputing.
type Document struct {
	ID         string
	TemplateID string
	Filter     string
	FormValues string
	Doc        string
	CreatedTS  int64
}

// TableName is the documents table.
func (Document) TableName() string {
	return "documents"
}
