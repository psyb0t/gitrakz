package aggregate

import "errors"

// ErrInvalidParams is returned by New when the "aggregate" step's params
// omit field or specify an op that isn't OpSum, OpAvg, OpMin, or OpMax.
var ErrInvalidParams = errors.New("invalid aggregate params")
