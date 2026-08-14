package transform

import "errors"

// ErrUnknownPrimitive is returned by Registry.Build when a template's transform
// step names a primitive that was never registered.
var ErrUnknownPrimitive = errors.New("unknown transform primitive")
