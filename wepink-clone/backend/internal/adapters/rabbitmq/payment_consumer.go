package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/pkg/logger"
	"github.com/awaken-rise/backend/internal/ports"
	"github.com/awaken-rise/backend/internal/usecase"
)

type PaymentConsumer struct {
	channel        *amqp.Channel
	usecase        *usecase.PaymentUseCase
	eventRepo      ports.ProcessedEventRepository
	txManager      ports.TransactionManager
}

func NewPaymentConsumer(
	channel *amqp.Channel, 
	usecase *usecase.PaymentUseCase,
	eventRepo ports.ProcessedEventRepository,
	txManager ports.TransactionManager,
) *PaymentConsumer {
	return &PaymentConsumer{
		channel:        channel,
		usecase:        usecase,
		eventRepo:      eventRepo,
		txManager:      txManager,
	}
}

func (c *PaymentConsumer) Consume(ctx context.Context) error {
	msgs, err := c.channel.Consume(
		"payment_queue",
		"",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	go func() {
		for d := range msgs {
			// Extract correlation_id from message
			ctxWithCorrelation := context.WithValue(ctx, logger.CorrelationIDKey, d.CorrelationId)
			
			var event entity.Event
			if err := json.Unmarshal(d.Body, &event); err != nil {
				logger.Error(ctxWithCorrelation, "Error unmarshaling event", "error", err)
				d.Ack(false)
				continue
			}

			// 1. Verificar se o evento já foi processado
			processed, err := c.eventRepo.IsProcessed(ctxWithCorrelation, event.ID)
			if err != nil {
				logger.Error(ctxWithCorrelation, "Error checking event status", "error", err)
				d.Nack(false, true)
				continue
			}
			if processed {
				logger.Info(ctxWithCorrelation, "Event already processed, skipping", "event_id", event.ID)
				d.Ack(false)
				continue
			}

			// 2. Extrair payload
			payloadData, _ := json.Marshal(event.Payload)
			var orderPayload entity.OrderCreatedPayload
			if err := json.Unmarshal(payloadData, &orderPayload); err != nil {
				logger.Error(ctxWithCorrelation, "Error unmarshaling payload", "error", err)
				d.Ack(false)
				continue
			}

			// 3. Processar evento dentro de uma transação
			err = c.txManager.Execute(ctxWithCorrelation, func(txCtx context.Context) error {
				// Processar pagamento (idempotente internamente também)
				input := usecase.ProcessPaymentInput{
					PaymentID:      uuid.New().String(),
					OrderID:        orderPayload.OrderID,
					IdempotencyKey: "event_" + event.ID,
				}

				_, err := c.usecase.ProcessPayment(txCtx, input)
				if err != nil {
					return err
				}

				// 4. Registrar evento como processado
				return c.eventRepo.MarkAsProcessed(txCtx, event.ID, event.Type)
			})

			if err != nil {
				logger.Error(ctxWithCorrelation, "Error processing event", "event_id", event.ID, "error", err)
				d.Nack(false, true)
				continue
			}

			// 5. ACK final
			d.Ack(false)
		}
	}()

	return nil
}
