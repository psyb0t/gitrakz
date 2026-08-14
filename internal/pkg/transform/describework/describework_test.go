package describework

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockCache is an in-memory CacheStore for tests. Zero value is unusable;
// build via newMockCache.
type mockCache struct {
	store    map[string]string
	getCalls int
	putCalls int
	getErr   error
	putErr   error
}

func newMockCache() *mockCache {
	return &mockCache{store: map[string]string{}}
}

func (m *mockCache) Get(
	_ context.Context, key string,
) (string, bool, error) {
	m.getCalls++

	if m.getErr != nil {
		return "", false, m.getErr
	}

	output, ok := m.store[key]

	return output, ok, nil
}

func (m *mockCache) Put(
	_ context.Context,
	key, _, _, _, output string,
) error {
	m.putCalls++

	if m.putErr != nil {
		return m.putErr
	}

	m.store[key] = output

	return nil
}

// mockLLM is a stub LLMClient that always returns output (or err).
type mockLLM struct {
	calls  int
	output string
	err    error
}

func (m *mockLLM) Describe(
	_ context.Context, _ string,
) (string, error) {
	m.calls++

	if m.err != nil {
		return "", m.err
	}

	return m.output, nil
}

// mockGH is a stub GHDiffer that always returns output (or err), recording
// the arguments of its last call.
type mockGH struct {
	calls     int
	output    string
	err       error
	lastOwner string
	lastRepo  string
	lastSHA   string
}

func (m *mockGH) Diff(
	_ context.Context, owner, repo, sha string,
) (string, error) {
	m.calls++
	m.lastOwner = owner
	m.lastRepo = repo
	m.lastSHA = sha

	if m.err != nil {
		return "", m.err
	}

	return m.output, nil
}

// commitEventOwner is the fixed owner every commitEvent test fixture
// carries — none of the tests exercise owner variation, only repo/sha/
// title/ts do.
const commitEventOwner = "acme"

// errMockFailure is a shared sentinel the mock CacheStore/LLMClient/
// GHDiffer return to verify Apply wraps and propagates their errors
// instead of swallowing them.
var errMockFailure = errors.New("mock failure")

// commitEvent builds a minimal commit-type Event for tests.
func commitEvent(repo, sha, title string, ts int64) types.Event {
	return types.Event{
		Type:  types.EventTypeCommit,
		Owner: commitEventOwner,
		Repo:  repo,
		SHA:   sha,
		Title: title,
		TS:    ts,
	}
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("missing dependency errors", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name  string
			cache CacheStore
			llm   LLMClient
			gh    GHDiffer
		}{
			{"nil cache", nil, &mockLLM{}, &mockGH{}},
			{"nil llm", newMockCache(), nil, &mockGH{}},
			{"nil gh", newMockCache(), &mockLLM{}, nil},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				_, err := New(tc.cache, tc.llm, tc.gh, nil)
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrMissingDependency)
			})
		}
	})

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()

		prim, err := New(newMockCache(), &mockLLM{}, &mockGH{}, nil)
		require.NoError(t, err)
		assert.Equal(t, Name, prim.Name())

		p, ok := prim.(primitive)
		require.True(t, ok)
		assert.Equal(t, ByDay, p.by)
		assert.Equal(t, defaultThinMessageMaxLen, p.thinMessageMaxLen)
		assert.Equal(
			t,
			hashHex(defaultPromptVersion+keySeparator),
			p.processingVersion,
		)
	})

	t.Run("custom params", func(t *testing.T) {
		t.Parallel()

		rawParams := []byte(
			`{"by":"repo","promptVersion":"v2","model":"glm-4",` +
				`"thinMessageMaxLen":5}`,
		)

		prim, err := New(newMockCache(), &mockLLM{}, &mockGH{}, rawParams)
		require.NoError(t, err)

		p, ok := prim.(primitive)
		require.True(t, ok)
		assert.Equal(t, ByRepo, p.by)
		assert.Equal(t, 5, p.thinMessageMaxLen)
		assert.Equal(
			t, hashHex("v2"+keySeparator+"glm-4"), p.processingVersion,
		)
	})

	t.Run("non-positive thinMessageMaxLen falls back to default", func(
		t *testing.T,
	) {
		t.Parallel()

		rawParams := []byte(`{"thinMessageMaxLen":0}`)

		prim, err := New(newMockCache(), &mockLLM{}, &mockGH{}, rawParams)
		require.NoError(t, err)

		p, ok := prim.(primitive)
		require.True(t, ok)
		assert.Equal(t, defaultThinMessageMaxLen, p.thinMessageMaxLen)
	})

	t.Run("unknown by errors", func(t *testing.T) {
		t.Parallel()

		_, err := New(
			newMockCache(), &mockLLM{}, &mockGH{}, []byte(`{"by":"nope"}`),
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnknownBy)
		assert.Contains(t, err.Error(), "nope")
	})

	t.Run("malformed params errors", func(t *testing.T) {
		t.Parallel()

		_, err := New(newMockCache(), &mockLLM{}, &mockGH{}, []byte(`{`))
		require.Error(t, err)
	})
}

func TestPrimitive_Name(t *testing.T) {
	t.Parallel()

	prim, err := New(newMockCache(), &mockLLM{}, &mockGH{}, nil)
	require.NoError(t, err)
	assert.Equal(t, "describe-work", prim.Name())
}

func TestPrimitive_Apply(t *testing.T) {
	t.Parallel()

	t.Run("cache miss calls llm once and puts", func(t *testing.T) {
		t.Parallel()

		cache := newMockCache()
		llm := &mockLLM{output: "shipped the retry-backoff rewrite"}
		gh := &mockGH{}

		prim, err := New(cache, llm, gh, nil)
		require.NoError(t, err)

		timeline := types.Timeline{
			commitEvent(
				"widgets", "sha1",
				"implement retry backoff logic", 1700000000,
			),
		}

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		assert.Equal(t, 1, llm.calls)
		assert.Equal(t, 1, cache.putCalls)
		require.Len(t, s.Rows, 1)
		assert.Equal(
			t,
			"shipped the retry-backoff rewrite",
			s.Rows[0].Labels[labelKeyDescription],
		)
		assert.Equal(
			t, float64(1), s.Rows[0].Values[valueKeyCommits],
		)
	})

	t.Run("cache hit skips llm entirely", func(t *testing.T) {
		t.Parallel()

		cache := newMockCache()
		llm := &mockLLM{output: "must never be used"}
		gh := &mockGH{}

		prim, err := New(cache, llm, gh, nil)
		require.NoError(t, err)

		p, ok := prim.(primitive)
		require.True(t, ok)

		title := "implement retry backoff logic"
		cacheKey := hashHex(
			Name + keySeparator + p.processingVersion +
				keySeparator + hashHex(title),
		)
		cache.store[cacheKey] = "cached summary from a previous run"

		timeline := types.Timeline{
			commitEvent("widgets", "sha1", title, 1700000000),
		}

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		assert.Equal(t, 0, llm.calls)
		assert.Equal(t, 1, cache.getCalls)
		assert.Equal(t, 0, cache.putCalls)
		require.Len(t, s.Rows, 1)
		assert.Equal(
			t,
			"cached summary from a previous run",
			s.Rows[0].Labels[labelKeyDescription],
		)
	})

	t.Run("thin commit triggers exactly one diff call", func(t *testing.T) {
		t.Parallel()

		cache := newMockCache()
		llm := &mockLLM{output: "fixed the thing"}
		gh := &mockGH{output: "diff --git a/x b/x\n+fixed"}

		prim, err := New(cache, llm, gh, nil)
		require.NoError(t, err)

		timeline := types.Timeline{
			commitEvent("widgets", "deadbeef", "short", 1700000000),
		}

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		assert.Equal(t, 1, gh.calls)
		assert.Equal(t, "acme", gh.lastOwner)
		assert.Equal(t, "widgets", gh.lastRepo)
		assert.Equal(t, "deadbeef", gh.lastSHA)
	})

	t.Run("junk pattern alone triggers diff regardless of length", func(
		t *testing.T,
	) {
		t.Parallel()

		cache := newMockCache()
		llm := &mockLLM{output: "cleaned up formatting"}
		gh := &mockGH{output: "diff content"}

		// 15 dots: not "thin" by length (>= thinMessageMaxLen 12) but
		// matches the junk pattern (whole title is just punctuation).
		title := strings.Repeat(".", 15)

		timeline := types.Timeline{
			commitEvent("widgets", "sha1", title, 1700000000),
		}

		prim, err := New(cache, llm, gh, nil)
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		assert.Equal(t, 1, gh.calls)
	})

	t.Run("non-thin commit does not fetch diff", func(t *testing.T) {
		t.Parallel()

		cache := newMockCache()
		llm := &mockLLM{output: "described without a diff"}
		gh := &mockGH{}

		timeline := types.Timeline{
			commitEvent(
				"widgets", "sha1",
				"add pagination to the timeline endpoint", 1700000000,
			),
		}

		prim, err := New(cache, llm, gh, nil)
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		assert.Equal(t, 0, gh.calls)
	})

	t.Run(
		"same input and version across runs hits cache once",
		func(t *testing.T) {
			t.Parallel()

			cache := newMockCache()
			llm := &mockLLM{output: "consistent summary"}
			gh := &mockGH{}

			timeline := types.Timeline{
				commitEvent(
					"widgets", "sha1",
					"refactor the session clustering algorithm",
					1700000000,
				),
			}

			prim1, err := New(cache, llm, gh, nil)
			require.NoError(t, err)

			s1 := &transform.State{Timeline: timeline}
			require.NoError(t, prim1.Apply(context.Background(), s1))

			prim2, err := New(cache, llm, gh, nil)
			require.NoError(t, err)

			s2 := &transform.State{Timeline: timeline}
			require.NoError(t, prim2.Apply(context.Background(), s2))

			assert.Equal(t, 1, llm.calls)
			require.Len(t, s1.Rows, 1)
			require.Len(t, s2.Rows, 1)
			assert.Equal(
				t,
				s1.Rows[0].Labels[labelKeyDescription],
				s2.Rows[0].Labels[labelKeyDescription],
			)
		},
	)

	t.Run("groups by repo when configured", func(t *testing.T) {
		t.Parallel()

		cache := newMockCache()
		llm := &mockLLM{output: "repo summary"}
		gh := &mockGH{}

		timeline := types.Timeline{
			commitEvent(
				"beta", "s1",
				"add the billing webhook handler", 1700000000,
			),
			commitEvent(
				"alpha", "s2",
				"improve the session gap heuristic", 1700003600,
			),
			commitEvent(
				"alpha", "s3",
				"polish the onboarding docs further", 1700007200,
			),
		}

		prim, err := New(cache, llm, gh, []byte(`{"by":"repo"}`))
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		require.Len(t, s.Rows, 2)
		assert.Equal(t, "alpha", s.Rows[0].Key)
		assert.Equal(t, float64(2), s.Rows[0].Values[valueKeyCommits])
		assert.Equal(t, "beta", s.Rows[1].Key)
		assert.Equal(t, float64(1), s.Rows[1].Values[valueKeyCommits])
	})

	t.Run("groups by day in UTC and sorts ascending", func(t *testing.T) {
		t.Parallel()

		const (
			day1Start = 1699920000 // 2023-11-14T00:00:00Z
			day1End   = 1700006399 // 2023-11-14T23:59:59Z
			day2Start = 1700006400 // 2023-11-15T00:00:00Z
		)

		cache := newMockCache()
		llm := &mockLLM{output: "day summary"}
		gh := &mockGH{}

		timeline := types.Timeline{
			commitEvent(
				"widgets", "s1",
				"add pagination to the timeline endpoint", day1Start,
			),
			commitEvent(
				"widgets", "s2",
				"improve the session gap heuristic", day1End,
			),
			commitEvent(
				"widgets", "s3",
				"polish the onboarding docs further", day2Start,
			),
		}

		prim, err := New(cache, llm, gh, nil)
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		require.Len(t, s.Rows, 2)
		assert.Equal(t, "2023-11-14", s.Rows[0].Key)
		assert.Equal(t, float64(2), s.Rows[0].Values[valueKeyCommits])
		assert.Equal(t, "2023-11-15", s.Rows[1].Key)
		assert.Equal(t, float64(1), s.Rows[1].Values[valueKeyCommits])
	})

	t.Run("non-commit events are ignored", func(t *testing.T) {
		t.Parallel()

		cache := newMockCache()
		llm := &mockLLM{output: "unused"}
		gh := &mockGH{}

		timeline := types.Timeline{
			{Type: types.EventTypePR, TS: 1700000000, Title: "a PR"},
			{Type: types.EventTypeIssue, TS: 1700000000, Title: "an issue"},
		}

		prim, err := New(cache, llm, gh, nil)
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		assert.Empty(t, s.Rows)
		assert.Equal(t, 0, llm.calls)
	})

	t.Run("cache get error is wrapped and returned", func(t *testing.T) {
		t.Parallel()

		cache := newMockCache()
		cache.getErr = errMockFailure

		prim, err := New(cache, &mockLLM{}, &mockGH{}, nil)
		require.NoError(t, err)

		timeline := types.Timeline{
			commitEvent(
				"widgets", "sha1",
				"add pagination to the timeline endpoint", 1700000000,
			),
		}

		s := &transform.State{Timeline: timeline}
		err = prim.Apply(context.Background(), s)
		require.Error(t, err)
		assert.ErrorIs(t, err, errMockFailure)
	})

	t.Run("llm error is wrapped and returned", func(t *testing.T) {
		t.Parallel()

		llm := &mockLLM{err: errMockFailure}

		prim, err := New(newMockCache(), llm, &mockGH{}, nil)
		require.NoError(t, err)

		timeline := types.Timeline{
			commitEvent(
				"widgets", "sha1",
				"add pagination to the timeline endpoint", 1700000000,
			),
		}

		s := &transform.State{Timeline: timeline}
		err = prim.Apply(context.Background(), s)
		require.Error(t, err)
		assert.ErrorIs(t, err, errMockFailure)
	})

	t.Run("cache put error is wrapped and returned", func(t *testing.T) {
		t.Parallel()

		cache := newMockCache()
		cache.putErr = errMockFailure

		prim, err := New(cache, &mockLLM{output: "x"}, &mockGH{}, nil)
		require.NoError(t, err)

		timeline := types.Timeline{
			commitEvent(
				"widgets", "sha1",
				"add pagination to the timeline endpoint", 1700000000,
			),
		}

		s := &transform.State{Timeline: timeline}
		err = prim.Apply(context.Background(), s)
		require.Error(t, err)
		assert.ErrorIs(t, err, errMockFailure)
	})

	t.Run("gh diff error is wrapped and returned", func(t *testing.T) {
		t.Parallel()

		gh := &mockGH{err: errMockFailure}

		prim, err := New(newMockCache(), &mockLLM{}, gh, nil)
		require.NoError(t, err)

		timeline := types.Timeline{
			commitEvent("widgets", "sha1", "short", 1700000000),
		}

		s := &transform.State{Timeline: timeline}
		err = prim.Apply(context.Background(), s)
		require.Error(t, err)
		assert.ErrorIs(t, err, errMockFailure)
	})

	t.Run("empty timeline produces no rows", func(t *testing.T) {
		t.Parallel()

		prim, err := New(newMockCache(), &mockLLM{}, &mockGH{}, nil)
		require.NoError(t, err)

		s := &transform.State{}
		require.NoError(t, prim.Apply(context.Background(), s))
		assert.Empty(t, s.Rows)
	})
}
