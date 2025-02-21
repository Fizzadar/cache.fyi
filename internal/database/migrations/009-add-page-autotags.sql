CREATE TABLE page_autotags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tag_id INTEGER NOT NULL REFERENCES tags(id),
    path_regex TEXT,
    UNIQUE(tag_id, path_regex)
); 