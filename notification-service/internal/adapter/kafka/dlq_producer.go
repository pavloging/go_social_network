package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"notification-service/internal/domain"

	"github.com/IBM/sarama"
)

type DLQProducer struct {
	producer sarama.SyncProducer
	topic    string
	log      *slog.Logger
}

func NewDLQProducer(brokers []string, topic string, log *slog.Logger) (*DLQProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5
	config.Producer.Retry.Backoff = time.Second

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	log.Info("DLQ producer initialized",
		slog.Any("brokers", brokers),
		slog.String("topic", topic))

	return &DLQProducer{
		producer: producer,
		topic:    topic,
		log:      log,
	}, nil
}

func (p *DLQProducer) Publish(ctx context.Context, event domain.DeadLetterEvent) error {
	_ = ctx

	if p == nil || p.producer == nil {
		return errors.New("dlq producer not initialized")
	}

	msgBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	key := event.OriginalKey
	if key == "" {
		key = "unknown"
	}

	msg := &sarama.ProducerMessage{
		Topic: p.topic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(msgBytes),
	}

	partition, offset, err := p.producer.SendMessage(msg)
	if err != nil {
		p.log.Error("failed to publish DLQ message",
			slog.String("topic", p.topic),
			slog.String("error", err.Error()))
		return err
	}

	p.log.Info("DLQ message published",
		slog.String("topic", p.topic),
		slog.Int("partition", int(partition)),
		slog.Int64("offset", offset))

	return nil
}

func (p *DLQProducer) Close() error {
	if p.producer != nil {
		return p.producer.Close()
	}
	return nil
}
