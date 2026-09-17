package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/a1234/notify-service/internal/api"
	"github.com/a1234/notify-service/internal/config"
	"github.com/a1234/notify-service/internal/db"
	"github.com/a1234/notify-service/internal/queue"
	"github.com/a1234/notify-service/internal/repository"
	"github.com/a1234/notify-service/internal/service"
)

func main() {
	migrateOnly := flag.Bool("migrate", false, "run migrations and exit")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool); err != nil {
		slog.Error("run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations completed")

	if *migrateOnly {
		return
	}

	q, err := queue.NewRedisQueue(cfg.RedisAddr, cfg.RedisQueue)
	if err != nil {
		slog.Error("connect to redis", "error", err)
		os.Exit(1)
	}

	notifRepo := repository.NewNotificationRepository(pool)
	vendorRepo := repository.NewVendorConfigRepository(pool)
	notifService := service.NewNotificationService(notifRepo, q)
	handler := api.NewHandler(notifService, vendorRepo, pool)
	router := api.NewRouter(handler)

	srv := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("server starting", "addr", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}
	fmt.Println("server stopped")
}
