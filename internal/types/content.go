package types

import "time"

type ContentType string

const (
	ContentTypeURL  ContentType = "url"
	ContentTypeFile ContentType = "file"
)

type Content struct {
	ID int64

	Type ContentType
	Hash string

	URL string

	Data        []byte
	ContentType *string
	Filename    *string
	SizeBytes   int

	CreatedAt   time.Time
	ProcessedAt *time.Time
	ArchivedAt  *time.Time
	ParentID    *int64
	HasChildren bool

	LinkwardenLinkID *string

	Tags []string
}
