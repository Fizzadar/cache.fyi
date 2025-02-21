--
-- Pages
--

CREATE TABLE pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    path TEXT NOT NULL,
    name TEXT NOT NULL, -- dropped in migration 008

    content TEXT NOT NULL,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME,

    UNIQUE(path, name)
);

--
-- Content
--

CREATE TABLE content (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    type TEXT NOT NULL,
    -- Hash (part defined by the type), globally unique
    hash TEXT NOT NULL UNIQUE,

    -- URL this content was found on, or is (if type=url,file)
    url TEXT,

    -- Optional data, content type and size (if type=file)
    data BLOB,
    content_type TEXT,
    size_bytes INTEGER,

    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CHECK (
        -- URL type only requires url field
        (type = 'url' AND url IS NOT NULL)
        -- Data type requires content type & size bytes
        OR (
            type = 'file'
            AND data IS NOT NULL
            AND content_type IS NOT NULL
            AND size_bytes IS NOT NULL
        )
        -- Anything else is invalid
    )
);

--
-- Tags
--

CREATE TABLE tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE content_autotags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tag_id INTEGER NOT NULL REFERENCES tags(id),
    url_regex TEXT,
    UNIQUE(tag_id, url_regex)
);

CREATE TABLE page__tag (
    tag_id INTEGER NOT NULL REFERENCES tags(id),
    page_id INTEGER NOT NULL REFERENCES pages(id),
    UNIQUE(tag_id, page_id)
);

CREATE TABLE content__tag (
    tag_id INTEGER NOT NULL REFERENCES tags(id),
    content_id INTEGER NOT NULL REFERENCES contents(id),
    UNIQUE(tag_id, content_id)
);

--
-- System Log
--

CREATE TABLE log(
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    message TEXT NOT NULL,
    at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- all references dropped in migration 008
    page_id INTEGER REFERENCES pages(id),
    content_id INTEGER REFERENCES content(id),
    tag_id INTEGER REFERENCES tags(id)
);
