package workers

import (
	"context"
	"log/slog"
	"time"

	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/ports"
)

type OutboxRelayWorker struct {
	outboxRepo ports.OutboxRepository
	publisher  ports.EventPublisher
	interval   time.Duration
	batchSize  int
}

func NewOutboxRelayWorker(
	outboxRepo ports.OutboxRepository,
	publisher ports.EventPublisher,
	interval time.Duration,
	batchSize int,
) *OutboxRelayWorker {
	return &OutboxRelayWorker{
		outboxRepo: outboxRepo,
		publisher:  publisher,
		interval:   interval,
		batchSize:  batchSize,
	}
}

func (w *OutboxRelayWorker) Start(ctx context.Context) {
	slog.Info("Outbox Relay Worker started", "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Outbox Relay Worker stopping...")
			return
		case <-ticker.C:
			w.processPendingEvents(ctx)
		}
	}
}

func (w *OutboxRelayWorker) processPendingEvents(ctx context.Context) {
	events, err := w.outboxRepo.FindPending(ctx, w.batchSize)
	if err != nil {
		slog.Error("Failed to fetch pending outbox events", "error", err)
		return
	}

	if len(events) == 0 {
		return // No pending events
	}

	for _, evt := range events {
		
		// In a real scenario, determine the exchange based on the event type
		exchange := "default_exchange"
		if len(evt.EventType) > 6 && evt.EventType[:6] == "order." {
			exchange = "orders_exchange"
		} else if len(evt.EventType) > 8 && evt.EventType[:8] == "payment." {
			exchange = "payments_exchange"
		}

		// Reconstruct standard Event format for the publisher
		// Since Payload is interface{}, we wrap it back into entity.Event
		integrationEvent := entity.Event{
			ID:            evt.ID,
			CorrelationID: evt.ID, // We could store correlationID in outbox_events to be accurate
			Type:          evt.EventType,
			Timestamp:     evt.CreatedAt,
			Payload:       evt.Payload,
		}

		err := w.publisher.Publish(ctx, exchange, evt.EventType, integrationEvent)
		if err != nil {
			slog.Error("Failed to publish outbox event", "event_id", evt.ID, "error", err)
			
			// Optional: Mark as failed after X retries
			// _ = w.outboxRepo.MarkAsFailed(ctx, evt.ID, err.Error())
			
			continue // We don't mark it as processed, it will be retried next tick
		}

		// Mark as processed
		if err := w.outboxRepo.MarkAsProcessed(ctx, evt.ID); err != nil {
			slog.Error("Failed to mark outbox event as processed", "event_id", evt.ID, "error", err)
		}
	}
}
