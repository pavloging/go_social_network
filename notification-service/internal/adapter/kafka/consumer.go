package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"notification-service/internal/domain"
	"notification-service/internal/usecase"

	"github.com/IBM/sarama"
)

type Consumer struct {
	log *slog.Logger
	uc  *usecase.NotificationUsecase
}

func NewConsumer(log *slog.Logger, uc *usecase.NotificationUsecase) *Consumer {
	return &Consumer{log: log, uc: uc}
}

func (c *Consumer) Start(ctx context.Context, brokers []string, topic string, group string) error {
	config := sarama.NewConfig()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest

	client, err := sarama.NewConsumerGroup(brokers, group, config)
	if err != nil {
		return err
	}

	go func() {
		for {
			if err := client.Consume(ctx, []string{topic}, c); err != nil {
				c.log.Error("error consuming", "err", err)
			}
			if ctx.Err() != nil {
				return
			}
		}
	}()

	return nil
}

// --- интерфейс sarama.ConsumerGroupHandler ---

func (c *Consumer) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (c *Consumer) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (c *Consumer) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var n domain.Notification
		if err := json.Unmarshal(msg.Value, &n); err != nil {
			c.log.Error("failed to unmarshal message", "err", err)
			continue
		}

		if err := c.uc.Process(sess.Context(), n); err != nil {
			c.log.Error("failed to process notification", "err", err)
		}
		sess.MarkMessage(msg, "")
	}
	return nil
}
