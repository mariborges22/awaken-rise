package entity

import "time"

type PlanType string

const (
	PlanStarter    PlanType = "starter"
	PlanPro        PlanType = "pro"
	PlanEnterprise PlanType = "enterprise"
)

type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending"
	VerificationApproved VerificationStatus = "approved"
	VerificationRejected VerificationStatus = "rejected"
)

type Tenant struct {
	ID                 string
	LegalName          string
	CNPJ               string
	MPAccessToken      string
	ContactEmail       string
	ContactPhone       string
	Status             string
	Plan               PlanType
	VerificationStatus VerificationStatus
	MonthlyUsageCount  int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (t *Tenant) CanProcessSale() bool {
	if t.Status != "active" || t.VerificationStatus != VerificationApproved {
		return false
	}

	// Regra de Negócio: Limites de Plano
	switch t.Plan {
	case PlanStarter:
		return t.MonthlyUsageCount < 100 // Exemplo: 100 vendas/mês
	case PlanPro:
		return t.MonthlyUsageCount < 1000 // Exemplo: 1000 vendas/mês
	case PlanEnterprise:
		return true // Ilimitado
	}

	return false
}
