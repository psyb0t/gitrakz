package groupby

import "errors"

// ErrUnknownBy is returned by New when the "by" param names a field
// group-by doesn't know how to bucket on.
var ErrUnknownBy = errors.New("unknown group-by field")
