package ports

import "context"

type EventPublisher interface {
	Publish(ctx context.Context, exchange string, routingKey string, body interface{}) error
}
