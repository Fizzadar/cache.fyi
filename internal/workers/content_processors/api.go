package content_processors

import (
	"context"

	"github.com/fizzadar/cache.fyi/internal/types"
)

type ContentProcessor interface {
	ProcessContent(context.Context, *types.Content) error
}
