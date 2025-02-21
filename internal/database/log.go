package database

import "database/sql"

const (
	queryInsertLog = `
		INSERT INTO log (message, page_id, page_content, content_id, tag_id)
		VALUES (?, ?, ?, ?, ?)
	`
)

func (d *Database) initLogStatements() error {
	var err error

	if d.stmtInsertLog, err = d.db.Prepare(queryInsertLog); err != nil {
		return err
	}

	return nil
}

func (d *Database) txCreateLog(
	tx *sql.Tx,
	message string,
	pageID *int64,
	page_content *string,
	contentID,
	tagID *int64,
) error {
	_, err := tx.Stmt(d.stmtInsertLog).Exec(message, pageID, page_content, contentID, tagID)
	return err
}
