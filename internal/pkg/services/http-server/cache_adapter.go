package httpserver

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
)

// describeWorkCacheStore implements describework.CacheStore over *db.Store.
// *db.Store's own cache methods are named LLMCacheGet/LLMCachePut rather
// than Get/Put, so it does not satisfy describework.CacheStore by itself —
// this is the thin adapter that renames the two calls.
type describeWorkCacheStore struct {
	store *db.Store
}

func newDescribeWorkCacheStore(store *db.Store) *describeWorkCacheStore {
	return &describeWorkCacheStore{store: store}
}

// Get returns the cached output for key, or ok=false with a nil error on a
// miss.
func (c *describeWorkCacheStore) Get(
	ctx context.Context,
	key string,
) (string, bool, error) {
	output, hit, err := c.store.LLMCacheGet(ctx, key)
	if err != nil {
		return "", false, ctxerrors.Wrap(err, "llm cache get")
	}

	return output, hit, nil
}

// Put stores output under key, replacing any previous entry for that key.
func (c *describeWorkCacheStore) Put(
	ctx context.Context,
	key, step, processingVersion, inputHash, output string,
) error {
	if err := c.store.LLMCachePut(
		ctx, key, step, processingVersion, inputHash, output,
	); err != nil {
		return ctxerrors.Wrap(err, "llm cache put")
	}

	return nil
}
