// Package aggregate implements the "aggregate" transform primitive: it folds
// every existing State.Rows entry's Values[field] down to one summary value
// (sum, avg, min, or max) and appends the result as a new Row.
package aggregate

import (
	"context"
	"encoding/json"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
)

// Name is the primitive's registry key.
const Name = "aggregate"

// defaultKey is the appended Row's Key when params.Key is empty.
const defaultKey = "total"

// Op is the finite set of aggregation operations "aggregate" supports.
type Op = string

const (
	OpSum Op = "sum"
	OpAvg Op = "avg"
	OpMin Op = "min"
	OpMax Op = "max"
)

// paramsInput is the JSON shape of an "aggregate" pipeline step's params.
type paramsInput struct {
	Op    Op     `json:"op"`
	Field string `json:"field"`
	Key   string `json:"key"`
}

// primitive folds s.Rows down to one summary Row over a single Values field.
type primitive struct {
	op    Op
	field string
	key   string
}

// New builds an "aggregate" primitive from its JSON params. It returns
// ErrInvalidParams (wrapped) when field is empty or op is not one of
// OpSum, OpAvg, OpMin, OpMax.
//

func New(params []byte) (transform.Primitive, error) {
	var p paramsInput
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, ctxerrors.Wrap(err, "unmarshal aggregate params")
	}

	if p.Field == "" {
		return nil, ctxerrors.Wrap(ErrInvalidParams, "field is required")
	}

	switch p.Op {
	case OpSum, OpAvg, OpMin, OpMax:
	default:
		return nil, ctxerrors.Wrapf(ErrInvalidParams, "unknown op %q", p.Op)
	}

	key := p.Key
	if key == "" {
		key = defaultKey
	}

	return primitive{op: p.Op, field: p.Field, key: key}, nil
}

// Name returns the primitive's registry key.
func (primitive) Name() string {
	return Name
}

// Apply computes p.op over every s.Rows entry's Values[p.field] (rows
// lacking the field are skipped) and appends one summary Row keyed p.key.
func (p primitive) Apply(_ context.Context, s *transform.State) error {
	stats := collectFieldStats(s.Rows, p.field)
	result := computeOpResult(p.op, stats)

	s.Rows = append(s.Rows, transform.Row{
		Key: p.key,
		Values: map[string]float64{
			p.field: result,
		},
	})

	return nil
}

// fieldStats accumulates the sum, min, max, and contributing-row count of
// one Values field across a set of Rows.
type fieldStats struct {
	sum    float64
	minVal float64
	maxVal float64
	count  int
}

// collectFieldStats scans rows and accumulates fieldStats for field. Rows
// lacking field are skipped.
func collectFieldStats(rows []transform.Row, field string) fieldStats {
	var stats fieldStats

	for _, row := range rows {
		val, ok := row.Values[field]
		if !ok {
			continue
		}

		if stats.count == 0 || val < stats.minVal {
			stats.minVal = val
		}

		if stats.count == 0 || val > stats.maxVal {
			stats.maxVal = val
		}

		stats.sum += val
		stats.count++
	}

	return stats
}

// computeOpResult applies op to stats. avg over zero contributing rows
// yields 0.
func computeOpResult(op Op, stats fieldStats) float64 {
	switch op {
	case OpSum:
		return stats.sum
	case OpAvg:
		if stats.count == 0 {
			return 0
		}

		return stats.sum / float64(stats.count)
	case OpMin:
		return stats.minVal
	case OpMax:
		return stats.maxVal
	default:
		return 0
	}
}
