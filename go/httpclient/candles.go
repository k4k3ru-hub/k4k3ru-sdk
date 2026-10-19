package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/candles"
)

type jsonRPCCaller interface {
	call(context.Context, jsonrpc.Method, json.RawMessage) (json.RawMessage, error)
}

type CandlesClient struct {
	caller jsonRPCCaller
}

func newCandlesClient(caller jsonRPCCaller) (*CandlesClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create candles client: caller=null")
	}
	return &CandlesClient{caller: caller}, nil
}

// Get retrieves one page of Composite or resolved-market candles, ordered by UTC start.
// Set params.From to result.To to continue until the original params.To is reached.
// With IncludeCurrent, refresh overlapping recent intervals and replace bars by start time.
// Each call can consume quota and credits; this method does not retry or fetch
// subsequent pages automatically.
//
// Version:
//   - 2026-10-19: Accept resolved market metadata and optional developing candles.
//   - 2026-10-11: Added.
func (c *CandlesClient) Get(ctx context.Context, params candles.Params) (*candles.Result, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to get historical candles: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get historical candles: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get historical candles: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get historical candles: %w", err)
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode candle parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubCandlesGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get historical candles: %w", err)
	}
	var result *candles.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode historical candles: %w", err)
	}
	if err := validateCandlePage(result, params); err != nil {
		return nil, fmt.Errorf("failed to get historical candles: %w", err)
	}
	return result, nil
}

func validateCandlePage(result *candles.Result, params candles.Params) error {
	if result == nil || result.Candles == nil {
		return fmt.Errorf("failed to validate candle response: result_or_candles=null")
	}
	step := int64(params.Step() / time.Second)
	limit := params.Limit
	if limit == 0 {
		limit = candles.DefaultLimit
	}
	if result.Asset != params.Asset || result.MarketType != params.MarketType || result.Interval != params.Interval {
		return fmt.Errorf("failed to validate candle response: selectors=invalid")
	}
	if params.Market == nil {
		if result.Definition != "composite" || result.Market != nil {
			return fmt.Errorf("failed to validate candle response: series=invalid")
		}
	} else {
		if result.Definition != "venue" || result.Market == nil || result.Currency != "USD" {
			return fmt.Errorf("failed to validate candle response: series=invalid")
		}
		if err := result.Market.Validate(); err != nil {
			return fmt.Errorf("failed to validate candle response: %w", err)
		}
		if err := result.Symbol.Validate(); err != nil {
			return fmt.Errorf("failed to validate candle response: %w", err)
		}
		target, resolved := params.Market.Normalize(), result.Market.Normalize()
		if target.Venue != resolved.Venue || target.Network != resolved.Network ||
			target.Chain != "" && target.Chain != resolved.Chain ||
			target.PoolID != "" && target.PoolID != resolved.PoolID ||
			target.VenueSymbol != "" && target.VenueSymbol != resolved.VenueSymbol {
			return fmt.Errorf("failed to validate candle response: market=invalid")
		}
	}
	if result.From != params.From || result.To <= result.From || result.To > params.To || result.To%step != 0 {
		return fmt.Errorf("failed to validate candle response: range=out_of_range")
	}
	if len(result.Candles) > limit || (len(result.Candles) < limit && result.To != params.To) {
		return fmt.Errorf("failed to validate candle response: candle_count=out_of_range")
	}
	previous := result.From - step
	for _, candle := range result.Candles {
		if !params.IncludeCurrent && !candle.Closed {
			return fmt.Errorf("failed to validate candle response: closed=invalid")
		}
		if candle.Time < result.From || candle.Time >= result.To || candle.Time%step != 0 || candle.Time <= previous {
			return fmt.Errorf("failed to validate candle response: candle_time=invalid")
		}
		previous = candle.Time
	}
	if len(result.Candles) == limit && previous+step != result.To {
		return fmt.Errorf("failed to validate candle response: continuation=invalid")
	}
	return nil
}
