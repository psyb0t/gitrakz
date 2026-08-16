// Package passthrough implements the "passthrough" transform primitive: it
// projects the raw event timeline directly into a display table with no
// aggregation, grouping, or filtering.
package passthrough

import (
	"context"
	"strconv"
	"time"

	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
)

// Name is the primitive's registry key.
const Name = "passthrough"

// primitive reads State.Timeline and appends a blocks.Table listing every
// event, one row each. It takes no configuration.
type primitive struct{}

// New builds a passthrough primitive. The params argument is accepted per
// the transform.Factory signature but ignored — passthrough takes no config.
func New(_ []byte) (transform.Primitive, error) {
	return primitive{}, nil
}

// Name returns the primitive's registry key.
func (primitive) Name() string {
	return Name
}

// Apply appends a blocks.Table to s.Blocks with one row per event in
// s.Timeline.
func (primitive) Apply(_ context.Context, s *transform.State) error {
	columns := []string{
		"ts",
		"type",
		"owner",
		"repo",
		"title",
		"additions",
		"deletions",
		"url",
	}

	rows := make([][]string, 0, len(s.Timeline))

	for _, event := range s.Timeline {
		rows = append(rows, []string{
			time.Unix(event.TS, 0).UTC().Format(time.RFC3339),
			string(event.Type),
			event.Owner,
			event.Repo,
			event.Title,
			strconv.Itoa(event.Additions),
			strconv.Itoa(event.Deletions),
			event.URL,
		})
	}

	s.Blocks = append(s.Blocks, blocks.NewTable(columns, rows, nil))

	return nil
}
