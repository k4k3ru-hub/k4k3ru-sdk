package scalping

import (
	"fmt"
	"math/big"

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

// Validate validates a complete evaluation snapshot and its candidate invariants.
// It does not establish inventory ownership, fillability, or current freshness.
//
// Version:
//   - 2026-09-25: Use SDK finance market types and canonical perpetual values.
//   - 2026-09-23: Added.
func (r Result) Validate() error {
	const op = "validate scalping result"
	if err := v.Text(op, "evaluation_id", r.EvaluationID, 128); err != nil {
		return err
	}
	if err := validateMarketType(r.MarketType.Normalize()); err != nil {
		return fmt.Errorf("failed to validate scalping result: %w", err)
	}
	if err := r.BaseAsset.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping result: %w: asset=%q", err, "base")
	}
	if err := r.QuoteAsset.Validate(); err != nil {
		return fmt.Errorf("failed to validate scalping result: %w: asset=%q", err, "quote")
	}
	if r.BaseAsset.Reference.Normalize() == r.QuoteAsset.Reference.Normalize() {
		return v.Invalid(op, "asset_pair", "invalid")
	}
	if r.EvaluatedAt <= 0 {
		return v.Invalid(op, "evaluated_at", "out_of_range")
	}
	markets := make([]rule.MarketRef, len(r.Markets))
	ids := make(map[string]struct{})
	for i, evaluation := range r.Markets {
		markets[i] = evaluation.Market
		if err := validateEvaluation(evaluation, r.EvaluatedAt); err != nil {
			return fmt.Errorf("failed to validate scalping result: %w: market_index=%d", err, i)
		}
		if evaluation.Candidate != nil {
			id := evaluation.Candidate.CandidateID
			if _, duplicate := ids[id]; duplicate {
				return v.Invalid(op, "duplicate_candidate", "invalid")
			}
			ids[id] = struct{}{}
		}
	}
	if err := rule.ValidateMarkets(markets); err != nil {
		return fmt.Errorf("failed to validate scalping result: %w", err)
	}
	return nil
}

func validateEvaluation(e MarketEvaluation, evaluatedAt int64) error {
	const op = "validate market evaluation"
	switch e.Status {
	case EvaluationStatusMatched:
		if e.Candidate == nil || e.Metrics == nil {
			return v.Invalid(op, "matched", "invalid")
		}
		if len(e.Reasons) != 0 {
			return v.Invalid(op, "reasons", "invalid")
		}
		if err := v.Text(op, "candidate_id", e.Candidate.CandidateID, 128); err != nil {
			return err
		}
		if e.Candidate.Revision == 0 {
			return v.Invalid(op, "revision", "empty")
		}
		if e.Candidate.ExpiresAt <= evaluatedAt {
			return v.Invalid(op, "expires_at", "out_of_range")
		}
	case EvaluationStatusNotMatched:
		if e.Candidate != nil || e.Metrics == nil {
			return v.Invalid(op, "not_matched", "invalid")
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
	if e.Metrics != nil {
		if err := e.Metrics.Validate(); err != nil {
			return fmt.Errorf("failed to validate market evaluation: %w", err)
		}
		if e.Metrics.WindowEnd > evaluatedAt || e.Metrics.LastObservedAt > evaluatedAt {
			return v.Invalid(op, "timestamps", "out_of_range")
		}
		if e.Status != EvaluationStatusUnavailable && e.Metrics.PriceChangeBPS == nil && e.Metrics.QuoteVolume == nil && e.Metrics.TradeCount == nil && e.Metrics.BuyVolumeRatioBPS == nil {
			return v.Invalid(op, "metrics", "empty")
		}
	}
	return nil
}

// Validate validates timestamps and available metrics without inventing zero values.
//
// Version:
//   - 2026-09-23: Added.
func (m Metrics) Validate() error {
	const op = "validate scalping metrics"
	if m.WindowStart <= 0 || m.WindowEnd <= m.WindowStart || m.LastObservedAt <= 0 {
		return v.Invalid(op, "timestamps", "out_of_range")
	}
	if m.PriceChangeBPS != nil {
		if _, err := v.Number(op, "price_change_bps", *m.PriceChangeBPS, false, true); err != nil {
			return err
		}
	}
	if m.QuoteVolume != nil {
		if _, err := v.Number(op, "quote_volume", *m.QuoteVolume, true, false); err != nil {
			return err
		}
	}
	if m.BuyVolumeRatioBPS != nil {
		number, err := v.Number(op, "buy_volume_ratio_bps", *m.BuyVolumeRatioBPS, false, false)
		if err != nil {
			return err
		}
		if number.Cmp(big.NewRat(10000, 1)) > 0 {
			return v.Invalid(op, "buy_volume_ratio_bps", "out_of_range")
		}
	}
	return nil
}

// ValidateFor checks that a snapshot belongs to the requested markets and assets.
// Evaluated markets must include the requested indicators within the data age bound.
// It does not recompute the server's signal or test the candidate against a clock.
//
// Version:
//   - 2026-09-25: Use SDK finance market types and canonical perpetual values.
//   - 2026-09-23: Added.
func (r Result) ValidateFor(params Params) error {
	const op = "match scalping result"
	params = params.Normalize()
	if err := params.Validate(); err != nil {
		return fmt.Errorf("failed to match scalping result: %w", err)
	}
	if err := r.Validate(); err != nil {
		return fmt.Errorf("failed to match scalping result: %w", err)
	}
	marketType := r.MarketType.Normalize()
	if marketType != params.MarketType || r.BaseAsset.Reference.Normalize() != params.BaseAsset || r.QuoteAsset.Reference.Normalize() != params.QuoteAsset {
		return v.Invalid(op, "request", "invalid")
	}
	if len(r.Markets) != len(params.Markets) {
		return v.Invalid(op, "markets", "invalid")
	}
	expected := make(map[rule.MarketRef]struct{}, len(params.Markets))
	for _, market := range params.Markets {
		expected[market] = struct{}{}
	}
	for _, evaluation := range r.Markets {
		if _, exists := expected[evaluation.Market.Normalize()]; !exists {
			return v.Invalid(op, "market", "invalid")
		}
		if evaluation.Status == EvaluationStatusUnavailable {
			continue
		}
		m, c := evaluation.Metrics, params.Conditions
		if (c.PriceChangeBPS != nil && m.PriceChangeBPS == nil) || (c.QuoteVolume != nil && m.QuoteVolume == nil) || (c.TradeCount != nil && m.TradeCount == nil) || (c.BuyVolumeRatioBPS != nil && m.BuyVolumeRatioBPS == nil) {
			return v.Invalid(op, "metrics", "null")
		}
		if uint64(r.EvaluatedAt-m.LastObservedAt) > c.MaximumDataAgeMS {
			return v.Invalid(op, "data_age_ms", "out_of_range")
		}
	}
	return nil
}

// UnmarshalJSON decodes a validated complete evaluation snapshot.
//
// Version:
//   - 2026-09-25: Use SDK finance market types and canonical perpetual values.
//   - 2026-09-23: Added.
func (r *Result) UnmarshalJSON(data []byte) error {
	if r == nil {
		return v.Invalid("decode scalping result", "destination", "null")
	}
	type wire Result
	var decoded wire
	if err := v.Decode(data, &decoded, "evaluationId", "marketType", "baseAsset", "quoteAsset", "evaluatedAt", "markets"); err != nil {
		return fmt.Errorf("failed to decode scalping result: %w", err)
	}
	value := Result(decoded)
	value.MarketType = value.MarketType.Normalize()
	if err := value.Validate(); err != nil {
		return fmt.Errorf("failed to decode scalping result: %w", err)
	}
	*r = value
	return nil
}
