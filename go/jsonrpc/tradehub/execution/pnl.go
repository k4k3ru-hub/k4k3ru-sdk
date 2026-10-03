package execution

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
)

type PnLStatus string

const (
	PnLStatusPending       PnLStatus = "pending"
	PnLStatusRealized      PnLStatus = "realized"
	PnLStatusNotApplicable PnLStatus = "not_applicable"
	PnLStatusUnavailable   PnLStatus = "unavailable"
)

// ExecutionPnL reports only the settlement attributable to this execution.
type ExecutionPnL struct {
	Status     PnLStatus      `json:"status"`
	Settlement *PnLSettlement `json:"settlement,omitempty"`
}

// PnLSettlement uses atomic asset quantities and decimal reporting-currency amounts.
// Acquisition and disposal fees are included once; gas and unrealized values are excluded.
type PnLSettlement struct {
	AssetID   string          `json:"assetId"`
	Quantity  market.Quantity `json:"quantity"`
	Currency  string          `json:"currency"`
	CostBasis string          `json:"costBasis"`
	Proceeds  string          `json:"proceeds"`
	Amount    string          `json:"amount"`
}

// Complete reports whether transaction observation and PnL evaluation have finished.
// Transaction finality alone remains available through ObservationStatus.Terminal.
//
// Version:
//   - 2026-09-29: Added.
func (s ExecutionSnapshot) Complete() bool {
	return s.Status.Terminal() && (s.OMS == nil || s.OMS.PnL.Status == PnLStatusRealized || s.OMS.PnL.Status == PnLStatusNotApplicable || s.OMS.PnL.Status == PnLStatusUnavailable)
}

// Validate validates one execution's settlement without interpreting unknown values as zero.
//
// Version:
//   - 2026-09-29: Added.
func (p ExecutionPnL) Validate() error {
	switch p.Status {
	case PnLStatusPending, PnLStatusNotApplicable, PnLStatusUnavailable:
		if p.Settlement != nil {
			return observationInvalid("pnl_settlement=invalid")
		}
	case PnLStatusRealized:
		v := p.Settlement
		if v == nil {
			return observationInvalid("pnl_settlement=null")
		}
		if v.AssetID == "" || len(v.AssetID) > 1024 || strings.TrimSpace(v.AssetID) != v.AssetID || v.Currency != "USDC" {
			return observationInvalid("pnl_asset=invalid")
		}
		if err := v.Quantity.Validate(); err != nil {
			return fmt.Errorf("failed to validate execution pnl: %w", err)
		}
		q, ok := new(big.Int).SetString(v.Quantity.Amount, 10)
		if !ok || q.Sign() <= 0 {
			return observationInvalid("pnl_quantity=out_of_range")
		}
		if !pnlDecimal(v.CostBasis, false) || !pnlDecimal(v.Proceeds, false) || !pnlDecimal(v.Amount, true) {
			return observationInvalid("pnl_amount=invalid")
		}
	default:
		return observationInvalid("pnl_status=invalid")
	}
	return nil
}

func pnlDecimal(s string, signed bool) bool {
	if len(s) == 0 || len(s) > 384 {
		return false
	}
	if signed && s[0] == '-' {
		s = s[1:]
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 || len(parts[0]) == 0 || len(parts[0]) > 1 && parts[0][0] == '0' {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return len(parts) == 1 || len(parts[1]) <= 18
}

func validateSnapshotPnL(s *ExecutionSnapshot) error {
	p := s.OMS.PnL
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Status == PnLStatusRealized {
		if s.Status != ObservationStatusSuccess || s.OMS.Fill == nil {
			return observationInvalid("pnl_settlement=invalid")
		}
		f, v := s.OMS.Fill, p.Settlement
		if !(v.AssetID == f.TokenInAssetID && v.Quantity.Decimals == f.TokenInDecimals || v.AssetID == f.TokenOutAssetID && v.Quantity.Decimals == f.TokenOutDecimals) {
			return observationInvalid("pnl_settlement=invalid")
		}
	}
	if s.Status == ObservationStatusPending && p.Status == PnLStatusNotApplicable || s.Status == ObservationStatusFailed && p.Status == PnLStatusPending {
		return observationInvalid("pnl_status=invalid")
	}
	return nil
}
