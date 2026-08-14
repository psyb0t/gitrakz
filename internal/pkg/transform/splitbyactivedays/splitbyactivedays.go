// Package splitbyactivedays implements a transform.Primitive that groups
// State.Sessions by the UTC calendar day of each session's start time and
// emits one aggregated Row per active day.
package splitbyactivedays

import (
	"context"
	"sort"
	"time"

	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
)

// Name is this primitive's registered identifier.
const Name = "split-by-active-days"

const (
	dayKeyFormat = "2006-01-02"

	valueKeySeconds = "seconds"
	valueKeyHours   = "hours"

	secondsPerHour = 3600.0
)

// primitive groups sessions by their UTC calendar day and sums
// DurationSeconds per day.
type primitive struct{}

// New builds the split-by-active-days primitive. It has no configuration, so
// params is accepted and ignored.
//

func New(_ []byte) (transform.Primitive, error) {
	return &primitive{}, nil
}

// Name returns this primitive's registered identifier.
func (p *primitive) Name() string {
	return Name
}

// Apply reads s.Sessions, groups them by the UTC calendar day of each
// session's Start timestamp, sums DurationSeconds per day, and writes one
// Row per active day into s.Rows, sorted by day ascending.
func (p *primitive) Apply(
	_ context.Context,
	s *transform.State,
) error {
	secondsByDay := map[string]int64{}

	for _, session := range s.Sessions {
		day := time.Unix(session.Start, 0).UTC().Format(dayKeyFormat)
		secondsByDay[day] += session.DurationSeconds
	}

	days := make([]string, 0, len(secondsByDay))
	for day := range secondsByDay {
		days = append(days, day)
	}

	sort.Strings(days)

	rows := make([]transform.Row, 0, len(days))

	for _, day := range days {
		seconds := secondsByDay[day]
		rows = append(rows, transform.Row{
			Key: day,
			Values: map[string]float64{
				valueKeySeconds: float64(seconds),
				valueKeyHours:   float64(seconds) / secondsPerHour,
			},
			Labels: map[string]string{},
		})
	}

	s.Rows = rows

	return nil
}
