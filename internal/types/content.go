package types

import (
	"time"

	"github.com/dustin/go-humanize"
)

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
	Depth       int

	LinkwardenLinkID *string

	Tags []string
}

func (c *Content) PrettyBytes() string {
	return humanize.Bytes(uint64(c.SizeBytes))
}

func (c *Content) PrettyCreatedAt() string {
	return humanize.Time(c.CreatedAt)
}

func (c *Content) PrettyProcessedAt() string {
	if c.ProcessedAt == nil {
		return ""
	}
	return humanize.Time(*c.ProcessedAt)
}

func (c *Content) PrettyArchivedAt() string {
	if c.ArchivedAt == nil {
		return ""
	}
	return humanize.Time(*c.ArchivedAt)
}
