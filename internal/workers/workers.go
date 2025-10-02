package workers

import (
	"context"

	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/rs/zerolog"
)

type Workers struct {
	config config.CachefyiConfig
	log    zerolog.Logger
	db     *database.Database

	contentProcessWorker *ContentProcessWorker
	contentArchiveWorker *ContentArchiveWorker
}

func NewWorkers(cfg config.CachefyiConfig, log zerolog.Logger, db *database.Database) *Workers {
	log = log.With().Str("component", "wokrers").Logger()

	return &Workers{
		config: cfg,
		log:    log,
		db:     db,

		contentProcessWorker: NewContentProcessWorker(cfg, log, db),
		contentArchiveWorker: NewContentArchiveWorker(cfg, log, db),
	}
}

func (w *Workers) Start() {
	ctx := context.TODO()

	w.log.Info().Msg("Starting content process worker...")
	go w.contentProcessWorker.loop(ctx)
	w.log.Info().Msg("Starting content archive worker...")
	go w.contentArchiveWorker.loop(ctx)
}
