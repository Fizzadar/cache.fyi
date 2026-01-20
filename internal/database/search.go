package database

import (
	"context"

	"github.com/fizzadar/cache.fyi/internal/types"
)

type SearchResults struct {
	Pages   []*types.Page
	Content []*types.Content
	Tags    []types.Tag
}

const (
	querySearchPages = `
		SELECT
			p.id, p.path, p.title, p.content, p.pinned, p.created_at, p.updated_at,
			group_concat(t.name) AS tag_names
		FROM pages AS p
		LEFT JOIN page__tag AS pt ON p.id = pt.page_id
		LEFT JOIN tags AS t ON t.id = pt.tag_id
		WHERE p.path LIKE ? OR p.title LIKE ? OR p.content LIKE ?
		GROUP BY p.id
		ORDER BY p.updated_at DESC, p.created_at DESC
		LIMIT ?
	`
	querySearchContent = `
		SELECT
			c.id, c.type, c.hash, c.url, c.content_type, c.filename, c.size_bytes,
			c.created_at, c.processed_at, c.archived_at, c.parent_id, c.linkwarden_link_id,
			group_concat(t.name) AS tag_names
		FROM content AS c
		LEFT JOIN content__tag AS ct ON c.id = ct.content_id
		LEFT JOIN tags AS t ON t.id = ct.tag_id
		WHERE c.url LIKE ?
		AND c.type = 'url'
		GROUP BY c.id
		ORDER BY c.created_at DESC
		LIMIT ?
	`
	querySearchTags = `
		SELECT id, name
		FROM tags
		WHERE name LIKE ?
		ORDER BY name
		LIMIT ?
	`
)

func (d *Database) initSearchStatements() error {
	var err error

	if d.stmtSearchPages, err = d.db.Prepare(querySearchPages); err != nil {
		return err
	} else if d.stmtSearchContent, err = d.db.Prepare(querySearchContent); err != nil {
		return err
	} else if d.stmtSearchTags, err = d.db.Prepare(querySearchTags); err != nil {
		return err
	}

	return nil
}

func (d *Database) Search(ctx context.Context, query string, limit int) (*SearchResults, error) {
	results := &SearchResults{
		Pages:   make([]*types.Page, 0),
		Content: make([]*types.Content, 0),
		Tags:    make([]types.Tag, 0),
	}

	// Search pattern for LIKE queries
	searchPattern := "%" + query + "%"

	// Search pages
	rows, err := d.stmtSearchPages.QueryContext(ctx, searchPattern, searchPattern, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if page, err := scanPageRow(rows); err != nil {
			return nil, err
		} else {
			results.Pages = append(results.Pages, page)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Search content
	rows, err = d.stmtSearchContent.QueryContext(ctx, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if content, err := scanContentRow(rows); err != nil {
			return nil, err
		} else {
			results.Content = append(results.Content, content)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Search tags
	rows, err = d.stmtSearchTags.QueryContext(ctx, searchPattern, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tag types.Tag
		if err := rows.Scan(&tag.ID, &tag.Name); err != nil {
			return nil, err
		}
		results.Tags = append(results.Tags, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
