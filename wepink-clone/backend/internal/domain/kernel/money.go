package kernel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrNegativeAmount = errors.New("amount cannot be negative")
	ErrCurrencyMismatch = errors.New("currency mismatch")
)

type Money struct {
	amount   int64  // value in cents
	currency string
}

func NewMoney(amount int64, currency string) (Money, error) {
	if amount < 0 {
		return Money{}, ErrNegativeAmount
	}
	if currency == "" {
		currency = "BRL"
	}
	return Money{amount: amount, currency: currency}, nil
}

func NewBRL(amount int64) Money {
	m, _ := NewMoney(amount, "BRL")
	return m
}

func (m Money) Amount() int64 {
	return m.amount
}

func (m Money) Currency() string {
	return m.currency
}

func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	return Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

func (m Money) Subtract(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, ErrCurrencyMismatch
	}
	newAmount := m.amount - other.amount
	if newAmount < 0 {
		return Money{}, ErrNegativeAmount
	}
	return Money{amount: newAmount, currency: m.currency}, nil
}

func (m Money) Multiply(multiplier int64) Money {
	return Money{amount: m.amount * multiplier, currency: m.currency}
}

func (m Money) String() string {
	return fmt.Sprintf("%s %.2f", m.currency, float64(m.amount)/100.0)
}

// MarshalJSON ensures Money is serialized as a float for the frontend
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(float64(m.amount) / 100.0)
}

// UnmarshalJSON allows Money to be created from a float in the JSON
func (m *Money) UnmarshalJSON(data []byte) error {
	var val float64
	if err := json.Unmarshal(data, &val); err != nil {
		return err
	}
	m.amount = int64(val*100 + 0.5)
	m.currency = "BRL"
	return nil
}

// Value implements driver.Valuer interface
func (m Money) Value() (driver.Value, error) {
	// Convertermos de volta para float64 para o banco (DECIMAL 10,2)
	return float64(m.amount) / 100.0, nil
}

// Scan implements sql.Scanner interface
func (m *Money) Scan(value interface{}) error {
	if value == nil {
		m.amount = 0
		m.currency = "BRL"
		return nil
	}

	switch v := value.(type) {
	case int64:
		m.amount = v * 100 // Caso o banco já retorne centavos
	case int:
		m.amount = int64(v) * 100
	case float64:
		// Se o banco é DECIMAL(10,2), ele retorna 10.50
		m.amount = int64(v*100 + 0.5) // +0.5 para arredondamento seguro
	case []byte:
		var val float64
		_, err := fmt.Sscanf(string(v), "%f", &val)
		if err != nil {
			return err
		}
		m.amount = int64(val*100 + 0.5)
	default:
		return fmt.Errorf("cannot scan %T into Money", value)
	}

	m.currency = "BRL"
	return nil
}
