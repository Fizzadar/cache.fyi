package database

import (
	"context"

	"github.com/dustin/go-humanize"
)

type DatabaseStats struct {
	PageSize              int64
	PageCount             int64
	DatabaseSize          int64
	DatabaseSizeFormatted string
	PageSizeFormatted     string
	ContentCount          int64
	PageRecordCount       int64
	TagCount              int64
	FreelistPages         int64
	SchemaVersion         string
	DatabaseFilename      string
}

func (d *Database) GetStats(ctx context.Context) (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// Get page size
	row := d.db.QueryRowContext(ctx, "PRAGMA page_size")
	if err := row.Scan(&stats.PageSize); err != nil {
		return nil, err
	}

	// Get page count
	row = d.db.QueryRowContext(ctx, "PRAGMA page_count")
	if err := row.Scan(&stats.PageCount); err != nil {
		return nil, err
	}

	// Calculate database size
	stats.DatabaseSize = stats.PageSize * stats.PageCount
	stats.DatabaseSizeFormatted = humanize.Bytes(uint64(stats.DatabaseSize))
	stats.PageSizeFormatted = humanize.Bytes(uint64(stats.PageSize))

	// Get freelist page count
	row = d.db.QueryRowContext(ctx, "PRAGMA freelist_count")
	if err := row.Scan(&stats.FreelistPages); err != nil {
		return nil, err
	}

	// Get schema version
	row = d.db.QueryRowContext(ctx, "PRAGMA schema_version")
	if err := row.Scan(&stats.SchemaVersion); err != nil {
		return nil, err
	}

	// Get database filename
	row = d.db.QueryRowContext(ctx, "PRAGMA database_list")
	var seq int64
	var name string
	if err := row.Scan(&seq, &name, &stats.DatabaseFilename); err != nil {
		return nil, err
	}

	// Get content count
	row = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM content")
	if err := row.Scan(&stats.ContentCount); err != nil {
		return nil, err
	}

	// Get page count
	row = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM pages")
	if err := row.Scan(&stats.PageRecordCount); err != nil {
		return nil, err
	}

	// Get tag count
	row = d.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags")
	if err := row.Scan(&stats.TagCount); err != nil {
		return nil, err
	}

	return stats, nil
}
