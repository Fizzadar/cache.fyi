-- Create new table with the created_at column added
CREATE TABLE content_autotags_new (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    tag_id INTEGER NOT NULL REFERENCES tags(id),
    url_regex TEXT,
    UNIQUE(tag_id, url_regex)
);

-- Copy data from old table to new
INSERT INTO content_autotags_new (tag_id, url_regex)
SELECT tag_id, url_regex FROM content_autotags;

-- Drop old table, rename new table
DROP TABLE content_autotags;
ALTER TABLE content_autotags_new RENAME TO content_autotags;
