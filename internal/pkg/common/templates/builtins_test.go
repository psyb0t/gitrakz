package templates

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/engine"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimeline returns a synthetic timeline spanning two owners, two
// repos, two event types, and two UTC calendar days — enough for every
// built-in's transform pipeline (group-by owner/repo; sessionize +
// split-by-active-days) to produce more than one row.
func testTimeline() types.Timeline {
	return types.Timeline{
		{
			ID: "1", TS: 1700000000, Type: types.EventTypeCommit,
			Owner: "alice", Repo: "repoA", Title: "fix bug",
			Additions: 10, Deletions: 2, Branch: "main",
		},
		{
			ID: "2", TS: 1700000900, Type: types.EventTypePR,
			Owner: "alice", Repo: "repoA", Title: "add feature",
			Additions: 5, Deletions: 1, Branch: "main",
		},
		{
			ID: "3", TS: 1700086400, Type: types.EventTypeCommit,
			Owner: "alice", Repo: "repoB", Title: "next day work",
			Additions: 7, Deletions: 0, Branch: "main",
		},
		{
			ID: "4", TS: 1700003600, Type: types.EventTypeCommit,
			Owner: "bob", Repo: "repoA", Title: "bob's commit",
			Additions: 3, Deletions: 1, Branch: "main",
		},
	}
}

func TestMustMarshal_PanicsOnUnmarshalableValue(t *testing.T) {
	t.Parallel()

	assert.Panics(t, func() {
		mustMarshal(make(chan int))
	})
}

func TestBuiltins_UniqueIDsAndBuiltinFlag(t *testing.T) {
	t.Parallel()

	builtins := Builtins()
	require.GreaterOrEqual(t, len(builtins), 2)

	seenIDs := make(map[string]struct{}, len(builtins))

	for _, tmpl := range builtins {
		assert.True(t, tmpl.Builtin, "template %q must be Builtin", tmpl.ID)
		require.NotEmpty(t, tmpl.ID)
		require.NotEmpty(t, tmpl.Name)

		_, dup := seenIDs[tmpl.ID]
		assert.False(t, dup, "duplicate builtin id %q", tmpl.ID)
		seenIDs[tmpl.ID] = struct{}{}
	}
}

func TestBuiltins_RunThroughRealEngine(t *testing.T) {
	t.Parallel()

	eng := engine.NewEngine(registry.Default())

	byID := make(map[string]template.Template, len(Builtins()))
	for _, tmpl := range Builtins() {
		byID[tmpl.ID] = tmpl
	}

	testCases := []struct {
		name           string
		templateID     string
		form           types.FormValues
		wantBlockTypes []blocks.BlockType
		wantMinRows    int
	}{
		{
			name:       "activity summary",
			templateID: IDActivitySummary,
			form:       nil,
			wantBlockTypes: []blocks.BlockType{
				blocks.BlockTypeHeading,
				blocks.BlockTypeTable,
			},
			wantMinRows: 1,
		},
		{
			name:       "commits per repo",
			templateID: IDCommitsPerRepo,
			form:       nil,
			wantBlockTypes: []blocks.BlockType{
				blocks.BlockTypeHeading,
				blocks.BlockTypeTable,
				blocks.BlockTypeMetric,
			},
			wantMinRows: 1,
		},
		{
			name:       "work sessions timesheet",
			templateID: IDWorkSessionsTimesheet,
			form:       types.FormValues{"rate": 50.0},
			wantBlockTypes: []blocks.BlockType{
				blocks.BlockTypeHeading,
				blocks.BlockTypeTable,
				blocks.BlockTypeMetric,
			},
			wantMinRows: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			tmpl, ok := byID[tc.templateID]
			require.True(t, ok, "builtin %q not found", tc.templateID)

			doc, err := eng.Run(
				context.Background(), tmpl, testTimeline(), tc.form,
			)
			require.NoError(t, err)
			require.NotEmpty(t, doc)
			require.Len(t, doc, len(tc.wantBlockTypes))

			var tableRowCount int

			for i, block := range doc {
				assert.Equal(
					t, tc.wantBlockTypes[i], block.Type, "block %d type", i,
				)

				if block.Type != blocks.BlockTypeTable {
					continue
				}

				table, tableErr := block.AsTable()
				require.NoError(t, tableErr)

				tableRowCount = len(table.Rows)
			}

			assert.GreaterOrEqual(t, tableRowCount, tc.wantMinRows)
		})
	}
}
