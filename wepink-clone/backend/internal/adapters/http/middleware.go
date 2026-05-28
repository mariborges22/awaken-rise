package http

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/pkg/logger"
)

// statusCapture é um wrapper de ResponseWriter que intercepta o status code real da resposta.
type statusCapture struct {
	http.ResponseWriter
	statusCode int
}

func (sc *statusCapture) WriteHeader(code int) {
	sc.statusCode = code
	sc.ResponseWriter.WriteHeader(code)
}

func CorrelationIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		ctx := context.WithValue(r.Context(), logger.CorrelationIDKey, correlationID)
		w.Header().Set("X-Correlation-ID", correlationID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func TenantIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")

		ctx := kernel.WithTenantID(r.Context(), tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// MetricsMiddleware captura latência e status code de cada request e grava
// de forma assíncrona na tabela api_metrics — sem impacto no tempo de resposta.
func MetricsMiddleware(db *sql.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sc := &statusCapture{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(sc, r)

			latencyMs := int(time.Since(start).Milliseconds())
			statusCode := sc.statusCode
			method := r.Method
			path := r.URL.Path
			tenantID, _ := kernel.GetTenantID(r.Context())

			// Fire-and-forget: não bloqueia a resposta ao cliente.
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				_, err := db.ExecContext(ctx,
					`INSERT INTO api_metrics (method, path, status_code, latency_ms, tenant_id)
					 VALUES (?, ?, ?, ?, ?)`,
					method, path, statusCode, latencyMs, tenantID,
				)
				if err != nil {
					slog.Warn("Failed to write api_metric", "error", err)
				}
			}()
		})
	}
}

