package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/k4k3ru-hub/k4k3ru-sdk/go/finance/market"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc"
	"github.com/k4k3ru-hub/k4k3ru-sdk/go/jsonrpc/markethub/quote"
	"github.com/k4k3ru-hub/onchain/go/core"
	"github.com/k4k3ru-hub/onchain/go/sui"
)

type QuoteClient struct{ caller jsonRPCCaller }

func newQuoteClient(caller jsonRPCCaller) (*QuoteClient, error) {
	if caller == nil {
		return nil, fmt.Errorf("failed to create quote client: caller=null")
	}
	return &QuoteClient{caller: caller}, nil
}

// Get retrieves estimates for one direction and quantity without retries or caller mutation.
// It verifies response selectors and resolved markets before returning the snapshot.
//
// Version:
//   - 2026-10-20: Added.
func (c *QuoteClient) Get(ctx context.Context, params quote.Params) (*quote.Result, error) {
	if c == nil || c.caller == nil {
		return nil, fmt.Errorf("failed to get quotes: client=null")
	}
	if ctx == nil {
		return nil, fmt.Errorf("failed to get quotes: context=null")
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("failed to get quotes: %w", err)
	}
	if err := params.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get quotes: %w", err)
	}
	params = params.Normalize()
	encoded, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to encode quote parameters: %w", err)
	}
	body, err := c.caller.call(ctx, jsonrpc.MethodMarketHubQuoteGet, encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to get quotes: %w", err)
	}
	var result *quote.Result
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode quote response: %w", err)
	}
	if result == nil {
		return nil, fmt.Errorf("failed to get quotes: result=null")
	}
	if err := result.Validate(); err != nil {
		return nil, fmt.Errorf("failed to get quotes: %w", err)
	}
	if result.MarketType != params.MarketType || result.Symbol != params.Symbol || result.Side != params.Side || !equalQuoteQuantity(result.Quantity, params.Quantity) {
		return nil, fmt.Errorf("failed to get quotes: selectors=invalid")
	}
	matched := make([]bool, len(params.Markets))
	for i, q := range result.Quotes {
		found := false
		for j, target := range params.Markets {
			if quoteMarketMatches(target, q.Market) {
				found, matched[j] = true, true
			}
		}
		if !found {
			return nil, fmt.Errorf("failed to get quotes: market=invalid quote_index=%d", i)
		}
	}
	for i, found := range matched {
		if !found {
			return nil, fmt.Errorf("failed to get quotes: market=empty market_index=%d", i)
		}
	}
	return result, nil
}

func equalQuoteQuantity(a, b market.Quantity) bool {
	// Inputs have already passed Quantity.Validate; compare scaled integers exactly.
	left, ok := new(big.Int).SetString(a.Amount, 10)
	if !ok {
		return false
	}
	right, ok := new(big.Int).SetString(b.Amount, 10)
	if !ok {
		return false
	}
	left.Mul(left, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(b.Decimals)), nil))
	right.Mul(right, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(a.Decimals)), nil))
	return left.Cmp(right) == 0
}

func quoteMarketMatches(target market.MarketTarget, resolved market.MarketRef) bool {
	target, resolved = target.Normalize(), resolved.Normalize()
	if target.Venue != resolved.Venue || target.Network != resolved.Network || target.Chain != "" && target.Chain != resolved.Chain || target.VenueSymbol != "" && target.VenueSymbol != resolved.VenueSymbol {
		return false
	}
	if target.PoolID == "" {
		return true
	}
	if resolved.Chain == core.ChainSui {
		left, err := sui.ParseAddress(target.PoolID)
		if err != nil {
			return false
		}
		right, err := sui.ParseAddress(resolved.PoolID)
		return err == nil && left == right
	}
	return target.PoolID == resolved.PoolID
}
