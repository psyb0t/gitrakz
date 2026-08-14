CREATE TABLE sync_state (
    owner TEXT NOT NULL,
    repo TEXT NOT NULL,
    last_synced_ts INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (owner, repo)
);
