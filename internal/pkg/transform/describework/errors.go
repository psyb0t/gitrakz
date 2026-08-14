package describework

import "errors"

// ErrUnknownBy is returned by New when the "by" param names a field
// describe-work doesn't know how to group on.
var ErrUnknownBy = errors.New("unknown describe-work group field")

// ErrMissingDependency is returned by New when cache, llm, or gh is
// nil — describe-work cannot run without all three.
var ErrMissingDependency = errors.New(
	"describe-work missing required dependency",
)
