package logger

import (
	"context"
	"log/slog"
	"os"
)

type contextKey string

const CorrelationIDKey contextKey = "correlation_id"

var DefaultLogger *slog.Logger

func init() {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	})
	DefaultLogger = slog.New(handler)
}

func FromContext(ctx context.Context) *slog.Logger {
	if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return DefaultLogger.With(slog.String("correlation_id", correlationID))
	}
	return DefaultLogger
}

func Info(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Info(msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Error(msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Warn(msg, args...)
}
