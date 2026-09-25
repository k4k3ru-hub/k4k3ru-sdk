package execution

import (
	"math/big"
	"strconv"

	"github.com/k4k3ru-hub/onchain/go/sui"
)

type ExecutionOMS struct {
	OrderID         string        `json:"orderId"`
	OpenExecutionID string        `json:"openExecutionId,omitempty"`
	Fill            *SwapFill     `json:"fill,omitempty"`
	Fee             *ExecutionFee `json:"fee,omitempty"`
	PnL             ExecutionPnL  `json:"pnl"`
}

// All amounts are integer strings in the identified token's smallest units.
type SwapFill struct {
	TokenInAssetID   string `json:"tokenInAssetId"`
	TokenOutAssetID  string `json:"tokenOutAssetId"`
	TokenInDecimals  uint8  `json:"tokenInDecimals"`
	TokenOutDecimals uint8  `json:"tokenOutDecimals"`
	AmountIn         string `json:"amountIn"`
	AmountOut        string `json:"amountOut"`
}

// ExecutionFee retains negative net SUI gas when storage rebates exceed costs.
type ExecutionFee struct {
	AssetID  string `json:"assetId"`
	Decimals uint8  `json:"decimals"`
	Amount   string `json:"amount"`
}

// ExecutionPnL is realized only on a successful explicit full Close; it excludes gas.
type ExecutionPnL struct {
	Status   string `json:"status"` // unavailable or realized
	AssetID  string `json:"assetId,omitempty"`
	Decimals uint8  `json:"decimals,omitempty"`
	Amount   string `json:"amount,omitempty"`
}

func validateSuiSnapshot(s *ExecutionSnapshot) error {
	o := s.Onchain
	if o.Chain != "sui" || o.Network != "testnet" || o.BlockNumber != nil || o.BlockHash != "" {
		return observationInvalid("onchain=invalid")
	}
	if _, err := sui.ParseTransactionDigest(o.TransactionID); err != nil {
		return observationInvalid("transaction_id=invalid")
	}
	m := s.OMS
	if m == nil {
		return observationInvalid("oms=null")
	}
	id, err := strconv.ParseUint(m.OrderID, 10, 64)
	if err != nil || id == 0 || strconv.FormatUint(id, 10) != m.OrderID {
		return observationInvalid("order_id=invalid")
	}
	if m.OpenExecutionID != "" {
		if err := validateObservationID(m.OpenExecutionID, "open_execution_id"); err != nil {
			return err
		}
	}
	if s.Status == ObservationStatusPending {
		if o.Checkpoint != nil || s.Failure != nil || m.Fill != nil || m.Fee != nil {
			return observationInvalid("pending=invalid")
		}
	} else {
		if !s.Status.Terminal() || o.Checkpoint == nil || m.Fee == nil {
			return observationInvalid("checkpoint=invalid")
		}
		asset, err := sui.NormalizeMoveType(m.Fee.AssetID)
		if err != nil || asset != "0x0000000000000000000000000000000000000000000000000000000000000002::sui::SUI" || m.Fee.Decimals != 9 || !atomicInteger(m.Fee.Amount, false) {
			return observationInvalid("fee=invalid")
		}
		if s.Status == ObservationStatusFailed {
			if m.Fill != nil || s.Failure == nil || s.Failure.Code != "transaction_failed" {
				return observationInvalid("failure=invalid")
			}
		} else {
			if s.Failure != nil || m.Fill == nil || !atomicInteger(m.Fill.AmountIn, true) || !atomicInteger(m.Fill.AmountOut, true) {
				return observationInvalid("fill=invalid")
			}
			in, errIn := sui.NormalizeMoveType(m.Fill.TokenInAssetID)
			out, errOut := sui.NormalizeMoveType(m.Fill.TokenOutAssetID)
			if errIn != nil || errOut != nil || in == out {
				return observationInvalid("fill_assets=invalid")
			}
		}
	}
	if m.PnL.Status == "unavailable" {
		if m.PnL.Amount != "" || m.PnL.AssetID != "" || m.PnL.Decimals != 0 {
			return observationInvalid("pnl=invalid")
		}
	} else if m.PnL.Status == "realized" {
		if s.Status != ObservationStatusSuccess || m.OpenExecutionID == "" || m.PnL.AssetID != m.Fill.TokenOutAssetID || m.PnL.Decimals != m.Fill.TokenOutDecimals || !atomicInteger(m.PnL.Amount, false) {
			return observationInvalid("pnl=invalid")
		}
	} else {
		return observationInvalid("pnl_status=invalid")
	}
	return nil
}

func atomicInteger(s string, positive bool) bool {
	if len(s) == 0 || len(s) > 79 {
		return false
	}
	n, ok := new(big.Int).SetString(s, 10)
	return ok && n.String() == s && (!positive || n.Sign() > 0 && n.IsUint64())
}
