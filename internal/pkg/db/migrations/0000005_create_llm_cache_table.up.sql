CREATE TABLE llm_cache (
    key TEXT PRIMARY KEY,
    step TEXT NOT NULL DEFAULT '',
    processing_version TEXT NOT NULL DEFAULT '',
    input_hash TEXT NOT NULL DEFAULT '',
    output TEXT NOT NULL DEFAULT '',
    created_ts INTEGER NOT NULL DEFAULT 0
);
