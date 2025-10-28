-- Re-process everything to populate linkwarden_link_id so archiving works
UPDATE content SET processed_at = NULL;
