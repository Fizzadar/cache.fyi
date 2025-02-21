package content_processors

import (
	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/fizzadar/cache.fyi/internal/types"
	"github.com/rs/zerolog"
)

var _ ContentProcessor = (*ArchiveboxContentProcessor)(nil)

type ArchiveboxContentProcessor struct {
	config config.CachefyiConfig
	db     *database.Database
}

func NewArchiveboxContentProcessor(
	cfg config.CachefyiConfig,
	log zerolog.Logger,
	db *database.Database,
) *ArchiveboxContentProcessor {
	return &ArchiveboxContentProcessor{
		config: cfg,
		db:     db,
	}
}

func (acp *ArchiveboxContentProcessor) ProcessContent(content *types.Content) error {
	if content.URL == "" {
		return nil
	}

	return nil
}
