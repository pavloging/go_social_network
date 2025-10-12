package repository

import (
	"context"
	"encoding/json"
	"errors"

	"post-service/internal/domain"

	"github.com/IBM/sarama"
)

type KafkaProducer struct {
	Producer sarama.SyncProducer
	Topic    string
}

func NewKafkaProducer(brokers []string, topic string) (*KafkaProducer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &KafkaProducer{
		Producer: producer,
		Topic:    topic,
	}, nil
}

func (k *KafkaProducer) Publish(ctx context.Context, post *domain.Post) error {
	if k == nil || k.Producer == nil {
		return errors.New("kafka producer not initialized")
	}

	msgBytes, err := json.Marshal(post)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: k.Topic,
		Value: sarama.ByteEncoder(msgBytes),
	}

	_, _, err = k.Producer.SendMessage(msg)

	return err
}
