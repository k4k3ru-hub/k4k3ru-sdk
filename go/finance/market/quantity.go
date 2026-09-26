package market

import (
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/apperror"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
)

type Quantity struct {
	Amount   string `json:"amount"`
	Decimals uint8  `json:"decimals"`
}

// Validate validates a nonnegative integer amount with an explicit decimal scale.
// Operation-specific validation must reject zero when a positive input is required.
//
// Version:
//   - 2026-09-26: Added.
func (q Quantity) Validate() error {
	if q.Amount == "" {
		return fmt.Errorf("failed to validate quantity: %w: amount=empty", apperror.InvalidParameter())
	}
	if len(q.Amount) > 384 {
		return fmt.Errorf("failed to validate quantity: %w: amount=too_long max_length=384", apperror.InvalidParameter())
	}
	for _, digit := range q.Amount {
		if digit < '0' || digit > '9' {
			return fmt.Errorf("failed to validate quantity: %w: amount=invalid", apperror.InvalidParameter())
		}
	}
	return nil
}

// UnmarshalJSON requires both the integer amount and its explicit decimal scale.
// A decoding failure leaves the receiver unchanged.
//
// Version:
//   - 2026-09-26: Added.
func (q *Quantity) UnmarshalJSON(data []byte) error {
	if q == nil {
		return fmt.Errorf("failed to decode quantity: %w: destination=null", apperror.InvalidParameter())
	}
	type wire Quantity
	var value wire
	if err := jsonobject.Decode(data, &value, "amount", "decimals"); err != nil {
		return fmt.Errorf("failed to decode quantity: %w", err)
	}
	if err := Quantity(value).Validate(); err != nil {
		return fmt.Errorf("failed to decode quantity: %w", err)
	}
	*q = Quantity(value)
	return nil
}
