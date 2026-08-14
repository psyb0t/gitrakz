package ghsync

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/psyb0t/commander"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
)

const (
	// ghBinary is the gh CLI executable shelled out to for every call.
	ghBinary = "gh"

	// repoListLimit caps how many repos `gh repo list` returns in one call.
	repoListLimit = 1000
)

// commanderGHClient implements GHClient by shelling out to the gh CLI via
// commander.Commander — the container is authed via mounted ~/.config/gh
// or GH_TOKEN, so no credentials are handled here.
type commanderGHClient struct {
	cmd commander.Commander
}

// NewCommanderGHClient returns a GHClient backed by the gh CLI, invoked
// through cmd. Production callers pass commander.New(); tests pass a
// commander.NewMock().
func NewCommanderGHClient(cmd commander.Commander) GHClient {
	return &commanderGHClient{cmd: cmd}
}

// ghRepoListEntry is one row of `gh repo list --json nameWithOwner` output.
type ghRepoListEntry struct {
	NameWithOwner string `json:"nameWithOwner"`
}

// DiscoverRepos lists every repo `gh repo list <user>` can see.
func (c *commanderGHClient) DiscoverRepos(
	ctx context.Context,
	user string,
) ([]RepoRef, error) {
	stdout, _, err := c.cmd.Output(
		ctx,
		ghBinary,
		[]string{
			"repo", "list", user,
			"--json", "nameWithOwner",
			"--limit", strconv.Itoa(repoListLimit),
		},
	)
	if err != nil {
		return nil, ctxerrors.Wrapf(err, "gh repo list %s", user)
	}

	var entries []ghRepoListEntry
	if err := json.Unmarshal(stdout, &entries); err != nil {
		return nil, ctxerrors.Wrapf(
			ctxerrors.Join(err, commerr.ErrUnmarshalFailed),
			"unmarshal gh repo list %s output", user,
		)
	}

	repos := make([]RepoRef, 0, len(entries))

	for _, entry := range entries {
		owner, repo, ok := strings.Cut(entry.NameWithOwner, "/")
		if !ok {
			return nil, ctxerrors.Wrapf(
				commerr.ErrParseFailed,
				"malformed nameWithOwner %q",
				entry.NameWithOwner,
			)
		}

		repos = append(repos, RepoRef{Owner: owner, Repo: repo})
	}

	return repos, nil
}

// ghCommit is one row of `gh api repos/{owner}/{repo}/commits` output —
// only the fields gitrakz's Event shape needs.
type ghCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Date time.Time `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	//nolint:tagliatelle // GitHub API field name, not ours to rename
	HTMLURL string `json:"html_url"`
	Stats   struct {
		Additions int `json:"additions"`
		Deletions int `json:"deletions"`
	} `json:"stats"`
}

// ListEvents fetches commits authored by user in r, at or after since, via
// `gh api repos/{owner}/{repo}/commits?author=...&since=...`.
func (c *commanderGHClient) ListEvents(
	ctx context.Context,
	r RepoRef,
	user string,
	since int64,
) ([]types.Event, error) {
	endpoint := buildCommitsEndpoint(r, user, since)

	stdout, _, err := c.cmd.Output(ctx, ghBinary, []string{"api", endpoint})
	if err != nil {
		return nil, ctxerrors.Wrapf(
			err, "gh api commits for %s/%s", r.Owner, r.Repo,
		)
	}

	var commits []ghCommit
	if err := json.Unmarshal(stdout, &commits); err != nil {
		return nil, ctxerrors.Wrapf(
			ctxerrors.Join(err, commerr.ErrUnmarshalFailed),
			"unmarshal gh api commits for %s/%s", r.Owner, r.Repo,
		)
	}

	events := make([]types.Event, 0, len(commits))

	for _, commit := range commits {
		event, err := commitToEvent(r, commit)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

// buildCommitsEndpoint builds the `gh api` path+query for r's commits
// authored by user, optionally bounded by since (0 means unbounded).
func buildCommitsEndpoint(r RepoRef, user string, since int64) string {
	query := url.Values{}
	query.Set("author", user)

	if since > 0 {
		query.Set("since", time.Unix(since, 0).UTC().Format(time.RFC3339))
	}

	return fmt.Sprintf(
		"repos/%s/%s/commits?%s", r.Owner, r.Repo, query.Encode(),
	)
}

// commitToEvent normalizes a ghCommit into gitrakz's Event shape. Title is
// the commit's subject line (message up to the first newline); Raw carries
// the original JSON for re-derivation.
func commitToEvent(r RepoRef, commit ghCommit) (types.Event, error) {
	title, _, _ := strings.Cut(commit.Commit.Message, "\n")

	raw, err := json.Marshal(commit)
	if err != nil {
		return types.Event{}, ctxerrors.Wrapf(
			ctxerrors.Join(err, commerr.ErrMarshalFailed),
			"marshal raw commit %s", commit.SHA,
		)
	}

	return types.Event{
		ID: fmt.Sprintf(
			"%s:%s/%s:%s",
			types.EventTypeCommit, r.Owner, r.Repo, commit.SHA,
		),
		TS:        commit.Commit.Author.Date.Unix(),
		Type:      types.EventTypeCommit,
		Owner:     r.Owner,
		Repo:      r.Repo,
		SHA:       commit.SHA,
		Title:     title,
		URL:       commit.HTMLURL,
		Additions: commit.Stats.Additions,
		Deletions: commit.Stats.Deletions,
		Raw:       raw,
	}, nil
}
