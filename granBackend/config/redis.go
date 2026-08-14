package config

import (
	"context"
	"log/slog"
	"os"

	"github.com/redis/go-redis/v9"
)

func InitRedis() *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	password := os.Getenv("REDIS_PASSWORD")

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		slog.Error("Error fatal conectando a Redis", "error", err)
		os.Exit(1)
	}

	slog.Info("Redis conectado: Rate Limiting distribuido listo, alegria todos ")
	return rdb
}
