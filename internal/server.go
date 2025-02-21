package internal

import (
	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/fizzadar/cache.fyi/internal/database"
	"github.com/fizzadar/cache.fyi/internal/routes"
	"github.com/fizzadar/cache.fyi/internal/workers"
	"github.com/rs/zerolog/log"
)

type cachefyi struct {
	routes  *routes.Routes
	workers *workers.Workers
}

func NewCachefyiServer(cfg config.CachefyiConfig) *cachefyi {
	log := log.With().Logger()
	db := database.NewDatabase(cfg, log)
	return &cachefyi{
		routes:  routes.NewRoutes(cfg, log, db),
		workers: workers.NewWorkers(cfg, log, db),
	}
}

func (s *cachefyi) Start() {
	s.routes.Start()
	s.workers.Start()
}

func (s *cachefyi) Stop() {

}
