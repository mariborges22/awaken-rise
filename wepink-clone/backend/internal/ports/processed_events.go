package ports

import "context"

type ProcessedEventRepository interface {
	IsProcessed(ctx context.Context, eventID string) (bool, error)
	MarkAsProcessed(ctx context.Context, eventID string, eventType string) error
}
