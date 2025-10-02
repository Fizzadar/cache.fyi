package content_archivers

import (
	"context"

	"github.com/fizzadar/cache.fyi/internal/types"
)

type ContentArchiver interface {
	ArchiveContent(context.Context, *types.Content) error
}
