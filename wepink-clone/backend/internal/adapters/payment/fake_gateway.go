package payment

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/awaken-rise/backend/internal/ports"
)

type FakePaymentGateway struct{}

func NewFakePaymentGateway() *FakePaymentGateway {
	return &FakePaymentGateway{}
}

func (g *FakePaymentGateway) Process(ctx context.Context, req ports.PaymentRequest) (*ports.PaymentGatewayResponse, error) {
	// Simulate external network latency
	select {
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	success := true
	errorMessage := ""
	
	// Simulate some failures based on amount cents for testing
	if req.Amount.Amount()%10 == 9 {
		success = false
		errorMessage = "insufficient funds"
	}

	return &ports.PaymentGatewayResponse{
		Success:       success,
		TransactionID: uuid.New().String(),
		ErrorMessage:  errorMessage,
	}, nil
}

func (g *FakePaymentGateway) GetPaymentStatus(ctx context.Context, transactionID string, tenantToken string) (*ports.PaymentGatewayResponse, error) {
	return &ports.PaymentGatewayResponse{
		Success:       true,
		TransactionID: transactionID,
		Status:        "approved",
	}, nil
}
