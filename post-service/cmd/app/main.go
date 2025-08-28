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
	route "post-service/internal/router"

	"post-service/internal/config"
	"post-service/internal/lib/logger"
	"post-service/internal/usecase"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
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

	// оборачиваем pool в репозиторий (если утка крякает как утка, то вероятно это и есть утка)
	postRepo := postgres.NewPostgresPostRepository(pool)

	producer, err := repository.NewKafkaProducer(cfg.Brokers, cfg.Topic)
	if err != nil {
		log.Error("failed to create kafka producer:", slog.Any("err", err))
	}

	postUC := usecase.NewPostUsecase(postRepo, producer)

	// Init router
	router := route.New(log, postUC)

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

	// http.HandleFunc("/create-post", handler.CreatePost)
}
