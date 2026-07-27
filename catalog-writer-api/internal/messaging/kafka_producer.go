package messaging

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
)

type Event struct {
	Type      string `json:"type"` // product.created | product.updated | product.deleted | product.reembed
	TenantID  string `json:"tenant_id"`
	ProductID string `json:"product_id"`
}

// KafkaProducer publishes catalog change events to a topic partitioned
// by product_id — keying on product_id (not tenant_id) keeps all events
// for one product strictly ordered on one partition, which matters here:
// a created->updated->deleted sequence processed out of order would
// leave stale vectors behind.
//
// Any Kafka-API-compatible broker works: self-hosted Kafka (e.g. via the
// Strimzi operator), Redpanda, or a managed offering — nothing here
// assumes a specific one.
type KafkaProducer struct {
	writer *kafka.Writer
}

func NewKafkaProducer(brokers []string, topic string) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.Hash{},
		},
	}
}

func (p *KafkaProducer) Publish(ctx context.Context, ev Event) error {
	body, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(ev.ProductID),
		Value: body,
	})
}

func (p *KafkaProducer) Close() error {
	return p.writer.Close()
}
