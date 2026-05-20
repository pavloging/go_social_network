package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"notification-service/internal/domain"
	"notification-service/internal/usecase"

	"github.com/IBM/sarama"
)

type Consumer struct {
	log          *slog.Logger
	uc           *usecase.NotificationUsecase
	dlq          *DLQProducer
	retryMax     int
	retryBackoff time.Duration
}

func NewConsumer(
	log *slog.Logger,
	uc *usecase.NotificationUsecase,
	dlq *DLQProducer,
	retryMax int,
	retryBackoff time.Duration,
) *Consumer {
	return &Consumer{
		log:          log,
		uc:           uc,
		dlq:          dlq,
		retryMax:     retryMax,
		retryBackoff: retryBackoff,
	}
}

func (c *Consumer) Start(ctx context.Context, brokers []string, topic string, group string) error {
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_0_0
	config.Consumer.Offsets.Initial = sarama.OffsetOldest
	config.Consumer.Offsets.AutoCommit.Enable = false

	client, err := sarama.NewConsumerGroup(brokers, group, config)
	if err != nil {
		return err
	}
	defer client.Close()

	for {
		if err := client.Consume(ctx, []string{topic}, c); err != nil {
			c.log.Error("consumer error", slog.Any("error", err))
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}
	}
}

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		c.handleMessage(sess, msg)
	}
	return nil
}

func (c *Consumer) handleMessage(sess sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	raw := msg.Value
	key := string(msg.Key)

	var event domain.EventEnvelope
	if err := json.Unmarshal(raw, &event); err != nil {
		c.log.Error("failed to unmarshal message", slog.Any("error", err))
		dlqEvent := domain.DeadLetterEvent{
			OriginalTopic: msg.Topic,
			OriginalKey:   key,
			OriginalValue: domain.RawMessageForDLQ(raw),
			Error:         "invalid json: " + err.Error(),
			Attempts:      1,
			FailedAt:      time.Now().UTC(),
		}
		if c.publishDLQAndCommit(sess, msg, dlqEvent) {
			return
		}
		return
	}

	if err := validateEvent(event); err != nil {
		c.log.Error("invalid event envelope", slog.Any("error", err))
		dlqEvent := domain.DeadLetterEvent{
			OriginalTopic: msg.Topic,
			OriginalKey:   key,
			OriginalValue: domain.RawMessageForDLQ(raw),
			Error:         err.Error(),
			Attempts:      1,
			FailedAt:      time.Now().UTC(),
		}
		if c.publishDLQAndCommit(sess, msg, dlqEvent) {
			return
		}
		return
	}

	if err := c.processWithRetry(sess.Context(), event); err != nil {
		c.log.Error("processing failed after retries",
			slog.String("event_id", event.EventID),
			slog.Any("error", err))

		dlqEvent := domain.DeadLetterEvent{
			OriginalTopic: msg.Topic,
			OriginalKey:   key,
			OriginalValue: domain.RawMessageForDLQ(raw),
			Error:         err.Error(),
			Attempts:      c.retryMax,
			FailedAt:      time.Now().UTC(),
		}
		c.publishDLQAndCommit(sess, msg, dlqEvent)
		return
	}

	commitMessage(sess, msg)
}

func (c *Consumer) publishDLQAndCommit(sess sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage, dlqEvent domain.DeadLetterEvent) bool {
	if err := c.dlq.Publish(sess.Context(), dlqEvent); err != nil {
		c.log.Error("failed to publish DLQ message", slog.Any("error", err))
		return false
	}

	commitMessage(sess, msg)
	return true
}

func (c *Consumer) processWithRetry(ctx context.Context, event domain.EventEnvelope) error {
	var lastErr error

	for attempt := 1; attempt <= c.retryMax; attempt++ {
		created, err := c.uc.Process(ctx, event)
		if err == nil {
			if created {
				c.log.Info("notification processed",
					slog.String("event_id", event.EventID),
					slog.String("post_id", event.Payload.ID))
			} else {
				c.log.Info("duplicate event skipped", slog.String("event_id", event.EventID))
			}
			return nil
		}

		lastErr = err
		c.log.Warn("processing failed",
			slog.String("event_id", event.EventID),
			slog.Int("attempt", attempt),
			slog.Any("error", err))

		if attempt < c.retryMax {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryBackoff):
			}
		}
	}

	return lastErr
}

func validateEvent(event domain.EventEnvelope) error {
	if event.EventID == "" {
		return errors.New("event_id is required")
	}
	if event.EventType != domain.EventTypePostCreated {
		return errors.New("unsupported event_type")
	}
	if event.Version != domain.EventVersion {
		return errors.New("unsupported event version")
	}
	if event.Payload.ID == "" {
		return errors.New("payload.id is required")
	}
	if event.Payload.Author == "" {
		return errors.New("payload.author is required")
	}
	if event.Payload.CreatedAt.IsZero() {
		return errors.New("payload.created_at is required")
	}
	return nil
}

func commitMessage(sess sarama.ConsumerGroupSession, msg *sarama.ConsumerMessage) {
	sess.MarkMessage(msg, "")
	sess.Commit()
}
