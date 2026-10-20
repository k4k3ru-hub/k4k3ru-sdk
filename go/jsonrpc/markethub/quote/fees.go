package quote

import (
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
	"github.com/k4k3ru-hub/onchain/go/core"
)

// FeeAccount identifies the trading account, not the API signing wallet.
// Omission uses the venue's standard fee schedule. Account lookup failures must
// not silently substitute standard fees. Initially only Hyperliquid is supported.
type FeeAccount struct {
	Venue   market.Venue `json:"venue"`
	Network core.Network `json:"network"`
	Address string       `json:"address"`
}

// Fees describes known charges already included in the net estimate; gas is excluded.
// Swap includes LP and protocol fees. Taker describes immediate book execution.
// Unknown breakdowns are omitted even when the net estimate can be calculated.
type Fees struct {
	Swap  *Fee `json:"swap,omitempty"`
	Taker *Fee `json:"taker,omitempty"`
}

type Fee struct {
	Token    FeeToken        `json:"token"`
	Quantity market.Quantity `json:"quantity"` // Explicit zero is a known zero charge.
}

// FeeToken identifies an asset in the enclosing market's chain/network or
// venue/network namespace. Symbol alone is not an asset identity.
type FeeToken struct {
	AssetID string `json:"assetId"`
	Symbol  string `json:"symbol"`
}

// Normalize returns canonical fee-account selectors without changing the receiver.
//
// Version:
//   - 2026-10-20: Added.
func (a FeeAccount) Normalize() FeeAccount {
	a.Venue = market.Venue(strings.ToLower(strings.TrimSpace(string(a.Venue))))
	a.Network = core.Network(strings.ToLower(strings.TrimSpace(string(a.Network))))
	a.Address = strings.ToLower(strings.TrimSpace(a.Address))
	return a
}

// Validate validates the supported account scope and its public address.
//
// Version:
//   - 2026-10-20: Added.
func (a FeeAccount) Validate() error {
	a = a.Normalize()
	if a.Venue != market.Hyperliquid {
		return invalid("fee_account_venue", "invalid")
	}
	if err := a.Network.Validate(); err != nil {
		return fmt.Errorf("failed to validate quote fee account: %w", err)
	}
	if len(a.Address) != 42 || !strings.HasPrefix(a.Address, "0x") {
		return invalid("fee_account_address", "invalid")
	}
	if _, err := hex.DecodeString(a.Address[2:]); err != nil {
		return invalid("fee_account_address", "invalid")
	}
	return nil
}

// UnmarshalJSON decodes one complete fee account without accepting unknown fields.
//
// Version:
//   - 2026-10-20: Added.
func (a *FeeAccount) UnmarshalJSON(data []byte) error {
	if a == nil {
		return invalid("fee_account", "null")
	}
	type wire FeeAccount
	var decoded wire
	if err := jsonobject.Decode(data, &decoded, "venue", "network", "address"); err != nil {
		return fmt.Errorf("failed to decode quote fee account: %w", err)
	}
	value := FeeAccount(decoded).Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode quote fee account: %w", err)
	}
	*a = value
	return nil
}

func validateFeeAccounts(p Params) error {
	if len(p.FeeAccounts) > MaximumMarkets {
		return invalid("fee_accounts", "too_long")
	}
	seen := make(map[string]bool, len(p.FeeAccounts))
	for _, a := range p.FeeAccounts {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("failed to validate quote fee accounts: %w", err)
		}
		key := string(a.Venue) + ":" + string(a.Network)
		if seen[key] {
			return invalid("duplicate_fee_account", "invalid")
		}
		seen[key] = true
		found := false
		for _, m := range p.Markets {
			if m.Venue == a.Venue && m.Network == a.Network {
				found = true
				break
			}
		}
		if !found {
			return invalid("fee_account_market", "invalid")
		}
	}
	return nil
}

func validateFees(f *Fees) error {
	if f == nil {
		return nil
	}
	if f.Swap == nil && f.Taker == nil {
		return invalid("fees", "empty")
	}
	for _, fee := range []*Fee{f.Swap, f.Taker} {
		if fee == nil {
			continue
		}
		for _, value := range []string{fee.Token.AssetID, fee.Token.Symbol} {
			if value == "" || value != strings.TrimSpace(value) || strings.IndexFunc(value, unicode.IsControl) >= 0 {
				return invalid("fee_token", state(value == ""))
			}
		}
		if err := fee.Quantity.Validate(); err != nil {
			return fmt.Errorf("failed to validate quote fees: %w", err)
		}
	}
	return nil
}
