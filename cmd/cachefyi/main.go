package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/fizzadar/cache.fyi/internal"
	"github.com/fizzadar/cache.fyi/internal/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	configFilename := flag.String("config", "config.yaml", "Config filename")
	debug := flag.Bool("debug", false, "Enable debug logging")
	trace := flag.Bool("trace", false, "Enable trace logging")
	prettyLogs := flag.Bool("prettyLogs", false, "Enable pretty console logging")
	flag.Parse()

	if *prettyLogs {
		log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	}

	zerolog.DefaultContextLogger = &log.Logger

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if *trace {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
		log.Debug().Msg("Debug logging enabled")
		log.Trace().Msg("Trace logging enabled")
	} else if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug logging enabled")
	}

	log.Info().Msg("Create cache.fyi config")
	cfg := config.NewCachefyiConfig(*configFilename)

	if cfg.AuthHeader == "" {
		panic("config authHeader must be set")
	}

	log.Info().Msg("Create cache.fyi server")
	server := internal.NewCachefyiServer(cfg)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info().Msg("Start cache.fyi server")
	server.Start()

	<-done
	server.Stop()

	log.Info().Msg("Stopped cache.fyi")

}
