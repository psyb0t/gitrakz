package models

// SyncState records the last synced unix timestamp per (owner, repo) so the
// incremental sync only fetches events newer than the last run.
type SyncState struct {
	Owner        string
	Repo         string
	LastSyncedTS int64
}

// TableName is the sync_state table.
func (SyncState) TableName() string {
	return "sync_state"
}
