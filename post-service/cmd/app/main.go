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

	repository "post-service/internal/repository/kafka"
	"post-service/internal/repository/postgres"
	"post-service/internal/repository/redis"
	route "post-service/internal/router"

	"post-service/internal/config"
	"post-service/internal/lib/logger"
	"post-service/internal/usecase"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// Configurate system
	cfg := config.MustLoad()

	// Settings logger
	log := logger.SetupLogger(cfg.Env)
	log.Info("starting the project...", slog.String("env", cfg.Env))

	// Подключаемся к БД
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to connect to db:", slog.Any("err", err))
	}
	defer pool.Close() // закрываем при завершении приложения

	// Репозиторий для Postgres
	postRepo := postgres.NewPostgresPostRepository(pool) // Сущность для работы с posts

	// Подключаем Redis (cache)
	cache := redis.NewRedisCache(cfg.Redis.Addr, cfg.Redis.DB)

	var producer *repository.KafkaProducer
	for i := 0; i < 10; i++ {
		producer, err = repository.NewKafkaProducer(cfg.Brokers, cfg.Topic)
		if err == nil {
			break
		}
		log.Warn("Kafka not ready, retrying in 3s...", slog.Any("err", err))
		time.Sleep(3 * time.Second)
	}
	if producer == nil {
		log.Error("cannot connect to Kafka after retries", slog.Any("err", err))
	}

	postUC := usecase.NewPostUsecase(postRepo, producer, cache) // Бизнес-логика для posts

	// Передаем ctx в обработчики
	router := route.New(ctx, log, postUC)

	// Settings and started server + Grasful shortdown
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
		log.Info("server stopped listening") // Когда выйдет из ListenAndServe
	}()

	<-done
	log.Info("server stopping...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("server forced to shutdown", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("server stopped gracefully")
}
