package models

// Event is one GitHub activity item (commit / pr / review / issue / release),
// keyed by a stable "{type}:{owner}/{repo}:{sha|number}" id. Query-only model;
// the schema (columns + indexes) is owned by the SQL migrations.
type Event struct {
	ID        string
	TS        int64
	Type      string
	Owner     string
	Repo      string
	SHA       string
	Number    int
	Title     string
	URL       string
	Additions int
	Deletions int
	Branch    string
	Raw       string
}

// TableName is the events table.
func (Event) TableName() string {
	return "events"
}
