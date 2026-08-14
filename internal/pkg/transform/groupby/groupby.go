// Package groupby implements the "group-by" transform primitive: it buckets
// the timeline by a chosen Event field and writes one Row per bucket to
// State.Rows, each carrying the bucket's event count and additions/
// deletions totals.
package groupby

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
)

// Name is the primitive's registry key.
const Name = "group-by"

// By selects which Event field group-by buckets on.
type By string

const (
	ByOwner  By = "owner"
	ByRepo   By = "repo"
	ByType   By = "type"
	ByBranch By = "branch"
	ByDay    By = "day"
)

// dayLayout is the UTC date format used to bucket events when By is ByDay.
const dayLayout = "2006-01-02"

const (
	valueKeyCount     = "count"
	valueKeyAdditions = "additions"
	valueKeyDeletions = "deletions"
)

// primitive reads State.Timeline and writes State.Rows: one Row per bucket,
// keyed by the chosen Event field and sorted by Key ascending.
type primitive struct {
	by By
}

// New builds the group-by primitive from its JSON params, shaped
// {"by": "owner"|"repo"|"type"|"branch"|"day"}. by defaults to ByOwner
// when params is empty or "by" is omitted. Returns a wrapped ErrUnknownBy
// when by names anything else.
//

func New(rawParams []byte) (transform.Primitive, error) {
	cfg := struct {
		By By `json:"by"`
	}{By: ByOwner}

	if len(rawParams) > 0 {
		if err := json.Unmarshal(rawParams, &cfg); err != nil {
			return nil, ctxerrors.Wrap(err, "unmarshal group-by params")
		}
	}

	if cfg.By == "" {
		cfg.By = ByOwner
	}

	switch cfg.By {
	case ByOwner, ByRepo, ByType, ByBranch, ByDay:
	default:
		return nil, ctxerrors.Wrapf(ErrUnknownBy, "by %q", cfg.By)
	}

	return primitive{by: cfg.By}, nil
}

// Name returns the primitive's registry key.
func (p primitive) Name() string {
	return Name
}

// Apply buckets s.Timeline by p.by and writes one Row per bucket to
// s.Rows, sorted by Key ascending.
func (p primitive) Apply(_ context.Context, s *transform.State) error {
	buckets := make(map[string]*transform.Row, len(s.Timeline))
	keys := make([]string, 0, len(s.Timeline))

	for _, event := range s.Timeline {
		key := bucketKey(p.by, event)

		row, ok := buckets[key]
		if !ok {
			row = &transform.Row{
				Key:    key,
				Values: map[string]float64{},
			}
			buckets[key] = row
			keys = append(keys, key)
		}

		row.Values[valueKeyCount]++
		row.Values[valueKeyAdditions] += float64(event.Additions)
		row.Values[valueKeyDeletions] += float64(event.Deletions)
	}

	sort.Strings(keys)

	rows := make([]transform.Row, 0, len(keys))
	for _, key := range keys {
		rows = append(rows, *buckets[key])
	}

	s.Rows = rows

	return nil
}

// bucketKey extracts the grouping key for event according to by.
func bucketKey(by By, event types.Event) string {
	switch by {
	case ByOwner:
		return event.Owner
	case ByRepo:
		return event.Repo
	case ByType:
		return string(event.Type)
	case ByBranch:
		return event.Branch
	case ByDay:
		return time.Unix(event.TS, 0).UTC().Format(dayLayout)
	}

	return ""
}
