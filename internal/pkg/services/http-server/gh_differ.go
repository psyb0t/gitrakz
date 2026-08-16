package httpserver

import (
	"context"
	"fmt"

	"github.com/psyb0t/commander"
	"github.com/psyb0t/ctxerrors"
)

const (
	// ghBinaryName is the gh CLI executable shelled out to — matches
	// ghsync's own commander_client.go, duplicated here since that
	// const is unexported.
	ghBinaryName = "gh"

	// ghDiffAcceptHeader asks the GitHub REST API for the commit's
	// unified diff instead of its default JSON representation.
	ghDiffAcceptHeader = "Accept: application/vnd.github.v3.diff"
)

// commanderGHDiffer implements describework.GHDiffer by shelling out to the
// gh CLI via commander, the same invocation pattern ghsync uses.
type commanderGHDiffer struct {
	cmd commander.Commander
}

func newCommanderGHDiffer(cmd commander.Commander) *commanderGHDiffer {
	return &commanderGHDiffer{cmd: cmd}
}

// Diff fetches sha's unified diff in owner/repo via
// `gh api repos/{owner}/{repo}/commits/{sha}` with the diff accept header.
func (d *commanderGHDiffer) Diff(
	ctx context.Context,
	owner, repo, sha string,
) (string, error) {
	endpoint := fmt.Sprintf("repos/%s/%s/commits/%s", owner, repo, sha)

	stdout, _, err := d.cmd.Output(
		ctx,
		ghBinaryName,
		[]string{"api", endpoint, "-H", ghDiffAcceptHeader},
	)
	if err != nil {
		return "", ctxerrors.Wrapf(
			err, "gh api diff for %s/%s@%s", owner, repo, sha,
		)
	}

	return string(stdout), nil
}
