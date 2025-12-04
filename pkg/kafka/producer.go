package kafka

import (
	"context"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// EventProducer defines the behavior required by the service.
// This interface allows us to mock the producer in tests.
type EventProducer interface {
	Publish(ctx context.Context, topic string, key string, value []byte) error
	Close() error
}

// Producer implements EventProducer using segmentio/kafka-go
type Producer struct {
	writer *kafka.Writer
}

// Ensure Producer implements the interface at compile time
var _ EventProducer = (*Producer)(nil)

func NewProducer(brokers []string) *Producer {
	writer := kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    1024,
		Async:        true,
		BatchTimeout: 10 * time.Millisecond,
	}

	return &Producer{
		writer: &writer,
	}
}

func (p *Producer) Publish(ctx context.Context, topic string, key string, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}

	err := p.writer.WriteMessages(ctx, msg)
	if err != nil {
		log.Printf("❌ Failed to publish message to topic %s: %v", topic, err)
		return err
	}

	return nil
}

func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		log.Printf("❌ Failed to close Kafka producer: %v", err)
		return err
	}
	return nil
}
