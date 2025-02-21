DROP TABLE content__tag;

CREATE TABLE content__tag (
    tag_id INTEGER NOT NULL REFERENCES tags(id),
    content_id INTEGER NOT NULL REFERENCES content(id),
    UNIQUE(tag_id, content_id)
);
