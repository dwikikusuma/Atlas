package kafka

import (
	"context"
	"time"

	"github.com/segmentio/kafka-go"
)

type EventConsumer interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type Consumer struct {
	Reader *kafka.Reader
}

func NewConsumer(broker []string, groupID string, topic string) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  broker,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
		MaxWait:  1 * time.Second,
	})

	return &Consumer{
		Reader: reader,
	}
}

func (c *Consumer) Close() error {
	err := c.Reader.Close()
	if err != nil {
		return err
	}
	return nil
}

func (c *Consumer) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	return c.Reader.CommitMessages(ctx, msgs...)
}

func (c *Consumer) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return c.Reader.FetchMessage(ctx)
}
