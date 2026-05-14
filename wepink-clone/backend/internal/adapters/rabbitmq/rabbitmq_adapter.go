package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/awaken-rise/backend/internal/pkg/logger"
)

type RabbitMQAdapter struct {
	conn    *amqp.Connection
	channel *amqp.Channel
}

func NewRabbitMQAdapter(url string) (*RabbitMQAdapter, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to rabbitmq: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	return &RabbitMQAdapter{
		conn:    conn,
		channel: ch,
	}, nil
}

func (a *RabbitMQAdapter) Publish(ctx context.Context, exchange string, routingKey string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	correlationID, _ := ctx.Value(logger.CorrelationIDKey).(string)

	return a.channel.PublishWithContext(ctx,
		exchange,   // exchange
		routingKey, // routing key
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: correlationID,
			Body:          data,
		})
}

func (r *RabbitMQAdapter) Channel() *amqp.Channel {
	return r.channel
}

func (r *RabbitMQAdapter) DeclareResilientQueue(name string) error {
	// 1. Declarar a Exchange de Dead Letter
	dlxName := name + "_dlx"
	err := r.channel.ExchangeDeclare(
		dlxName,
		"direct",
		true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLX: %w", err)
	}

	// 2. Declarar a Fila de Dead Letter (DLQ) - O "Hospital"
	dlqName := name + "_dlq"
	_, err = r.channel.QueueDeclare(
		dlqName,
		true, false, false, false, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare DLQ: %w", err)
	}

	// Bind DLQ to DLX
	err = r.channel.QueueBind(dlqName, "dead", dlxName, false, nil)
	if err != nil {
		return fmt.Errorf("failed to bind DLQ: %w", err)
	}

	// 3. Declarar a Fila Principal com apontamento para a DLX
	args := amqp.Table{
		"x-dead-letter-exchange":    dlxName,
		"x-dead-letter-routing-key": "dead",
	}

	_, err = r.channel.QueueDeclare(
		name,
		true, false, false, false, args,
	)
	return err
}

func (a *RabbitMQAdapter) Close() {
	a.channel.Close()
	a.conn.Close()
}
