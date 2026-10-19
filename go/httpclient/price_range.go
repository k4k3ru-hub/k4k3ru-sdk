package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	pricerange "github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price/range"
)

type PriceRangeClient struct{ caller jsonRPCCaller }

func newPriceRangeClient(caller jsonRPCCaller) (*PriceRangeClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create price range client: caller=null")
	}
	return &PriceRangeClient{caller: caller}, nil
}

// Get retrieves selected Composite price ranges in one HTTP request without retries.
// Omitted periods select all six supported periods; the caller's parameters are not mutated.
//
// Version:
//   - 2026-10-19: Added.
func (c *PriceRangeClient) Get(ctx context.Context, params pricerange.Params) (*pricerange.Result, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to get price ranges: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get price ranges: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get price ranges: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get price ranges: %w", err)
	}
	params = params.Normalize()
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode price range parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubPriceRangeGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get price ranges: %w", err)
	}
	var result *pricerange.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode price range response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to get price ranges: result=null")
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get price ranges: %w", err)
	}
	if result.Asset != params.Asset || result.MarketType != params.MarketType {
		return nil, fmt.Errorf("failed to get price ranges: selectors=invalid")
	}
	if len(result.Ranges) != len(params.Periods) {
		return nil, fmt.Errorf("failed to get price ranges: periods=invalid")
	}
	for i, period := range params.Periods {
		if result.Ranges[i].Period != period {
			return nil, fmt.Errorf("failed to get price ranges: period=invalid index=%d", i)
		}
	}
	return result, nil
}
