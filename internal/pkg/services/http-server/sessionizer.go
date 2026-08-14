package httpserver

import (
	"context"
	"encoding/json"
	"time"

	"github.com/psyb0t/ctxerrors"
	ctransform "github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/sessionize"
)

// sessionizeParams mirrors sessionize's own JSON step-params shape — gap
// and lead-in expressed as whole seconds.
type sessionizeParams struct {
	GapSeconds    int `json:"gapSeconds"`
	LeadInSeconds int `json:"leadInSeconds"`
}

// sessionizer implements server.Sessionizer by running the sessionize
// transform primitive, configured once at wiring time from
// cfg.SessionGap/SessionLeadIn, over an ad hoc transform.State per call.
type sessionizer struct {
	primitive ctransform.Primitive
}

// newSessionizer builds a sessionizer configured with gap/leadIn.
func newSessionizer(gap, leadIn time.Duration) (*sessionizer, error) {
	params, err := json.Marshal(sessionizeParams{
		GapSeconds:    int(gap.Seconds()),
		LeadInSeconds: int(leadIn.Seconds()),
	})
	if err != nil {
		return nil, ctxerrors.Wrap(err, "marshal sessionize params")
	}

	primitive, err := sessionize.New(params)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "build sessionize primitive")
	}

	return &sessionizer{primitive: primitive}, nil
}

// Sessions derives per-owner work sessions from timeline.
func (s *sessionizer) Sessions(
	ctx context.Context,
	timeline types.Timeline,
) ([]types.Session, error) {
	state := &ctransform.State{Timeline: timeline}

	if err := s.primitive.Apply(ctx, state); err != nil {
		return nil, ctxerrors.Wrap(err, "sessionize timeline")
	}

	return state.Sessions, nil
}
