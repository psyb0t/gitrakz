CREATE TABLE events (
    id TEXT PRIMARY KEY,
    ts INTEGER NOT NULL,
    type TEXT NOT NULL,
    owner TEXT NOT NULL,
    repo TEXT NOT NULL,
    sha TEXT NOT NULL DEFAULT '',
    number INTEGER NOT NULL DEFAULT 0,
    title TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    additions INTEGER NOT NULL DEFAULT 0,
    deletions INTEGER NOT NULL DEFAULT 0,
    branch TEXT NOT NULL DEFAULT '',
    raw TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_events_owner_ts ON events(owner, ts);
CREATE INDEX idx_events_repo_ts ON events(repo, ts);
CREATE INDEX idx_events_type_ts ON events(type, ts);
