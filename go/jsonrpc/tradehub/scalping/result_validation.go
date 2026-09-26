package scalping

import (
	"fmt"
	"github.com/k4k3ru-hub/onchain/go/sui"
	"math"
	"math/big"
	"strings"
	"unicode"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	observations "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/scalping"

	rule "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/executionrule"
	v "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/tradehub/internal/validation"
)

// Validate validates descriptive metadata without asserting provider correctness.
//
// Version:
//   - 2026-09-23: Added.
func (a AssetMetadata) Validate() error {
	if err := a.Reference.Validate(); err != nil {
		return fmt.Errorf("failed to validate asset metadata: %w", err)
	}
	return v.Text("validate asset metadata", "symbol", a.Symbol, 64)
}

// UnmarshalJSON decodes asset metadata, requiring an explicit decimal count.
// Zero decimals are valid and distinct from an absent or null field.
//
// Version:
//   - 2026-09-23: Added.
func (a *AssetMetadata) UnmarshalJSON(data []byte) error {
	if a == nil {
		return v.Invalid("decode asset metadata", "destination", "null")
	}
	type wire AssetMetadata
	var decoded wire
	if err := v.Decode(data, &decoded, "reference", "symbol", "decimals"); err != nil {
		return fmt.Errorf("failed to decode asset metadata: %w", err)
	}
	value := AssetMetadata(decoded)
	if err := value.Validate(); err != nil {
		return err
	}
	*a = value
	return nil
}

// Validate validates consolidated metrics and concrete market candidates.
// It does not establish execution asset equivalence, inventory, or fillability.
//
// Version:
//   - 2026-09-26: Separate consolidated observations from per-market candidates.
func (r Result) Validate() error {
	const op = "validate scalping result"
	if err := v.Text(op, "evaluation_id", r.EvaluationID, 128); err != nil {
		return err
	}
	if err := validateMarketType(r.MarketType.Normalize()); err != nil {
		return fmt.Errorf("failed to validate scalping result: %w", err)
	}
	if err := r.Symbol.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping result: %w", err)
	}
	parts := strings.Split(string(r.Symbol), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.EqualFold(parts[0], parts[1]) || strings.ContainsAny(string(r.Symbol), ",;*?[]") || strings.IndexFunc(strings.TrimSpace(string(r.Symbol)), func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) >= 0 {
		return v.Invalid(op, "symbol", "invalid")
	}
	for _, a := range []rule.AssetRef{r.BaseAsset, r.QuoteAsset} {
		if err := a.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping result: %w", err)
		}
	}
	if r.BaseAsset.Normalize() == r.QuoteAsset.Normalize() {
		return v.Invalid(op, "asset_pair", "invalid")
	}
	if r.EvaluatedAt <= 0 {
		return v.Invalid(op, "evaluated_at", "out_of_range")
	}
	if r.Markets == nil || len(r.Markets) > observations.MaximumMarkets {
		return v.Invalid(op, "markets", "invalid")
	}
	if err := validateConsolidatedMetrics(r.Metrics); err != nil {
		return err
	}
	seen := make(map[market.MarketRef]bool)
	ids := make(map[string]bool)
	for _, e := range r.Markets {
		ref := e.Price.Market.Normalize()
		if err := ref.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping result: %w", err)
		}
		if seen[ref] {
			return v.Invalid(op, "duplicate_market", "invalid")
		}
		seen[ref] = true
		if err := validateEvaluation(e, r.EvaluatedAt); err != nil {
			return err
		}
		if e.Status != EvaluationStatusUnavailable && (r.Metrics == nil || r.Metrics.PriceChangeBPS == nil && r.Metrics.QuoteVolume == nil && r.Metrics.TradeCount == nil && r.Metrics.BuyVolumeRatioBPS == nil) {
			return v.Invalid(op, "metrics", "empty")
		}
		if e.Candidate != nil {
			if ids[e.Candidate.CandidateID] {
				return v.Invalid(op, "duplicate_candidate", "invalid")
			}
			ids[e.Candidate.CandidateID] = true
		}
	}
	return nil
}

func validateEvaluation(e MarketEvaluation, at int64) error {
	const op = "validate market evaluation"
	switch e.Price.Status {
	case observations.PriceStatusReference, observations.PriceStatusVWAP, observations.PriceStatusFallbackReference:
		if e.Price.ObservedAt == nil {
			return v.Invalid(op, "observed_at", "null")
		}
		if e.Price.Price == nil {
			return v.Invalid(op, "price", "null")
		}
		n, err := v.Number(op, "price", *e.Price.Price, false, false)
		if err != nil {
			return err
		}
		if n.Sign() <= 0 {
			return v.Invalid(op, "price", "out_of_range")
		}
	case observations.PriceStatusUnavailable:
		if e.Price.Price != nil || e.Price.QuoteQuantity != nil || e.Price.Fees != nil {
			return v.Invalid(op, "unavailable_price", "invalid")
		}
	default:
		return v.Invalid(op, "price_status", "invalid")
	}
	if e.Price.QuoteQuantity != nil {
		if err := e.Price.QuoteQuantity.Validate(); err != nil {
			return fmt.Errorf("failed to validate market evaluation: %w", err)
		}
	}
	if e.Price.Status == observations.PriceStatusVWAP && e.Price.QuoteQuantity == nil {
		return v.Invalid(op, "quote_quantity", "null")
	}
	if e.Price.Status != observations.PriceStatusVWAP && (e.Price.QuoteQuantity != nil || e.Price.Fees != nil) {
		return v.Invalid(op, "reference_quantity", "invalid")
	}
	for _, timestamp := range []*int64{e.Price.ObservedAt, e.Price.LastTradeAt} {
		if timestamp != nil && (*timestamp <= 0 || *timestamp > at) {
			return v.Invalid(op, "timestamps", "out_of_range")
		}
	}
	if e.Price.Fees != nil {
		for _, fee := range []*observations.Fee{e.Price.Fees.Swap, e.Price.Fees.Taker} {
			if fee != nil {
				if err := fee.Quantity.Validate(); err != nil {
					return fmt.Errorf("failed to validate market evaluation: %w", err)
				}
				if err := v.Text(op, "fee_asset_id", fee.Token.AssetID, 512); err != nil {
					return err
				}
				if err := v.Text(op, "fee_symbol", fee.Token.Symbol, 64); err != nil {
					return err
				}
			}
		}
	}
	switch e.Status {
	case EvaluationStatusMatched:
		if e.Candidate == nil || e.Price.Status == observations.PriceStatusUnavailable || e.Price.LastTradeAt == nil || len(e.Reasons) != 0 {
			return v.Invalid(op, "matched", "invalid")
		}
		if err := v.Text(op, "candidate_id", e.Candidate.CandidateID, 128); err != nil {
			return err
		}
		if e.Candidate.Revision == 0 || e.Candidate.ExpiresAt <= at {
			return v.Invalid(op, "candidate", "invalid")
		}
	case EvaluationStatusNotMatched:
		if e.Candidate != nil || e.Price.Status == observations.PriceStatusUnavailable {
			return v.Invalid(op, "candidate", "invalid")
		}
	case EvaluationStatusUnavailable:
		if e.Candidate != nil || len(e.Reasons) == 0 {
			return v.Invalid(op, "unavailable", "invalid")
		}
	default:
		return v.Invalid(op, "status", "invalid")
	}
	for _, reason := range e.Reasons {
		if err := v.Text(op, "reason", reason, 64); err != nil {
			return err
		}
	}
	return nil
}

func validateConsolidatedMetrics(m *observations.Metrics) error {
	if m == nil {
		return nil
	}
	const op = "validate scalping metrics"
	if m.PriceChangeBPS != nil {
		if _, err := v.Number(op, "price_change_bps", *m.PriceChangeBPS, false, true); err != nil {
			return err
		}
	}
	if m.QuoteVolume != nil {
		if err := m.QuoteVolume.Validate(); err != nil {
			return fmt.Errorf("failed to validate scalping metrics: %w", err)
		}
	}
	if m.BuyVolumeRatioBPS != nil {
		n, err := v.Number(op, "buy_volume_ratio_bps", *m.BuyVolumeRatioBPS, false, false)
		if err != nil {
			return err
		}
		if n.Cmp(big.NewRat(10000, 1)) > 0 {
			return v.Invalid(op, "buy_volume_ratio_bps", "out_of_range")
		}
	}
	for name, value := range map[string]*string{"trade_vwap": m.TradeVWAP, "realized_volatility_bps": m.RealizedVolatilityBPS} {
		if value != nil {
			n, err := v.Number(op, name, *value, false, false)
			if err != nil {
				return err
			}
			if name == "trade_vwap" && n.Sign() <= 0 {
				return v.Invalid(op, name, "out_of_range")
			}
		}
	}
	if m.Trend != nil {
		for name, value := range map[string]*string{"price_change_delta_bps": m.Trend.PriceChangeDeltaBPS, "quote_volume_change_bps": m.Trend.QuoteVolumeChangeBPS, "buy_volume_ratio_delta_bps": m.Trend.BuyVolumeRatioDeltaBPS} {
			if value != nil {
				if _, err := v.Number(op, name, *value, false, true); err != nil {
					return err
				}
			}
		}
	}
	if m.Spread != nil {
		switch m.Spread.Status {
		case observations.PriceStatusReference, observations.PriceStatusVWAP, observations.PriceStatusFallbackReference:
			if m.Spread.BPS == nil {
				return v.Invalid(op, "spread_bps", "null")
			}
			if _, err := v.Number(op, "spread_bps", *m.Spread.BPS, false, true); err != nil {
				return err
			}
		case observations.PriceStatusUnavailable:
			if m.Spread.BPS != nil {
				return v.Invalid(op, "spread_bps", "invalid")
			}
		default:
			return v.Invalid(op, "spread_status", "invalid")
		}
	}

	return nil
}

// ValidateFor binds consolidated observations and candidates to the requested scope.
// Snapshot transport-age limits are enforced by the receiver's clock.
//
// Version:
//   - 2026-09-26: Validate expanded targets, scaled metrics and explicit observation quantities.
func (r Result) ValidateFor(params Params) error {
	const op = "match scalping result"
	params = params.Normalize()
	if err := params.Validate(); err != nil {
		return fmt.Errorf("failed to match scalping result: %w", err)
	}
	if err := r.Validate(); err != nil {
		return fmt.Errorf("failed to match scalping result: %w", err)
	}
	if r.MarketType.Normalize() != params.MarketType || market.Symbol(strings.ToUpper(strings.TrimSpace(string(r.Symbol)))) != params.Symbol || r.BaseAsset.Normalize() != params.BaseAsset || r.QuoteAsset.Normalize() != params.QuoteAsset {
		return v.Invalid(op, "request", "invalid")
	}
	for _, e := range r.Markets {
		found := false
		for _, target := range params.Markets {
			m := e.Price.Market.Normalize()
			if m.Venue == target.Venue && m.Network == target.Network && (target.Chain == "" || m.Chain == target.Chain) && (target.PoolID == "" || matchingPool(m, target.PoolID)) && (target.VenueSymbol == "" || m.VenueSymbol == target.VenueSymbol) {
				found = true
				break
			}
		}
		if !found {
			return v.Invalid(op, "market", "invalid")
		}
		if params.BaseQuantity == nil && (e.Price.Status == observations.PriceStatusVWAP || e.Price.Status == observations.PriceStatusFallbackReference) || params.BaseQuantity != nil && e.Price.Status == observations.PriceStatusReference {
			return v.Invalid(op, "price_status", "invalid")
		}
		if e.Status == EvaluationStatusUnavailable {
			continue
		}
		m, c := r.Metrics, params.Conditions
		if m == nil || c.PriceChangeBPS != nil && m.PriceChangeBPS == nil || c.QuoteVolume != nil && m.QuoteVolume == nil || c.TradeCount != nil && m.TradeCount == nil || c.BuyVolumeRatioBPS != nil && m.BuyVolumeRatioBPS == nil {
			return v.Invalid(op, "metrics", "null")
		}
		if e.Price.LastTradeAt == nil || uint64(r.EvaluatedAt-*e.Price.LastTradeAt) > c.MaximumDataAgeMS {
			return v.Invalid(op, "data_age_ms", "out_of_range")
		}
		if e.Candidate != nil {
			limit := expiryAfter(*e.Price.LastTradeAt, c.MaximumDataAgeMS)
			if c.MaximumSnapshotAgeMS != nil {
				limit = min(limit, expiryAfter(r.EvaluatedAt, *c.MaximumSnapshotAgeMS))
			}
			if e.Candidate.ExpiresAt > limit {
				return v.Invalid(op, "expires_at", "out_of_range")
			}
		}
	}
	return nil
}

// UnmarshalJSON decodes a validated complete evaluation snapshot.
//
// Version:
//   - 2026-09-26: Decode consolidated metrics and ranked market observations.
//   - 2026-09-25: Use SDK finance market types and canonical perpetual values.
//   - 2026-09-23: Added.
func (r *Result) UnmarshalJSON(data []byte) error {
	if r == nil {
		return v.Invalid("decode scalping result", "destination", "null")
	}
	type wire Result
	var decoded wire
	if err := v.Decode(data, &decoded, "evaluationId", "marketType", "symbol", "baseAsset", "quoteAsset", "evaluatedAt", "markets"); err != nil {
		return fmt.Errorf("failed to decode scalping result: %w", err)
	}
	value := Result(decoded)
	value.MarketType = value.MarketType.Normalize()
	value.Symbol = market.Symbol(strings.ToUpper(strings.TrimSpace(string(value.Symbol))))
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping result: %w", err)
	}
	*r = value
	return nil
}

func matchingPool(m market.MarketRef, pool string) bool {
	if m.PoolID == pool {
		return true
	}
	if m.Chain != "sui" {
		return false
	}
	left, err := sui.ParseAddress(m.PoolID)
	if err != nil {
		return false
	}
	right, err := sui.ParseAddress(pool)
	return err == nil && left == right
}

func expiryAfter(at int64, age uint64) int64 {
	if age >= uint64(math.MaxInt64-at) {
		return math.MaxInt64
	}
	return at + int64(age) + 1
}
