CREATE TABLE llm_settings (
    id TEXT PRIMARY KEY,
    model TEXT NOT NULL DEFAULT '',
    reasoning_effort TEXT NOT NULL DEFAULT '',
    temperature REAL NOT NULL DEFAULT 0,
    updated_ts INTEGER NOT NULL DEFAULT 0
);
