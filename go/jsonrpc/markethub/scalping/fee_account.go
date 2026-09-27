package scalping

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/internal/jsonobject"
	"github.com/k4k3ru-hub/onchain/go/core"
)

// FeeAccount identifies the actual trading account, not an API signing wallet.
// Omission selects the venue's standard fee schedule without account discounts.
type FeeAccount struct {
	Venue   market.Venue `json:"venue"`
	Network core.Network `json:"network"`
	Address string       `json:"address"`
}

// Normalize returns canonical fee account identity.
//
// Version:
//   - 2026-09-28: Added.
func (a FeeAccount) Normalize() FeeAccount {
	a.Venue = market.Venue(strings.ToLower(strings.TrimSpace(string(a.Venue))))
	a.Network = core.Network(strings.ToLower(strings.TrimSpace(string(a.Network))))
	a.Address = strings.ToLower(strings.TrimSpace(a.Address))
	return a
}

// Validate validates the supported Hyperliquid account scope and public address.
//
// Version:
//   - 2026-09-28: Added.
func (a FeeAccount) Validate() error {
	a = a.Normalize()
	if a.Venue != market.Hyperliquid {
		return invalid("fee_account_venue", "invalid")
	}
	if err := a.Network.Validate(); err != nil {
		return fmt.Errorf("failed to validate fee account: %w", err)
	}
	if len(a.Address) != 42 || !strings.HasPrefix(a.Address, "0x") {
		return invalid("fee_account_address", "invalid")
	}
	if _, err := hex.DecodeString(a.Address[2:]); err != nil {
		return invalid("fee_account_address", "invalid")
	}
	return nil
}

// UnmarshalJSON decodes a complete fee account without accepting unknown fields.
//
// Version:
//   - 2026-09-28: Added.
func (a *FeeAccount) UnmarshalJSON(data []byte) error {
	if a == nil {
		return invalid("fee_account", "null")
	}
	type wire FeeAccount
	var value wire
	if err := jsonobject.Decode(data, &value, "venue", "network", "address"); err != nil {
		return fmt.Errorf("failed to decode fee account: %w", err)
	}
	n := FeeAccount(value).Normalize()
	if err := n.Validate(); err != nil {
		return fmt.Errorf("failed to decode fee account: %w", err)
	}
	*a = n
	return nil
}

func validateFeeAccounts(p Params) error {
	if len(p.FeeAccounts) > MaximumMarkets {
		return invalid("fee_accounts", "too_long")
	}
	seen := make(map[string]bool)
	for _, a := range p.FeeAccounts {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping fee accounts: %w", err)
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
