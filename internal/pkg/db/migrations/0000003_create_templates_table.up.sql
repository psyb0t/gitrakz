CREATE TABLE templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    form TEXT NOT NULL DEFAULT '',
    transform TEXT NOT NULL DEFAULT '',
    layout TEXT NOT NULL DEFAULT '',
    exports TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    builtin INTEGER NOT NULL DEFAULT 0,
    created_ts INTEGER NOT NULL DEFAULT 0
);
