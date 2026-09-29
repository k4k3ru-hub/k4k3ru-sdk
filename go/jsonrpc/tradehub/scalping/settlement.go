package scalping

import (
	"fmt"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observations "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
	"strings"
)

// ExecutionState describes OMS-derived inventory owned by this setting.
// Revision fences inventory changes, independently of price/candidate revisions.
type ExecutionState struct {
	Revision      string           `json:"revision"`
	Status        string           `json:"status"`
	Owner         string           `json:"owner,omitempty"`
	Quantity      *market.Quantity `json:"quantity,omitempty"`
	AcquiredAt    *int64           `json:"acquiredAt,omitempty"`
	ExecutionID   string           `json:"executionId,omitempty"`
	TransactionID string           `json:"transactionId,omitempty"`
}

type Settlement struct {
	Quantity market.Quantity    `json:"quantity"`
	Markets  []SettlementMarket `json:"markets"`
}

type SettlementMarket struct {
	Price     observations.MarketPrice `json:"price"`
	Trigger   string                   `json:"trigger,omitempty"`
	ReturnBPS *string                  `json:"returnBps,omitempty"`
}

// Validate validates an OMS inventory reference without inferring wallet balances.
//
// Version:
//   - 2026-09-29: Added.
func (s ExecutionState) Validate() error {
	const op = "validate scalping execution state"
	if len(s.Revision) != 64 || strings.Trim(s.Revision, "0123456789abcdef") != "" {
		return v.Invalid(op, "revision", "invalid")
	}
	switch s.Status {
	case "idle", "pending", "holding", "closed", "unavailable":
	default:
		return v.Invalid(op, "status", "invalid")
	}
	if (s.ExecutionID == "") != (s.TransactionID == "") {
		return v.Invalid(op, "execution", "invalid")
	}
	if s.Status != "idle" {
		for field, value := range map[string]string{"owner": s.Owner, "execution_id": s.ExecutionID, "transaction_id": s.TransactionID} {
			if err := v.Text(op, field, value, 128); err != nil {
				return err
			}
		}
	}
	if s.Quantity != nil {
		if err := s.Quantity.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping inventory: %w", err)
		}
		if strings.Trim(s.Quantity.Amount, "0") == "" {
			return v.Invalid(op, "quantity", "out_of_range")
		}
	}
	if s.Status == "holding" && (s.Quantity == nil || s.AcquiredAt == nil) {
		return v.Invalid(op, "inventory", "null")
	}
	if s.AcquiredAt != nil && *s.AcquiredAt <= 0 {
		return v.Invalid(op, "acquired_at", "out_of_range")
	}
	if (s.Status == "idle" || s.Status == "closed") && s.Quantity != nil {
		return v.Invalid(op, "quantity", "invalid")
	}
	return nil
}

func (r Result) validateSettlement(params *Params) error {
	const op = "validate scalping settlement"
	if r.State != nil {
		if err := r.State.Validate(); err != nil {
			return err
		}
	}
	if r.Settlement == nil {
		return nil
	}
	s := r.Settlement
	if r.MarketType != market.MarketTypeSpot || r.State == nil || r.State.Status != "holding" || r.State.Quantity == nil || *r.State.Quantity != s.Quantity || len(r.Markets) != 0 {
		return v.Invalid(op, "state", "invalid")
	}
	if s.Markets == nil || len(s.Markets) > observations.MaximumMarkets {
		return v.Invalid(op, "markets", "invalid")
	}
	seen := map[market.MarketRef]bool{}
	for _, m := range s.Markets {
		ref := m.Price.Market.Normalize()
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping settlement market: %w", err)
		}
		if seen[ref] {
			return v.Invalid(op, "duplicate_market", "invalid")
		}
		seen[ref] = true
		e := MarketEvaluation{Price: m.Price, Status: EvaluationStatusNotMatched}
		if m.Price.Status == observations.PriceStatusUnavailable {
			e.Status = EvaluationStatusUnavailable
			e.Reasons = []string{"price_unavailable"}
		}
		if err := validateEvaluation(e, r.EvaluatedAt, r.MarketType); err != nil {
			return err
		}
		switch m.Trigger {
		case "", "take_profit", "stop_loss", "maximum_holding":
		default:
			return v.Invalid(op, "trigger", "invalid")
		}
		if m.Trigger != "" && (m.Price.Status != observations.PriceStatusVWAP || m.Price.NetReceiveQuantity == nil) {
			return v.Invalid(op, "trigger_quantity", "invalid")
		}
		if m.ReturnBPS != nil {
			if _, err := v.Number(op, "return_bps", *m.ReturnBPS, false, true); err != nil {
				return err
			}
		}
		if params != nil {
			found := false
			if params.ExecutionRule.Close.Spot != nil {
				for _, target := range params.ExecutionRule.Close.Spot.Markets {
					if string(ref.Venue) == target.Venue && string(ref.Network) == target.Network && string(ref.Chain) == target.Chain && (target.PoolID == "" || matchingPool(ref, target.PoolID)) && (target.VenueSymbol == "" || ref.VenueSymbol == target.VenueSymbol) {
						found = true
						break
					}
				}
			}
			if !found {
				return v.Invalid(op, "market", "invalid")
			}
		}
	}
	return nil
}
