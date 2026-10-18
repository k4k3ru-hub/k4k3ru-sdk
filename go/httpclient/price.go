package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/price"
)

type PriceClient struct{ caller jsonRPCCaller }

func newPriceClient(caller jsonRPCCaller) (*PriceClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create price client: caller=null")
	}
	return &PriceClient{caller: caller}, nil
}

// Get retrieves the latest in-memory Composite price without automatic retries.
// Each request shares the account's method quota and may consume credits.
//
// Version:
//   - 2026-10-18: Added.
func (c *PriceClient) Get(ctx context.Context, params price.Params) (*price.Result, error) {
	if c == nil || c.caller == nil || ctx == nil {
		return nil, fmt.Errorf("failed to get price: dependency=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get price: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get price: %w", err)
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode price parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubPriceGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get price: %w", err)
	}
	var result *price.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode price response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to get price: result=null")
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get price: %w", err)
	}
	if result.Asset != params.Asset || result.MarketType != params.MarketType {
		return nil, fmt.Errorf("failed to get price: selectors=invalid")
	}
	return result, nil
}
