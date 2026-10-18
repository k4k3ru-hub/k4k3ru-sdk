package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/change"
)

type PriceChangeClient struct{ caller jsonRPCCaller }

func newPriceChangeClient(caller jsonRPCCaller) (*PriceChangeClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create price change client: caller=null")
	}
	return &PriceChangeClient{caller: caller}, nil
}

// Get retrieves selected Composite price changes in one HTTP request.
// Omitted periods select all six supported periods. It does not mutate the input,
// retry requests or fetch candles. Gateway authentication and credit rules apply.
//
// Version:
//   - 2026-10-18: Added.
func (c *PriceChangeClient) Get(ctx context.Context, params change.Params) (*change.Result, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to get price changes: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get price changes: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get price changes: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get price changes: %w", err)
	}
	params = params.Normalize()
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode price change parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubPriceChangeGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get price changes: %w", err)
	}
	var result *change.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode price change response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to get price changes: result=null")
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get price changes: %w", err)
	}
	if result.Asset != params.Asset || result.MarketType != params.MarketType {
		return nil, fmt.Errorf("failed to get price changes: selectors=invalid")
	}
	if len(result.Changes) != len(params.Periods) {
		return nil, fmt.Errorf("failed to get price changes: periods=invalid")
	}
	requested := make(map[change.Period]bool, len(params.Periods))
	for _, period := range params.Periods {
		requested[period] = true
	}
	for _, value := range result.Changes {
		if !requested[value.Period] {
			return nil, fmt.Errorf("failed to get price changes: period=invalid")
		}
	}
	return result, nil
}
