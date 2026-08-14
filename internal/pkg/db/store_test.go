package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openTestStore(t *testing.T) *Store {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "gitrakz.db")

	store, err := Open(context.Background(), dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, store.Close())
	})

	return store
}

func schemaObjectExists(
	t *testing.T,
	sqlDB *sql.DB,
	objType, name string,
) bool {
	t.Helper()

	var count int

	err := sqlDB.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = ? AND name = ?",
		objType, name,
	).Scan(&count)
	require.NoError(t, err)

	return count == 1
}

func TestOpen_CreatesSchema(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)

	sqlDB, err := store.query.UnderlyingDB().DB()
	require.NoError(t, err)

	wantTables := []string{
		"events", "sync_state", "templates", "documents", "llm_cache",
	}
	for _, table := range wantTables {
		assert.True(t,
			schemaObjectExists(t, sqlDB, "table", table), "table %s", table)
	}

	wantIndexes := []string{
		"idx_events_owner_ts",
		"idx_events_repo_ts",
		"idx_events_type_ts",
	}
	for _, index := range wantIndexes {
		assert.True(t,
			schemaObjectExists(t, sqlDB, "index", index), "index %s", index)
	}
}

func TestStore_EventsTimeline(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	const owner, repo = "psyb0t", "gitrakz"

	events := make([]types.Event, 0, 5)
	for i := range 5 {
		events = append(events, types.Event{
			ID:    fmt.Sprintf("commit:%s/%s:%d", owner, repo, i),
			TS:    int64(1000 + i),
			Type:  types.EventTypeCommit,
			Owner: owner,
			Repo:  repo,
			SHA:   fmt.Sprintf("sha%d", i),
			Title: fmt.Sprintf("commit %d", i),
		})
	}

	require.NoError(t, store.UpsertEvents(ctx, events))

	// Re-upsert one row with a changed title — proves OnConflict/UpdateAll.
	events[0].Title = "updated title"
	require.NoError(t, store.UpsertEvents(ctx, events[:1]))

	const perPage = 2

	page, hasMore, err := store.QueryTimeline(ctx, TimelineFilter{
		Owner:   owner,
		Repo:    repo,
		PerPage: perPage,
	})
	require.NoError(t, err)
	assert.True(t, hasMore)
	require.Len(t, page, perPage)
	assert.Equal(t, int64(1004), page[0].TS)
	assert.Equal(t, int64(1003), page[1].TS)

	lastPage, hasMoreLast, err := store.QueryTimeline(ctx, TimelineFilter{
		Owner:   owner,
		Repo:    repo,
		PerPage: perPage,
		Page:    2,
	})
	require.NoError(t, err)
	assert.False(t, hasMoreLast)
	require.Len(t, lastPage, 1)
	assert.Equal(t, int64(1000), lastPage[0].TS)
	assert.Equal(t, "updated title", lastPage[0].Title)

	owners, err := store.ListOwners(ctx)
	require.NoError(t, err)
	assert.Contains(t, owners, owner)

	repos, err := store.ListRepos(ctx, owner)
	require.NoError(t, err)
	assert.Contains(t, repos, repo)
}

func TestStore_QueryTimeline_FilterVariants(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.UpsertEvents(ctx, []types.Event{
		{
			ID: "commit:psyb0t/a:1", TS: 100, Type: types.EventTypeCommit,
			Owner: "psyb0t", Repo: "a",
		},
		{
			ID: "pr:psyb0t/a:2", TS: 200, Type: types.EventTypePR,
			Owner: "psyb0t", Repo: "a",
		},
		{
			ID: "commit:psyb0t/b:3", TS: 300, Type: types.EventTypeCommit,
			Owner: "psyb0t", Repo: "b",
		},
	}))

	byType, _, err := store.QueryTimeline(ctx, TimelineFilter{
		Type: string(types.EventTypePR), PerPage: 10,
	})
	require.NoError(t, err)
	require.Len(t, byType, 1)
	assert.Equal(t, "pr:psyb0t/a:2", byType[0].ID)

	fromOnly, _, err := store.QueryTimeline(ctx, TimelineFilter{
		From: 200, PerPage: 10,
	})
	require.NoError(t, err)
	require.Len(t, fromOnly, 2)

	toOnly, _, err := store.QueryTimeline(ctx, TimelineFilter{
		To: 200, PerPage: 10,
	})
	require.NoError(t, err)
	require.Len(t, toOnly, 2)

	// Exactly PerPage rows on the first page — the off-by-one trap:
	// hasMore MUST be false, not true.
	exact, hasMore, err := store.QueryTimeline(ctx, TimelineFilter{PerPage: 3})
	require.NoError(t, err)
	assert.False(t, hasMore)
	assert.Len(t, exact, 3)
}

func TestStore_SyncState(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	const owner, repo = "owner", "repo"

	ts, err := store.GetSyncState(ctx, owner, repo)
	require.NoError(t, err)
	assert.Equal(t, int64(0), ts)

	const firstTS = 12345

	require.NoError(t, store.UpsertSyncState(ctx, owner, repo, firstTS))

	ts, err = store.GetSyncState(ctx, owner, repo)
	require.NoError(t, err)
	assert.Equal(t, int64(firstTS), ts)

	const secondTS = 67890

	require.NoError(t, store.UpsertSyncState(ctx, owner, repo, secondTS))

	ts, err = store.GetSyncState(ctx, owner, repo)
	require.NoError(t, err)
	assert.Equal(t, int64(secondTS), ts)
}

func TestStore_TemplateCRUD(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	tmpl := template.Template{
		ID:          "tmpl-1",
		Name:        "Weekly Report",
		Description: "Summarizes weekly activity",
		Form: []template.FormField{
			{
				Key:      "rate",
				Label:    "Hourly rate",
				Type:     template.FieldTypeNumber,
				Required: true,
			},
		},
		Transform: []template.Step{
			{Name: "bucketByWeek", Params: json.RawMessage(`{"tz":"UTC"}`)},
		},
		Layout: []template.LayoutBlock{
			{Type: "table", Source: "weeks"},
		},
		Exports: []string{"pdf", "csv"},
		Model:   "gpt-5",
		Builtin: false,
	}

	require.NoError(t, store.SaveTemplate(ctx, tmpl))

	got, err := store.GetTemplate(ctx, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, tmpl, got)

	list, err := store.ListTemplates(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, tmpl, list[0])

	tmpl.Name = "Weekly Report v2"
	require.NoError(t, store.SaveTemplate(ctx, tmpl))

	got, err = store.GetTemplate(ctx, tmpl.ID)
	require.NoError(t, err)
	assert.Equal(t, "Weekly Report v2", got.Name)

	require.NoError(t, store.DeleteTemplate(ctx, tmpl.ID))

	_, err = store.GetTemplate(ctx, tmpl.ID)
	require.ErrorIs(t, err, commerr.ErrNotFound)
}

func TestStore_SaveDocument(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.SaveDocument(
		ctx, "doc-1", "tmpl-1", `{"owner":"x"}`, `{"rate":50}`, `{"pdf":"..."}`,
	))

	sqlDB, err := store.query.UnderlyingDB().DB()
	require.NoError(t, err)

	var count int

	err = sqlDB.QueryRowContext(
		ctx, "SELECT COUNT(*) FROM documents WHERE id = ?", "doc-1",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestStore_LLMCache(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	_, ok, err := store.LLMCacheGet(ctx, "missing-key")
	require.NoError(t, err)
	assert.False(t, ok)

	require.NoError(t, store.LLMCachePut(
		ctx, "key-1", "summarize", "v1", "hash-1", "cached output",
	))

	output, ok, err := store.LLMCacheGet(ctx, "key-1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "cached output", output)

	require.NoError(t, store.LLMCachePut(
		ctx, "key-1", "summarize", "v1", "hash-1", "updated output",
	))

	output, ok, err = store.LLMCacheGet(ctx, "key-1")
	require.NoError(t, err)
	require.True(t, ok)
	assert.Equal(t, "updated output", output)
}

func TestOpen_DirectoryAsPathErrors(t *testing.T) {
	t.Parallel()

	// A directory can't be opened as a SQLite database file — this hits
	// Open's "open sqlite database" error branch (gorm.Open failing).
	_, err := Open(context.Background(), t.TempDir())
	require.Error(t, err)
}

func TestStore_EventsTimeline_RawFieldRoundTrip(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	raw := json.RawMessage(`{"sha":"abc123"}`)

	require.NoError(t, store.UpsertEvents(ctx, []types.Event{
		{
			ID: "commit:psyb0t/gitrakz:raw", TS: 1,
			Type: types.EventTypeCommit, Owner: "psyb0t", Repo: "gitrakz",
			Raw: raw,
		},
	}))

	page, _, err := store.QueryTimeline(ctx, TimelineFilter{
		Owner: "psyb0t", Repo: "gitrakz", PerPage: 10,
	})
	require.NoError(t, err)
	require.Len(t, page, 1)
	assert.JSONEq(t, string(raw), string(page[0].Raw))
}

func TestStore_UpsertEvents_EmptyIsNoOp(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.UpsertEvents(ctx, nil))
	require.NoError(t, store.UpsertEvents(ctx, []types.Event{}))

	owners, err := store.ListOwners(ctx)
	require.NoError(t, err)
	assert.Empty(t, owners)
}

func TestStore_SaveTemplate_MarshalFormError(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	// FormField.Default is `any` — a channel is a value encoding/json
	// cannot marshal, so templateToModel's "marshal form" branch fires.
	tmpl := template.Template{
		ID: "bad-default",
		Form: []template.FormField{
			{Key: "rate", Default: make(chan int)},
		},
	}

	err := store.SaveTemplate(ctx, tmpl)
	require.Error(t, err)
}

func TestStore_TemplateDecodeErrors(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	// Seed a row directly through the generated repo (same package, so
	// the unexported `query` field is reachable) with an invalid-JSON
	// Form column — templateFromModel's "unmarshal form" branch can only
	// be reached via a row SaveTemplate itself would never produce.
	err := store.query.Template.WithContext(ctx).Create(&models.Template{
		ID:        "bad-form-json",
		Form:      "not json",
		Transform: "[]",
		Layout:    "[]",
		Exports:   "[]",
	})
	require.NoError(t, err)

	_, err = store.GetTemplate(ctx, "bad-form-json")
	require.Error(t, err)

	_, err = store.ListTemplates(ctx)
	require.Error(t, err)
}

func TestStore_GetTemplate_EmptyJSONColumnsDecodeAsZeroValue(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	// An empty-string column (never produced by SaveTemplate, which
	// always marshals to at least "null" or "[]") exercises
	// decodeJSONText's "leave the field at its zero value" branch,
	// distinct from the "malformed JSON" error branch covered above.
	seedErr := store.query.Template.WithContext(ctx).Create(&models.Template{
		ID: "empty-json-columns",
	})
	require.NoError(t, seedErr)

	got, err := store.GetTemplate(ctx, "empty-json-columns")
	require.NoError(t, err)
	assert.Equal(t, "empty-json-columns", got.ID)
	assert.Empty(t, got.Form)
	assert.Empty(t, got.Transform)
	assert.Empty(t, got.Layout)
	assert.Empty(t, got.Exports)
}

func TestStore_ClosedStoreErrors(t *testing.T) {
	t.Parallel()

	store := openTestStore(t)
	ctx := context.Background()

	// Seed one template row while the store is still open, so GetTemplate
	// / ListTemplates / DeleteTemplate below exercise the SAME code path
	// as a healthy store, just against a closed connection.
	require.NoError(t, store.SaveTemplate(ctx, template.Template{ID: "t1"}))
	require.NoError(t, store.Close())

	t.Run("UpsertEvents", func(t *testing.T) {
		t.Parallel()

		err := store.UpsertEvents(ctx, []types.Event{{ID: "x"}})
		assert.Error(t, err)
	})

	t.Run("QueryTimeline", func(t *testing.T) {
		t.Parallel()

		_, _, err := store.QueryTimeline(ctx, TimelineFilter{PerPage: 5})
		assert.Error(t, err)
	})

	t.Run("ListOwners", func(t *testing.T) {
		t.Parallel()

		_, err := store.ListOwners(ctx)
		assert.Error(t, err)
	})

	t.Run("ListRepos", func(t *testing.T) {
		t.Parallel()

		_, err := store.ListRepos(ctx, "owner")
		assert.Error(t, err)
	})

	t.Run("GetSyncState", func(t *testing.T) {
		t.Parallel()

		_, err := store.GetSyncState(ctx, "owner", "repo")
		assert.Error(t, err)
	})

	t.Run("UpsertSyncState", func(t *testing.T) {
		t.Parallel()

		err := store.UpsertSyncState(ctx, "owner", "repo", 1)
		assert.Error(t, err)
	})

	t.Run("ListTemplates", func(t *testing.T) {
		t.Parallel()

		_, err := store.ListTemplates(ctx)
		assert.Error(t, err)
	})

	t.Run("GetTemplate", func(t *testing.T) {
		t.Parallel()

		_, err := store.GetTemplate(ctx, "t1")
		assert.Error(t, err)
		assert.NotErrorIs(t, err, commerr.ErrNotFound)
	})

	t.Run("SaveTemplate", func(t *testing.T) {
		t.Parallel()

		err := store.SaveTemplate(ctx, template.Template{ID: "t2"})
		assert.Error(t, err)
	})

	t.Run("DeleteTemplate", func(t *testing.T) {
		t.Parallel()

		err := store.DeleteTemplate(ctx, "t1")
		assert.Error(t, err)
	})

	t.Run("SaveDocument", func(t *testing.T) {
		t.Parallel()

		err := store.SaveDocument(ctx, "d1", "t1", "{}", "{}", "{}")
		assert.Error(t, err)
	})

	t.Run("LLMCacheGet", func(t *testing.T) {
		t.Parallel()

		_, _, err := store.LLMCacheGet(ctx, "key")
		assert.Error(t, err)
	})

	t.Run("LLMCachePut", func(t *testing.T) {
		t.Parallel()

		err := store.LLMCachePut(ctx, "key", "step", "v1", "hash", "output")
		assert.Error(t, err)
	})
}
