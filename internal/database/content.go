package database

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/fizzadar/cache.fyi/internal/types"
)

const (
	queryGetContentBase = `
		SELECT
			c.id, c.type, c.hash, c.url, c.content_type, c.filename, c.size_bytes,
			c.created_at, c.processed_at, c.parent_id,
			group_concat(t.name) AS tag_names
		FROM content AS c
		LEFT JOIN content__tag AS ct ON c.id = ct.content_id
		LEFT JOIN tags AS t ON t.id = ct.tag_id
	`
	queryGetContent = queryGetContentBase + `
		WHERE c.id = ?
		GROUP BY c.id
	`
	queryListContent = queryGetContentBase + `
		GROUP BY c.id
		ORDER BY c.id DESC
	`
	queryListContentForTag = queryGetContentBase + `
		WHERE t.id = ?
		GROUP BY c.id
		ORDER BY c.id DESC
	`
	queryListContentToProcess = queryGetContentBase + `
		WHERE c.processed_at IS NULL
		GROUP BY c.id
		ORDER BY c.id
		LIMIT ?
	`
	queryInsertContent = `
		INSERT INTO content (type, hash, url, data, content_type, filename, size_bytes, created_at, parent_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
	`
	queryCreateContentAutoTag = `
		INSERT INTO content_autotags (tag_id, url_regex)
		VALUES (?, ?)
	`
	queryListContentAutoTags = `
		SELECT id, tag_id, url_regex, created_at
		FROM content_autotags
	`
	queryDeleteContentAutoTag = `
		DELETE FROM content_autotags
		WHERE id = ?
	`
	queryInsertContentTag = `
		INSERT INTO content__tag (content_id, tag_id)
		VALUES (?, ?)
		ON CONFLICT DO NOTHING
	`
	queryDeleteContentTag = `
		DELETE FROM content__tag
		WHERE content_id = ? AND tag_id = ?
	`
	queryGetContentData = `
		SELECT data FROM content WHERE id = ?
	`
	querySetContentProcessed = `
		UPDATE content
		SET processed_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
)

func (d *Database) initContentStatements() error {
	var err error

	if d.stmtGetContent, err = d.db.Prepare(queryGetContent); err != nil {
		return err
	} else if d.stmtGetContentData, err = d.db.Prepare(queryGetContentData); err != nil {
		return err
	} else if d.stmtListContent, err = d.db.Prepare(queryListContent); err != nil {
		return err
	} else if d.stmtListContentForTag, err = d.db.Prepare(queryListContentForTag); err != nil {
		return err
	} else if d.stmtInsertContent, err = d.db.Prepare(queryInsertContent); err != nil {
		return err
	} else if d.stmtCreateContentAutoTag, err = d.db.Prepare(queryCreateContentAutoTag); err != nil {
		return err
	} else if d.stmtListContentAutoTags, err = d.db.Prepare(queryListContentAutoTags); err != nil {
		return err
	} else if d.stmtDeleteContentAutoTag, err = d.db.Prepare(queryDeleteContentAutoTag); err != nil {
		return err
	} else if d.stmtInsertContentTag, err = d.db.Prepare(queryInsertContentTag); err != nil {
		return err
	} else if d.stmtDeleteContentTag, err = d.db.Prepare(queryDeleteContentTag); err != nil {
		return err
	} else if d.stmtSetContentProcessedAt, err = d.db.Prepare(querySetContentProcessed); err != nil {
		return err
	} else if d.stmtListContentToProcess, err = d.db.Prepare(queryListContentToProcess); err != nil {
		return err
	}

	return nil
}

func scanContentRow(row rowScanner) (*types.Content, error) {
	var content types.Content
	var tagNames *string
	if err := row.Scan(
		&content.ID,
		&content.Type,
		&content.Hash,
		&content.URL,
		&content.ContentType,
		&content.Filename,
		&content.SizeBytes,
		&content.CreatedAt,
		&content.ProcessedAt,
		&content.ParentID,
		&tagNames,
	); err != nil {
		return nil, err
	}

	if tagNames != nil {
		content.Tags = strings.Split(*tagNames, ",")
	}

	return &content, nil
}

func (d *Database) GetContent(ctx context.Context, id int64) (*types.Content, error) {
	row := d.stmtGetContent.QueryRowContext(ctx, id)
	if content, err := scanContentRow(row); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		return content, nil
	}
}

func (d *Database) GetContentData(ctx context.Context, id int64) ([]byte, error) {
	var data []byte
	if err := d.stmtGetContentData.QueryRowContext(ctx, id).Scan(&data); errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	} else {
		return data, nil
	}
}

func (d *Database) CreateContent(
	ctx context.Context,
	cType types.ContentType,
	hash, url string,
	data []byte,
	contentType, filename *string,
	tagIDs []int64,
	parentID *int64,
) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return -1, err
	}
	defer tx.Rollback()

	result, err := tx.StmtContext(ctx, d.stmtInsertContent).ExecContext(ctx, cType, hash, url, data, contentType, filename, len(data), parentID)
	if err != nil {
		return -1, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}

	for _, tagID := range tagIDs {
		if _, err := tx.StmtContext(ctx, d.stmtInsertContentTag).ExecContext(ctx, id, tagID); err != nil {
			return -1, err
		}
	}

	d.txCreateLog(tx, fmt.Sprintf("Create content type=%s, url=%s", cType, url), nil, nil, &id, nil)

	if err := tx.Commit(); err != nil {
		return -1, err
	}

	return id, nil
}

func (d *Database) CreateURLContent(ctx context.Context, url string, tagIDs []int64, parentID *int64) (int64, error) {
	hashStr := string(types.ContentTypeURL) + url
	hashB := sha256.Sum256([]byte(hashStr))
	hash := base64.RawURLEncoding.EncodeToString(hashB[:])
	return d.CreateContent(ctx, types.ContentTypeURL, hash, url, nil, nil, nil, tagIDs, parentID)
}

type ListContentOptions struct {
	Limit int
}

func (d *Database) ListContent(ctx context.Context) ([]*types.Content, error) {
	rows, err := d.stmtListContent.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contents := make([]*types.Content, 0, 1000) // TODO: size

	for rows.Next() {
		if content, err := scanContentRow(rows); err != nil {
			return nil, err
		} else {
			contents = append(contents, content)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return contents, nil
}

func (d *Database) ListContentForTagID(ctx context.Context, tagID int64) ([]*types.Content, error) {
	rows, err := d.stmtListContentForTag.QueryContext(ctx, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contents := make([]*types.Content, 0, 1000) // TODO: size

	for rows.Next() {
		if content, err := scanContentRow(rows); err != nil {
			return nil, err
		} else {
			contents = append(contents, content)
		}
	}

	return contents, rows.Err()
}

func (d *Database) ListContentToProcess(ctx context.Context, limit int) ([]*types.Content, error) {
	rows, err := d.stmtListContentToProcess.QueryContext(ctx, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	contents := make([]*types.Content, 0, limit)

	for rows.Next() {
		if content, err := scanContentRow(rows); err != nil {
			return nil, err
		} else {
			contents = append(contents, content)
		}
	}

	return contents, rows.Err()
}

func (d *Database) CreateContentAutoTag(tagID int, urlRegex string) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.Stmt(d.stmtCreateContentAutoTag).Exec(tagID, urlRegex)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	tagIDInt64 := int64(tagID)
	if err := d.txCreateLog(tx, fmt.Sprintf("Create content auto-tag: tag_id=%d regex=%s", tagID, urlRegex), nil, nil, nil, &tagIDInt64); err != nil {
		return 0, err
	}

	return id, tx.Commit()
}

func (d *Database) ListContentAutoTags() ([]types.ContentAutoTag, error) {
	rows, err := d.stmtListContentAutoTags.Query()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []types.ContentAutoTag
	for rows.Next() {
		var tag types.ContentAutoTag
		if err := rows.Scan(&tag.ID, &tag.TagID, &tag.URLRegex, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

func (d *Database) DeleteContentAutoTag(id int) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Stmt(d.stmtDeleteContentAutoTag).Exec(id); err != nil {
		return err
	}

	if err := d.txCreateLog(tx, fmt.Sprintf("Delete content auto-tag: id=%d", id), nil, nil, nil, nil); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) SetContentProcessed(id int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Stmt(d.stmtSetContentProcessedAt).Exec(id); err != nil {
		return err
	}

	if err := d.txCreateLog(tx, fmt.Sprintf("Mark content as processed: content_id=%d", id), nil, nil, &id, nil); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) AddContentTag(ctx context.Context, contentID, tagID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.StmtContext(ctx, d.stmtInsertContentTag).ExecContext(ctx, contentID, tagID); err != nil {
		return err
	}

	if err := d.txCreateLog(tx, fmt.Sprintf("Add tag to content: content_id=%d tag_id=%d", contentID, tagID), nil, nil, &contentID, &tagID); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) DeleteContentTag(ctx context.Context, contentID, tagID int64) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.StmtContext(ctx, d.stmtDeleteContentTag).ExecContext(ctx, contentID, tagID); err != nil {
		return err
	}

	if err := d.txCreateLog(tx, fmt.Sprintf("Remove tag from content: content_id=%d tag_id=%d", contentID, tagID), nil, nil, &contentID, &tagID); err != nil {
		return err
	}

	return tx.Commit()
}
