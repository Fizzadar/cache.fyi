package content_processors

import (
	"github.com/fizzadar/cache.fyi/internal/types"
)

type ContentProcessor interface {
	ProcessContent(*types.Content) error
}
