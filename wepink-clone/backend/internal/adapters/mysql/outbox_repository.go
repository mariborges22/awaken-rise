package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/awaken-rise/backend/internal/ports"
)

type OutboxRepository struct {
	db *sql.DB
}

func NewOutboxRepository(db *sql.DB) *OutboxRepository {
	return &OutboxRepository{db: db}
}

func (r *OutboxRepository) getExecutor(ctx context.Context) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
} {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *OutboxRepository) Save(ctx context.Context, event *ports.OutboxEvent) error {
	exec := r.getExecutor(ctx)

	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal outbox event payload: %w", err)
	}

	query := `INSERT INTO outbox_events 
		(id, aggregate_type, aggregate_id, event_type, payload, status, created_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`
	
	_, err = exec.ExecContext(ctx, query, 
		event.ID, event.AggregateType, event.AggregateID, event.EventType, payloadBytes, event.Status, event.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to save outbox event: %w", err)
	}

	return nil
}

func (r *OutboxRepository) FindPending(ctx context.Context, limit int) ([]*ports.OutboxEvent, error) {
	exec := r.getExecutor(ctx)

	query := `SELECT id, aggregate_type, aggregate_id, event_type, payload, status, created_at 
			  FROM outbox_events 
			  WHERE status = 'PENDING' 
			  ORDER BY created_at ASC 
			  LIMIT ?`
	
	rows, err := exec.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending outbox events: %w", err)
	}
	defer rows.Close()

	var events []*ports.OutboxEvent
	for rows.Next() {
		var evt ports.OutboxEvent
		var payloadBytes []byte
		
		err := rows.Scan(
			&evt.ID, &evt.AggregateType, &evt.AggregateID, &evt.EventType, &payloadBytes, &evt.Status, &evt.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox event: %w", err)
		}

		// Deserializar para map[string]interface{} provisoriamente
		var payloadData interface{}
		if err := json.Unmarshal(payloadBytes, &payloadData); err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to unmarshal payload for event %s: %v\n", evt.ID, err)
		}
		evt.Payload = payloadData

		events = append(events, &evt)
	}

	return events, nil
}

func (r *OutboxRepository) MarkAsProcessed(ctx context.Context, id string) error {
	exec := r.getExecutor(ctx)
	
	query := `UPDATE outbox_events SET status = 'PROCESSED', processed_at = ? WHERE id = ?`
	_, err := exec.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to mark outbox event as processed: %w", err)
	}
	return nil
}

func (r *OutboxRepository) MarkAsFailed(ctx context.Context, id string, errMessage string) error {
	exec := r.getExecutor(ctx)
	
	// Poderíamos salvar a mensagem de erro em uma coluna separada se tivéssemos.
	// Por enquanto, apenas mudamos o status para FAILED.
	query := `UPDATE outbox_events SET status = 'FAILED', processed_at = ? WHERE id = ?`
	_, err := exec.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to mark outbox event as failed: %w", err)
	}
	return nil
}
