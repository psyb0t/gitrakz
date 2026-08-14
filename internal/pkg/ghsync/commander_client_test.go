package ghsync

import (
	"context"
	"testing"
	"time"

	"github.com/psyb0t/commander"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommanderGHClient_DiscoverRepos_Success(t *testing.T) {
	t.Parallel()

	cmd := commander.NewMock()
	cmd.Expect(
		ghBinary,
		"repo", "list", "psyb0t",
		"--json", "nameWithOwner",
		"--limit", "1000",
	).ReturnOutput([]byte(`[
		{"nameWithOwner":"psyb0t/gitrakz"},
		{"nameWithOwner":"octocat/hello-world"}
	]`))

	client := NewCommanderGHClient(cmd)

	repos, err := client.DiscoverRepos(context.Background(), "psyb0t")
	require.NoError(t, err)
	assert.Equal(t, []RepoRef{
		{Owner: "psyb0t", Repo: "gitrakz"},
		{Owner: "octocat", Repo: "hello-world"},
	}, repos)
}

func TestCommanderGHClient_DiscoverRepos_CommandError(t *testing.T) {
	t.Parallel()

	cmdErr := ctxerrors.New("gh not authenticated")

	cmd := commander.NewMock()
	cmd.Expect(
		ghBinary,
		"repo", "list", "psyb0t",
		"--json", "nameWithOwner",
		"--limit", "1000",
	).ReturnError(cmdErr)

	client := NewCommanderGHClient(cmd)

	_, err := client.DiscoverRepos(context.Background(), "psyb0t")
	require.Error(t, err)
	assert.ErrorIs(t, err, cmdErr)
}

func TestCommanderGHClient_DiscoverRepos_UnmarshalError(t *testing.T) {
	t.Parallel()

	cmd := commander.NewMock()
	cmd.Expect(
		ghBinary,
		"repo", "list", "psyb0t",
		"--json", "nameWithOwner",
		"--limit", "1000",
	).ReturnOutput([]byte("not json"))

	client := NewCommanderGHClient(cmd)

	_, err := client.DiscoverRepos(context.Background(), "psyb0t")
	require.Error(t, err)
	assert.ErrorIs(t, err, commerr.ErrUnmarshalFailed)
}

func TestCommanderGHClient_DiscoverRepos_MalformedNameWithOwner(t *testing.T) {
	t.Parallel()

	cmd := commander.NewMock()
	cmd.Expect(
		ghBinary,
		"repo", "list", "psyb0t",
		"--json", "nameWithOwner",
		"--limit", "1000",
	).ReturnOutput([]byte(`[{"nameWithOwner":"noSlashHere"}]`))

	client := NewCommanderGHClient(cmd)

	_, err := client.DiscoverRepos(context.Background(), "psyb0t")
	require.Error(t, err)
	assert.ErrorIs(t, err, commerr.ErrParseFailed)
}

func TestCommanderGHClient_ListEvents_Success(t *testing.T) {
	t.Parallel()

	const (
		user        = "psyb0t"
		since int64 = 0
	)

	r := RepoRef{Owner: "psyb0t", Repo: "gitrakz"}
	endpoint := buildCommitsEndpoint(r, user, since)

	cmd := commander.NewMock()
	cmd.Expect(ghBinary, "api", endpoint).ReturnOutput([]byte(`[
		{
			"sha": "abc123",
			"commit": {
				"message": "Fix bug\n\nLonger description here",
				"author": {"date": "2024-01-15T10:30:00Z"}
			},
			"html_url": "https://github.com/psyb0t/gitrakz/commit/abc123",
			"stats": {"additions": 10, "deletions": 3}
		},
		{
			"sha": "def456",
			"commit": {
				"message": "Add feature",
				"author": {"date": "2024-01-16T08:00:00Z"}
			},
			"html_url": "https://github.com/psyb0t/gitrakz/commit/def456",
			"stats": {"additions": 5, "deletions": 0}
		}
	]`))

	client := NewCommanderGHClient(cmd)

	events, err := client.ListEvents(context.Background(), r, user, since)
	require.NoError(t, err)
	require.Len(t, events, 2)

	wantTS0 := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC).Unix()
	assert.Equal(t, wantTS0, events[0].TS)
	assert.Equal(t, "commit:psyb0t/gitrakz:abc123", events[0].ID)
	assert.Equal(t, "Fix bug", events[0].Title)
	assert.Equal(t, "psyb0t", events[0].Owner)
	assert.Equal(t, "gitrakz", events[0].Repo)
	assert.Equal(t, "abc123", events[0].SHA)
	assert.Equal(t,
		"https://github.com/psyb0t/gitrakz/commit/abc123", events[0].URL)
	assert.Equal(t, 10, events[0].Additions)
	assert.Equal(t, 3, events[0].Deletions)
	assert.NotEmpty(t, events[0].Raw)

	wantTS1 := time.Date(2024, 1, 16, 8, 0, 0, 0, time.UTC).Unix()
	assert.Equal(t, wantTS1, events[1].TS)
	assert.Equal(t, "commit:psyb0t/gitrakz:def456", events[1].ID)
	assert.Equal(t, "Add feature", events[1].Title)
	assert.Equal(t, 5, events[1].Additions)
	assert.Equal(t, 0, events[1].Deletions)
}

func TestCommanderGHClient_ListEvents_CommandError(t *testing.T) {
	t.Parallel()

	r := RepoRef{Owner: "psyb0t", Repo: "gitrakz"}
	endpoint := buildCommitsEndpoint(r, "psyb0t", 0)

	cmdErr := ctxerrors.New("rate limited")

	cmd := commander.NewMock()
	cmd.Expect(ghBinary, "api", endpoint).ReturnError(cmdErr)

	client := NewCommanderGHClient(cmd)

	_, err := client.ListEvents(context.Background(), r, "psyb0t", 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, cmdErr)
}

func TestCommanderGHClient_ListEvents_UnmarshalError(t *testing.T) {
	t.Parallel()

	r := RepoRef{Owner: "psyb0t", Repo: "gitrakz"}
	endpoint := buildCommitsEndpoint(r, "psyb0t", 0)

	cmd := commander.NewMock()
	cmd.Expect(ghBinary, "api", endpoint).ReturnOutput([]byte("not json"))

	client := NewCommanderGHClient(cmd)

	_, err := client.ListEvents(context.Background(), r, "psyb0t", 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, commerr.ErrUnmarshalFailed)
}

func TestBuildCommitsEndpoint(t *testing.T) {
	t.Parallel()

	r := RepoRef{Owner: "psyb0t", Repo: "gitrakz"}

	testCases := []struct {
		name  string
		since int64
		want  string
	}{
		{
			"since zero omits the since query param",
			0,
			"repos/psyb0t/gitrakz/commits?author=psyb0t",
		},
		{
			"since positive adds an RFC3339 UTC since query param",
			1700000000,
			//nolint:lll // single URL literal; splitting would change the value
			"repos/psyb0t/gitrakz/commits?author=psyb0t&since=2023-11-14T22%3A13%3A20Z",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := buildCommitsEndpoint(r, "psyb0t", tc.since)
			assert.Equal(t, tc.want, got)
		})
	}
}
