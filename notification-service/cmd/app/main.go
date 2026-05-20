package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	kafkaadapter "notification-service/internal/adapter/kafka"
	redisadapter "notification-service/internal/adapter/redis"
	"notification-service/internal/config"
	"notification-service/internal/lib/logger"
	"notification-service/internal/usecase"
)

func main() {
	cfg := config.MustLoad()

	log := logger.SetupLogger(cfg.Env)
	log.Info("starting the project...", slog.String("env", cfg.Env))

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	store, err := redisadapter.NewStore(
		cfg.Redis.Addr,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Redis.ProcessedTTL,
		log.With(slog.String("component", "redis")),
	)
	if err != nil {
		log.Error("failed to connect to redis", slog.Any("error", err))
		os.Exit(1)
	}

	uc := usecase.NewNotificationUsecase(store, log.With(slog.String("component", "usecase")))

	var dlq *kafkaadapter.DLQProducer
	for i := 0; i < 10; i++ {
		dlq, err = kafkaadapter.NewDLQProducer(cfg.Kafka.Brokers, cfg.Kafka.DLQTopic, log.With(slog.String("component", "dlq")))
		if err == nil {
			break
		}
		log.Warn("Kafka not ready, retrying in 3s...", slog.Any("err", err))
		time.Sleep(3 * time.Second)
	}
	if dlq == nil {
		log.Error("cannot connect to Kafka DLQ producer after retries", slog.Any("err", err))
		os.Exit(1)
	}

	consumer := kafkaadapter.NewConsumer(
		log.With(slog.String("component", "kafka-consumer")),
		uc,
		dlq,
		cfg.Retry.MaxAttempts,
		cfg.Retry.Backoff,
	)

	go func() {
		if err := consumer.Start(appCtx, cfg.Kafka.Brokers, cfg.Kafka.Topic, cfg.Kafka.GroupID); err != nil {
			log.Error("consumer stopped with error", slog.Any("error", err))
			cancel()
		}
	}()

	log.Info("notification consumer started",
		slog.String("topic", cfg.Kafka.Topic),
		slog.String("group", cfg.Kafka.GroupID))

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Info("shutting down service...")
	cancel()

	time.Sleep(500 * time.Millisecond)

	if err := dlq.Close(); err != nil {
		log.Error("failed to close DLQ producer", slog.Any("error", err))
	}

	if err := store.Close(); err != nil {
		log.Error("failed to close redis store", slog.Any("error", err))
	}

	log.Info("graceful shutdown completed")
}
