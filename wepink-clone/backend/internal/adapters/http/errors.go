package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/awaken-rise/backend/internal/domain/kernel"
	"github.com/awaken-rise/backend/internal/pkg/logger"
)

// HandleError mapeia erros de domínio para respostas HTTP padronizadas.
// Erros 500 são sempre logados de forma estruturada para triagem em produção.
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	statusCode := http.StatusInternalServerError
	message := err.Error()

	switch {
	case errors.Is(err, kernel.ErrValidation) || errors.Is(err, kernel.ErrInvalidInput):
		statusCode = http.StatusBadRequest
	case errors.Is(err, kernel.ErrUnauthorized):
		statusCode = http.StatusUnauthorized
	case errors.Is(err, kernel.ErrForbidden):
		statusCode = http.StatusForbidden
	case errors.Is(err, kernel.ErrNotFound):
		statusCode = http.StatusNotFound
	case errors.Is(err, kernel.ErrAlreadyExists) || errors.Is(err, kernel.ErrPrecondition):
		statusCode = http.StatusConflict
	}

	// Erros internos (500) são logados de forma estruturada e separada dos logs de acesso.
	// O correlation_id permite rastrear o erro ponta-a-ponta no sistema.
	if statusCode == http.StatusInternalServerError {
		correlationID, _ := r.Context().Value(logger.CorrelationIDKey).(string)
		tenantID, _ := kernel.GetTenantID(r.Context())

		slog.Error("Internal server error",
			"correlation_id", correlationID,
			"tenant_id",      tenantID,
			"method",         r.Method,
			"path",           r.URL.Path,
			"error",          message,
		)

		// Em produção, não expõe o detalhe do erro interno ao cliente.
		message = "internal server error — ref: " + correlationID
	}

	RespondWithError(w, r, statusCode, message)
}

