package database

import (
	"context"
	"database/sql"
	"embed"
	"path"
	"strings"

	"github.com/fizzadar/cache.fyi/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
)

//go:embed migrations
var MigrationsFS embed.FS

type Database struct {
	log zerolog.Logger
	db  *sql.DB

	stmtGetPage,
	stmtUpsertPage,
	stmtInsertPage,
	stmtListPages,
	stmtListPagesForTag,
	stmtGetPageID,
	stmtGetPagePath,
	stmtInsertPageTag,
	stmtDeletePageTag,
	stmtCreatePageAutoTag,
	stmtListPageAutoTags,
	stmtDeletePageAutoTag,

	stmtGetContent,
	stmtGetContentData,
	stmtListContent,
	stmtListContentForTag,
	stmtInsertContent,
	stmtCreateContentAutoTag,
	stmtListContentAutoTags,
	stmtDeleteContentAutoTag,
	stmtInsertContentTag,
	stmtDeleteContentTag,
	stmtSetContentProcessedAt,
	stmtListContentToProcess,

	stmtListTags,
	stmtCreateTag,
	stmtDeleteTag,
	stmtGetTagID,

	stmtInsertLog *sql.Stmt
}

func NewDatabase(cfg config.CachefyiConfig, log zerolog.Logger) *Database {
	log = log.With().Str("component", "database").Logger()

	db, err := sql.Open("sqlite3", cfg.Database)
	if err != nil {
		return nil
	}

	d := &Database{
		log: log,
		db:  db,
	}

	if err = d.runMigrations(); err != nil {
		panic(err)
	} else if err := d.initPageStatements(); err != nil {
		panic(err)
	} else if err = d.initContentStatements(); err != nil {
		panic(err)
	} else if err = d.initTagStatements(); err != nil {
		panic(err)
	} else if err = d.initLogStatements(); err != nil {
		panic(err)
	}

	return d
}

func (d *Database) runMigrations() error {
	migrations, err := MigrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}

	ctx := context.Background()

	sql := `
		CREATE TABLE IF NOT EXISTS migrations (
		    filename TEXT NOT NULL
		);
	`
	if _, err := d.db.ExecContext(ctx, sql); err != nil {
		return err
	}

	sql = "SELECT filename FROM migrations"
	rows, err := d.db.QueryContext(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()

	runMigrationsSlice := make([]string, 0)

	for rows.Next() {
		var filename string
		if err := rows.Scan(&filename); err != nil {
			return err
		}
		runMigrationsSlice = append(runMigrationsSlice, filename)
	}

	runMigrations := make(map[string]struct{})
	for _, file := range runMigrationsSlice {
		runMigrations[file] = struct{}{}
	}

	for _, file := range migrations {
		if _, found := runMigrations[file.Name()]; found {
			d.log.Debug().Msgf("Migration already run: %s", file.Name())
		} else {
			data, err := MigrationsFS.ReadFile(path.Join("migrations", file.Name()))
			if err != nil {
				return err
			}

			tx, err := d.db.Begin()
			if err != nil {
				return err
			}
			defer tx.Rollback()

			d.log.Debug().Msgf("Running migration: %s", file.Name())

			statements := strings.Split(string(data), ";")
			for _, statement := range statements {
				if _, err := tx.ExecContext(ctx, statement); err != nil {
					return err
				}
			}

			sql = "INSERT INTO migrations (filename) VALUES ($1)"

			if _, err := tx.ExecContext(ctx, sql, file.Name()); err != nil {
				return err
			}

			if err := tx.Commit(); err != nil {
				return err
			}

			d.log.Info().Msgf("Migration complete: %s", file.Name())
		}
	}

	return nil
}
