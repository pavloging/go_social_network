package main

import (
	"context"
	"log/slog"
	"notification-service/internal/adapter/kafka"
	"notification-service/internal/config"
	"notification-service/internal/lib/logger"
	"notification-service/internal/usecase"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Configurate system
	cfg := config.MustLoad()

	// Settings logger
	log := logger.SetupLogger(cfg.Env)
	log.Info("starting the project...", slog.String("env", cfg.Env))

	uc := usecase.NewNotificationUsecase()
	c := kafka.NewConsumer(log, uc)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := c.Start(ctx, cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID); err != nil {
			log.Error("consumer error:", slog.Any("err", err))
			cancel()
		}
	}()

	log.Info("server starting", slog.String("address", cfg.Address))

	// Ждем сигнал завершения
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	log.Info("shutting down service...")
}
