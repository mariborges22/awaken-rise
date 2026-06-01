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
	Success         bool
	TransactionID   string
	Status          string // 'approved', 'pending', 'rejected'
	PaymentURL      string // Link do boleto (se aplicável)
	PixQRCodeBase64 string // Imagem do QR Code do PIX em Base64
	PixCopyPaste    string // Chave PIX Copia e Cola
	ErrorMessage    string
}

// OAuthTokenResult é o resultado de uma operação de troca ou renovação de token.
type OAuthTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // segundos
	UserID       int64
}

// OAuthProvider define as operações de autorização do Gateway de Pagamento.
// Separado de PaymentGateway para respeitar SRP.
type OAuthProvider interface {
	OAuthAuthorizationURL(state string) string
	ExchangeOAuthCode(ctx context.Context, code string) (*OAuthTokenResult, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (*OAuthTokenResult, error)
}

type PaymentGateway interface {
	Process(ctx context.Context, req PaymentRequest) (*PaymentGatewayResponse, error)
	GetPaymentStatus(ctx context.Context, transactionID string, tenantToken string) (*PaymentGatewayResponse, error)
}

