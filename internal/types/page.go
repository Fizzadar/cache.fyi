package types

import (
	"time"
)

type Page struct {
	ID string

	Path string

	Title   string
	Content string

	Pinned bool

	CreatedAt time.Time
	UpdatedAt *time.Time

	Tags []string
}
