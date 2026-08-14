package sessionize

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		params        []byte
		wantGap       int64
		wantLeadIn    int64
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:       "nil params uses defaults",
			params:     nil,
			wantGap:    defaultGapSeconds,
			wantLeadIn: defaultLeadInSeconds,
		},
		{
			name:       "empty params uses defaults",
			params:     []byte(""),
			wantGap:    defaultGapSeconds,
			wantLeadIn: defaultLeadInSeconds,
		},
		{
			name:       "partial params fills in remaining default",
			params:     []byte(`{"gapSeconds":60}`),
			wantGap:    60,
			wantLeadIn: defaultLeadInSeconds,
		},
		{
			name:       "full params override defaults",
			params:     []byte(`{"gapSeconds":60,"leadInSeconds":30}`),
			wantGap:    60,
			wantLeadIn: 30,
		},
		{
			name:          "invalid json errors",
			params:        []byte(`{not json`),
			wantErr:       true,
			wantErrSubstr: "unmarshal sessionize params",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			prim, err := New(tc.params)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrSubstr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, Name, prim.Name())

			p, ok := prim.(*primitive)
			require.True(t, ok)
			assert.Equal(t, tc.wantGap, p.gapSeconds)
			assert.Equal(t, tc.wantLeadIn, p.leadInSeconds)
		})
	}
}

func TestPrimitive_Apply(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		timeline     types.Timeline
		wantSessions []types.Session
	}{
		{
			name:         "empty timeline produces no sessions",
			timeline:     types.Timeline{},
			wantSessions: nil,
		},
		{
			name: "owner A: two within gap, one after a big gap",
			timeline: types.Timeline{
				{TS: 3500, Owner: "A"},
				{TS: 1000, Owner: "A"},
				{TS: 1500, Owner: "A"},
			},
			wantSessions: []types.Session{
				{
					Owner:           "A",
					Start:           1000,
					End:             1500,
					DurationSeconds: 500 + defaultLeadInSeconds,
					EventCount:      2,
				},
				{
					Owner:           "A",
					Start:           3500,
					End:             3500,
					DurationSeconds: 0 + defaultLeadInSeconds,
					EventCount:      1,
				},
			},
		},
		{
			name: "groups distinct owners into separate sessions",
			timeline: types.Timeline{
				{TS: 1000, Owner: "B"},
				{TS: 1100, Owner: "A"},
				{TS: 1200, Owner: "B"},
			},
			wantSessions: []types.Session{
				{
					Owner:           "B",
					Start:           1000,
					End:             1200,
					DurationSeconds: 200 + defaultLeadInSeconds,
					EventCount:      2,
				},
				{
					Owner:           "A",
					Start:           1100,
					End:             1100,
					DurationSeconds: 0 + defaultLeadInSeconds,
					EventCount:      1,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			prim, err := New(nil)
			require.NoError(t, err)
			assert.Equal(t, Name, prim.Name())

			wantTimeline := make(types.Timeline, len(tc.timeline))
			copy(wantTimeline, tc.timeline)

			s := &transform.State{Timeline: tc.timeline}

			err = prim.Apply(context.Background(), s)
			require.NoError(t, err)
			assert.Equal(t, tc.wantSessions, s.Sessions)

			// s.Timeline must stay untouched — Apply sorts a COPY.
			assert.Equal(t, wantTimeline, s.Timeline)
		})
	}
}
