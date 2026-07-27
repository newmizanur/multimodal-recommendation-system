// Package kafka is this service's "delivery" layer — the equivalent of
// internal/delivery/http in the API services, just consuming events
// instead of serving requests. It knows how to read a Kafka message and
// hand raw bytes to a usecase; it has no embedding or vector-DB logic
// of its own.
package kafka

import (
	"context"

	segmentio "github.com/segmentio/kafka-go"
)

// Consumer wraps a Reader bound to a consumer group. KEDA's `kafka`
// scaler trigger watches this group's lag on the topic to decide worker
// replica count — see infra/helm/worker/templates/scaledobject.yaml.
type Consumer struct {
	reader *segmentio.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: segmentio.NewReader(segmentio.ReaderConfig{
			Brokers: brokers,
			Topic:   topic,
			GroupID: groupID,
		}),
	}
}

// Run calls handle for every message, forever. It commits the offset
// only after handle returns nil.
//
// NOTE: unlike SQS's visibility-timeout redelivery, a Kafka consumer's
// fetch cursor advances regardless of whether you commit — not
// committing here does NOT automatically retry this message. Production
// should route handle failures to a dead-letter topic instead of
// relying on non-commit retry; this scaffold just logs and moves on.
func (c *Consumer) Run(ctx context.Context, handle func(ctx context.Context, body []byte) error, onError func(error)) {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			onError(err)
			continue
		}

		if err := handle(ctx, msg.Value); err != nil {
			onError(err)
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			onError(err)
		}
	}
}
