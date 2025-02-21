package workers

import (
	"context"
	"time"

	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/fizzadar/cache.fyi/internal/workers/content_processors"
	"github.com/rs/zerolog"
)

type ContentProcessWorker struct {
	config config.CachefyiConfig
	log    zerolog.Logger
	db     *database.Database

	processors []content_processors.ContentProcessor
}

func NewContentProcessWorker(cfg config.CachefyiConfig, log zerolog.Logger, db *database.Database) *ContentProcessWorker {
	log = log.With().Str("worker", "content_process").Logger()

	if cfg.ProcessInterval == 0 {
		panic("cannot have no process interval set")
	}

	return &ContentProcessWorker{
		config: cfg,
		log:    log,
		db:     db,
		processors: []content_processors.ContentProcessor{
			content_processors.NewArchiveboxContentProcessor(cfg, log, db),
			content_processors.NewNestedURLContentProcessor(cfg, log, db),
		},
	}
}

func (cpw *ContentProcessWorker) loop() {
	ticker := time.NewTicker(cpw.config.ProcessInterval)

	for range ticker.C {
		contents, err := cpw.db.ListContentToProcess(context.Background(), 10)
		if err != nil {
			cpw.log.Err(err).Msg("Failed to fetch content to process")
			continue
		}

		cpw.log.Info().Int("contents", len(contents)).Msg("Processing contents")

		for _, content := range contents {
			cpw.log.Debug().Int64("content_id", content.ID).Msg("Processing content")
			failed := false

			for _, processor := range cpw.processors {
				if err = processor.ProcessContent(content); err != nil {
					failed = true
					break
				}
			}

			if failed {
				cpw.log.Err(err).Int64("content_id", content.ID).Msg("Failed to process content")
			} else {
				if err := cpw.db.SetContentProcessed(content.ID); err != nil {
					cpw.log.Err(err).Msg("Failed to set content processed!")
				} else {
					cpw.log.Debug().Int64("content_id", content.ID).Msg("Processed content")
				}
			}
		}
	}
}
