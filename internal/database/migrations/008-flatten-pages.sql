-- Drop log references as overly strict
CREATE TABLE temp__log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    message TEXT NOT NULL,
    at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    page_content TEXT,

    page_id INTEGER,
    content_id INTEGER,
    tag_id INTEGER
);

INSERT INTO temp__log (id, message, at, page_content, page_id, content_id, tag_id)
SELECT id, message, at, page_content, page_id, content_id, tag_id FROM log;

DROP TABLE log;
ALTER TABLE temp__log RENAME TO log;


-- Merge page path+name
CREATE TABLE temp__pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    path TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME
);

INSERT INTO temp__pages (id, path, content, created_at, updated_at)
SELECT id, concat(path, name), content, created_at, updated_at
FROM pages;

DROP TABLE pages;
ALTER TABLE temp__pages RENAME TO pages;
