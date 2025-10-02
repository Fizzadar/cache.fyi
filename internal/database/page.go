package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/fizzadar/cache.fyi/internal/types"
)

var ErrPageNotFound = errors.New("no such page")

const (
	queryGetPageBase = `
		SELECT
			p.id, p.path, p.title, p.content, p.pinned, p.created_at, p.updated_at,
			group_concat(t.name) AS tag_names
		FROM pages AS p
		LEFT JOIN page__tag AS pt ON p.id = pt.page_id
		LEFT JOIN tags AS t ON t.id = pt.tag_id
	`
	queryGetPage = queryGetPageBase + `
		WHERE path = ?
		GROUP BY p.id
	`
	queryListPages = queryGetPageBase + `
		GROUP BY p.id
		ORDER BY path
	`
	queryListPinnedPages = queryGetPageBase + `
		WHERE p.pinned
		GROUP BY p.id
		ORDER BY path
	`
	queryListPagesForTag = queryGetPageBase + `
		WHERE t.id = ?
		GROUP BY p.id
		ORDER BY path
	`
	queryGetPagePath = `SELECT path FROM pages WHERE id = ?`
	queryGetPageID   = `SELECT id FROM pages WHERE path = ?`
	queryInsertPage  = `
		INSERT INTO pages (path, content, created_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
	`
	queryUpsertPage = `
		INSERT INTO pages (path, title, content, pinned)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (path) DO UPDATE SET
			updated_at = CURRENT_TIMESTAMP,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			pinned = EXCLUDED.pinned
	`
	queryInsertPageTag = `
		INSERT INTO page__tag (page_id, tag_id)
		VALUES (?, ?)
		ON CONFLICT DO NOTHING
	`
	queryDeletePageTag = `
		DELETE FROM page__tag
		WHERE page_id = ? AND tag_id = ?
	`
	queryCreatePageAutoTag = `
		INSERT INTO page_autotags (tag_id, path_regex)
		VALUES (?, ?)
	`
	queryListPageAutoTags = `
		SELECT id, tag_id, path_regex, created_at
		FROM page_autotags
	`
	queryDeletePageAutoTag = `
		DELETE FROM page_autotags
		WHERE id = ?
	`
)

func (d *Database) initPageStatements() error {
	var err error

	if d.stmtGetPage, err = d.db.Prepare(queryGetPage); err != nil {
		return err
	} else if d.stmtInsertPage, err = d.db.Prepare(queryInsertPage); err != nil {
		return err
	} else if d.stmtUpsertPage, err = d.db.Prepare(queryUpsertPage); err != nil {
		return err
	} else if d.stmtListPages, err = d.db.Prepare(queryListPages); err != nil {
		return err
	} else if d.stmtListPinnedPages, err = d.db.Prepare(queryListPinnedPages); err != nil {
		return err
	} else if d.stmtListPagesForTag, err = d.db.Prepare(queryListPagesForTag); err != nil {
		return err
	} else if d.stmtGetPageID, err = d.db.Prepare(queryGetPageID); err != nil {
		return err
	} else if d.stmtGetPagePath, err = d.db.Prepare(queryGetPagePath); err != nil {
		return err
	} else if d.stmtInsertPageTag, err = d.db.Prepare(queryInsertPageTag); err != nil {
		return err
	} else if d.stmtDeletePageTag, err = d.db.Prepare(queryDeletePageTag); err != nil {
		return err
	} else if d.stmtCreatePageAutoTag, err = d.db.Prepare(queryCreatePageAutoTag); err != nil {
		return err
	} else if d.stmtListPageAutoTags, err = d.db.Prepare(queryListPageAutoTags); err != nil {
		return err
	} else if d.stmtDeletePageAutoTag, err = d.db.Prepare(queryDeletePageAutoTag); err != nil {
		return err
	}

	return nil
}

func scanPageRow(row rowScanner) (*types.Page, error) {
	var page types.Page
	var tagNames *string
	if err := row.Scan(
		&page.ID,
		&page.Path,
		&page.Title,
		&page.Content,
		&page.Pinned,
		&page.CreatedAt,
		&page.UpdatedAt,
		&tagNames,
	); err != nil {
		return nil, err
	}

	if tagNames != nil {
		page.Tags = strings.Split(*tagNames, ",")
	}

	return &page, nil
}

func (d *Database) GetPage(ctx context.Context, path string) (*types.Page, error) {
	if page, err := scanPageRow(d.stmtGetPage.QueryRowContext(ctx, path)); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		return page, nil
	}
}

func (d *Database) UpsertPage(ctx context.Context, path, title, content string, pinned bool) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert/update the page
	_, err = tx.StmtContext(ctx, d.stmtUpsertPage).ExecContext(ctx, path, title, content, pinned)
	if err != nil {
		return err
	}

	// Get the page ID
	pageID, err := d.txPageIDForPathName(tx, path)
	if err != nil {
		return err
	}

	// Get all autotags and check for matches
	rows, err := tx.StmtContext(ctx, d.stmtListPageAutoTags).Query()
	if err != nil {
		return fmt.Errorf("failed to list autotags: %w", err)
	}
	defer rows.Close()

	var autoTags []types.PageAutoTag
	for rows.Next() {
		var tag types.PageAutoTag
		if err := rows.Scan(&tag.ID, &tag.TagID, &tag.PathRegex, &tag.CreatedAt); err != nil {
			return fmt.Errorf("failed to scan autotag: %w", err)
		}
		autoTags = append(autoTags, tag)
	}
	if err = rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate autotags: %w", err)
	}

	// Check each autotag for a match
	for _, autoTag := range autoTags {
		matched, err := regexp.MatchString(autoTag.PathRegex, path)
		if err != nil {
			return fmt.Errorf("failed to match regex %s: %w", autoTag.PathRegex, err)
		}
		if matched {
			// Add the tag if it matches
			if _, err := tx.StmtContext(ctx, d.stmtInsertPageTag).ExecContext(
				ctx, pageID, autoTag.TagID,
			); err != nil {
				return fmt.Errorf("failed to insert autotag: %w", err)
			}
		}
	}

	// Create the log entry
	if err = d.txCreateLog(tx, fmt.Sprintf("Upsert page: %s", path), &pageID, &content, nil, nil); err != nil {
		return fmt.Errorf("failed to create log entry: %w", err)
	}

	return tx.Commit()
}

func (d *Database) txPageIDForPathName(tx *sql.Tx, path string) (int64, error) {
	var id int64
	if err := tx.Stmt(d.stmtGetPageID).QueryRow(path).Scan(&id); err != nil {
		return id, err
	} else if id == 0 {
		return 0, ErrPageNotFound
	} else {
		return id, nil
	}
}

func (d *Database) GetPagePathFromID(ctx context.Context, id int64) (string, error) {
	var path string
	if err := d.stmtGetPagePath.QueryRowContext(ctx, id).Scan(&path); err != nil {
		return "", err
	} else if path == "" {
		return "", ErrPageNotFound
	} else {
		return path, nil
	}
}

func (d *Database) ListPages(ctx context.Context) ([]*types.Page, error) {
	rows, err := d.stmtListPages.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []*types.Page
	for rows.Next() {
		if page, err := scanPageRow(rows); err != nil {
			return nil, err
		} else {
			pages = append(pages, page)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pages, nil
}

func (d *Database) ListPinnedPages(ctx context.Context) ([]*types.Page, error) {
	rows, err := d.stmtListPinnedPages.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []*types.Page
	for rows.Next() {
		if page, err := scanPageRow(rows); err != nil {
			return nil, err
		} else {
			pages = append(pages, page)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pages, nil
}

func (d *Database) ListPagesForTagID(ctx context.Context, tagID int64) ([]*types.Page, error) {
	rows, err := d.stmtListPagesForTag.QueryContext(ctx, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []*types.Page
	for rows.Next() {
		if page, err := scanPageRow(rows); err != nil {
			return nil, err
		} else {
			pages = append(pages, page)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return pages, nil
}

func (d *Database) AddPageTag(ctx context.Context, pageID, tagID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.StmtContext(ctx, d.stmtInsertPageTag).ExecContext(ctx, pageID, tagID); err != nil {
		return err
	}

	if err := d.txCreateLog(tx, "Add tag to page", &pageID, nil, nil, &tagID); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) DeletePageTag(ctx context.Context, pageID, tagID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.StmtContext(ctx, d.stmtDeletePageTag).ExecContext(ctx, pageID, tagID); err != nil {
		return err
	}

	if err := d.txCreateLog(tx, "Remove tag from page", &pageID, nil, nil, &tagID); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) CreatePageAutoTag(tagID int, pathRegex string) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.Stmt(d.stmtCreatePageAutoTag).Exec(tagID, pathRegex)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	tagIDInt64 := int64(tagID)
	if err := d.txCreateLog(tx, "Create page auto-tag", nil, nil, nil, &tagIDInt64); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

func (d *Database) ListPageAutoTags() ([]types.PageAutoTag, error) {
	rows, err := d.stmtListPageAutoTags.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []types.PageAutoTag
	for rows.Next() {
		var tag types.PageAutoTag
		if err := rows.Scan(&tag.ID, &tag.TagID, &tag.PathRegex, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

func (d *Database) DeletePageAutoTag(id int) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Stmt(d.stmtDeletePageAutoTag).Exec(id); err != nil {
		return err
	}

	if err := d.txCreateLog(tx, "Delete page auto-tag", nil, nil, nil, nil); err != nil {
		return err
	}

	return tx.Commit()
}
