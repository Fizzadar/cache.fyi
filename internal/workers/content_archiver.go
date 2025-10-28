package workers

import (
	"context"
	"time"

	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/fizzadar/cache.fyi/internal/workers/content_archivers"
	"github.com/rs/zerolog"
)

const ContentArchiveBatchSize = 10

type ContentArchiveWorker struct {
	config   config.CachefyiConfig
	log      zerolog.Logger
	db       *database.Database
	archiver content_archivers.ContentArchiver
}

func NewContentArchiveWorker(cfg config.CachefyiConfig, log zerolog.Logger, db *database.Database) *ContentArchiveWorker {
	log = log.With().Str("worker", "content_archive").Logger()

	if cfg.ProcessInterval == 0 {
		panic("cannot have no process interval set")
	}

	archiver := content_archivers.NewLinkwardenArchiver(cfg, log, db)

	return &ContentArchiveWorker{
		config:   cfg,
		log:      log,
		db:       db,
		archiver: archiver,
	}
}

func (cpw *ContentArchiveWorker) loop(ctx context.Context) {
	ticker := time.NewTicker(cpw.config.ProcessInterval)
	continueCh := make(chan struct{}, 1)
	continueCh <- struct{}{}

	for {
		select {
		case <-ticker.C:
		case <-continueCh:
		}

		contents, err := cpw.db.ListContentToArchive(context.Background(), ContentArchiveBatchSize)
		if err != nil {
			cpw.log.Err(err).Msg("Failed to fetch content to archive")
			continue
		} else if len(contents) == 0 {
			cpw.log.Trace().Msg("No content to archive")
			continue
		}

		cpw.log.Info().Int("contents", len(contents)).Msg("Archiving contents")

		for _, content := range contents {
			cpw.log.Debug().Int64("content_id", content.ID).Msg("Archiving content")

			if err := cpw.archiver.ArchiveContent(ctx, content); err != nil {
				cpw.log.Err(err).Int64("content_id", content.ID).Msg("Failed to archive content")
			} else {
				if err := cpw.db.SetContentArchived(content.ID); err != nil {
					cpw.log.Err(err).Msg("Failed to set content archived!")
				} else {
					cpw.log.Debug().Int64("content_id", content.ID).Msg("Processed content")
				}
			}
		}

		if len(contents) == ContentArchiveBatchSize {
			go func() {
				continueCh <- struct{}{}
			}()
		}
	}
}
