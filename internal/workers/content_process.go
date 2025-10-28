package workers

import (
	"context"
	"time"

	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/fizzadar/cache.fyi/internal/workers/content_processors"
	"github.com/rs/zerolog"
)

const ContentProcessBatchSize = 10

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
			content_processors.NewLinkwardenContentProcessor(cfg, log, db),
			content_processors.NewNestedURLContentProcessor(cfg, log, db),
		},
	}
}

func (cpw *ContentProcessWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(cpw.config.ProcessInterval)
	continueCh := make(chan struct{}, 1)
	continueCh <- struct{}{}

	for {
		select {
		case <-ticker.C:
		case <-continueCh:
		}

		contents, err := cpw.db.ListContentToProcess(context.Background(), ContentProcessBatchSize)
		if err != nil {
			cpw.log.Err(err).Msg("Failed to fetch content to process")
			continue
		} else if len(contents) == 0 {
			cpw.log.Trace().Msg("No content to process")
			continue
		}

		cpw.log.Info().Int("contents", len(contents)).Msg("Processing contents")

		for _, content := range contents {
			cpw.log.Debug().Int64("content_id", content.ID).Msg("Processing content")
			failed := false

			for _, processor := range cpw.processors {
				if err = processor.ProcessContent(ctx, content); err != nil {
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

		if len(contents) == ContentProcessBatchSize {
			go func() {
				continueCh <- struct{}{}
			}()
		}
	}
}
