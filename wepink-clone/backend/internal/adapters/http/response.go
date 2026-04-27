package http

import (
	"encoding/json"
	"net/http"

	"github.com/wepink-clone/backend/internal/pkg/logger"
)

type APIResponse struct {
	Status        string      `json:"status"`
	Data          interface{} `json:"data,omitempty"`
	Error         string      `json:"error,omitempty"`
	CorrelationID string      `json:"correlation_id"`
}

func RespondWithSuccess(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	correlationID, _ := r.Context().Value(logger.CorrelationIDKey).(string)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	json.NewEncoder(w).Encode(APIResponse{
		Status:        "success",
		Data:          data,
		CorrelationID: correlationID,
	})
}

func RespondWithError(w http.ResponseWriter, r *http.Request, code int, message string) {
	correlationID, _ := r.Context().Value(logger.CorrelationIDKey).(string)
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	json.NewEncoder(w).Encode(APIResponse{
		Status:        "error",
		Error:         message,
		CorrelationID: correlationID,
	})
}
