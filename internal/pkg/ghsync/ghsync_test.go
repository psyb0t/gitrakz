package ghsync

import (
	"context"
	"sync"
	"testing"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockGHClient is a hand-rolled GHClient test double — each test wires the
// exact behavior it needs via the two func fields instead of pulling in a
// generated mock.
type mockGHClient struct {
	discoverFn func(ctx context.Context, user string) ([]RepoRef, error)
	listFn     func(
		ctx context.Context,
		r RepoRef,
		user string,
		since int64,
	) ([]types.Event, error)
}

func (m *mockGHClient) DiscoverRepos(
	ctx context.Context,
	user string,
) ([]RepoRef, error) {
	return m.discoverFn(ctx, user)
}

func (m *mockGHClient) ListEvents(
	ctx context.Context,
	r RepoRef,
	user string,
	since int64,
) ([]types.Event, error) {
	return m.listFn(ctx, r, user, since)
}

// mockEventStore is a hand-rolled EventStore test double that records every
// call so tests can assert on exactly what Sync persisted.
type mockEventStore struct {
	mu sync.Mutex

	syncState        map[string]int64
	syncStateUpdates map[string]int64
	upsertedEvents   []types.Event
	upsertCalls      int

	getStateErr        error
	upsertEventsErr    error
	upsertSyncStateErr error
}

func newMockEventStore() *mockEventStore {
	return &mockEventStore{
		syncState:        map[string]int64{},
		syncStateUpdates: map[string]int64{},
	}
}

func stateKey(owner, repo string) string {
	return owner + "/" + repo
}

func (m *mockEventStore) UpsertEvents(
	_ context.Context,
	evs []types.Event,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.upsertEventsErr != nil {
		return m.upsertEventsErr
	}

	m.upsertCalls++
	m.upsertedEvents = append(m.upsertedEvents, evs...)

	return nil
}

func (m *mockEventStore) GetSyncState(
	_ context.Context,
	owner, repo string,
) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.getStateErr != nil {
		return 0, m.getStateErr
	}

	return m.syncState[stateKey(owner, repo)], nil
}

func (m *mockEventStore) UpsertSyncState(
	_ context.Context,
	owner, repo string,
	ts int64,
) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.upsertSyncStateErr != nil {
		return m.upsertSyncStateErr
	}

	m.syncStateUpdates[stateKey(owner, repo)] = ts

	return nil
}

func TestSyncer_Sync_DiscoversUpsertsAndUpdatesState(t *testing.T) {
	t.Parallel()

	const user = "psyb0t"

	repoA := RepoRef{Owner: "psyb0t", Repo: "aaa"}
	repoB := RepoRef{Owner: "octocat", Repo: "bbb"}

	eventsA := []types.Event{
		{
			ID: "commit:psyb0t/aaa:sha1", TS: 100,
			Type: types.EventTypeCommit, Owner: "psyb0t", Repo: "aaa",
		},
		{
			ID: "commit:psyb0t/aaa:sha2", TS: 200,
			Type: types.EventTypeCommit, Owner: "psyb0t", Repo: "aaa",
		},
	}
	eventsB := []types.Event{
		{
			ID: "commit:octocat/bbb:sha1", TS: 300,
			Type: types.EventTypeCommit, Owner: "octocat", Repo: "bbb",
		},
	}

	gh := &mockGHClient{
		discoverFn: func(_ context.Context, u string) ([]RepoRef, error) {
			require.Equal(t, user, u)

			return []RepoRef{repoA, repoB}, nil
		},
		listFn: func(
			_ context.Context, r RepoRef, u string, since int64,
		) ([]types.Event, error) {
			require.Equal(t, user, u)
			require.Equal(t, int64(0), since)

			switch r {
			case repoA:
				return eventsA, nil
			case repoB:
				return eventsB, nil
			default:
				require.FailNowf(t, "unexpected repo", "%+v", r)

				return nil, nil
			}
		},
	}

	store := newMockEventStore()
	syncer := NewSyncer(gh, store, user)

	result, err := syncer.Sync(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 2, result.ReposScanned)
	assert.Equal(t, 3, result.EventsUpserted)
	assert.Empty(t, result.Errors)

	wantEvents := append(append([]types.Event{}, eventsA...), eventsB...)
	assert.ElementsMatch(t, wantEvents, store.upsertedEvents)
	assert.Equal(t, 2, store.upsertCalls)

	keyA := stateKey(repoA.Owner, repoA.Repo)
	assert.Equal(t, int64(200), store.syncStateUpdates[keyA])

	keyB := stateKey(repoB.Owner, repoB.Repo)
	assert.Equal(t, int64(300), store.syncStateUpdates[keyB])
}

func TestSyncer_Sync_FailSoftOnPerRepoListEventsError(t *testing.T) {
	t.Parallel()

	const user = "psyb0t"

	repoOK := RepoRef{Owner: "psyb0t", Repo: "ok-repo"}
	repoBad := RepoRef{Owner: "psyb0t", Repo: "bad-repo"}

	eventsOK := []types.Event{
		{
			ID: "commit:psyb0t/ok-repo:sha1", TS: 111,
			Type: types.EventTypeCommit, Owner: "psyb0t", Repo: "ok-repo",
		},
	}

	listErr := ctxerrors.New("rate limited")

	gh := &mockGHClient{
		discoverFn: func(_ context.Context, _ string) ([]RepoRef, error) {
			return []RepoRef{repoOK, repoBad}, nil
		},
		listFn: func(
			_ context.Context, r RepoRef, _ string, _ int64,
		) ([]types.Event, error) {
			if r == repoBad {
				return nil, listErr
			}

			return eventsOK, nil
		},
	}

	store := newMockEventStore()
	syncer := NewSyncer(gh, store, user)

	result, err := syncer.Sync(context.Background())
	require.NoError(t, err) // fail-soft: Sync itself does not error

	assert.Equal(t, 1, result.ReposScanned)
	assert.Equal(t, 1, result.EventsUpserted)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "psyb0t/bad-repo")

	// The good repo's events + sync_state landed despite the bad repo's
	// failure — that's the fail-soft contract.
	assert.Equal(t, eventsOK, store.upsertedEvents)

	keyOK := stateKey(repoOK.Owner, repoOK.Repo)
	assert.Equal(t, int64(111), store.syncStateUpdates[keyOK])

	keyBad := stateKey(repoBad.Owner, repoBad.Repo)
	_, badRecorded := store.syncStateUpdates[keyBad]
	assert.False(t, badRecorded)
}

func TestSyncer_Sync_DiscoverReposErrorAbortsSync(t *testing.T) {
	t.Parallel()

	discoverErr := ctxerrors.New("token expired")

	gh := &mockGHClient{
		discoverFn: func(_ context.Context, _ string) ([]RepoRef, error) {
			return nil, discoverErr
		},
		listFn: func(
			_ context.Context, _ RepoRef, _ string, _ int64,
		) ([]types.Event, error) {
			require.FailNow(
				t, "ListEvents must not be called when DiscoverRepos fails",
			)

			return nil, nil
		},
	}

	store := newMockEventStore()
	syncer := NewSyncer(gh, store, "psyb0t")

	result, err := syncer.Sync(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, discoverErr)
	assert.Zero(t, result)
}

func TestSyncer_Sync_NoNewEventsSkipsUpsertAndState(t *testing.T) {
	t.Parallel()

	repo := RepoRef{Owner: "psyb0t", Repo: "quiet"}

	gh := &mockGHClient{
		discoverFn: func(_ context.Context, _ string) ([]RepoRef, error) {
			return []RepoRef{repo}, nil
		},
		listFn: func(
			_ context.Context, _ RepoRef, _ string, _ int64,
		) ([]types.Event, error) {
			return nil, nil
		},
	}

	store := newMockEventStore()
	syncer := NewSyncer(gh, store, "psyb0t")

	result, err := syncer.Sync(context.Background())
	require.NoError(t, err)

	assert.Equal(t, 1, result.ReposScanned)
	assert.Equal(t, 0, result.EventsUpserted)
	assert.Empty(t, result.Errors)
	assert.Zero(t, store.upsertCalls)

	_, recorded := store.syncStateUpdates[stateKey(repo.Owner, repo.Repo)]
	assert.False(t, recorded)
}

func TestSyncer_Sync_UsesRecordedSyncStateAsSince(t *testing.T) {
	t.Parallel()

	repo := RepoRef{Owner: "psyb0t", Repo: "resumed"}

	var gotSince int64

	gh := &mockGHClient{
		discoverFn: func(_ context.Context, _ string) ([]RepoRef, error) {
			return []RepoRef{repo}, nil
		},
		listFn: func(
			_ context.Context, _ RepoRef, _ string, since int64,
		) ([]types.Event, error) {
			gotSince = since

			return nil, nil
		},
	}

	store := newMockEventStore()
	store.syncState[stateKey(repo.Owner, repo.Repo)] = 999

	syncer := NewSyncer(gh, store, "psyb0t")

	_, err := syncer.Sync(context.Background())
	require.NoError(t, err)

	assert.Equal(t, int64(999), gotSince)
}

func TestSyncer_Sync_FailSoftOnGetSyncStateError(t *testing.T) {
	t.Parallel()

	repo := RepoRef{Owner: "psyb0t", Repo: "broken-state"}

	gh := &mockGHClient{
		discoverFn: func(_ context.Context, _ string) ([]RepoRef, error) {
			return []RepoRef{repo}, nil
		},
		listFn: func(
			_ context.Context, _ RepoRef, _ string, _ int64,
		) ([]types.Event, error) {
			require.FailNow(
				t, "ListEvents must not be called when GetSyncState fails",
			)

			return nil, nil
		},
	}

	store := newMockEventStore()
	store.getStateErr = ctxerrors.New("db locked")

	syncer := NewSyncer(gh, store, "psyb0t")

	result, err := syncer.Sync(context.Background())
	require.NoError(t, err) // fail-soft: Sync itself does not error

	assert.Zero(t, result.ReposScanned)
	assert.Zero(t, result.EventsUpserted)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "psyb0t/broken-state")
	assert.Contains(t, result.Errors[0], "get sync state")
}

func TestSyncer_Sync_FailSoftOnUpsertEventsError(t *testing.T) {
	t.Parallel()

	repo := RepoRef{Owner: "psyb0t", Repo: "unwritable"}

	events := []types.Event{
		{
			ID: "commit:psyb0t/unwritable:sha1", TS: 100,
			Type: types.EventTypeCommit, Owner: "psyb0t", Repo: "unwritable",
		},
	}

	gh := &mockGHClient{
		discoverFn: func(_ context.Context, _ string) ([]RepoRef, error) {
			return []RepoRef{repo}, nil
		},
		listFn: func(
			_ context.Context, _ RepoRef, _ string, _ int64,
		) ([]types.Event, error) {
			return events, nil
		},
	}

	store := newMockEventStore()
	store.upsertEventsErr = ctxerrors.New("disk full")

	syncer := NewSyncer(gh, store, "psyb0t")

	result, err := syncer.Sync(context.Background())
	require.NoError(t, err) // fail-soft: Sync itself does not error

	assert.Zero(t, result.ReposScanned)
	assert.Zero(t, result.EventsUpserted)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "upsert events")

	// The sync state must NOT advance when the events themselves failed
	// to persist — otherwise a re-run would skip the lost events.
	_, recorded := store.syncStateUpdates[stateKey(repo.Owner, repo.Repo)]
	assert.False(t, recorded)
}

func TestSyncer_Sync_FailSoftOnUpsertSyncStateError(t *testing.T) {
	t.Parallel()

	repo := RepoRef{Owner: "psyb0t", Repo: "state-write-fails"}

	events := []types.Event{
		{
			ID: "commit:psyb0t/state-write-fails:sha1", TS: 100,
			Type: types.EventTypeCommit, Owner: "psyb0t",
			Repo: "state-write-fails",
		},
	}

	gh := &mockGHClient{
		discoverFn: func(_ context.Context, _ string) ([]RepoRef, error) {
			return []RepoRef{repo}, nil
		},
		listFn: func(
			_ context.Context, _ RepoRef, _ string, _ int64,
		) ([]types.Event, error) {
			return events, nil
		},
	}

	store := newMockEventStore()
	store.upsertSyncStateErr = ctxerrors.New("write conflict")

	syncer := NewSyncer(gh, store, "psyb0t")

	result, err := syncer.Sync(context.Background())
	require.NoError(t, err) // fail-soft: Sync itself does not error

	assert.Zero(t, result.ReposScanned)
	assert.Zero(t, result.EventsUpserted)
	require.Len(t, result.Errors, 1)
	assert.Contains(t, result.Errors[0], "upsert sync state")

	// The events themselves still landed, even though advancing the
	// sync-state cursor failed.
	assert.Equal(t, events, store.upsertedEvents)
}
