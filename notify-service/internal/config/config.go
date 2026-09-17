package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ServerPort       string
	DatabaseURL      string
	RedisAddr        string
	RedisQueue       string
	WorkerCount      int
	RetryInterval    time.Duration
	RecoveryInterval time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:       getEnv("SERVER_PORT", ":8080"),
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://notify:notify@localhost:5432/notify?sslmode=disable"),
		RedisAddr:        getEnv("REDIS_ADDR", "localhost:6379"),
		RedisQueue:       getEnv("REDIS_QUEUE", "notifications"),
		WorkerCount:      getEnvInt("WORKER_COUNT", 5),
		RetryInterval:    getEnvDuration("RETRY_INTERVAL", 10*time.Second),
		RecoveryInterval: getEnvDuration("RECOVERY_INTERVAL", 30*time.Second),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
