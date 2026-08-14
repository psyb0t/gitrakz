// Package types defines the shared domain vocabulary for gitrakz — events,
// sessions, form values — as plain data structs. It has no behavior and no
// dependencies beyond the standard library, so every other package can
// depend on it without pulling anything else in.
package types

import "encoding/json"

// EventType identifies the kind of GitHub activity an Event represents.
type EventType string

const (
	EventTypeCommit  EventType = "commit"
	EventTypePR      EventType = "pr"
	EventTypeReview  EventType = "review"
	EventTypeIssue   EventType = "issue"
	EventTypeRelease EventType = "release"
)

// Event is a single unit of GitHub activity — a commit, PR, review, issue,
// or release — normalized into one shape for storage and display.
type Event struct {
	ID        string          `json:"id"`
	TS        int64           `json:"ts"`
	Type      EventType       `json:"type"`
	Owner     string          `json:"owner"`
	Repo      string          `json:"repo"`
	SHA       string          `json:"sha"`
	Number    int             `json:"number"`
	Title     string          `json:"title"`
	URL       string          `json:"url"`
	Additions int             `json:"additions"`
	Deletions int             `json:"deletions"`
	Branch    string          `json:"branch"`
	Raw       json.RawMessage `json:"raw"`
}

// Timeline is an ordered collection of events.
type Timeline = []Event

// Session is a derived cluster of events treated as one continuous block
// of work.
type Session struct {
	Owner           string `json:"owner"`
	Repo            string `json:"repo"`
	Start           int64  `json:"start"`
	End             int64  `json:"end"`
	DurationSeconds int64  `json:"durationSeconds"`
	EventCount      int    `json:"eventCount"`
}

// FormValues holds the arbitrary input a template's form collects at run
// time — off-hours schedule, rate, lead-in, or whatever fields the
// template declares.
type FormValues = map[string]any

// OffWindow is a single off-hours interval within a day. StartMinute and
// EndMinute are both minutes-of-day in the range [0, 1440) — 1440 being
// the number of minutes in a day.
type OffWindow struct {
	StartMinute int `json:"startMinute"`
	EndMinute   int `json:"endMinute"`
}

// OffHours is the set of OffWindow entries excluded from time-based
// calculations (e.g. lunch breaks, end-of-day cutoffs) for a given day.
type OffHours = []OffWindow
