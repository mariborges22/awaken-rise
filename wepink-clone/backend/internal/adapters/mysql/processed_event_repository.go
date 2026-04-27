package mysql

import (
	"context"
	"database/sql"
	"fmt"
)

type ProcessedEventRepository struct {
	db *sql.DB
}

func NewProcessedEventRepository(db *sql.DB) *ProcessedEventRepository {
	return &ProcessedEventRepository{db: db}
}

func (r *ProcessedEventRepository) getExecutor(ctx context.Context) interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
} {
	if tx := GetTx(ctx); tx != nil {
		return tx
	}
	return r.db
}

func (r *ProcessedEventRepository) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	exec := r.getExecutor(ctx)
	
	var exists int
	err := exec.QueryRowContext(ctx, "SELECT COUNT(1) FROM processed_events WHERE event_id = ?", eventID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check if event is processed: %w", err)
	}
	
	return exists > 0, nil
}

func (r *ProcessedEventRepository) MarkAsProcessed(ctx context.Context, eventID string, eventType string) error {
	exec := r.getExecutor(ctx)
	
	_, err := exec.ExecContext(ctx, "INSERT INTO processed_events (event_id, event_type) VALUES (?, ?)", eventID, eventType)
	if err != nil {
		return fmt.Errorf("failed to mark event as processed: %w", err)
	}
	
	return nil
}
