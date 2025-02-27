package types

import (
	"time"
)

type Page struct {
	ID string

	Path string

	Title   string
	Content string

	CreatedAt time.Time
	UpdatedAt *time.Time

	Tags []string
}
