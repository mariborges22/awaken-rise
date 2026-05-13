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

func (a *RabbitMQAdapter) Channel() *amqp.Channel {
	return a.channel
}

func (a *RabbitMQAdapter) Close() {
	a.channel.Close()
	a.conn.Close()
}
