package scalping

import (
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/perpetual"
)

type RunPositionSyncStatus string

const (
	RunPositionSyncing     RunPositionSyncStatus = "syncing"
	RunPositionSynced      RunPositionSyncStatus = "synced"
	RunPositionUnavailable RunPositionSyncStatus = "unavailable"
)

// RunPosition identifies a venue-managed position by market and actual account.
// It is a replaceable read view, not a new OMS position or per-order PnL ledger.
type RunPosition struct {
	Market         market.MarketRef      `json:"market"`
	AccountAddress string                `json:"accountAddress"`
	SyncStatus     RunPositionSyncStatus `json:"syncStatus"`
	// ObservedAt is the venue account-state observation time in Unix milliseconds.
	// It does not advance merely because a new Run snapshot is published.
	ObservedAt *int64 `json:"observedAt,omitempty"`
	// Position reuses the Perpetual API's venue data: signed Base quantity,
	// average entry price, margin settings and venue-reported unrealized PnL.
	// Absence means flat only when SyncStatus is synced. Otherwise any included
	// data is last-known information and cannot authorize a new order or exit.
	Position *perpetual.Position `json:"position,omitempty"`
	Exit     RunEvaluation       `json:"exit"`
	Reasons  []string            `json:"reasons,omitempty"`
}

func validateRunPositions(r RunSnapshot) error {
	const op = "validate scalping run positions"
	type scope struct {
		market  market.MarketRef
		account string
	}
	seen := make(map[scope]bool, len(r.Positions))
	for _, p := range r.Positions {
		if err := p.Market.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping run position: %w", err)
		}
		if p.Market.VenueSymbol == "" || p.Market.PoolID != "" {
			return v.Invalid(op, "position_market", "invalid")
		}
		if err := v.Text(op, "account_address", p.AccountAddress, 256); err != nil {
			return err
		}
		key := scope{p.Market.Normalize(), p.AccountAddress}
		if seen[key] {
			return v.Invalid(op, "position_scope", "invalid")
		}
		seen[key] = true
		if p.ObservedAt != nil && (*p.ObservedAt <= 0 || *p.ObservedAt > r.EvaluatedAt) {
			return v.Invalid(op, "observed_at", "out_of_range")
		}
		switch p.SyncStatus {
		case RunPositionSynced:
			if p.ObservedAt == nil {
				return v.Invalid(op, "observed_at", "null")
			}
			if len(p.Reasons) != 0 {
				return v.Invalid(op, "synced_reasons", "invalid")
			}
		case RunPositionSyncing, RunPositionUnavailable:
			if len(p.Reasons) == 0 {
				return v.Invalid(op, "reasons", "empty")
			}
			if r.Entry.Status != EvaluationStatusUnavailable {
				return v.Invalid(op, "unsynchronized_entry", "invalid")
			}
		default:
			return v.Invalid(op, "sync_status", "invalid")
		}
		for _, reason := range p.Reasons {
			if err := v.Text(op, "reason", reason, 128); err != nil {
				return err
			}
		}
		if p.Position != nil {
			if p.ObservedAt == nil {
				return v.Invalid(op, "observed_at", "null")
			}
			if err := validateRunVenuePosition(*p.Position); err != nil {
				return err
			}
		}
		if err := validateRunEvaluation(p.Exit, r.EvaluatedAt, r.MarketType, true); err != nil {
			return err
		}
		for _, m := range p.Exit.Markets {
			if m.Price.Market.Normalize() != p.Market.Normalize() {
				return v.Invalid(op, "position_exit_market", "invalid")
			}
		}
		if (p.SyncStatus != RunPositionSynced || p.Position == nil) && p.Exit.Status != EvaluationStatusUnavailable {
			return v.Invalid(op, "position_exit", "invalid")
		}
		if p.Exit.Status == EvaluationStatusMatched && r.Entry.Status == EvaluationStatusMatched {
			return v.Invalid(op, "settlement_priority", "invalid")
		}
	}
	return nil
}

func validateRunVenuePosition(p perpetual.Position) error {
	const op = "validate scalping run venue position"
	q, err := v.Number(op, "quantity", p.Quantity, false, true)
	if err != nil {
		return err
	}
	if q.Sign() == 0 {
		return v.Invalid(op, "quantity", "empty")
	}
	for _, field := range []struct {
		name  string
		value *string
	}{{"entry_price", p.EntryPrice}, {"liquidation_price", p.LiquidationPrice}} {
		if field.value != nil {
			value, err := v.Number(op, field.name, *field.value, false, false)
			if err != nil {
				return err
			}
			if value.Sign() <= 0 {
				return v.Invalid(op, field.name, "out_of_range")
			}
		}
	}
	if p.Leverage == 0 {
		return v.Invalid(op, "leverage", "empty")
	}
	if p.MarginMode != string(rule.MarginModeCross) && p.MarginMode != string(rule.MarginModeIsolated) {
		return v.Invalid(op, "margin_mode", "invalid")
	}
	if _, err := v.Number(op, "margin_used", p.MarginUsed, false, false); err != nil {
		return err
	}
	if _, err := v.Number(op, "unrealized_pnl", p.UnrealizedPnL, false, true); err != nil {
		return err
	}
	return nil
}
