package llmstep

import "context"

// LLMClient runs one user-authored LLM step over serialized data. schema
// nil/empty selects a plain-text response; a non-empty schema selects
// JSON-schema-constrained structured output. The engine wires this to
// elelem; llm never talks to a model provider directly.
type LLMClient interface {
	Complete(
		ctx context.Context,
		instruction, data string,
		schema []byte,
	) (string, error)
}
