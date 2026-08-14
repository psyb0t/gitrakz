package passthrough

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrimitive_Apply(t *testing.T) {
	t.Parallel()

	wantColumns := []string{
		"ts",
		"type",
		"owner",
		"repo",
		"title",
		"additions",
		"deletions",
		"url",
	}

	testCases := []struct {
		name     string
		timeline types.Timeline
		wantRows [][]string
	}{
		{
			name:     "empty timeline",
			timeline: types.Timeline{},
			wantRows: [][]string{},
		},
		{
			name: "two events",
			timeline: types.Timeline{
				{
					TS:        1700000000,
					Type:      types.EventTypeCommit,
					Owner:     "psyb0t",
					Repo:      "gitrakz",
					Title:     "fix bug",
					URL:       "https://example.com/commit/1",
					Additions: 10,
					Deletions: 2,
				},
				{
					TS:        1700003600,
					Type:      types.EventTypePR,
					Owner:     "psyb0t",
					Repo:      "gitrakz",
					Title:     "add feature",
					URL:       "https://example.com/pr/1",
					Additions: 50,
					Deletions: 5,
				},
			},
			wantRows: [][]string{
				{
					"2023-11-14T22:13:20Z",
					"commit",
					"psyb0t",
					"gitrakz",
					"fix bug",
					"10",
					"2",
					"https://example.com/commit/1",
				},
				{
					"2023-11-14T23:13:20Z",
					"pr",
					"psyb0t",
					"gitrakz",
					"add feature",
					"50",
					"5",
					"https://example.com/pr/1",
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

			s := &transform.State{Timeline: tc.timeline}

			err = prim.Apply(context.Background(), s)
			require.NoError(t, err)
			require.Len(t, s.Blocks, 1)

			block := s.Blocks[0]
			assert.Equal(t, blocks.BlockTypeTable, block.Type)

			table, err := block.AsTable()
			require.NoError(t, err)
			assert.Equal(t, wantColumns, table.Columns)
			assert.Equal(t, tc.wantRows, table.Rows)
		})
	}
}
