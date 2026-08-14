CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL DEFAULT '',
    filter TEXT NOT NULL DEFAULT '',
    form_values TEXT NOT NULL DEFAULT '',
    doc TEXT NOT NULL DEFAULT '',
    created_ts INTEGER NOT NULL DEFAULT 0
);
