package ports

import (
	"context"

	"github.com/awaken-rise/backend/internal/domain/kernel"
)

type EventDispatcher interface {
	Dispatch(ctx context.Context, events []kernel.DomainEvent) error
}
