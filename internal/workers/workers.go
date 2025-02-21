package workers

import (
	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/rs/zerolog"
)

type Workers struct {
	config config.CachefyiConfig
	log    zerolog.Logger
	db     *database.Database

	contentProcessWorker *ContentProcessWorker
}

func NewWorkers(cfg config.CachefyiConfig, log zerolog.Logger, db *database.Database) *Workers {
	log = log.With().Str("component", "wokrers").Logger()

	return &Workers{
		config: cfg,
		log:    log,
		db:     db,

		contentProcessWorker: NewContentProcessWorker(cfg, log, db),
	}
}

func (w *Workers) Start() {
	w.log.Info().Msg("Starting content process worker...")
	go w.contentProcessWorker.loop()
}
