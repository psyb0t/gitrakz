// Package sessionize implements the "sessionize" transform primitive: it
// clusters a timeline's events into per-owner work sessions using a
// gap-based heuristic, so downstream primitives can reason about
// continuous blocks of work instead of raw event timestamps.
package sessionize

import (
	"context"
	"encoding/json"
	"sort"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
)

// Name is the primitive's registry key.
const Name = "sessionize"

const (
	// defaultGapSeconds is the max gap, in seconds, between two
	// consecutive events for them to stay in the same session.
	defaultGapSeconds = 1800
	// defaultLeadInSeconds is folded into every session's duration to
	// account for work that precedes the first recorded event.
	defaultLeadInSeconds = 1500
)

// paramsConfig is the JSON shape a sessionize step config carries.
type paramsConfig struct {
	GapSeconds    int `json:"gapSeconds"`
	LeadInSeconds int `json:"leadInSeconds"`
}

// primitive reads State.Timeline and appends the resulting per-owner
// sessions to State.Sessions.
type primitive struct {
	gapSeconds    int64
	leadInSeconds int64
}

// New builds a sessionize primitive from its JSON params. Empty params
// fall back to defaultGapSeconds / defaultLeadInSeconds.
//

func New(params []byte) (transform.Primitive, error) {
	cfg := paramsConfig{
		GapSeconds:    defaultGapSeconds,
		LeadInSeconds: defaultLeadInSeconds,
	}

	if len(params) > 0 {
		if err := json.Unmarshal(params, &cfg); err != nil {
			return nil, ctxerrors.Wrap(err, "unmarshal sessionize params")
		}
	}

	return &primitive{
		gapSeconds:    int64(cfg.GapSeconds),
		leadInSeconds: int64(cfg.LeadInSeconds),
	}, nil
}

// Name returns the primitive's registry key.
func (p *primitive) Name() string {
	return Name
}

// Apply sorts a copy of s.Timeline by ts, groups it by owner, and appends
// one session per owner-level cluster to s.Sessions. Every other State
// field is left untouched.
func (p *primitive) Apply(_ context.Context, s *transform.State) error {
	if len(s.Timeline) == 0 {
		return nil
	}

	sorted := make(types.Timeline, len(s.Timeline))
	copy(sorted, s.Timeline)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TS < sorted[j].TS
	})

	owners, byOwner := groupByOwner(sorted)

	for _, owner := range owners {
		s.Sessions = append(
			s.Sessions,
			p.sessionizeOwner(owner, byOwner[owner])...,
		)
	}

	return nil
}

// groupByOwner splits a ts-sorted timeline into per-owner event slices,
// preserving each owner's first-appearance order so the output stays
// deterministic.
func groupByOwner(
	sorted types.Timeline,
) ([]string, map[string][]types.Event) {
	owners := make([]string, 0)
	byOwner := make(map[string][]types.Event)

	for _, event := range sorted {
		if _, ok := byOwner[event.Owner]; !ok {
			owners = append(owners, event.Owner)
		}

		byOwner[event.Owner] = append(byOwner[event.Owner], event)
	}

	return owners, byOwner
}

// sessionizeOwner walks one owner's ts-sorted events and splits them into
// sessions, starting a new session whenever the gap to the previous event
// exceeds p.gapSeconds.
func (p *primitive) sessionizeOwner(
	owner string,
	events []types.Event,
) []types.Session {
	sessions := make([]types.Session, 0, 1)

	start := events[0].TS
	end := events[0].TS
	count := 1

	for _, event := range events[1:] {
		if event.TS-end > p.gapSeconds {
			sessions = append(
				sessions,
				p.newSession(owner, start, end, count),
			)

			start = event.TS
			end = event.TS
			count = 1

			continue
		}

		end = event.TS
		count++
	}

	sessions = append(sessions, p.newSession(owner, start, end, count))

	return sessions
}

// newSession builds an owner-level session (Repo left empty) spanning
// [start, end], with p.leadInSeconds folded into the duration.
func (p *primitive) newSession(
	owner string,
	start, end int64,
	count int,
) types.Session {
	return types.Session{
		Owner:           owner,
		Start:           start,
		End:             end,
		DurationSeconds: (end - start) + p.leadInSeconds,
		EventCount:      count,
	}
}
