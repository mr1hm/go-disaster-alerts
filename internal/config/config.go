package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server  ServerConfig
	GRPC    GRPCConfig
	Worker  WorkerConfig
	Sources SourcesConfig
	DB      DatabaseConfig
	Logging LoggingConfig
}

type GRPCConfig struct {
	Port int
}

type ServerConfig struct {
	Host string
	Port int
}

type WorkerConfig struct {
	Count      int
	BufferSize int
}

type SourcesConfig struct {
	USGSEnabled       bool
	USGSURL           string
	USGSPollInterval  time.Duration
	GDACSEnabled      bool
	GDACSURL          string
	GDACSPollInterval time.Duration
}

type DatabaseConfig struct {
	Path string
}

type LoggingConfig struct {
	Level string
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		GRPC: GRPCConfig{
			Port: getEnvInt("GRPC_PORT", 50051),
		},
		Worker: WorkerConfig{
			Count:      getEnvInt("WORKER_COUNT", 2),
			BufferSize: getEnvInt("WORKER_BUFFER_SIZE", 20),
		},
		Sources: SourcesConfig{
			USGSEnabled:       getEnvBool("USGS_ENABLED", true),
			USGSURL:           getEnv("USGS_URL", "https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_hour.geojson"),
			USGSPollInterval:  getEnvDuration("USGS_POLL_INTERVAL", 5*time.Minute),
			GDACSEnabled:      getEnvBool("GDACS_ENABLED", true),
			GDACSURL:          getEnv("GDACS_URL", "https://www.gdacs.org/xml/rss.xml"),
			GDACSPollInterval: getEnvDuration("GDACS_POLL_INTERVAL", 10*time.Minute),
		},
		DB: DatabaseConfig{
			Path: getEnv("DB_PATH", "./data/disaster-alerts.db"),
		},
		Logging: LoggingConfig{
			Level: getEnv("LOG_LEVEL", "info"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	validLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLevels[c.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", c.Logging.Level)
	}

	if c.Sources.USGSPollInterval < time.Minute {
		return fmt.Errorf("USGS poll interval must be at least 1 minute")
	}
	if c.Sources.GDACSPollInterval < time.Minute {
		return fmt.Errorf("GDACS poll interval must be at least 1 minute")
	}

	return nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val := os.Getenv(key); val != "" {
		if i, err := strconv.Atoi(val); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val := os.Getenv(key); val != "" {
		if b, err := strconv.ParseBool(val); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil {
			return d
		}
	}
	return fallback
}
