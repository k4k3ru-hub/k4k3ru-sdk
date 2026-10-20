package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/markets"
)

type MarketsClient struct{ caller jsonRPCCaller }

func newMarketsClient(caller jsonRPCCaller) (*MarketsClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create markets client: caller=null")
	}
	return &MarketsClient{caller: caller}, nil
}

// List retrieves one market catalog page without retries or automatic pagination.
// It preserves caller parameters and binds the response to their selectors and page.
//
// Version:
//   - 2026-10-20: Added.
func (c *MarketsClient) List(ctx context.Context, params markets.Params) (*markets.Result, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to list markets: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to list markets: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to list markets: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to list markets: %w", err)
	}
	params = params.Normalize()
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode markets parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubMarketsList, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to list markets: %w", err)
	}
	var result *markets.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode markets response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to list markets: result=null")
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("failed to list markets: %w", err)
	}
	if result.Asset != params.Asset || result.MarketType != params.MarketType || result.Page != params.Page || result.Limit != params.Limit {
		return nil, fmt.Errorf("failed to list markets: response_scope=invalid")
	}
	return result, nil
}
