package main

import (
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/mr1hm/go-disaster-alerts/internal/config"
	"github.com/mr1hm/go-disaster-alerts/internal/logging"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		logging.Fatalf("Fatal while loading config: %v", err)
	}
	logging.Setup(cfg.Logging.Level)

	slog.Info("Server starting", "host", cfg.Server.Host, "port", cfg.Server.Port)
}
