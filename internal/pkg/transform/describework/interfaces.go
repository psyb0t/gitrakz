package describework

import "context"

// CacheStore persists and retrieves LLM step outputs keyed by a
// deterministic hash of (step, processingVersion, inputHash) — the
// llm_cache table's shape (see common/transform doc + git-trakz.md's
// "LLM steps are versioned + cached"). Get's second return reports a
// cache hit; a miss returns ("", false, nil) with no error.
type CacheStore interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Put(
		ctx context.Context,
		key, step, processingVersion, inputHash, output string,
	) error
}

// LLMClient produces a natural-language response to a prompt. The
// engine wires this to elelem; describe-work never talks to a model
// provider directly.
type LLMClient interface {
	Describe(ctx context.Context, prompt string) (string, error)
}

// GHDiffer fetches the diff/patch for a single commit. The engine wires
// this to commander's gh shell-out; describe-work never shells out
// directly.
type GHDiffer interface {
	Diff(ctx context.Context, owner, repo, sha string) (string, error)
}
