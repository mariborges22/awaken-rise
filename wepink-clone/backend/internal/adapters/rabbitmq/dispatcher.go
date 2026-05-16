package rabbitmq

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/pkg/logger"
	"github.com/awaken-rise/backend/internal/ports"
)

type DomainEventDispatcher struct {
	publisher ports.EventPublisher
}

func NewDomainEventDispatcher(publisher ports.EventPublisher) *DomainEventDispatcher {
	return &DomainEventDispatcher{publisher: publisher}
}

func (d *DomainEventDispatcher) Dispatch(ctx context.Context, events []kernel.DomainEvent) error {
	correlationID, _ := ctx.Value(logger.CorrelationIDKey).(string)
	if correlationID == "" {
		correlationID = uuid.New().String()
	}

	for _, evt := range events {
		integrationEvent := entity.Event{
			ID:            uuid.New().String(),
			CorrelationID: correlationID,
			Type:          evt.EventName(),
			Timestamp:     evt.OccurredOn(),
			Payload:       evt,
		}

		exchange := "default_exchange"
		if strings.HasPrefix(evt.EventName(), "order.") {
			exchange = "orders_exchange"
		} else if strings.HasPrefix(evt.EventName(), "payment.") {
			exchange = "payments_exchange"
		}

		if err := d.publisher.Publish(ctx, exchange, evt.EventName(), integrationEvent); err != nil {
			logger.Error(ctx, "Failed to dispatch domain event", "event_type", evt.EventName(), "error", err)
			return err // Na versão com Outbox, apenas marcaria como não enviado
		}
	}

	return nil
}
