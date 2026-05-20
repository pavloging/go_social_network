// cmd/main.go
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	kafkarepo "post-service/internal/repository/kafka"
	"post-service/internal/repository/postgres"
	redisrepo "post-service/internal/repository/redis"
	route "post-service/internal/router"

	"post-service/internal/config"
	"post-service/internal/lib/logger"
	"post-service/internal/outbox"
	"post-service/internal/usecase"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	log.Info("starting the project...", slog.String("env", cfg.Env))

	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to db:", slog.Any("err", err))
		os.Exit(1)
	}

	postRepo := postgres.NewPostgresPostRepository(pool)
	outboxRepo := postgres.NewPostgresOutboxRepository(pool)

	cache := redisrepo.NewRedisCache(cfg.Redis.Addr, cfg.Redis.DB, log.With(slog.String("component", "redis")))

	var producer *kafkarepo.KafkaProducer
	for i := 0; i < 10; i++ {
		producer, err = kafkarepo.NewKafkaProducer(cfg.Kafka.Brokers, cfg.Kafka.Topic, log.With(slog.String("component", "kafka")))
		if err == nil {
			break
		}
		log.Warn("Kafka not ready, retrying in 3s...", slog.Any("err", err))
		time.Sleep(3 * time.Second)
	}
	if producer == nil {
		log.Error("cannot connect to Kafka after retries", slog.Any("err", err))
		os.Exit(1)
	}

	postUC := usecase.NewPostUsecase(postRepo, cache, log.With(slog.String("component", "usecase")))

	workerID := uuid.NewString()
	outboxWorker := outbox.NewWorker(
		outboxRepo,
		producer,
		log.With(slog.String("component", "outbox-worker")),
		cfg.Outbox.PollInterval,
		cfg.Outbox.RetryDelay,
		cfg.Outbox.MaxAttempts,
		cfg.Outbox.BatchSize,
		workerID,
	)

	go outboxWorker.Run(appCtx)

	router := route.New(appCtx, log.With(slog.String("component", "http")), postUC)

	srv := &http.Server{
		Addr:         cfg.Address,
		ReadTimeout:  cfg.Timeout,
		WriteTimeout: cfg.Timeout,
		IdleTimeout:  cfg.IdleTimeout,
		Handler:      router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Info("server starting", slog.String("address", cfg.Address))

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed to start", slog.Any("error", err))
			os.Exit(1)
		}
		log.Info("server stopped listening")
	}()

	<-done
	log.Info("server stopping...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", slog.Any("error", err))
	}

	if err := producer.Close(); err != nil {
		log.Error("failed to close kafka producer", slog.Any("error", err))
	}

	if err := cache.Close(); err != nil {
		log.Error("failed to close redis cache", slog.Any("error", err))
	}

	pool.Close()

	log.Info("server stopped gracefully")
}
