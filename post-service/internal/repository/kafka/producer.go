package repository

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"post-service/internal/domain"

	"github.com/IBM/sarama"
)

type KafkaProducer struct {
	Producer sarama.SyncProducer
	Topic    string
	log      *slog.Logger
}

func NewKafkaProducer(brokers []string, topic string, log *slog.Logger) (*KafkaProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	start := time.Now()
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		log.Error("Failed to create Kafka producer",
			slog.String("error", err.Error()),
			slog.Any("brokers", brokers),
			slog.String("topic", topic))
		return nil, err
	}

	log.Info("Kafka producer initialized",
		slog.Any("brokers", brokers),
		slog.String("topic", topic),
		slog.Duration("duration", time.Since(start)))

	return &KafkaProducer{
		Producer: producer,
		Topic:    topic,
		log:      log,
	}, nil
}

func (k *KafkaProducer) Publish(ctx context.Context, post *domain.Post) error {
	if k == nil || k.Producer == nil {
		err := errors.New("kafka producer not initialized")
		k.log.Error("Kafka producer not initialized")
		return err
	}

	start := time.Now()

	msgBytes, err := json.Marshal(post)
	if err != nil {
		k.log.Error("Failed to marshal post for Kafka",
			slog.String("post_id", post.ID),
			slog.String("error", err.Error()))
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: k.Topic,
		Value: sarama.ByteEncoder(msgBytes),
		Key:   sarama.StringEncoder(post.ID),
	}

	partition, offset, err := k.Producer.SendMessage(msg)
	if err != nil {
		k.log.Error("Failed to publish message to Kafka",
			slog.String("post_id", post.ID),
			slog.String("topic", k.Topic),
			slog.String("error", err.Error()),
			slog.Duration("duration", time.Since(start)))
		return err
	}

	k.log.Info("Message published to Kafka",
		slog.String("post_id", post.ID),
		slog.String("topic", k.Topic),
		slog.Int("partition", int(partition)),
		slog.Int64("offset", offset),
		slog.Duration("duration", time.Since(start)),
		slog.Int("message_size", len(msgBytes)))

	return nil
}

func (k *KafkaProducer) Close() error {
	if k.Producer != nil {
		k.log.Info("Closing Kafka producer")
		return k.Producer.Close()
	}
	return nil
}
