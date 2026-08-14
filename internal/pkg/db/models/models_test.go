package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTableName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		got  string
		want string
	}{
		{"event", Event{}.TableName(), "events"},
		{"syncState", SyncState{}.TableName(), "sync_state"},
		{"template", Template{}.TableName(), "templates"},
		{"document", Document{}.TableName(), "documents"},
		{"llmCache", LLMCache{}.TableName(), "llm_cache"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, tc.got)
		})
	}
}
