package usecase

import "context"

type IngestUsecase interface {
	// HandleEvent dispatches a raw Kafka message body to the right
	// action (embed-and-upsert, or hard-delete) based on its event type.
	HandleEvent(ctx context.Context, body []byte) error
}
