package ports

import (
	"context"
	"time"
)

type OutboxEvent struct {
	ID            string      `json:"id"`
	AggregateType string      `json:"aggregate_type"`
	AggregateID   string      `json:"aggregate_id"`
	EventType     string      `json:"event_type"`
	Payload       interface{} `json:"payload"`
	Status        string      `json:"status"` // PENDING, PROCESSED, FAILED
	CreatedAt     time.Time   `json:"created_at"`
	ProcessedAt   *time.Time  `json:"processed_at"`
}

type OutboxRepository interface {
	Save(ctx context.Context, event *OutboxEvent) error
	FindPending(ctx context.Context, limit int) ([]*OutboxEvent, error)
	MarkAsProcessed(ctx context.Context, id string) error
	MarkAsFailed(ctx context.Context, id string, errMessage string) error
}
