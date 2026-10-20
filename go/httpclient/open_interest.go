package httpclient

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/openinterest"
)

type OpenInterestClient struct{ caller jsonRPCCaller }

func newOpenInterestClient(caller jsonRPCCaller) (*OpenInterestClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create open interest client: caller=null")
	}
	return &OpenInterestClient{caller: caller}, nil
}

// Get retrieves current normalized open interest without retries or caller mutation.
// Response markets must cover the requested scopes and cannot introduce other markets.
//
// Version:
//   - 2026-10-20: Added.
func (c *OpenInterestClient) Get(ctx context.Context, params openinterest.Params) (*openinterest.Result, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to get open interest: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get open interest: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get open interest: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get open interest: %w", err)
	}
	params = params.Normalize()
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode open interest parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubOpenInterestGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get open interest: %w", err)
	}
	var result *openinterest.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode open interest response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to get open interest: result=null")
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get open interest: %w", err)
	}
	if result.Symbol != params.Symbol {
		return nil, fmt.Errorf("failed to get open interest: symbol=invalid")
	}
	matched := make([]bool, len(params.Markets))
	for i, rate := range result.Interests {
		found := false
		for j, target := range params.Markets {
			if quoteMarketMatches(target, rate.Market) {
				found, matched[j] = true, true
			}
		}
		if !found {
			return nil, fmt.Errorf("failed to get open interest: market=invalid interest_index=%d", i)
		}
	}
	for i, found := range matched {
		if !found {
			return nil, fmt.Errorf("failed to get open interest: market=empty market_index=%d", i)
		}
	}
	return result, nil
}
