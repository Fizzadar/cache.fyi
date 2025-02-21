ALTER TABLE content ADD COLUMN parent_id INTEGER REFERENCES content(id);
