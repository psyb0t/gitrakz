// Package ghsync pulls a GitHub user's activity into gitrakz's local store,
// incrementally. It is built by dependency inversion: GHClient and
// EventStore are ordinary interfaces defined in this package, so Syncer is
// unit-testable without a real gh CLI or a real database. Production wiring
// supplies a commander-backed GHClient (see commander_client.go) and
// internal/pkg/db.DB, which already satisfies EventStore.
package ghsync

import (
	"context"
	"fmt"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
)

// RepoRef identifies a single GitHub repository by owner and name.
type RepoRef struct {
	Owner string
	Repo  string
}

// GHClient discovers repos for a user and lists that user's activity in
// each. The production implementation shells out to the gh CLI via
// commander (see commanderGHClient); tests supply a mock.
type GHClient interface {
	// AuthenticatedUser returns the login the gh CLI is authenticated as
	// (`gh api user`). It's used to default the sync target when
	// GITRAKZ_GH_USER is unset — track yourself without configuring anything.
	AuthenticatedUser(ctx context.Context) (string, error)

	// DiscoverRepos returns every repo the authenticated gh token can see
	// for user.
	DiscoverRepos(ctx context.Context, user string) ([]RepoRef, error)

	// ListEvents returns user's activity in r at or after since (a unix
	// timestamp; 0 means "from the beginning").
	ListEvents(
		ctx context.Context,
		r RepoRef,
		user string,
		since int64,
	) ([]types.Event, error)
}

// EventStore persists synced events and tracks per-repo sync progress.
// internal/pkg/db.DB already satisfies this interface in production —
// ghsync never imports db, so it stays unit-testable without a real
// database; the engine/handlers wire the real one in.
type EventStore interface {
	UpsertEvents(ctx context.Context, evs []types.Event) error
	GetSyncState(ctx context.Context, owner, repo string) (int64, error)
	UpsertSyncState(ctx context.Context, owner, repo string, ts int64) error
}

// Syncer pulls a GitHub user's activity from a GHClient into an EventStore,
// incrementally.
type Syncer struct {
	gh    GHClient
	store EventStore
	user  string
}

// NewSyncer returns a Syncer that syncs user's activity from gh into store.
// An empty user means "the gh CLI's authenticated login", resolved on the
// first Sync via GHClient.AuthenticatedUser.
func NewSyncer(gh GHClient, store EventStore, user string) *Syncer {
	return &Syncer{gh: gh, store: store, user: user}
}

// SyncResult summarizes one Sync run.
type SyncResult struct {
	ReposScanned   int
	EventsUpserted int
	Errors         []string
}

// Sync runs two passes: discover every repo for the configured user, then
// pull + persist each repo's events since its last recorded sync.
//
// A single repo failing (a network blip, an SSO-locked org, a rate limit)
// is recorded in Errors and does NOT abort the rest of the sync — per-owner
// fail-soft, so one bad repo never takes down every other owner's sync.
func (s *Syncer) Sync(ctx context.Context) (SyncResult, error) {
	logger := ctxscope.GetLogger(ctx)

	user, err := s.resolveUser(ctx)
	if err != nil {
		return SyncResult{}, err
	}

	repos, err := s.gh.DiscoverRepos(ctx, user)
	if err != nil {
		return SyncResult{}, ctxerrors.Wrapf(
			err, "discover repos for %s", user,
		)
	}

	result := SyncResult{}

	for _, repo := range repos {
		logger.Debug("repo sync started",
			"owner", repo.Owner,
			"repo", repo.Repo,
		)

		eventCount, err := s.syncRepo(ctx, repo, user)
		if err != nil {
			logger.Warn("repo sync failed, continuing",
				"owner", repo.Owner,
				"repo", repo.Repo,
				"err", err,
				"reason", "fail-soft — other repos still sync",
			)

			result.Errors = append(result.Errors, fmt.Sprintf(
				"%s/%s: %s", repo.Owner, repo.Repo, err,
			))

			continue
		}

		result.ReposScanned++
		result.EventsUpserted += eventCount

		logger.Info("repo sync completed",
			"owner", repo.Owner,
			"repo", repo.Repo,
			"events", eventCount,
		)
	}

	return result, nil
}

// resolveUser returns the configured sync target, or — when GITRAKZ_GH_USER is
// unset — the login the gh CLI is authenticated as. It errors only when neither
// is available (no env var AND gh is not authenticated).
func (s *Syncer) resolveUser(ctx context.Context) (string, error) {
	if s.user != "" {
		return s.user, nil
	}

	user, err := s.gh.AuthenticatedUser(ctx)
	if err != nil {
		return "", ctxerrors.Wrap(err, "resolve authenticated gh user")
	}

	if user == "" {
		return "", ctxerrors.Wrap(
			commerr.ErrNotAuthenticated,
			"GITRAKZ_GH_USER unset and gh reported no authenticated user",
		)
	}

	return user, nil
}

// syncRepo pulls + persists r's events since its last recorded sync, then
// advances sync_state to the newest event's timestamp. Returns the number
// of events upserted.
func (s *Syncer) syncRepo(
	ctx context.Context,
	r RepoRef,
	user string,
) (int, error) {
	since, err := s.store.GetSyncState(ctx, r.Owner, r.Repo)
	if err != nil {
		return 0, ctxerrors.Wrap(err, "get sync state")
	}

	events, err := s.gh.ListEvents(ctx, r, user, since)
	if err != nil {
		return 0, ctxerrors.Wrap(err, "list events")
	}

	if len(events) == 0 {
		return 0, nil
	}

	if err := s.store.UpsertEvents(ctx, events); err != nil {
		return 0, ctxerrors.Wrap(err, "upsert events")
	}

	latest := latestEventTS(since, events)

	if err := s.store.UpsertSyncState(
		ctx, r.Owner, r.Repo, latest,
	); err != nil {
		return 0, ctxerrors.Wrap(err, "upsert sync state")
	}

	return len(events), nil
}

// latestEventTS returns the newest TS across events, or current if events
// is empty or none is newer.
func latestEventTS(current int64, events []types.Event) int64 {
	latest := current

	for _, ev := range events {
		if ev.TS > latest {
			latest = ev.TS
		}
	}

	return latest
}
