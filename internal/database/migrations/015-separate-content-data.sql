-- Separate data column from content table into content_data table
-- NOTE: we can't delete the data column just set every value to an empty string
-- since the table self-references itself. And we can't use DROP COLUMN because
-- the column is part of the CHECK. To do the switcheroo we need to run w/o FK
-- enabled which is messy in the server context, so just empty the column. FIXME

-- Create the new content_data table
CREATE TABLE content_data (
    content_id INTEGER PRIMARY KEY REFERENCES content(id) ON DELETE CASCADE,
    data BLOB NOT NULL
);

-- Migrate existing data from content table to content_data table
INSERT INTO content_data (content_id, data)
SELECT id, data
FROM content
WHERE data IS NOT NULL;

-- Drop the data column from content table by emptying each row
UPDATE content SET data = X'';
