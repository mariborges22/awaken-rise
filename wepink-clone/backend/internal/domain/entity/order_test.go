package entity

import (
	"testing"
	"github.com/awaken-rise/backend/internal/domain/kernel"
)

func TestOrder_Confirm(t *testing.T) {
	order := NewOrder("1", "tenant-1", []OrderItem{{ProductID: "p1", Quantity: 1, Price: kernel.NewBRL(10000)}})
	
	if order.Status != OrderPending {
		t.Errorf("expected status PENDING, got %s", order.Status)
	}

	err := order.Confirm()
	if err != nil {
		t.Errorf("unexpected error on confirm: %v", err)
	}

	if order.Status != OrderConfirmed {
		t.Errorf("expected status CONFIRMED, got %s", order.Status)
	}

	// Test idempotency
	err = order.Confirm()
	if err != nil {
		t.Errorf("unexpected error on idempotent confirm: %v", err)
	}
}

func TestOrder_CancelAfterConfirm(t *testing.T) {
	order := NewOrder("1", "tenant-1", []OrderItem{{ProductID: "p1", Quantity: 1, Price: kernel.NewBRL(10000)}})
	_ = order.Confirm()

	err := order.Cancel()
	if err != ErrOrderAlreadyConfirmed {
		t.Errorf("expected ErrOrderAlreadyConfirmed, got %v", err)
	}
}
