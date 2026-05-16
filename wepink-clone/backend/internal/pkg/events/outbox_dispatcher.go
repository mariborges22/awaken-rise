package events

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/ports"
)

type OutboxEventDispatcher struct {
	outboxRepo ports.OutboxRepository
}

func NewOutboxEventDispatcher(outboxRepo ports.OutboxRepository) *OutboxEventDispatcher {
	return &OutboxEventDispatcher{outboxRepo: outboxRepo}
}

func (d *OutboxEventDispatcher) Dispatch(ctx context.Context, events []kernel.DomainEvent) error {
	for _, evt := range events {
		
		// In a real scenario, we might want to extract aggregate type and ID from the event if possible.
		// For now, we use defaults or simple mapping.
		
		outboxEvt := &ports.OutboxEvent{
			ID:            uuid.New().String(),
			AggregateType: "aggregate", // Simplification
			AggregateID:   "unknown",   // Simplification - could use reflection or add methods to DomainEvent
			EventType:     evt.EventName(),
			Payload:       evt,
			Status:        "PENDING",
			CreatedAt:     time.Now(),
		}

		if err := d.outboxRepo.Save(ctx, outboxEvt); err != nil {
			return err
		}
	}
	return nil
}
