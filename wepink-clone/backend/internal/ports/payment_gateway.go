package ports

import (
	"context"
)

type PaymentGatewayResponse struct {
	Success       bool
	TransactionID string
	ErrorMessage  string
}

type PaymentGateway interface {
	Process(ctx context.Context, amount float64) (*PaymentGatewayResponse, error)
}
