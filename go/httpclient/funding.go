package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/funding"
)

type FundingClient struct{ caller jsonRPCCaller }

func newFundingClient(caller jsonRPCCaller) (*FundingClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create funding client: caller=null")
	}
	return &FundingClient{caller: caller}, nil
}

// Get retrieves current estimated funding rates without retries or caller mutation.
// Response markets must cover the requested scopes and cannot introduce other markets.
//
// Version:
//   - 2026-10-20: Added.
func (c *FundingClient) Get(ctx context.Context, params funding.Params) (*funding.Result, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to get funding rates: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get funding rates: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get funding rates: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get funding rates: %w", err)
	}
	params = params.Normalize()
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode funding parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubFundingGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get funding rates: %w", err)
	}
	var result *funding.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode funding response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to get funding rates: result=null")
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get funding rates: %w", err)
	}
	if result.Symbol != params.Symbol {
		return nil, fmt.Errorf("failed to get funding rates: symbol=invalid")
	}
	matched := make([]bool, len(params.Markets))
	for i, rate := range result.Rates {
		found := false
		for j, target := range params.Markets {
			if quoteMarketMatches(target, rate.Market) {
				found, matched[j] = true, true
			}
		}
		if !found {
			return nil, fmt.Errorf("failed to get funding rates: market=invalid rate_index=%d", i)
		}
	}
	for i, found := range matched {
		if !found {
			return nil, fmt.Errorf("failed to get funding rates: market=empty market_index=%d", i)
		}
	}
	return result, nil
}
