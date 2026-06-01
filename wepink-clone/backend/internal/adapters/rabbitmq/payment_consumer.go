package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/awaken-rise/backend/internal/domain/entity"
	"github.com/awaken-rise/backend/internal/pkg/logger"
	"github.com/awaken-rise/backend/internal/pkg/metrics"
	"github.com/awaken-rise/backend/internal/ports"
	"github.com/awaken-rise/backend/internal/usecase"
)

type PaymentConsumer struct {
	channel   *amqp.Channel
	usecase   *usecase.PaymentUseCase
	eventRepo ports.ProcessedEventRepository
	txManager ports.TransactionManager
}

func NewPaymentConsumer(
	channel *amqp.Channel, 
	usecase *usecase.PaymentUseCase,
	eventRepo ports.ProcessedEventRepository,
	txManager ports.TransactionManager,
) *PaymentConsumer {
	return &PaymentConsumer{
		channel:   channel,
		usecase:   usecase,
		eventRepo: eventRepo,
		txManager: txManager,
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
			c.handleMessage(ctx, d)
		}
	}()

	return nil
}

func (c *PaymentConsumer) handleMessage(ctx context.Context, d amqp.Delivery) {
	// Setup do contexto com Correlation ID para rastreabilidade
	ctxWithCorrelation := context.WithValue(ctx, logger.CorrelationIDKey, d.CorrelationId)
	
	var event entity.Event
	if err := json.Unmarshal(d.Body, &event); err != nil {
		slog.Error("Error unmarshaling event", "error", err)
		d.Ack(false)
		return
	}

	// 1. Extrair payload do evento para ter o contexto (TenantID)
	payloadData, _ := json.Marshal(event.Payload)
	var orderPayload entity.OrderCreatedPayload
	if err := json.Unmarshal(payloadData, &orderPayload); err != nil {
		slog.Error("Error unmarshaling payload", "error", err)
		d.Ack(false)
		return
	}

	// 2. Verificação de Idempotência (Garante que não processamos duplicados)
	processed, err := c.eventRepo.IsProcessed(ctxWithCorrelation, event.ID)
	if err != nil {
		c.handleFailure(ctxWithCorrelation, d, event.ID, orderPayload.TenantID, "idempotency_check", err)
		return
	}
	if processed {
		slog.Info("Event already processed, skipping", "event_id", event.ID)
		d.Ack(false)
		return
	}

	// 3. Processar evento dentro de uma transação ACID
	err = c.txManager.Execute(ctxWithCorrelation, func(txCtx context.Context) error {
		input := usecase.ProcessPaymentInput{
			PaymentID:      uuid.New().String(),
			OrderID:        orderPayload.OrderID,
			IdempotencyKey: "event_" + event.ID,
		}

		payment, _, err := c.usecase.ProcessPayment(txCtx, input)
		if err != nil {
			return err
		}

		// 4. Registrar evento como processado (Parte da transação)
		return c.eventRepo.MarkAsProcessed(txCtx, event.ID, event.Type)
	})

	if err != nil {
		c.handleFailure(ctxWithCorrelation, d, event.ID, orderPayload.TenantID, "processing", err)
		return
	}

	// 5. Sucesso - Enviar ACK para o RabbitMQ
	d.Ack(false)
}

func (c *PaymentConsumer) handleFailure(ctx context.Context, d amqp.Delivery, eventID string, tenantID string, stage string, err error) {
	retryCount := c.getRetryCount(d)
	maxRetries := 5

	if retryCount < maxRetries {
		retryCount++
		metrics.PaymentRetriesTotal.WithLabelValues(tenantID).Inc()
		
		waitTime := c.calculateBackoff(retryCount)
		
		slog.Warn("Processing failed, scheduling retry", 
			"event_id", eventID, 
			"stage", stage,
			"attempt", retryCount, 
			"wait_time", waitTime,
			"error", err,
		)

		// Em uma arquitetura de alta escala, usaríamos uma Retry Queue com TTL no RabbitMQ.
		// Para simplificação robusta aqui, usamos sleep na goroutine do worker.
		time.Sleep(waitTime)
		d.Nack(false, true) // Requeue=true para tentar de novo
	} else {
		slog.Error("Processing failed after max retries, sending to DLQ", 
			"event_id", eventID, 
			"error", err,
		)
		d.Nack(false, false) // Requeue=false envia para a DLQ configurada
	}
}

func (c *PaymentConsumer) getRetryCount(d amqp.Delivery) int {
	if d.Headers == nil {
		return 0
	}
	if xDeath, ok := d.Headers["x-death"].([]interface{}); ok && len(xDeath) > 0 {
		if death, ok := xDeath[0].(amqp.Table); ok {
			if count, ok := death["count"].(int64); ok {
				return int(count)
			}
		}
	}
	return 0
}

func (c *PaymentConsumer) calculateBackoff(retryCount int) time.Duration {
	baseDelay := time.Duration(1<<uint(retryCount)) * time.Second
	jitter := time.Duration(float64(baseDelay) * 0.2 * (2*rand.Float64() - 1))
	return baseDelay + jitter
}
