// Package excludeofftime implements the "exclude-off-time" transform
// primitive: it subtracts recurring daily off-hours windows (lunch
// breaks, end-of-day cutoffs, etc.) from every session's duration.
//
// Off-hours windows come from State.Form["offHours"] when the run's
// form supplied one (round-tripped through JSON, since Form values are
// untyped any), falling back to the primitive's own params
// ({"offHours": [{"startMinute": .., "endMinute": ..}]}) otherwise.
//
// All day-boundary math runs in time.UTC for determinism — the
// primitive has no notion of a user's local timezone, so the overlap
// computed for a given session is reproducible regardless of where the
// pipeline runs.
package excludeofftime

import (
	"context"
	"encoding/json"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
)

// Name is the primitive's registry key.
const Name = "exclude-off-time"

const (
	// formOffHoursKey is the State.Form key a template's off-hours
	// field populates (see common/template.FieldTypeOffHours).
	formOffHoursKey = "offHours"

	secondsPerMinute = 60
	minutesPerDay    = 1440
	secondsPerDay    = minutesPerDay * secondsPerMinute
)

// stepParams is the JSON shape an exclude-off-time step config carries.
type stepParams struct {
	OffHours types.OffHours `json:"offHours"`
}

// primitive reads State.Sessions and the resolved off-hours windows,
// then rewrites each session's DurationSeconds to exclude the overlap.
type primitive struct {
	fallback types.OffHours
}

// New builds an exclude-off-time primitive from its JSON params. params
// may be empty when the run's Form is expected to supply "offHours"
// instead.
//

func New(params []byte) (transform.Primitive, error) {
	var sp stepParams

	if len(params) > 0 {
		if err := json.Unmarshal(params, &sp); err != nil {
			return nil, ctxerrors.Wrap(err, "unmarshal exclude-off-time params")
		}
	}

	return &primitive{fallback: sp.OffHours}, nil
}

// Name returns the primitive's registry key.
func (p *primitive) Name() string {
	return Name
}

// Apply reduces every session's DurationSeconds by its overlap with the
// resolved off-hours windows, never below zero. Start, End, and
// EventCount are left untouched.
func (p *primitive) Apply(_ context.Context, s *transform.State) error {
	offHours, err := p.resolveOffHours(s.Form)
	if err != nil {
		return ctxerrors.Wrap(err, "resolve off-hours windows")
	}

	if len(offHours) == 0 {
		return nil
	}

	for i := range s.Sessions {
		overlap := totalOverlap(
			s.Sessions[i].Start,
			s.Sessions[i].End,
			offHours,
		)

		s.Sessions[i].DurationSeconds = max(
			0,
			s.Sessions[i].DurationSeconds-overlap,
		)
	}

	return nil
}

// resolveOffHours prefers form[formOffHoursKey] (round-tripped through
// JSON, since Form values are untyped any) and falls back to the
// params this primitive was constructed with.
func (p *primitive) resolveOffHours(
	form map[string]any,
) (types.OffHours, error) {
	raw, ok := form[formOffHoursKey]
	if !ok || raw == nil {
		return p.fallback, nil
	}

	data, err := json.Marshal(raw)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "marshal form off-hours")
	}

	var offHours types.OffHours
	if err := json.Unmarshal(data, &offHours); err != nil {
		return nil, ctxerrors.Wrap(err, "unmarshal form off-hours")
	}

	return offHours, nil
}

// totalOverlap sums the overlap in seconds between session [start, end)
// and every recurring daily off-hours window, across every UTC calendar
// day the session touches.
func totalOverlap(start, end int64, offHours types.OffHours) int64 {
	if end <= start {
		return 0
	}

	day := dayStart(time.Unix(start, 0).UTC())

	var total int64

	for day.Unix() < end {
		for _, w := range offHours {
			total += windowOverlap(day.Unix(), w, start, end)
		}

		day = day.AddDate(0, 0, 1)
	}

	return total
}

// dayStart truncates t to UTC midnight of its calendar day.
func dayStart(t time.Time) time.Time {
	y, m, d := t.Date()

	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// overlapSeconds returns the overlap in seconds between [start, end) and
// [lo, hi), or 0 when they don't intersect.
func overlapSeconds(start, end, lo, hi int64) int64 {
	l := max(start, lo)
	h := min(end, hi)

	if h <= l {
		return 0
	}

	return h - l
}

// windowOverlap returns the overlap in seconds between [start, end) and
// the occurrence of off-hours window w anchored to the UTC calendar day
// starting at midnight (a UTC day boundary, in unix seconds).
//
// For a non-wrapping window (w.EndMinute > w.StartMinute) that's the
// single interval [midnight+StartMinute*60, midnight+EndMinute*60).
//
// For a wrapping window (w.EndMinute <= w.StartMinute, e.g. 23:00-06:00)
// every calendar day carries TWO disjoint pieces of the recurring
// window: a "tail" inherited from the PREVIOUS day's occurrence,
// [midnight, midnight+EndMinute*60), and a "head" that continues into
// the NEXT day, [midnight+StartMinute*60, midnight+86400). Each day is
// self-contained — day D's head ends exactly where day D+1's tail
// starts — so summing both pieces per day, for every day the session
// touches, covers the whole window with no gap and no double count.
func windowOverlap(midnight int64, w types.OffWindow, start, end int64) int64 {
	startSecs := int64(w.StartMinute) * secondsPerMinute
	endSecs := int64(w.EndMinute) * secondsPerMinute

	if w.EndMinute > w.StartMinute {
		return overlapSeconds(start, end, midnight+startSecs, midnight+endSecs)
	}

	dayEnd := midnight + secondsPerDay

	tail := overlapSeconds(start, end, midnight, midnight+endSecs)
	head := overlapSeconds(start, end, midnight+startSecs, dayEnd)

	return tail + head
}
