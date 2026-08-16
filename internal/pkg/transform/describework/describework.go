// Package describework implements the "describe-work" transform
// primitive: it turns a group of raw commit titles into a one-line,
// natural-language description of the work. A thin or junk title
// triggers a diff fetch so the description reflects what actually
// changed, not the subject line alone.
//
// CacheStore, LLMClient, and GHDiffer (interfaces.go) are small
// in-package interfaces the engine wires to real implementations (the
// db-backed llm_cache table, elelem, commander's gh shell-out), which
// keeps the package unit-testable with mocks.
//
// Every group is cached by (step, processingVersion, inputHash):
// processingVersion hashes the whole LLM config (prompt version +
// model), so a cache hit skips the LLM call entirely, and changing
// the prompt or model bumps the version, cleanly invalidating old
// entries without deleting them.
package describework

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
)

// Name is the primitive's registry key.
const Name = "describe-work"

// By selects which key describe-work groups commit events on.
type By string

const (
	ByDay  By = "day"
	ByRepo By = "repo"
)

const (
	defaultBy                = ByDay
	defaultPromptVersion     = "v1"
	defaultThinMessageMaxLen = 12
)

// dayLayout is the UTC date format used to bucket events when By is
// ByDay.
const dayLayout = "2006-01-02"

// junkPatternSource matches commit titles that carry no signal beyond
// the pattern itself (case-insensitive, whole-title match).
const junkPatternSource = `(?i)^(wip|fix|update|asdf|\.+)$`

// diffTruncateLen is the max number of bytes of a fetched diff
// appended to a thin commit's input segment.
const diffTruncateLen = 2000

// keySeparator joins the parts hashed into a processing version or a
// cache key.
const keySeparator = "|"

const (
	valueKeyCommits     = "commits"
	labelKeyDescription = "description"
)

// promptTemplate wraps the group's input text with the instruction
// sent to LLMClient.Describe.
const promptTemplate = `Write a one-line, professional summary of the
work described below, based on the commit titles (and any diffs)
provided. Respond with only the summary — no preamble, no quotes.

%s`

// stepParams is the JSON shape a "describe-work" pipeline step
// configures itself from. Every field is optional.
type stepParams struct {
	By                By     `json:"by"`
	PromptVersion     string `json:"promptVersion"`
	Model             string `json:"model"`
	ThinMessageMaxLen int    `json:"thinMessageMaxLen"`
}

// primitive reads State.Timeline and writes State.Rows: one Row per
// commit group, carrying a "description" label produced (or read from
// cache) by an LLM.
type primitive struct {
	cache CacheStore
	llm   LLMClient
	gh    GHDiffer

	by                By
	thinMessageMaxLen int
	processingVersion string
	junkPattern       *regexp.Regexp
}

// New builds a describe-work primitive from cache, llm, gh — the
// engine's wired implementations — and its JSON params, shaped
// {"by": "day"|"repo", "promptVersion": string, "model": string,
// "thinMessageMaxLen": int}. by defaults to "day", promptVersion to
// "v1", and thinMessageMaxLen to 12 when omitted. Returns
// ErrMissingDependency when any of cache, llm, gh is nil, or a
// wrapped ErrUnknownBy when by names anything else.
func New(
	cache CacheStore,
	llm LLMClient,
	gh GHDiffer,
	rawParams []byte,
) (transform.Primitive, error) {
	if cache == nil || llm == nil || gh == nil {
		return nil, ctxerrors.Wrap(ErrMissingDependency, Name)
	}

	sp := stepParams{
		By:                defaultBy,
		PromptVersion:     defaultPromptVersion,
		ThinMessageMaxLen: defaultThinMessageMaxLen,
	}

	if len(rawParams) > 0 {
		if err := json.Unmarshal(rawParams, &sp); err != nil {
			return nil, ctxerrors.Wrap(
				err, "unmarshal describe-work params",
			)
		}
	}

	if sp.By == "" {
		sp.By = defaultBy
	}

	switch sp.By {
	case ByDay, ByRepo:
	default:
		return nil, ctxerrors.Wrapf(ErrUnknownBy, "by %q", sp.By)
	}

	if sp.ThinMessageMaxLen <= 0 {
		sp.ThinMessageMaxLen = defaultThinMessageMaxLen
	}

	processingVersion := hashHex(
		sp.PromptVersion + keySeparator + sp.Model,
	)

	return primitive{
		cache: cache,
		llm:   llm,
		gh:    gh,

		by:                sp.By,
		thinMessageMaxLen: sp.ThinMessageMaxLen,
		processingVersion: processingVersion,
		junkPattern:       regexp.MustCompile(junkPatternSource),
	}, nil
}

// Name returns the primitive's registry key.
func (p primitive) Name() string {
	return Name
}

// Apply groups s.Timeline's commit events by p.by, describes each
// group (cache-first, LLM on miss), and appends one Row per group to
// s.Rows.
func (p primitive) Apply(ctx context.Context, s *transform.State) error {
	for _, group := range p.groupCommits(s.Timeline) {
		input, err := p.buildInput(ctx, group.commits)
		if err != nil {
			return ctxerrors.Wrapf(
				err, "build input for group %q", group.key,
			)
		}

		description, err := p.describe(ctx, input)
		if err != nil {
			return ctxerrors.Wrapf(
				err, "describe group %q", group.key,
			)
		}

		s.Rows = append(s.Rows, transform.Row{
			Key: group.key,
			Values: map[string]float64{
				valueKeyCommits: float64(len(group.commits)),
			},
			Labels: map[string]string{
				labelKeyDescription: description,
			},
		})
	}

	return nil
}

// commitGroup is one describe-work bucket: the group key (a day or a
// repo, per p.by) plus the commit events it holds, in Timeline order.
type commitGroup struct {
	key     string
	commits []types.Event
}

// groupCommits buckets every commit-type event in timeline by p.by,
// returning groups sorted by key ascending.
func (p primitive) groupCommits(timeline types.Timeline) []commitGroup {
	buckets := map[string][]types.Event{}
	keys := make([]string, 0, len(timeline))

	for _, event := range timeline {
		if event.Type != types.EventTypeCommit {
			continue
		}

		key := p.groupKey(event)

		if _, ok := buckets[key]; !ok {
			keys = append(keys, key)
		}

		buckets[key] = append(buckets[key], event)
	}

	sort.Strings(keys)

	groups := make([]commitGroup, 0, len(keys))
	for _, key := range keys {
		groups = append(
			groups, commitGroup{key: key, commits: buckets[key]},
		)
	}

	return groups
}

// groupKey extracts the grouping key for event according to p.by.
func (p primitive) groupKey(event types.Event) string {
	switch p.by {
	case ByDay:
		return time.Unix(event.TS, 0).UTC().Format(dayLayout)
	case ByRepo:
		return event.Repo
	}

	return ""
}

// buildInput joins commits into one input text, one line per commit.
// A commit whose title is thin (per p.isThin) has its truncated diff
// (fetched via p.gh) appended to its line.
func (p primitive) buildInput(
	ctx context.Context,
	commits []types.Event,
) (string, error) {
	segments := make([]string, 0, len(commits))

	for _, commit := range commits {
		segment := commit.Title

		if p.isThin(commit.Title) {
			diff, err := p.gh.Diff(
				ctx, commit.Owner, commit.Repo, commit.SHA,
			)
			if err != nil {
				return "", ctxerrors.Wrapf(
					err,
					"diff %s/%s@%s",
					commit.Owner, commit.Repo, commit.SHA,
				)
			}

			segment += "\n" + truncate(diff, diffTruncateLen)
		}

		segments = append(segments, segment)
	}

	return strings.Join(segments, "\n"), nil
}

// isThin reports whether title is too short or too generic to
// describe the work on its own — either signal means buildInput
// fetches the commit's diff for context.
func (p primitive) isThin(title string) bool {
	if len(title) < p.thinMessageMaxLen {
		return true
	}

	return p.junkPattern.MatchString(title)
}

// describe resolves input's description: a cache hit returns the
// stored output with no LLM call; a miss calls p.llm.Describe and
// stores the result under the computed cache key.
func (p primitive) describe(
	ctx context.Context,
	input string,
) (string, error) {
	inputHash := hashHex(input)
	cacheKey := hashHex(
		Name + keySeparator + p.processingVersion +
			keySeparator + inputHash,
	)

	output, hit, err := p.cache.Get(ctx, cacheKey)
	if err != nil {
		return "", ctxerrors.Wrap(err, "get cache")
	}

	if hit {
		return output, nil
	}

	output, err = p.llm.Describe(ctx, buildPrompt(input))
	if err != nil {
		return "", ctxerrors.Wrap(err, "llm describe")
	}

	if err := p.cache.Put(
		ctx, cacheKey, Name, p.processingVersion, inputHash, output,
	); err != nil {
		return "", ctxerrors.Wrap(err, "put cache")
	}

	return output, nil
}

// buildPrompt wraps input in the instruction sent to LLMClient.Describe.
func buildPrompt(input string) string {
	return fmt.Sprintf(promptTemplate, input)
}

// hashHex returns the hex-encoded SHA-256 digest of s.
func hashHex(s string) string {
	sum := sha256.Sum256([]byte(s))

	return hex.EncodeToString(sum[:])
}

// truncate returns the first maxLen bytes of s, or s unchanged when
// it's already within maxLen.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen]
}
