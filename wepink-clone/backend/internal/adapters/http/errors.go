package http

import (
	"errors"
	"net/http"

	"github.com/awaken-rise/backend/internal/domain/kernel"
)

// HandleError maps domain errors to standard HTTP responses.
func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	// 1. Identify the error type and map to HTTP status code
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

	// For internal server errors, we might want to hide details in production.
	// if statusCode == http.StatusInternalServerError && isProduction() {
	// 	message = "Internal Server Error"
	// }

	// 2. Respond using the standard error format
	RespondWithError(w, r, statusCode, message)
}
