package ports

import (
	"context"
	"github.com/awaken-rise/backend/internal/domain/kernel"
)

type PaymentRequest struct {
	Amount        kernel.Money
	Description   string
	PaymentMethod string // 'credit_card', 'pix', 'ticket'
	Token         string // Token do cartão (se credit_card)
	Email         string // Email do comprador (LGPD!)
	TenantToken   string // Access Token do lojista (Multi-tenant)
}

type PaymentGatewayResponse struct {
	Success       bool
	TransactionID string
	Status        string // 'approved', 'pending', 'rejected'
	PaymentURL    string // Para Pix (QR Code) ou Boleto
	ErrorMessage  string
}

type PaymentGateway interface {
	Process(ctx context.Context, req PaymentRequest) (*PaymentGatewayResponse, error)
	GetPaymentStatus(ctx context.Context, transactionID string, tenantToken string) (*PaymentGatewayResponse, error)
}
