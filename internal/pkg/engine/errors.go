package engine

import "errors"

// ErrUnknownLayoutType is returned when a template's layout names a block
// type the engine doesn't know how to render.
var ErrUnknownLayoutType = errors.New("unknown layout block type")
