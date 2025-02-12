package main

import (
	"github.com/linemk/avito-shop/internal/config"
	"github.com/linemk/avito-shop/internal/lib/logger"
	"log/slog"
)

func main() {
	// loading configurations
	cfg := config.MustLoad()

	// initialize logger - depends on env
	log := logger.SetupLogger(cfg.Env)

	log.Info("Starting server", slog.String("env", cfg.Env))
	log.Debug("Debug messages are enabled")
}
