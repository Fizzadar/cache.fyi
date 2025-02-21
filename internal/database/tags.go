package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fizzadar/cache.fyi/internal/types"
)

var ErrTagNotFound = errors.New("no such tag")

const (
	queryListTags = `
		SELECT id, name
		FROM tags
		ORDER BY id DESC
	`
	queryCreateTag = `
		INSERT INTO tags (name)
		VALUES (?)
	`
	queryDeleteTag = `
		DELETE FROM tags
		WHERE id = ?
	`
	queryGetTagID = `
		SELECT id
		FROM tags
		WHERE name = ?
	`
)

func (d *Database) initTagStatements() error {
	var err error

	d.stmtListTags, err = d.db.Prepare(queryListTags)
	if err != nil {
		return err
	}

	d.stmtCreateTag, err = d.db.Prepare(queryCreateTag)
	if err != nil {
		return err
	}

	d.stmtDeleteTag, err = d.db.Prepare(queryDeleteTag)
	if err != nil {
		return err
	}

	d.stmtGetTagID, err = d.db.Prepare(queryGetTagID)
	if err != nil {
		return err
	}

	return nil
}

func (d *Database) CreateTag(name string) (int64, error) {
	tx, err := d.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	result, err := tx.Stmt(d.stmtCreateTag).Exec(name)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	if err := d.txCreateLog(tx, fmt.Sprintf("Create tag: name=%s id=%d", name, id), nil, nil, nil, &id); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

func (d *Database) DeleteTag(id int) error {
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	idInt64 := int64(id)
	if _, err := tx.Stmt(d.stmtDeleteTag).Exec(id); err != nil {
		return err
	}

	if err := d.txCreateLog(tx, fmt.Sprintf("Delete tag: id=%d", id), nil, nil, nil, &idInt64); err != nil {
		return err
	}

	return tx.Commit()
}

func (d *Database) ListTags(ctx context.Context) ([]types.Tag, error) {
	rows, err := d.stmtListTags.QueryContext(ctx)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := make([]types.Tag, 0, 1000)

	for rows.Next() {
		var tag types.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}

	return tags, rows.Err()
}

func (d *Database) TagIDForName(ctx context.Context, name string) (int64, error) {
	var id int64
	if err := d.stmtGetTagID.QueryRowContext(ctx, name).Scan(&id); errors.Is(err, sql.ErrNoRows) {
		return 0, ErrTagNotFound
	} else if err != nil {
		return id, err
	}
	return id, nil
}

// Get or create tag - NOT parallel safe because it doesn't use a transaction but doesn't matter for
// cache.fyi, I'm just lazy.
func (d *Database) EnsureTag(ctx context.Context, name string) (int64, error) {
	if id, err := d.TagIDForName(ctx, name); err == nil {
		return id, nil
	} else if errors.Is(err, ErrTagNotFound) {
		return d.CreateTag(name)
	} else {
		return 0, err
	}
}
