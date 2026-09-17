package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/a1234/notify-service/internal/config"
	"github.com/a1234/notify-service/internal/db"
	"github.com/a1234/notify-service/internal/queue"
	"github.com/a1234/notify-service/internal/repository"
	"github.com/a1234/notify-service/internal/service"
)

func main() {
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

	q, err := queue.NewRedisQueue(cfg.RedisAddr, cfg.RedisQueue)
	if err != nil {
		slog.Error("connect to redis", "error", err)
		os.Exit(1)
	}

	notifRepo := repository.NewNotificationRepository(pool)
	vendorRepo := repository.NewVendorConfigRepository(pool)
	deliverySvc := service.NewDeliveryService(notifRepo, vendorRepo, q)

	var wg sync.WaitGroup

	for i := 0; i < cfg.WorkerCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			slog.Info("delivery worker started", "worker_id", id)
			deliverySvc.RunDeliveryLoop(ctx)
			slog.Info("delivery worker stopped", "worker_id", id)
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("retry sweep started", "interval", cfg.RetryInterval)
		deliverySvc.RunRetrySweep(ctx, cfg.RetryInterval)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		slog.Info("recovery sweep started", "interval", cfg.RecoveryInterval)
		deliverySvc.RunRecoverySweep(ctx, cfg.RecoveryInterval)
	}()

	slog.Info("worker started", "delivery_workers", cfg.WorkerCount)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	slog.Info("shutting down worker")
	cancel()
	wg.Wait()
	slog.Info("worker stopped")
}
