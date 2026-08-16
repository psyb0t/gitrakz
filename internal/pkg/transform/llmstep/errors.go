package llmstep

import "errors"

// ErrMissingDependency is returned by New when llm is nil — the llm
// primitive cannot run without a client.
var ErrMissingDependency = errors.New("llm missing required dependency")

// ErrMissingInstruction is returned by New when the step params omit the
// required "instruction" field.
var ErrMissingInstruction = errors.New("llm missing required instruction param")
