package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventTypeConstants(t *testing.T) {
	testCases := []struct {
		name     string
		value    EventType
		expected string
	}{
		{name: "commit", value: EventTypeCommit, expected: "commit"},
		{name: "pr", value: EventTypePR, expected: "pr"},
		{name: "review", value: EventTypeReview, expected: "review"},
		{name: "issue", value: EventTypeIssue, expected: "issue"},
		{name: "release", value: EventTypeRelease, expected: "release"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, EventType(tc.expected), tc.value)
		})
	}
}

func TestEventJSONRoundTrip(t *testing.T) {
	original := Event{
		ID:        "commit:psyb0t/gitrakz:abc123",
		TS:        1700000000,
		Type:      EventTypeCommit,
		Owner:     "psyb0t",
		Repo:      "gitrakz",
		SHA:       "abc123",
		Number:    42,
		Title:     "add types package",
		URL:       "https://github.com/psyb0t/gitrakz/commit/abc123",
		Additions: 12,
		Deletions: 3,
		Branch:    "main",
		Raw:       json.RawMessage(`{"foo":"bar"}`),
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Event

	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}
